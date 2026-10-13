package agoraservice

import (
	"container/list"
	"fmt"
	"math"
)

// AudioVadConfigV2 configures the Go-side SDK telemetry segmenter, not standalone
// native AudioVad. Counts are measured in incoming frames, conventionally 10 ms.
// NewAudioVadV2(nil) selects a complete default configuration. For a non-nil
// configuration, only specified invalid/zero values are normalized as documented;
// a zero struct is not equivalent to nil.
type AudioVadConfigV2 struct {
	// PreStartRecognizeCount is buffered pre-roll before speech onset.
	// Nil-config default: 16. Zero remains zero; negative values normalize to zero.
	// The combined start/pre-roll buffer is capped at 60000 frames.
	PreStartRecognizeCount int
	// StartRecognizeCount is the required consecutive active-frame count.
	// Nonpositive values use 30; positive values are capped at 60000.
	// The full start/pre-roll buffer must also be populated before onset.
	StartRecognizeCount int
	// StopRecognizeCount is the required consecutive inactive-frame count.
	// Nonpositive values use 65; positive values are capped at 60000.
	StopRecognizeCount int
	// ActivePercent is a compatibility setting normalized to 0.7 outside (0,1].
	// The current segmenter uses consecutive-frame counts rather than this ratio.
	ActivePercent float32
	// InactivePercent is a compatibility setting normalized to 0.5 outside (0,1].
	// The current segmenter uses consecutive-frame counts rather than this ratio.
	InactivePercent float32
	// StartVoiceProb is a compatibility threshold clamped to 0–100.
	// The current active-frame decision instead treats native VoiceProb == 1 as active.
	StartVoiceProb int
	// StartRms is the initial RMS threshold in -127–0 dB; values are clamped.
	// Nil-config default: -70. Internally 127 is added to match native RMS telemetry.
	StartRms int
	// StopVoiceProb is a compatibility threshold clamped to 0–100.
	// The current active-frame decision does not use a separate stop-probability threshold.
	StopVoiceProb int
	// StopRms is a compatibility threshold clamped to -127–0 dB.
	// The current active-frame decision uses the start/adaptive threshold instead.
	StopRms int
	// EnableAdaptiveRmsThreshold enables a threshold derived from the previous
	// speech onset's average native RMS. Nil-config default: true.
	EnableAdaptiveRmsThreshold bool
	// AdaptiveRmsThresholdFactor scales that average; values outside (0,1] normalize
	// to 0.67. It applies only when adaptive thresholds are enabled and an average exists.
	AdaptiveRmsThresholdFactor float32
}

// VadFrame is opaque per-frame state used internally by AudioVadV2.
// Its fields are intentionally inaccessible; applications use AudioFrame and Process.
type VadFrame struct {
	frame    *AudioFrame
	isActive bool
}

// VadBuffer is an opaque frame queue used internally by AudioVadV2.
// It has no independently usable zero-value or exported mutation API.
type VadBuffer struct {
	queue   *list.List
	maxSize int
}

// VadFrameFormat records the first input format accepted by AudioVadV2.
type VadFrameFormat struct {
	// BytesPerSample is the byte width of a sample; PCM16 uses 2.
	BytesPerSample int
	// Channels is the number of interleaved channels.
	Channels int
	// SamplesPerSec is the sampling rate in Hz.
	SamplesPerSec int
}

// AudioVadV2 segments SDK audio using native voice/RMS telemetry and consecutive
// active/inactive frame counts. It is not a standalone PCM classifier.
// Construct it with NewAudioVadV2; serialize Process and Release externally.
// It retains input frame pointers, and output may alias the current input.
type AudioVadV2 struct {
	config                  *AudioVadConfigV2
	expectFormat            *VadFrameFormat
	isSpeaking              bool
	startBuffer             *VadBuffer
	stopBuffer              *VadBuffer
	voiceCount              int
	silenceCount            int
	totalVoiceRms           int // range from 0 to 127, respond to db: -127db, to 0db
	refAvgRmsInLastSesseion int // range from 0 to 127, respond to db: -127db, to 0db
}

