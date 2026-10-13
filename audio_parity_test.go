package agorasdk

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestEncodedAudioOptions(t *testing.T) {
	for _, codec := range []AudioCodec{Opus, PCMA, PCMU, G722, AAC, HEAAC, HEAACv2, JC1, LPCNet} {
		if err := (EncodedAudioFrame{Data: []byte{1}, Codec: codec, SampleRate: 48000, Channels: 2}).validate(); err != nil {
			t.Fatal(codec, err)
		}
	}
	for channels := 3; channels <= 8; channels++ {
		if err := (EncodedAudioFrame{Codec: OpusMultiChannel, SampleRate: 48000, Channels: channels, SendEvenIfEmpty: true}).validate(); err != nil {
			t.Fatal(channels, err)
		}
	}
	for _, f := range []EncodedAudioFrame{
		{Codec: Opus, SampleRate: 48000, Channels: 1},
		{Data: []byte{1}, Codec: Opus, SampleRate: 48000, Channels: 3},
		{Data: []byte{1}, Codec: OpusMultiChannel, SampleRate: 48000, Channels: 2},
		{Data: []byte{1}, Codec: OpusMultiChannel, SampleRate: 48000, Channels: 9},
		{Data: []byte{1}, Codec: Opus, SampleRate: 48000, Channels: 1, SamplesPerChannel: math.MaxInt32 + 1},
	} {
		if err := f.validate(); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("accepted %+v", f)
		}
	}
}

func TestTrackControlValidationAndClose(t *testing.T) {
	s := &Session{closed: true}
	for _, err := range []error{s.SetAudioEnabled(false), s.SetVideoEnabled(false), s.SetAudioPublishVolume(100), s.SetAudioSendDelay(10 * time.Millisecond), s.SetAudioBufferCapacity(30), s.ClearAudioBuffer()} {
		if !errors.Is(err, ErrClosed) {
			t.Fatal(err)
		}
	}
	for _, err := range []error{s.SetAudioPublishVolume(-1), s.SetAudioPublishVolume(math.MaxInt32 + 1), s.SetAudioSendDelay(-1), s.SetAudioSendDelay(time.Duration(math.MaxInt32+1) * time.Millisecond), s.SetAudioBufferCapacity(0), s.SetAudioBufferCapacity(math.MaxInt32 + 1)} {
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
}

func TestPCMFrameExplicitMetadata(t *testing.T) {
	f := AudioFrame{Data: []byte{1, 2}, SampleRate: 16000, Channels: 1, SamplesPerChannel: 1, BytesPerSample: 2, TimestampMillis: math.MaxUint32, PresentationMillis: math.MinInt64}
	n := nativeSendAudioFrame(f)
	if n.ValidatePCM() != nil || n.RenderTimeMs != math.MaxUint32 || n.PresentTimeMs != math.MinInt64 || n.Buffer[0] != 1 {
		t.Fatal(n)
	}
	s := &Session{closed: true, cfg: ChannelConfig{Audio: AudioConfig{Send: PCMFormat{SampleRate: 16000, Channels: 1}}}}
	if err := s.SendPCMFrame(f); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	f.TimestampMillis = -1
	if err := s.SendPCMFrame(f); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	f.TimestampMillis = 0
	f.SampleRate = 48000
	if err := s.SendPCMFrame(f); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}
