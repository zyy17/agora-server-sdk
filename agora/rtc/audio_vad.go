package agoraservice

/*
#cgo CFLAGS: -I${SRCDIR}/../headers/include/c/api2 -I${SRCDIR}/../headers/include/c/base
#cgo CXXFLAGS: -I${SRCDIR}/../headers/include/c/api2 -std=c++11
#cgo darwin LDFLAGS: -luap_aed -lc++
#cgo linux LDFLAGS: -lagora_uap_aed -lstdc++
#include <stdlib.h>
#include "audio_vad_bridge.h"
*/
import "C"

import (
	"math"
	"sync"
	"unsafe"
)

// AudioVadConfig configures the standalone native AudioVad detector.
// Counts refer to 10 ms, 16 kHz mono frames and must be 0–10000.
// NewAudioVad(nil) uses the defaults documented on each field; a non-nil
// configuration replaces all defaults, including zero values.
type AudioVadConfig struct {
	// StartRecognizeCount is the native speech-start frame count; nil-config default: 30.
	StartRecognizeCount int
	// StopRecognizeCount is the native speech-stop frame count; nil-config default: 48.
	StopRecognizeCount int
	// PreStartRecognizeCount is the buffered pre-roll frame count; nil-config default: 16.
	PreStartRecognizeCount int
	// ActivePercent is the native active-frame ratio in 0–1; nil-config default: 0.8.
	ActivePercent float32
	// InactivePercent is the native inactive-frame ratio in 0–1; nil-config default: 0.2.
	InactivePercent float32
	// VoiceProb is the native voice-probability threshold in 0–1; nil-config default: 0.7.
	VoiceProb float32
	// RmsThr is the native RMS threshold in -100–0 dB; nil-config default: -40.
	RmsThr float32
	// JointThr is the native joint-probability threshold in 0–1; nil-config default: 0.
	JointThr float32
	// Aggressive is the native aggressiveness setting in 0–3; nil-config default: 2.
	Aggressive float32
}

// AudioVad owns one standalone native VAD detector. It is distinct from AudioVadV2.
// Construct it with NewAudioVad and call Release when finished. ProcessPcmFrame
// and Release are serialized; use separate detectors for independent speakers.
// A detector retired after a native exception cannot be reused.
type AudioVad struct {
	mu        sync.Mutex
	vadCfg    *AudioVadConfig
	cVad      unsafe.Pointer
	lastOutTs int64
}

// NewAudioVad copies cfg and creates a standalone native detector.
// Nil uses the defaults documented on AudioVadConfig. Invalid values or native
// allocation failure return nil. The native VAD extension must be installed;
// this constructor does not create an RTC channel or Service.
func NewAudioVad(cfg *AudioVadConfig) *AudioVad {
	settings := AudioVadConfig{StartRecognizeCount: 30, StopRecognizeCount: 48, PreStartRecognizeCount: 16, ActivePercent: 0.8, InactivePercent: 0.2, RmsThr: -40, JointThr: 0, Aggressive: 2, VoiceProb: 0.7}
	if cfg != nil {
		settings = *cfg
	}
	if !validStandaloneVADConfig(settings) {
		return nil
	}
	vad := &AudioVad{vadCfg: &settings}
	native := C.struct_Vad_Config_{}
	native.fftSz = 1024
	native.anaWindowSz = 768
	native.hopSz = 160
	native.voiceProbThr = C.float(settings.VoiceProb)
	native.rmsThr = C.float(settings.RmsThr)
	native.jointThr = C.float(settings.JointThr)
	native.aggressive = C.float(settings.Aggressive)
	native.startRecognizeCount = C.int(settings.StartRecognizeCount)
	native.stopRecognizeCount = C.int(settings.StopRecognizeCount)
	native.preStartRecognizeCount = C.int(settings.PreStartRecognizeCount)
	native.activePercent = C.float(settings.ActivePercent)
	native.inactivePercent = C.float(settings.InactivePercent)
	if int(C.agora_go_vad_create(&vad.cVad, &native)) != 0 || vad.cVad == nil {
		if vad.cVad != nil {
			C.agora_go_vad_destroy(&vad.cVad)
		}
		return nil
	}
	return vad
}

func validStandaloneVADConfig(c AudioVadConfig) bool {
	for _, count := range []int{c.StartRecognizeCount, c.StopRecognizeCount, c.PreStartRecognizeCount} {
		if count < 0 || count > 10000 {
			return false
		}
	}
	for _, value := range []float32{c.ActivePercent, c.InactivePercent, c.VoiceProb, c.JointThr} {
		if math.IsNaN(float64(value)) || value < 0 || value > 1 {
			return false
		}
	}
	return !math.IsNaN(float64(c.RmsThr)) && c.RmsThr >= -100 && c.RmsThr <= 0 && !math.IsNaN(float64(c.Aggressive)) && c.Aggressive >= 0 && c.Aggressive <= 3
}

// Release destroys the standalone detector. It waits for an in-flight
// ProcessPcmFrame call and is safe to repeat or call on a nil receiver.
// Subsequent valid processing requests return the retired-handle error -1.
func (vad *AudioVad) Release() {
	if vad == nil {
		return
	}
	vad.mu.Lock()
	defer vad.mu.Unlock()
	if vad.cVad != nil {
		C.agora_go_vad_destroy(&vad.cVad)
		vad.cVad = nil
	}
}

func validVADFrame(frame *AudioFrame, channels int) bool {
	return frame != nil && frame.Type == AudioFrameTypePCM16 && frame.SamplesPerSec == 16000 && frame.Channels == channels && frame.BytesPerSample == 2 && len(frame.Buffer) == 320*channels && frame.SamplesPerChannel == 160
}