func newVadFrame(frame *AudioFrame, isActive bool) *VadFrame {
	return &VadFrame{
		frame:    frame,
		isActive: isActive,
	}
}

func newVadBuffer(maxSize int) *VadBuffer {
	if maxSize < 1 {
		maxSize = 1
	}
	return &VadBuffer{
		queue:   list.New(),
		maxSize: maxSize,
	}
}

func (buf *VadBuffer) pushBack(v *VadFrame) bool {
	l := buf.queue
	if l.Len() >= buf.maxSize {
		l.Remove(l.Front())
	}
	l.PushBack(v)
	return l.Len() >= buf.maxSize
}

func (buf *VadBuffer) clear() {
	buf.queue.Init()
}

func (buf *VadBuffer) getActivePercent(lastN int) float32 {
	l := buf.queue
	count := 0
	curIndex := 0
	startElem := l.Len() - lastN
	for e := l.Front(); e != nil; e = e.Next() {
		if curIndex < startElem {
			curIndex++
			continue
		}
		v := e.Value.(*VadFrame)
		if v.isActive {
			count++
		}
		curIndex++
	}
	// fmt.Printf("[vad] getActivePercent: %d, %d, %f\n", count, lastN, float32(count)/float32(lastN))
	return float32(count) / float32(lastN)
}

func (buf *VadBuffer) flushAudio() *AudioFrame {
	l := buf.queue
	if l.Len() == 0 {
		return nil
	}
	// copy a frame
	samplesCount := 0
	ret := *(l.Front().Value.(*VadFrame).frame)
	data := make([]byte, 0, l.Len()*ret.SamplesPerChannel*ret.BytesPerSample*ret.Channels)
	for e := l.Front(); e != nil; e = e.Next() {
		v := e.Value.(*VadFrame)
		data = append(data, v.frame.Buffer...)
		samplesCount += v.frame.SamplesPerChannel
	}
	ret.Buffer = data
	ret.SamplesPerChannel = samplesCount
	l.Init()
	return &ret
}

// NewAudioVadV2 copies and normalizes cfg, then creates segmenter buffers.
// Nil selects 16 pre-roll, 30 start, and 65 stop frames; -70 dB RMS; and enabled
// adaptive RMS with factor 0.67. At 10 ms per frame those counts are 160, 300,
// and 650 ms. A non-nil zero configuration differs from nil; see AudioVadConfigV2.
// The current implementation writes diagnostic configuration/state lines to stdout.
func NewAudioVadV2(cfg *AudioVadConfigV2) *AudioVadV2 {

	if cfg == nil {
		cfg = &AudioVadConfigV2{
			PreStartRecognizeCount:     16,
			StartRecognizeCount:        30,
			StopRecognizeCount:         65,
			ActivePercent:              0.7,
			InactivePercent:            0.5,
			StartVoiceProb:             70,
			StartRms:                   -70.0,
			StopVoiceProb:              70,
			StopRms:                    -70.0,
			EnableAdaptiveRmsThreshold: true,
			AdaptiveRmsThresholdFactor: 0.67,
		}
	}
	config := *cfg
	cfg = &config
	if cfg.PreStartRecognizeCount < 0 {
		cfg.PreStartRecognizeCount = 0
	}
	if cfg.StartRecognizeCount <= 0 {
		cfg.StartRecognizeCount = 30
	}
	if cfg.StopRecognizeCount <= 0 {
		cfg.StopRecognizeCount = 65
	}
	// Bound buffered frames (10 ms each) and normalize probability thresholds.
	const maxFrames = 60000
	cfg.StartRecognizeCount = min(cfg.StartRecognizeCount, maxFrames)
	cfg.StopRecognizeCount = min(cfg.StopRecognizeCount, maxFrames)
	cfg.PreStartRecognizeCount = min(cfg.PreStartRecognizeCount, maxFrames-cfg.StartRecognizeCount)
	cfg.StartVoiceProb = max(0, min(cfg.StartVoiceProb, 100))
	cfg.StopVoiceProb = max(0, min(cfg.StopVoiceProb, 100))
	cfg.StartRms = max(-127, min(cfg.StartRms, 0))
	cfg.StopRms = max(-127, min(cfg.StopRms, 0))
	if math.IsNaN(float64(cfg.ActivePercent)) || cfg.ActivePercent <= 0 || cfg.ActivePercent > 1 {
		cfg.ActivePercent = 0.7
	}
	if math.IsNaN(float64(cfg.InactivePercent)) || cfg.InactivePercent <= 0 || cfg.InactivePercent > 1 {
		cfg.InactivePercent = 0.5
	}
	if math.IsNaN(float64(cfg.AdaptiveRmsThresholdFactor)) || cfg.AdaptiveRmsThresholdFactor <= 0 || cfg.AdaptiveRmsThresholdFactor > 1 {
		cfg.AdaptiveRmsThresholdFactor = 0.67
	}
	// fmt.Printf("[vad] NewAudioVadV2: %v\n", cfg)
	startQueueSize := cfg.StartRecognizeCount + cfg.PreStartRecognizeCount
	ret := &AudioVadV2{
		config:                  cfg,
		expectFormat:            nil,
		isSpeaking:              false,
		startBuffer:             newVadBuffer(startQueueSize),
		stopBuffer:              newVadBuffer(cfg.StopRecognizeCount),
		voiceCount:              0,
		silenceCount:            0,
		totalVoiceRms:           0,
		refAvgRmsInLastSesseion: 0,
	}
	// for this version, convert rms from db to rms,i.e convert from -127db to 0db to 0 to 127
	ret.config.StartRms = ret.config.StartRms + 127
	ret.config.StopRms = ret.config.StopRms + 127

	fmt.Printf("[vad] NewAudioVadV2: %v\n", ret.config)
	return ret
}

