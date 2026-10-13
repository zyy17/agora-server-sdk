package agorasdk

import (
	"errors"
	"testing"
)

func TestDecoderConfigurationAndClosed(t *testing.T) {
	cfg, err := normalizedOpusDecoderConfig(OpusDecoderConfig{})
	if err != nil || cfg.InputChannels != 1 || cfg.Output.SampleRate != 48000 || cfg.Output.Channels != 1 {
		t.Fatal(cfg, err)
	}
	cfg, err = normalizedOpusDecoderConfig(OpusDecoderConfig{InputChannels: 2, Output: PCMFormat{SampleRate: 24000}})
	if err != nil || cfg.Output.Channels != 2 {
		t.Fatal(cfg, err)
	}
	for _, cfg := range []OpusDecoderConfig{{InputChannels: 3}, {Output: PCMFormat{SampleRate: -1}}, {Output: PCMFormat{Channels: 3}}, {Output: PCMFormat{SampleRate: 192001}}} {
		if _, err := NewOpusDecoder(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
	var video *VideoDecoder
	video.Close()
	if _, err := video.Decode([]byte{1}, H264, VideoDecodeOptions{}); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if _, err := video.Decode([]byte{1}, AV1, VideoDecodeOptions{}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if _, err := video.Decode([]byte{1}, H264, VideoDecodeOptions{2, 0}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	var opus *OpusDecoder
	opus.Close()
	if _, err := opus.Decode([]byte{1}); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if _, err := opus.Decode(nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if !avcodecAvailable {
		if _, err := NewVideoDecoder(); !errors.Is(err, ErrUnsupported) {
			t.Fatal(err)
		}
		if _, err := NewOpusDecoder(OpusDecoderConfig{}); !errors.Is(err, ErrUnsupported) {
			t.Fatal(err)
		}
	}
}
