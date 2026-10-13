package agoraservice

// #cgo CFLAGS: -I${SRCDIR}/../headers/include/c/api2 -I${SRCDIR}/../headers/include/c/base
// #include "agora_media_node_factory.h"
import "C"
import (
	"math"
	"sync"
	"unsafe"
)

// AudioFrame describes native PCM samples and optional observer telemetry.
// Buffer uses interleaved channels. Keep input fields and bytes unchanged during
// synchronous native sending; observer-generated buffers are copied into Go memory.
// Do not assume raw VAD processors always copy their input or output.
type AudioFrame struct {
	// Type identifies the sample representation; PCM16 uses AudioFrameTypePCM16.
	Type AudioFrameType
	// SamplesPerChannel counts samples in each channel, not all channels combined.
	SamplesPerChannel int
	// BytesPerSample is the size of one sample in bytes; PCM16 requires 2.
	BytesPerSample int
	// Channels is 1 for mono or 2 for interleaved stereo on the PCM sender.
	Channels int
	// SamplesPerSec is the sampling frequency in Hz.
	SamplesPerSec int
	// Buffer contains the PCM bytes and must match sample count, channel count,
	// and sample width. Its lifetime depends on the API using this frame.
	Buffer []byte
	// RenderTimeMs is the native render timestamp in milliseconds.
	// The PCM sender converts it to the native unsigned timestamp field.
	RenderTimeMs int64
	// PresentTimeMs is forwarded as a signed presentation value without rewriting.
	// Some applications use it as an opaque tag; it is not necessarily Unix time.
	PresentTimeMs int64
	// AvsyncType preserves the native synchronization indicator.
	AvsyncType int
	// FarFieldFlag is observer telemetry from the selected native audio path.
	FarFieldFlag int
	// Rms is native observer RMS telemetry.
	Rms int
	// VoiceProb is native voice telemetry. Its interpretation is version/path-specific;
	// the current AudioVadV2 segmenter treats a value of 1 as active.
	VoiceProb int
	// MusicProb is native music telemetry and is not normalized by this binding.
	MusicProb int
	// Pitch is native pitch telemetry and is not normalized by this binding.
	Pitch int
}

// AudioPcmDataSender owns a native PCM sender created by MediaNodeFactory.
// Release is coordinated with in-flight sends, but callers should serialize
// native operations according to the SDK's threading contract.
// Release it after dependent tracks and before the process-wide service.
type AudioPcmDataSender struct {
	cSender       unsafe.Pointer
	mu            sync.RWMutex
	closed        bool
	audioScenario AudioScenario
}

// AudioVolumeInfo is copied native per-user volume telemetry.
type AudioVolumeInfo struct {
	// UserId identifies the user reported by the native callback.
	UserId string
	// Volume is the native volume level.
	Volume uint32
	// VAD is the native voice-activity indicator.
	VAD uint32
	// VoicePitch is native voice-pitch telemetry; support depends on the audio path.
	VoicePitch float64
}

// NewAudioPcmDataSender allocates a sender owned by the caller.
// The factory and process-wide service must be alive. A nil/invalid factory or
// native allocation failure returns nil. Release the sender when no track uses it.
func (mediaNodeFactory *MediaNodeFactory) NewAudioPcmDataSender() *AudioPcmDataSender {
	if mediaNodeFactory == nil || mediaNodeFactory.cFactory == nil {
		return nil
	}
	sender := C.agora_media_node_factory_create_audio_pcm_data_sender(mediaNodeFactory.cFactory)
	if sender == nil {
		return nil
	}
	return &AudioPcmDataSender{
		cSender:       sender,
		closed:        false,
		audioScenario: AudioScenarioChorus, //default to chorus mode
	}
}

// Release destroys the sender after in-flight sends finish.
// It is idempotent and accepts nil. Dependent tracks must no longer use the sender.
func (sender *AudioPcmDataSender) Release() {
	if sender == nil {
		return
	}
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if sender.closed || sender.cSender == nil {
		return
	}
	sender.closed = true
	if sender.cSender == nil {
		return
	}
	C.agora_audio_pcm_data_sender_destroy(sender.cSender)
	sender.cSender = nil
}

// SendAudioPcmData synchronously submits one nonempty PCM16 buffer without pacing.
// Channels must be 1 or 2; sample count and rate must be positive uint32 values,
// and Buffer length must exactly match SamplesPerChannel*Channels*2.
// The caller must keep frame and its Buffer unchanged during the call.
// The binding returns -1 for nil/closed/empty input and -2 for invalid layout;
// otherwise it returns the native result code. A zero result does not acknowledge
// remote playback. This low-level method does not enforce the high-level wrapper's
// supported-rate or integral-millisecond restrictions.
func (sender *AudioPcmDataSender) SendAudioPcmData(frame *AudioFrame) int {
	if sender == nil || frame == nil {
		return -1
	}
	sender.mu.RLock()
	defer sender.mu.RUnlock()
	if sender.closed || sender.cSender == nil || frame == nil || len(frame.Buffer) == 0 {
		return -1
	}
	if frame.BytesPerSample != 2 || frame.Channels < 1 || frame.Channels > 2 || frame.SamplesPerChannel <= 0 || uint64(frame.SamplesPerChannel) > math.MaxUint32 || frame.SamplesPerSec <= 0 || uint64(frame.SamplesPerSec) > math.MaxUint32 {
		return -2
	}
	bytesPerSample := frame.BytesPerSample * frame.Channels
	if len(frame.Buffer)%bytesPerSample != 0 || frame.SamplesPerChannel != len(frame.Buffer)/bytesPerSample {
		return -2
	}
	cData, pinner := unsafeCBytes(frame.Buffer)
	defer pinner.Unpin()
	return int(C.agora_audio_pcm_data_sender_send(sender.cSender, cData,
		C.uint(frame.RenderTimeMs), C.int64_t(frame.PresentTimeMs),
		C.uint(frame.SamplesPerChannel),
		C.uint(frame.BytesPerSample), C.uint(frame.Channels),
		C.uint(frame.SamplesPerSec)))
}