// ProcessPcmFrame processes exactly one 10 ms, 16 kHz mono PCM16 frame:
// Type AudioFrameTypePCM16, 160 SamplesPerChannel, 2 BytesPerSample,
// 1 Channel, and a 320-byte Buffer. Keep the input unchanged during the call.
// Invalid input returns nil, -2; a released handle returns nil, -1.
//
// A nonnegative result is the native VAD state and the returned speech frame
// owns its copied buffer, which may be empty. RenderTimeMs is cumulative output
// duration in milliseconds, not the input timestamp. Native failures return
// negative codes; a C++ exception returns -2002 and retires the detector.
func (vad *AudioVad) ProcessPcmFrame(frame *AudioFrame) (*AudioFrame, int) {
	if vad == nil || !validVADFrame(frame, 1) {
		return nil, -2
	}
	vad.mu.Lock()
	defer vad.mu.Unlock()
	if vad.cVad == nil {
		return nil, -1
	}
	data := C.CBytes(frame.Buffer)
	defer C.free(data)
	in := C.Vad_AudioData{audioData: data, size: C.int(len(frame.Buffer))}
	var state C.enum_VAD_STATE
	var out C.Vad_AudioData
	if code := int(C.agora_go_vad_process(vad.cVad, &in, &out, &state)); code < 0 {
		if code == int(C.AGORA_GO_VAD_EXCEPTION) {
			C.agora_go_vad_destroy(&vad.cVad)
			vad.cVad = nil
		}
		return nil, code
	}
	if out.size < 0 || out.size > 64*1024*1024 || out.size%2 != 0 || (out.size > 0 && out.audioData == nil) {
		return nil, -1
	}
	samples := int(out.size) / 2
	result := &AudioFrame{Type: AudioFrameTypePCM16, Buffer: C.GoBytes(out.audioData, out.size), RenderTimeMs: vad.lastOutTs, SamplesPerChannel: samples, BytesPerSample: 2, Channels: 1, SamplesPerSec: 16000}
	vad.lastOutTs += int64(samples) * 1000 / 16000
	return result, int(state)
}

// SteroAudioVad owns independent left/right standalone native detectors.
// The spelling is retained for raw-binding compatibility. Construct it with
// NewSteroVad and release it once finished. Do not mutate or independently
// release exported components while the stereo wrapper is in use.
type SteroAudioVad struct {
	mu sync.Mutex
	// LeftVadInstance and RightVadInstance are owned component detectors.
	// Treat these handles as borrowed from the stereo wrapper.
	LeftVadInstance, RightVadInstance *AudioVad
	// LeftVadConfigure and RightVadConfigure are snapshots of construction settings.
	// Changing them does not reconfigure the native detectors.
	LeftVadConfigure, RightVadConfigure *AudioVadConfig
}

// NewSteroVad creates independent left and right detectors using copied settings.
// Either configuration may be nil to select defaults. It returns nil on invalid
// configuration or allocation failure and releases any partially created detector.
func NewSteroVad(leftConfig, rightConfig *AudioVadConfig) *SteroAudioVad {
	left := NewAudioVad(leftConfig)
	if left == nil {
		return nil
	}
	right := NewAudioVad(rightConfig)
	if right == nil {
		left.Release()
		return nil
	}
	leftCopy, rightCopy := *left.vadCfg, *right.vadCfg
	return &SteroAudioVad{LeftVadInstance: left, RightVadInstance: right, LeftVadConfigure: &leftCopy, RightVadConfigure: &rightCopy}
}

// ProcessAudioFrame accepts exactly 10 ms of 16 kHz interleaved stereo PCM16:
// 160 SamplesPerChannel, 2 BytesPerSample, 2 Channels, and 640 buffer bytes.
// It preserves left/right ordering and returns left frame/state followed by
// right frame/state. Each result owns its speech bytes.
// Invalid input produces -2 states; missing detectors produce -1 states.
// Component processing follows AudioVad.ProcessPcmFrame's error contract.
func (vad *SteroAudioVad) ProcessAudioFrame(frame *AudioFrame) (*AudioFrame, int, *AudioFrame, int) {
	if vad == nil || !validVADFrame(frame, 2) {
		return nil, -2, nil, -2
	}
	vad.mu.Lock()
	defer vad.mu.Unlock()
	if vad.LeftVadInstance == nil || vad.RightVadInstance == nil {
		return nil, -1, nil, -1
	}
	left, right := splitStereoVADFrame(frame)
	l, ls := vad.LeftVadInstance.ProcessPcmFrame(left)
	r, rs := vad.RightVadInstance.ProcessPcmFrame(right)
	return l, ls, r, rs
}

func splitStereoVADFrame(frame *AudioFrame) (*AudioFrame, *AudioFrame) {
	left, right := *frame, *frame
	left.Channels, right.Channels = 1, 1
	left.Buffer = make([]byte, len(frame.Buffer)/2)
	right.Buffer = make([]byte, len(frame.Buffer)/2)
	for src, dst := 0, 0; src < len(frame.Buffer); src, dst = src+4, dst+2 {
		copy(left.Buffer[dst:dst+2], frame.Buffer[src:src+2])
		copy(right.Buffer[dst:dst+2], frame.Buffer[src+2:src+4])
	}
	return &left, &right
}

// Release serializes against stereo processing, releases both owned detectors,
// and clears their handles. It is safe to repeat or call on a nil receiver.
func (vad *SteroAudioVad) Release() {
	if vad == nil {
		return
	}
	vad.mu.Lock()
	defer vad.mu.Unlock()
	vad.LeftVadInstance.Release()
	vad.RightVadInstance.Release()
	vad.LeftVadInstance, vad.RightVadInstance = nil, nil
}
