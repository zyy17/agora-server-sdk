package agorasdk

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestPlayPCMPaddingAndTimestamp(t *testing.T) {
	var frames [][]byte
	var timestamps []int64
	input := make([]byte, 322)
	input[320] = 7
	err := playPCM(context.Background(), bytes.NewReader(input), 16000, 1, 1000, func(p []byte, pts int64) error {
		frames = append(frames, slices.Clone(p))
		timestamps = append(timestamps, pts)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || len(frames[1]) != 320 || frames[1][0] != 7 || frames[1][2] != 0 || timestamps[0] != 1000 || timestamps[1] != 1010 {
		t.Fatal("incorrect paced frames", timestamps)
	}
}
func TestPlayPCMCancellationStopsFutureFrames(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sent := 0
	err := playPCM(ctx, bytes.NewReader(make([]byte, 960)), 16000, 1, 0, func([]byte, int64) error { sent++; cancel(); return nil })
	if !errors.Is(err, context.Canceled) || sent != 1 {
		t.Fatal(sent, err)
	}
}
func TestPlayPCMRejectsIncompleteSample(t *testing.T) {
	err := playPCM(context.Background(), bytes.NewReader([]byte{1}), 16000, 1, 0, func([]byte, int64) error { t.Fatal("sent partial sample"); return nil })
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}

type pcmCaptureTransport struct {
	fakeTransport
	sends []struct {
		data           []byte
		rate, channels int
		pts            int64
	}
	publishCode, sendCode int
}

func (f *pcmCaptureTransport) PublishAudio() int { return f.publishCode }
func (f *pcmCaptureTransport) PushAudioPcmData(data []byte, rate, channels int, pts int64) int {
	f.sends = append(f.sends, struct {
		data           []byte
		rate, channels int
		pts            int64
	}{slices.Clone(data), rate, channels, pts})
	return f.sendCode
}

func TestPCMIndependentFormatsAndOpaquePTS(t *testing.T) {
	for _, channels := range []int{1, 2} {
		config := validChannel()
		config.Audio.Send = PCMFormat{16000, channels}
		config.Audio.Receive = PCMFormat{16000, 1}
		config, err := normalized(config)
		if err != nil {
			t.Fatal(err)
		}
		s := emptySession(nil, config)
		transport := new(pcmCaptureTransport)
		s.transport = transport
		params := s.audioObserver().OnGetPlaybackAudioFrameParam(nil)
		if params.SampleRate != 16000 || params.Channels != 1 || params.SamplesPerCall != 160 {
			t.Fatal(params)
		}
		for i, ms := range []int{1, 7, 10, 1003} {
			data := bytes.Repeat([]byte{byte(i), 0}, 16000/1000*channels*ms)
			pts := int64(0x7123456789abcdef) + int64(i)
			if err := s.SendPCMAt(data, pts); err != nil {
				t.Fatal(err)
			}
			got := transport.sends[i]
			if !bytes.Equal(got.data, data) || got.pts != pts || got.rate != 16000 || got.channels != channels {
				t.Fatal("PCM or PTS changed", got)
			}
		}
		if err := s.SendPCMAt([]byte{0}, 1); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
		if channels == 2 {
			if err := s.SendPCM(make([]byte, 32)); !errors.Is(err, ErrInvalidConfig) {
				t.Fatal("used receive format to send", err)
			}
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		if err := s.SendPCM(make([]byte, 32*channels)); !errors.Is(err, ErrClosed) {
			t.Fatal(err)
		}
	}
}

func TestPCMSubmissionErrors(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	tr := &pcmCaptureTransport{publishCode: -11}
	s.transport = tr
	var native *Error
	if err := s.SendPCM(make([]byte, 32)); !errors.As(err, &native) || native.Code != -11 {
		t.Fatal(err)
	}
	if len(tr.sends) != 0 {
		t.Fatal("sent after publish failure")
	}
	tr.publishCode = 0
	tr.sendCode = -12
	if err := s.SendPCM(make([]byte, 32)); !errors.As(err, &native) || native.Code != -12 {
		t.Fatal(err)
	}
}

func TestPCMQuantumAndProcessorFrames(t *testing.T) {
	for _, rate := range []int{8000, 16000, 32000, 48000} {
		for _, ch := range []int{1, 2} {
			if err := validatePCM(make([]byte, rate/1000*ch*2), rate, ch); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, size := range []int{2, 88, 880, 884} {
		if err := validatePCM(make([]byte, size), 44100, 1); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(size, err)
		}
	}
	if err := validatePCM(make([]byte, 882), 44100, 1); err != nil {
		t.Fatal(err)
	}
	if validateProcessorPCM(make([]byte, 32), 16000, 1) == nil {
		t.Fatal("processor accepted short frame")
	}
	if err := validateProcessorPCM(make([]byte, 320), 16000, 1); err != nil {
		t.Fatal(err)
	}
}

func TestReceivedPCMMetadataWithoutVAD(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	frame := &rtc.AudioFrame{Type: rtc.AudioFrameTypePCM16, Buffer: []byte{1, 0, 2, 0}, SamplesPerChannel: 1, BytesPerSample: 2, Channels: 2, SamplesPerSec: 16000, RenderTimeMs: 42, PresentTimeMs: 0x7123456789abcdef, AvsyncType: 2, FarFieldFlag: 1, Rms: 63, VoiceProb: 1, MusicProb: 7, Pitch: 200}
	s.audioObserver().OnPlaybackAudioFrameBeforeMixing(nil, "room", "12345", frame, rtc.VadStateInvalid, nil)
	frame.Buffer[0] = 9
	got, err := s.Audio().Next(context.Background())
	if err != nil || got.Data[0] != 1 || got.SamplesPerChannel != 1 || got.BytesPerSample != 2 || got.AVSyncType != 2 || got.FarField != 1 || got.PresentationMillis != frame.PresentTimeMs || got.RMS != 63 || got.MusicProbability != 7 || got.Pitch != 200 || got.Speech != nil {
		t.Fatal(got, err)
	}
}

func TestPlayPCMZeroTimestampStaysZero(t *testing.T) {
	sent := 0
	err := playPCM(context.Background(), bytes.NewReader(make([]byte, 640)), 16000, 1, 0, func(_ []byte, pts int64) error {
		sent++
		if pts != 0 {
			t.Fatal(pts)
		}
		return nil
	})
	if err != nil || sent != 2 {
		t.Fatal(sent, err)
	}
}

func TestBinaryAudioMetadataPreservesOwnership(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	data := []byte{0, 0xff, 1, 0}
	s.localObserver().OnAudioMetaDataReceived(nil, "12345", data)
	data[1] = 9
	event, err := s.Events().Next(context.Background())
	payload, ok := event.Data.([]byte)
	if err != nil || !ok || event.Kind != AudioMetadata || event.UserID != "12345" || !bytes.Equal(payload, []byte{0, 0xff, 1, 0}) {
		t.Fatal(event, err)
	}
}
