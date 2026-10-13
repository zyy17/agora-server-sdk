package agoraservice

import "sync"

type AudioTransCodeInData struct {
	data       []byte
	codecType  int
	sampleRate int
	channels   int
}

// AudioTranscodingWorker decodes raw Opus packets to mono 48 kHz S16 PCM.
// Stop is terminal and also releases a worker that has never been started.
type AudioTranscodingWorker struct {
	isRunning       bool
	stopped         bool
	encodeDataQueue chan *AudioTransCodeInData
	stopChan        chan bool
	conn            *RtcConnection
	mutex           sync.Mutex
	wg              sync.WaitGroup
	queueWG         sync.WaitGroup
	decoder         *OpusDecoder
	out_channels    int
	out_sampleRate  int
	in_channels     int
	pushPCM         func([]byte, int, int, int64) int
}

func NewAudioTranscodingWorker(conn *RtcConnection, in_channels int) *AudioTranscodingWorker {
	if in_channels == 0 {
		in_channels = 1
	}
	d := NewOpusDecoder(in_channels, 48000, 1)
	if d == nil {
		return nil
	}
	return &AudioTranscodingWorker{encodeDataQueue: make(chan *AudioTransCodeInData, 1000), conn: conn, stopChan: make(chan bool), decoder: d, in_channels: in_channels, out_channels: 1, out_sampleRate: 48000, pushPCM: conn.PushAudioPcmData}
}

func (worker *AudioTranscodingWorker) Start() int {
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
	worker.wg.Add(1)
	go worker.run()
	return 0
}

func (worker *AudioTranscodingWorker) run() {
	defer worker.wg.Done()
	channels := worker.in_channels
	bytesPer10ms := worker.out_sampleRate / 100 * worker.out_channels * 2
	var pending []byte
	for {
		select {
		case <-worker.stopChan:
			return
		default:
		}
		select {
		case in := <-worker.encodeDataQueue:
			if in.channels != channels {
				decoder := NewOpusDecoder(in.channels, worker.out_sampleRate, worker.out_channels)
				if decoder == nil {
					continue
				}
				worker.decoder.Release()
				worker.decoder = decoder
				channels = in.channels
			}
			pcm, err := worker.decoder.Decode(in.data)
			if err == nil && len(pcm) > 0 {
				// Opus permits 2.5 ms and 5 ms packets. Agora accepts PCM in
				// multiples of 10 ms, so preserve shorter decoded fragments.
				pending = append(pending, pcm...)
				n := len(pending) / bytesPer10ms * bytesPer10ms
				if n > 0 {
					worker.pushPCM(pending[:n], worker.out_sampleRate, worker.out_channels, 0)
					pending = append(pending[:0], pending[n:]...)
				}
			}
		case <-worker.stopChan:
			return
		}
	}
}

func (worker *AudioTranscodingWorker) PushEncodedData(data []byte, channels int, codecType int) {
	if channels == 0 {
		channels = 1
	}
	if worker == nil || len(data) == 0 || codecType != int(AudioCodecOpus) || channels < 1 || channels > 2 {
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
	in := &AudioTransCodeInData{data: append([]byte(nil), data...), channels: channels, codecType: codecType, sampleRate: 48000}
	select {
	case worker.encodeDataQueue <- in:
	case <-worker.stopChan:
	}
}

func (worker *AudioTranscodingWorker) Stop() {
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
	if worker.decoder != nil {
		worker.decoder.Release()
		worker.decoder = nil
	}
}
