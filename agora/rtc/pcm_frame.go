package agoraservice

import "math"

// makePCMFrame preserves complete samples and opaque presentation values.
// Integer millisecond packets need 1 ms at integer-kHz rates and 10 ms at 44.1 kHz.
func makePCMFrame(data []byte, rate, channels int, pts int64) (*AudioFrame, bool) {
	switch rate {
	case 8000, 16000, 32000, 44100, 48000:
	default:
		return nil, false
	}
	if channels != 1 && channels != 2 {
		return nil, false
	}
	if len(data) == 0 || len(data)%(channels*2) != 0 {
		return nil, false
	}
	samples := len(data) / (channels * 2)
	a, b := rate, 1000
	for b != 0 {
		a, b = b, a%b
	}
	if uint64(samples) > math.MaxUint32 || samples%(rate/a) != 0 {
		return nil, false
	}
	return &AudioFrame{Type: AudioFrameTypePCM16, Buffer: data, PresentTimeMs: pts, SamplesPerChannel: samples, BytesPerSample: 2, Channels: channels, SamplesPerSec: rate}, true
}