// Release clears retained frames and retires the segmenter.
// It is safe to repeat sequentially or call on nil, but must not race with Process.
func (vad *AudioVadV2) Release() {
	if vad == nil || vad.startBuffer == nil || vad.stopBuffer == nil {
		return
	}
	vad.startBuffer.clear()
	vad.stopBuffer.clear()
	vad.startBuffer = nil
	vad.stopBuffer = nil
}

func (vad *AudioVadV2) isActive(frame *AudioFrame) bool {
	voiceProb := 0
	rmsProb := 0
	if vad.isSpeaking {
		voiceProb = vad.config.StopVoiceProb
		rmsProb = vad.config.StopRms
	} else {
		voiceProb = vad.config.StartVoiceProb
		rmsProb = vad.config.StartRms
	}

	active := frame.FarFieldFlag == 1 && frame.VoiceProb > voiceProb && frame.Rms > rmsProb
	// date: 2025-10-29 for sdk which support apm filter, and in this version, we don't need to use farfield flag to detect speech, so we don't need to use farfield flag to detect speech
	refRms := vad.getRefActiveAvgRms()
	active = (frame.VoiceProb == 1) || (frame.Rms > refRms)
	//fmt.Printf("[vad] voiceProb: %d, rms: %d, pitch: %d, refRms: %d, active: %v\n", frame.VoiceProb, frame.Rms, frame.Pitch, refRms, active)
	return active
}

func (vad *AudioVadV2) getRefActiveAvgRms() int {
	if vad.config.EnableAdaptiveRmsThreshold && vad.refAvgRmsInLastSesseion > 0 {
		return int(float32(vad.refAvgRmsInLastSesseion) * vad.config.AdaptiveRmsThresholdFactor) // half of avg as threshold value
	}
	// convert from db to rms
	return vad.config.StartRms
}

