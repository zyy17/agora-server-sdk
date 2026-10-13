//go:build avcodec

package agorasdk

import (
	"encoding/base64"
	"errors"
	"sync"
	"testing"
)

func TestOptionalMediaDecoders(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString("AAAAAWdCwAraEewEQAAAAwBAAAAMo8SJqAAAAAFozg/IAAAAAWWIhDoRigACGPHAAED2OAAIeUnJyddddddddeA=")
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewVideoDecoder()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	frame, err := d.Decode(data, H264, VideoDecodeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if frame.Width != 64 || frame.Height != 48 || frame.Format != I420 || len(frame.Data) != 64*48*3/2 {
		t.Fatal(frame)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 5 {
				_, err := d.Decode(data, H264, VideoDecodeOptions{32, 24})
				if err != nil && !errors.Is(err, ErrClosed) {
					t.Error(err)
				}
			}
		})
	}
	wg.Go(func() { d.Close() })
	wg.Wait()
	if frame.Data[0] != 81 {
		t.Fatal("returned pixels did not survive close")
	}
	if _, err := d.Decode(data, H264, VideoDecodeOptions{}); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	opus, err := NewOpusDecoder(OpusDecoderConfig{InputChannels: 1, Output: PCMFormat{SampleRate: 16000, Channels: 2}})
	if err != nil {
		t.Fatal(err)
	}
	defer opus.Close()
	// Standard Opus silence packet, 20 ms, single-frame encoding.
	pcm, err := opus.Decode([]byte{0xf8, 0xff, 0xfe})
	if err != nil {
		t.Fatal(err)
	}
	if pcm.SampleRate != 16000 || pcm.Channels != 2 || pcm.BytesPerSample != 2 || len(pcm.Data) == 0 || pcm.SamplesPerChannel*4 != len(pcm.Data) {
		t.Fatal(pcm)
	}
	opus.Close()
	opus.Close()
	if _, err := opus.Decode([]byte{0xf8, 0xff, 0xfe}); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
