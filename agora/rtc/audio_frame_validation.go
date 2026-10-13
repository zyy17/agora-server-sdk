package agoraservice

import (
	"fmt"
	"math"
)

// ValidatePCM checks the complete PCM16 send layout and native scalar bounds.
// It accepts positive uint32 sample rates/counts and one or two interleaved
// channels; native format support is checked by the SDK. RenderTimeMs must fit
// the C sender's uint32 capture timestamp, while PresentTimeMs is signed int64.
// Observer telemetry is not transmitted by the PCM sender.
func (f *AudioFrame) ValidatePCM() error {
	if f == nil {
		return fmt.Errorf("frame is nil")
	}
	if f.Type != AudioFrameTypePCM16 || f.BytesPerSample != 2 || f.Channels < 1 || f.Channels > 2 {
		return fmt.Errorf("requires interleaved PCM16 with one or two channels")
	}
	if f.SamplesPerChannel <= 0 || uint64(f.SamplesPerChannel) > math.MaxUint32 || f.SamplesPerSec <= 0 || uint64(f.SamplesPerSec) > math.MaxUint32 {
		return fmt.Errorf("sample count and rate must be positive uint32")
	}
	if len(f.Buffer)%(f.Channels*2) != 0 || len(f.Buffer)/(f.Channels*2) != f.SamplesPerChannel {
		return fmt.Errorf("buffer must match sample count and channel layout")
	}
	if f.RenderTimeMs < 0 || uint64(f.RenderTimeMs) > math.MaxUint32 {
		return fmt.Errorf("capture timestamp must fit uint32")
	}
	return nil
}