// Process consumes one telemetry-bearing AudioFrame and returns selected speech
// plus a VadState. Nil input, nil receiver, or a released segmenter returns
// VadStateInvalid. The first input fixes sample width, channel count, and rate;
// a later mismatch returns VadStateNoSpeeking without changing that format.
// This method does not enforce exact 10 ms PCM shape like AudioVad.ProcessPcmFrame.
//
// Speech onset returns a newly concatenated pre-roll/start buffer. During speech
// and at its end, the returned frame may be the input pointer itself. Input frames
// are retained in internal queues: do not mutate/reuse them while retained.
// The caller must serialize processing and cleanup. A non-speaking state is not
// an error, and the current implementation may emit diagnostic lines to stdout.
func (vad *AudioVadV2) Process(frame *AudioFrame) (*AudioFrame, VadState) {
	if vad == nil || frame == nil || vad.startBuffer == nil || vad.stopBuffer == nil {
		return nil, VadStateInvalid
	}
	if vad.expectFormat == nil {
		vad.expectFormat = &VadFrameFormat{
			BytesPerSample: frame.BytesPerSample,
			Channels:       frame.Channels,
			SamplesPerSec:  frame.SamplesPerSec,
		}
	} else {
		if vad.expectFormat.BytesPerSample != frame.BytesPerSample ||
			vad.expectFormat.Channels != frame.Channels ||
			vad.expectFormat.SamplesPerSec != frame.SamplesPerSec {
			return nil, VadStateNoSpeeking
		}
	}
	// if vad.isSpeaking {
	// 	fmt.Printf("[vad] +++++++++++++++++\n")
	// } else {
	// 	fmt.Printf("[vad] -----------------\n")
	// }
	isActive := vad.isActive(frame)
	vadFrame := newVadFrame(frame, isActive)
	if !vad.isSpeaking {
		full := vad.startBuffer.pushBack(vadFrame)
		if isActive {
			vad.totalVoiceRms += frame.Rms
			vad.voiceCount++
		} else {
			vad.voiceCount = 0
			vad.totalVoiceRms = 0
		}
		// todo： 是否需要根据N个isActive来计算avg rms? 而不是开始的时候，直接计算avg rms?
		// fmt.Printf("[vad] isSpeaking: false, startBuffer: %d\n", vad.startBuffer.queue.Len())
		if full {
			//activePercent = float32(vad.voiceCount) / float32(vad.config.StartRecognizeCount)
			//todo: disable activePercent check, use voiceCount instead
			if vad.voiceCount >= vad.config.StartRecognizeCount {
				vad.isSpeaking = true
				vad.stopBuffer.clear()
				ret := vad.startBuffer.flushAudio()

				// update ref rms
				vad.refAvgRmsInLastSesseion = vad.totalVoiceRms / vad.voiceCount
				fmt.Printf("[vad] StartSpeeking: %d, %d, %d, %d\n", vad.voiceCount, vad.totalVoiceRms, vad.refAvgRmsInLastSesseion, vad.config.StartRecognizeCount)

				//reset
				vad.voiceCount = 0
				vad.totalVoiceRms = 0
				vad.silenceCount = 0

				// return the frame, and the state
				return ret, VadStateStartSpeeking
			}
		}
		return nil, VadStateNoSpeeking
	} else {
		full := vad.stopBuffer.pushBack(vadFrame)
		// fmt.Printf("[vad] isSpeaking: true, stopBuffer: %d\n", vad.stopBuffer.queue.Len())
		if isActive {
			vad.silenceCount = 0
		} else {
			vad.silenceCount++
		}
		if full {
			//inactivePercent := float32(vad.silenceCount) / float32(vad.config.StopRecognizeCount)
			if vad.silenceCount >= vad.config.StopRecognizeCount {
				vad.isSpeaking = false
				vad.stopBuffer.clear()

				fmt.Printf("[vad] StopSpeeking: %d, %d, %d, %d\n", vad.silenceCount, vad.totalVoiceRms, vad.refAvgRmsInLastSesseion, vad.config.StopRecognizeCount)

				// reset
				vad.voiceCount = 0
				vad.totalVoiceRms = 0
				vad.silenceCount = 0

				//return
				return frame, VadStateStopSpeeking
			}
		}
		return frame, VadStateSpeeking
	}
}
