package agoraservice

/*
#cgo CFLAGS: -I${SRCDIR}/../headers/include/c/api2 -I${SRCDIR}/../headers/include/c/base
#include <string.h>
#include <stdlib.h>
#include "agora_audio_track.h"
#include "agora_media_node_factory.h"
#include "audio_sink_callback_cgo.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

//export goOnSinkAudioFrame
func goOnSinkAudioFrame(sink unsafe.Pointer, frame unsafe.Pointer) C.int {
	value, ok := audioSinkProcessors.Load(sink)
	if !ok || frame == nil {
		return -1
	}
	processor := value.(*ExternalAudioProcessor)
	if !processor.callbacks.begin() {
		return -1
	}
	defer processor.callbacks.end()
	goFrame := GoSinkAudioFrame((*C.struct__audio_pcm_frame)(frame))
	return C.int(processor.doResultFrame(goFrame))
}

var audioSinkProcessors sync.Map

// ExternalAudioProcessorObserver receives native processed PCM callbacks.
// Callbacks run on the native callback path; return promptly and dispatch lifecycle
// changes elsewhere. Do not release or reconfigure the processor synchronously
// inside a callback.
type ExternalAudioProcessorObserver struct {
	// OnProcessedAudioFrame receives copied processed samples, optional VAD state,
	// and selected speech. It must be non-nil when an observer is supplied.
	// The speech frame may alias the processed frame; copy before independent mutation.
	OnProcessedAudioFrame func(processor *ExternalAudioProcessor, frame *AudioFrame, vadResultStat VadState, vadResultFrame *AudioFrame)
}

// AudioSink is an opaque native sink owned by ExternalAudioProcessor.
// Applications do not construct or release it independently.
type AudioSink struct {
	cSinkCallback *C.audio_sink
	cSink         unsafe.Pointer
}

// ExternalAudioProcessor owns a raw PCM sender, processing track, sink, and
// optional Go-side AudioVadV2 segmenter. It is independent of RTC publication.
// Construct it after native service initialization and release it before the
// service. Serialize initialization, sending, and Release externally.
// Prefer the root package's owned AudioProcessor for application use.
type ExternalAudioProcessor struct {
	callbacks   callbackGate
	pcmSender   *AudioPcmDataSender
	audioTrack  *LocalAudioTrack
	audioSinks  *AudioSink
	initialized bool
	vadInstance *AudioVadV2
	observer    *ExternalAudioProcessorObserver
	// InStreamInMs counts submitted input duration in milliseconds after each
	// sender call, including a call whose native result is nonzero. It is telemetry,
	// not a reliable outstanding-input or successful-processing counter.
	InStreamInMs atomic.Int64
	// ProcessedStreamInMs advances by 10 for each native processed-frame callback.
	// It is callback-duration telemetry, not a remote-playback acknowledgement.
	ProcessedStreamInMs atomic.Int64
}

// NewExternalAudioProcessor allocates native processing resources from the
// process-global media factory. The service must already be initialized.
// Allocation failure returns nil and releases partial resources.
// Call Initialize before use and Release when finished, including after failed initialization.
func NewExternalAudioProcessor() *ExternalAudioProcessor {
	processor := &ExternalAudioProcessor{
		pcmSender:           nil,
		audioTrack:          nil,
		audioSinks:          nil,
		initialized:         false,
		vadInstance:         nil,
		observer:            nil,
		InStreamInMs:        atomic.Int64{},
		ProcessedStreamInMs: atomic.Int64{},
	}

	//and initialize the pcmSender and audioTrack
	factory := agoraService.mediaFactory
	processor.pcmSender = factory.NewAudioPcmDataSender()
	if processor.pcmSender == nil {
		return nil
	}
	processor.audioTrack = NewCustomAudioTrackPcm(processor.pcmSender, AudioScenarioDefault, false)
	if processor.audioTrack == nil {
		processor.Release()
		return nil
	}
	processor.audioSinks = newAudioSink(processor)
	if processor.audioSinks == nil {
		processor.Release()
		return nil
	}
	processor.initialized = false
	return processor
}

// Initialize configures the processing track, sink, APM extensions, optional
// Go-side VAD, and observer. outputSampleRate is in Hz and outputChannels is the
// requested output channel count. The native service's APM mode must support
// external processing, and apmConfig and vadConfig cannot both be nil.
// A non-nil observer must provide OnProcessedAudioFrame.
//
// A nonzero return is a binding or native failure; release the object on failure.
// Requested extension libraries and models must be present in the installed SDK.
// The method can print diagnostic settings/errors to stdout. It is not safe to
// reinitialize a running processor or race with sending or Release.
func (p *ExternalAudioProcessor) Initialize(apmConfig *APMConfig, outputSampleRate int, outputChannels int, vadConfig *AudioVadConfigV2, observer *ExternalAudioProcessorObserver) int {
	var ret int = 0

	//check apm model: must support amp but can only do vad
	if isSupportExternalAudioProcessor(agoraService.apmModel) == false {
		fmt.Printf("[ExternalAudioProcessor] apm model is not supported, error code: %d, model: %d\n", ret, agoraService.apmModel)
		return -3000
	}
	if apmConfig == nil && vadConfig == nil {
		fmt.Printf("[ExternalAudioProcessor] apm config and vad config are both nil, error code: %d\n", ret)
		return -3002
	}
	// check vad config and create vad instance if needed
	if vadConfig != nil {
		p.vadInstance = NewAudioVadV2(vadConfig)
		if p.vadInstance == nil {
			fmt.Printf("[ExternalAudioProcessor] failed to create vad instance, error code: %d\n", ret)
			return -3001
		}
	}

	ret = p.setFilterProperties(apmConfig)
	if ret != 0 {
		fmt.Printf("[ExternalAudioProcessor] failed to set filter properties, error code: %d\n", ret)
		return ret
	}

	//1. add sink and set filter properties
	ret = p.addAudioSink(outputSampleRate, outputChannels)
	if ret != 0 {
		fmt.Printf("[ExternalAudioProcessor] failed to add audio sink, error code: %d\n", ret)
		return ret
	}

	//3. set track properties
	ret = p.audioTrack.SetSendDelayMs(10)
	if ret != 0 {
		fmt.Printf("[ExternalAudioProcessor] failed to set send delay ms, error code: %d\n", ret)
		return ret
	}

	p.audioTrack.SetMaxBufferedAudioFrameNumber(100000) // up to 300 frames, 3000ms

	// 4 enable & publish audio track
	p.audioTrack.SetEnabled(true)

	// 4. register observer
	p.observer = observer

	// 5 check if initialized successfully
	p.initialized = true
	return 0
}

// PushAudioPcmData submits nonempty interleaved PCM16 input in complete 10 ms
// multiples. sampleRate is in Hz, must be 8000–48000 and divisible by 100;
// channels must be 1 or 2. startPtsInMs is forwarded as the signed presentation
// value. Keep data unchanged until the synchronous call returns.
//
// The call does not pace input or wait for output. It returns -2000 for missing
// resources, -2001 for invalid layout/rate/channels, otherwise the native sender
// result. Submission counters advance even for nonzero native results.
// Serialize this call with other raw lifecycle operations.
func (p *ExternalAudioProcessor) PushAudioPcmData(data []byte, sampleRate int, channels int, startPtsInMs int64) int {
	if p == nil || p.audioTrack == nil || p.pcmSender == nil {
		return -2000
	}
	if len(data) == 0 || sampleRate < 8000 || sampleRate > 48000 || sampleRate%100 != 0 || channels < 1 || channels > 2 {
		return -2001
	}
	readLen := len(data)
	bytesPer10ms := (sampleRate / 100) * 2 * channels
	// validity check: only accepts data with lengths that are integer multiples of 10ms​​
	if readLen%bytesPer10ms != 0 {
		fmt.Printf("PushAudioPcmData data length is not integer multiples of 10ms, readLen: %d, bytesPerFrame: %d\n", readLen, bytesPer10ms)
		return -2001
	}
	packnumInMs := (readLen / bytesPer10ms) * 10

	frame := &AudioFrame{
		Buffer:            nil,
		RenderTimeMs:      0,
		PresentTimeMs:     startPtsInMs,
		SamplesPerChannel: sampleRate / 100,
		BytesPerSample:    2,
		Channels:          channels,
		SamplesPerSec:     sampleRate,
		Type:              AudioFrameTypePCM16,
	}

	frame.Buffer = data
	frame.SamplesPerChannel = sampleRate * packnumInMs / 1000

	ret := p.pcmSender.SendAudioPcmData(frame)
	p.InStreamInMs.Add(int64(packnumInMs))

	return ret
}

// Release stops callbacks and releases the processor's sink, track, sender,
// and VAD buffers. It accepts nil and is safe to repeat sequentially.
// Do not invoke it synchronously from a processed-frame callback or race it
// with Initialize or PushAudioPcmData. Pending input need not produce output.
func (p *ExternalAudioProcessor) Release() {
	if p == nil {
		return
	}
	p.callbacks.pause()

	// release track
	// release pcmSender
	// release audioSinks
	if p.audioSinks != nil {
		p.removeAudioSink()
	}
	if p.audioTrack != nil {
		p.audioTrack.SetEnabled(false)
		p.audioTrack.ClearSenderBuffer()
		p.audioTrack.Release()
		p.audioTrack = nil
	}

	if p.audioSinks != nil {
		p.audioSinks.release()
		p.audioSinks = nil
	}

	if p.pcmSender != nil {
		p.pcmSender.Release()
		p.pcmSender = nil
	}

	if p.vadInstance != nil {
		p.vadInstance.Release()
		p.vadInstance = nil
	}

	p.observer = nil
	p.initialized = false

}

//
// private methods
//

func (p *ExternalAudioProcessor) setFilterProperties(apmConfig *APMConfig) int {
	if p.audioTrack == nil || p.audioTrack.cTrack == nil {
		return -2002
	}

	if apmConfig == nil {
		// in this case, no need to set filter properties.
		return 0
	}
	cLocalTrackHandle := p.audioTrack.cTrack

	// Step 1: Enable filter on Track
	ret := enableAudioFilterByTrack(cLocalTrackHandle, "audio_processing_pcm_source", true, true)
	if ret != 0 {
		fmt.Printf("[LocalAudioProcessor] failed to enable filter on Track, error code: %d\n", ret)
		return -2003
	}

	// Step 2: Load AINS resources (if needed)
	ret = setFilterPropertyByTrack(cLocalTrackHandle, "audio_processing_pcm_source", "apm_load_resource", "ains", true)
	if ret != 0 {
		fmt.Printf("[LocalAudioProcessor] failed to load AINS model resources, error code: %d\n", ret)
		return -2004
	}

	// Step 3: Build and apply configuration
	apmConfigJSON := apmConfig.toJson()
	ret = setFilterPropertyByTrack(cLocalTrackHandle, "audio_processing_pcm_source", "apm_config", apmConfigJSON, true)
	if ret != 0 {
		fmt.Printf("[LocalAudioProcessor] failed to configure audio processing parameters, error code: %d\n", ret)
		return -2005
	}
	fmt.Printf("[LocalAudioProcessor] apmConfigJSON: %s\n", apmConfigJSON)

	// Step 4: Enable dump (if debugging is needed)
	if apmConfig.EnableDump {
		ret = setFilterPropertyByTrack(cLocalTrackHandle, "audio_processing_pcm_source", "apm_dump", "true", true)
		if ret != 0 {
			fmt.Printf("[LocalAudioProcessor] failed to enable dump, error code: %d (non-critical)\n", ret)
			return -2006
		}
	}

	return ret
}

func newAudioSink(processor *ExternalAudioProcessor) *AudioSink {
	sink := &AudioSink{
		cSinkCallback: nil,
		cSink:         nil,
	}
	// allocate audio_sink	 structure
	csink_callback := (*C.audio_sink)(C.malloc(C.sizeof_audio_sink))
	if csink_callback == nil {
		return nil
	}
	C.memset(unsafe.Pointer(csink_callback), 0, C.sizeof_audio_sink)

	// set callback function pointer - using C wrapper function
	csink_callback.on_audio_frame = (*[0]byte)(C.cgo_onSinkAudioFrameCallback)
	csink_callback.user_data = unsafe.Pointer(csink_callback)
	audioSinkProcessors.Store(unsafe.Pointer(csink_callback), processor)

	// create audio sink
	sink.cSink = C.agora_audio_sink_create(csink_callback)
	sink.cSinkCallback = csink_callback
	if sink.cSink == nil {
		sink.release()
		return nil
	}
	return sink
}

func (s *AudioSink) release() {
	if s == nil {
		return
	}
	if s.cSink != nil {
		C.agora_audio_sink_destroy(s.cSink)
		s.cSink = nil
	}
	if s.cSinkCallback != nil {
		audioSinkProcessors.Delete(unsafe.Pointer(s.cSinkCallback))
		C.free(unsafe.Pointer(s.cSinkCallback))
		s.cSinkCallback = nil
	}
}

func (p *ExternalAudioProcessor) addAudioSink(sampleRate int, channels int) int {
	if p.audioTrack == nil || p.audioTrack.cTrack == nil || p.audioSinks == nil || p.audioSinks.cSink == nil {
		return -1000
	}

	wants := &C.struct__audio_sink_wants{
		samples_per_sec: C.int(sampleRate),
		channels:        C.uint32_t(channels),
	}

	return int(C.agora_audio_track_add_audio_sink(p.audioTrack.cTrack, p.audioSinks.cSink, wants))
}

// RemoveAudioSink 从 audio track 移除音频接收器
func (p *ExternalAudioProcessor) removeAudioSink() int {
	if p.audioTrack == nil || p.audioTrack.cTrack == nil || p.audioSinks == nil {
		return -1001
	}
	return int(C.agora_audio_track_remove_audio_sink(p.audioTrack.cTrack, p.audioSinks.cSink))
}

func (p *ExternalAudioProcessor) doResultFrame(frame *AudioFrame) int {
	// TODO: implement the logic to process the audio frame
	//fmt.Printf("[ExternalAudioProcessor] doResultFrame, voice prob: %d, rms: %d, pitch: %d\n", frame.VoiceProb, frame.Rms, frame.Pitch)
	var vadResultState VadState = VadStateInvalid
	var vadResultFrame *AudioFrame = nil
	if p.vadInstance != nil {
		vadResultFrame, vadResultState = p.vadInstance.Process(frame)
	}
	p.ProcessedStreamInMs.Add(int64(10))
	if p.observer != nil {
		p.observer.OnProcessedAudioFrame(p, frame, vadResultState, vadResultFrame)
	}
	return 0
}
