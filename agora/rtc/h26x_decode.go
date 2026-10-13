//go:build avcodec

package agoraservice

// #cgo CFLAGS: -DUSE_AVCODEC
// #cgo pkg-config: libavcodec libavutil
// #include "h26x_decode_frame.h"
import "C"
import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// VideoDecoder decodes complete Annex-B H.264 or H.265 access units to 8-bit I420.
// Calls to DecodeAndSave and Close may be made concurrently.
type VideoDecoder struct {
	decoder    *C.H264Decoder
	codecType  int
	frameCount int
	mutex      sync.Mutex
}

func NewVideoDecoder() (*VideoDecoder, error) {
	var code C.int
	d := C.h26x_decoder_init(C.int(VideoCodecTypeH264), &code)
	if d == nil {
		return nil, fmt.Errorf("initialize H.264 decoder: %d", code)
	}
	return &VideoDecoder{decoder: d, codecType: int(VideoCodecTypeH264)}, nil
}

// DecodeAndSave returns the latest decoded frame, 1 if the decoder needs another
// access unit, or a negative FFmpeg error. Despite its historical name, it never
// writes files. Dimensions of zero preserve the source size; nonzero dimensions
// must be positive and even. Unsupported pixel formats are rejected.
func (vd *VideoDecoder) DecodeAndSave(data []byte, width, height, codecType int) (*ExternalVideoFrame, int) {
	if vd == nil || len(data) == 0 || len(data) > 1<<31-1 || width < 0 || height < 0 || width > 1<<31-1 || height > 1<<31-1 || width%2 != 0 || height%2 != 0 {
		return nil, -1
	}
	if codecType != int(VideoCodecTypeH264) && codecType != int(VideoCodecTypeH265) {
		return nil, -1
	}
	vd.mutex.Lock()
	defer vd.mutex.Unlock()
	if vd.decoder == nil {
		return nil, -1
	}
	if vd.codecType != codecType {
		var code C.int
		d := C.h26x_decoder_init(C.int(codecType), &code)
		if d == nil {
			return nil, int(code)
		}
		C.h264_decoder_free(vd.decoder)
		vd.decoder, vd.codecType = d, codecType
	}
	var yuv *C.YUVFrame
	ret := int(C.h264_decode_frame(vd.decoder, (*C.uint8_t)(unsafe.Pointer(&data[0])), C.int(len(data)), C.int(width), C.int(height), &yuv))
	if yuv != nil {
		defer C.yuv_frame_free(yuv)
	}
	if ret != 0 || yuv == nil {
		return nil, ret
	}
	buf := make([]byte, int(yuv.total_size))
	ySize, uvSize := int(yuv.y_size), int(yuv.uv_size)
	copy(buf, unsafe.Slice((*byte)(unsafe.Pointer(yuv.y_data)), ySize))
	copy(buf[ySize:], unsafe.Slice((*byte)(unsafe.Pointer(yuv.u_data)), uvSize))
	copy(buf[ySize+uvSize:], unsafe.Slice((*byte)(unsafe.Pointer(yuv.v_data)), uvSize))
	vd.frameCount++
	return &ExternalVideoFrame{Type: VideoBufferRawData, Format: VideoPixelI420, Buffer: buf, Stride: int(yuv.width), Height: int(yuv.height)}, 0
}

func (vd *VideoDecoder) Close() {
	if vd == nil {
		return
	}
	vd.mutex.Lock()
	defer vd.mutex.Unlock()
	if vd.decoder != nil {
		C.h264_decoder_free(vd.decoder)
		vd.decoder = nil
	}
}

type TransCodeInData struct {
	data      []byte
	frameInfo *EncodedVideoFrameInfo
	timestamp int64
}
type TransCodeStatus struct {
	orgEncodedDataInFrameCount     int
	transcodedYuvDataOutFrameCount int
	pushToRtcFrameCount            int
	pushToRtcFailedCount           int
	lastCheckCount                 int
}

func (status *TransCodeStatus) Reset() { *status = TransCodeStatus{} }

