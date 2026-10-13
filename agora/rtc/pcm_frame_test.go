package agoraservice

import "testing"

func TestPCMFramePreservesSamplesAndPTS(t *testing.T) {
	for _, rate := range []int{8000, 16000, 32000, 44100, 48000} {
		for _, ch := range []int{1, 2} {
			quantum := 1
			if rate == 44100 {
				quantum = 10
			}
			for _, n := range []int{1, 7, 103} {
				samples := rate * quantum * n / 1000
				data := make([]byte, samples*ch*2)
				pts := int64(-0x7123456789abcdef)
				f, ok := makePCMFrame(data, rate, ch, pts)
				if !ok || f.SamplesPerChannel != samples || f.PresentTimeMs != pts || f.Channels != ch || len(f.Buffer) != len(data) {
					t.Fatal(f, ok)
				}
			}
		}
	}
	for _, test := range []struct{ bytes, rate, ch int }{{0, 16000, 1}, {31, 16000, 1}, {2, 16000, 1}, {32, 16000, 2}, {88, 44100, 1}, {32, 11025, 1}, {32, 16000, 3}} {
		if _, ok := makePCMFrame(make([]byte, test.bytes), test.rate, test.ch, 0); ok {
			t.Fatal(test)
		}
	}
}
