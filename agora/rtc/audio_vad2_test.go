package agoraservice

import (
	"math"
	"testing"
)

func TestVadConfigIsNotMutatedBetweenUsers(t *testing.T) {
	cfg := AudioVadConfigV2{StartRecognizeCount: 30, StopRecognizeCount: 65, StartRms: -70, StopRms: -60}
	original := cfg
	first := NewAudioVadV2(&cfg)
	second := NewAudioVadV2(&cfg)
	defer first.Release()
	defer second.Release()
	if cfg != original {
		t.Fatal("constructor mutated caller configuration")
	}
	if first.config.StartRms != second.config.StartRms || first.config.StartRms != 57 {
		t.Fatal("per-user thresholds differ")
	}
}

func TestVadInvalidConfigIsBounded(t *testing.T) {
	vad := NewAudioVadV2(&AudioVadConfigV2{PreStartRecognizeCount: -100, StartRecognizeCount: -5, StopRecognizeCount: -1, ActivePercent: float32(math.NaN()), StartRms: 999})
	defer vad.Release()
	if vad.startBuffer.maxSize < 1 || vad.stopBuffer.maxSize < 1 {
		t.Fatal("invalid buffer capacity")
	}
	frame := &AudioFrame{BytesPerSample: 2, Channels: 1, SamplesPerSec: 16000, SamplesPerChannel: 160, Buffer: make([]byte, 320)}
	for range 100 {
		vad.Process(frame)
	}
	vad.Release()
	vad.Release()
	if _, state := vad.Process(frame); state != VadStateInvalid {
		t.Fatal("released VAD accepted frame")
	}
}
