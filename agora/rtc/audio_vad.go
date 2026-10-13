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

type AudioVadConfig struct {
	StartRecognizeCount    int     // start recognize count, buffer size for 10ms 16KHz 16bit 1channel PCM, default value is 30
	StopRecognizeCount     int     // max recognize count, buffer size for 10ms 16KHz 16bit 1channel PCM, default value is 48
	PreStartRecognizeCount int     // pre start recognize count, buffer size for 10ms 16KHz 16bit 1channel PCM, default value is 16
	ActivePercent          float32 // active percent, if over this percent, will be recognized as speaking, default value is 0.8
	InactivePercent        float32 // inactive percent, if below this percent, will be recognized as non-speaking, default value is 0.2
	VoiceProb              float32 // voice probability threshold, default value is 0.7; range from 0 to 1
	RmsThr                 float32 // rms threshold, default value is -40; range from -100 to 0
	JointThr               float32 // joint threshold, default value is 0.0; range from 0 to 1
	Aggressive             float32 // default value is 2.0; range from 0 to 3
}

// AudioVad owns one standalone native VAD. ProcessPcmFrame and Release are
// serialized; use separate detectors for independent speakers/channels.
type AudioVad struct {
	mu        sync.Mutex
	vadCfg    *AudioVadConfig
	cVad      unsafe.Pointer
	lastOutTs int64
}

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

// ProcessPcmFrame accepts exactly one 10 ms 16 kHz mono PCM16 frame. Returned
// speech owns its buffer; a native C++ exception returns -2002 and retires the
// detector. RenderTimeMs is the detector's cumulative output time.
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

// SteroAudioVad owns independent left/right detectors (the spelling is inherited
// from the raw binding). Do not replace exported components while methods run.
type SteroAudioVad struct {
	mu                                  sync.Mutex
	LeftVadInstance, RightVadInstance   *AudioVad
	LeftVadConfigure, RightVadConfigure *AudioVadConfig
}

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

// ProcessAudioFrame splits interleaved stereo PCM16 without changing channel
// order. Each result uses its own detector state and owns its speech buffer.
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