// TranscodingWorker owns a bounded queue and a decoder. Stop is terminal and
// idempotent, including when called before Start. Inputs are copied on enqueue.
type TranscodingWorker struct {
	isRunning        bool
	stopped          bool
	encodeDataQueue  chan *TransCodeInData
	decodeVideoFrame *ExternalVideoFrame
	timer            *time.Ticker
	interval         time.Duration
	conn             *RtcConnection
	decoder          *VideoDecoder
	stopChan         chan bool
	mutex            sync.Mutex
	wg               sync.WaitGroup
	queueWG          sync.WaitGroup
	status           *TransCodeStatus
}

func NewTranscodingWorker(conn *RtcConnection, fps int) *TranscodingWorker {
	if fps <= 0 {
		fps = 15
	}
	if fps > 30 {
		fps = 30
	}
	d, err := NewVideoDecoder()
	if err != nil {
		return nil
	}
	return &TranscodingWorker{encodeDataQueue: make(chan *TransCodeInData, 1000), interval: time.Second / time.Duration(fps), conn: conn, decoder: d, stopChan: make(chan bool), status: &TransCodeStatus{}}
}

func (worker *TranscodingWorker) Start() int {
	if worker == nil {
		return -10001
	}
	worker.mutex.Lock()
	defer worker.mutex.Unlock()
	if worker.isRunning || worker.stopped {
		return -1000
	}
	if worker.decoder == nil {
		return -10001
	}
	worker.isRunning = true
	worker.timer = time.NewTicker(worker.interval)
	worker.wg.Add(1)
	go worker.run()
	return 0
}

func (worker *TranscodingWorker) run() {
	defer worker.wg.Done()
	for {
		// Prefer shutdown even if producers have left a full queue.
		select {
		case <-worker.stopChan:
			return
		default:
		}
		select {
		case in := <-worker.encodeDataQueue:
			frame, ret := worker.decoder.DecodeAndSave(in.data, in.frameInfo.Width, in.frameInfo.Height, int(in.frameInfo.CodecType))
			worker.status.orgEncodedDataInFrameCount++
			if ret == 0 && frame != nil {
				worker.decodeVideoFrame = frame
				worker.status.transcodedYuvDataOutFrameCount++
			}
		case <-worker.timer.C:
			if worker.decodeVideoFrame != nil {
				if worker.conn.handleDecodedVideoFrameForTranscode(worker.decodeVideoFrame) == 0 {
					worker.status.pushToRtcFrameCount++
				} else {
					worker.status.pushToRtcFailedCount++
				}
			}
		case <-worker.stopChan:
			return
		}
	}
}

func (worker *TranscodingWorker) PushEncodedData(data []byte, info *EncodedVideoFrameInfo) {
	if worker == nil || len(data) == 0 || info == nil {
		return
	}
	worker.mutex.Lock()
	if !worker.isRunning || worker.stopped {
		worker.mutex.Unlock()
		return
	}
	worker.queueWG.Add(1)
	worker.mutex.Unlock()
	defer worker.queueWG.Done()
	frameInfo := *info
	frameInfo.SeiData = append([]byte(nil), info.SeiData...)
	in := &TransCodeInData{data: append([]byte(nil), data...), frameInfo: &frameInfo, timestamp: time.Now().UnixMilli()}
	select {
	case worker.encodeDataQueue <- in:
	case <-worker.stopChan:
	}
}

func (worker *TranscodingWorker) Stop() {
	if worker == nil {
		return
	}
	worker.mutex.Lock()
	if !worker.stopped {
		worker.stopped = true
		worker.isRunning = false
		close(worker.stopChan)
	}
	worker.mutex.Unlock()
	worker.wg.Wait()
	worker.queueWG.Wait()
	worker.mutex.Lock()
	defer worker.mutex.Unlock()
	for len(worker.encodeDataQueue) > 0 {
		<-worker.encodeDataQueue
	}
	if worker.timer != nil {
		worker.timer.Stop()
		worker.timer = nil
	}
	if worker.decoder != nil {
		worker.decoder.Close()
		worker.decoder = nil
	}
	worker.decodeVideoFrame = nil
}
