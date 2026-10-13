package agorasdk

import (
	"context"
	"errors"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestAudioReceiveParametersAndCopies(t *testing.T) {
	cfg := validChannel()
	cfg.Audio.Receive = PCMFormat{SampleRate: 48000, Channels: 2}
	cfg.Audio.Fallback = &AudioFallbackPolicy{Scenario: ScenarioMeeting}
	cfg.Audio.Callbacks = &AudioCallbacks{OnFrame: func(AudioFrame) bool { return true }}
	cfg.Audio.ReceiveParameters = &AudioReceiveParameters{
		Playback: &AudioFrameParameters{Format: PCMFormat{SampleRate: 32000}, SamplesPerCall: 1280, ReadWrite: true},
		Record:   &AudioFrameParameters{Format: PCMFormat{SampleRate: 16000, Channels: 1}},
		Mixed:    &AudioFrameParameters{SamplesPerCall: 1920},
	}
	copy, err := normalized(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Audio.Fallback.Scenario = ScenarioChorus
	cfg.Audio.ReceiveParameters.Playback.SamplesPerCall = 12
	cfg.Audio.Callbacks.OnFrame = nil
	s := emptySession(nil, copy)
	defer s.Close()
	for _, test := range []struct {
		position                AudioPosition
		rate, channels, samples int
		writable                bool
	}{
		{AudioPlayback, 32000, 2, 1280, true}, {AudioRecord, 16000, 1, 160, false},
		{AudioMixed, 48000, 2, 1920, false}, {AudioEarMonitoring, 48000, 2, 960, false},
		{AudioBeforeMixing, 48000, 2, 960, false},
	} {
		params := s.audioFrameParameters(test.position)
		if params.SampleRate != test.rate || params.Channels != test.channels || params.SamplesPerCall != test.samples || (params.Mode == rtc.RawAudioFrameOpModeReadWrite) != test.writable {
			t.Fatalf("position %d = %+v", test.position, params)
		}
	}
	if copy.Audio.Fallback.Scenario != ScenarioMeeting || copy.Audio.Callbacks.OnFrame == nil {
		t.Fatal("retained caller-owned pointers")
	}
}

func TestAudioReceiveParameterValidation(t *testing.T) {
	for _, params := range []AudioFrameParameters{
		{Format: PCMFormat{SampleRate: 99}}, {Format: PCMFormat{Channels: 3}},
		{SamplesPerCall: -1}, {SamplesPerCall: 3, Format: PCMFormat{Channels: 2}},
	} {
		cfg := validChannel()
		cfg.Audio.ReceiveParameters = &AudioReceiveParameters{Playback: &params}
		if _, err := normalized(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
	cfg := validChannel()
	cfg.Audio.Fallback = &AudioFallbackPolicy{Scenario: 99}
	if _, err := normalized(cfg); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}

func TestAudioCallbackDecisionsAndPanic(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	frame := AudioFrame{Data: []byte{1, 2}, Position: AudioPlayback}
	if !s.processAudioCallback(frame) {
		t.Fatal("missing hook rejected frame")
	}
	s.cfg.Audio.Callbacks = &AudioCallbacks{OnFrame: func(f AudioFrame) bool {
		f.Data[0] = 9
		return false
	}}
	if s.processAudioCallback(frame) || frame.Data[0] != 9 {
		t.Fatal("hook changes not preserved")
	}
	s.cfg.Audio.Callbacks.OnFrame = func(AudioFrame) bool {
		// Snapshot APIs must remain safe without acquiring the native lock.
		_ = s.Info()
		_ = s.Role()
		_ = s.Done()
		if err := s.SetAudioFallback(nil); err != nil {
			t.Error(err)
		}
		panic("application bug")
	}
	s.mu.Lock()
	accepted := s.processAudioCallback(frame)
	s.mu.Unlock()
	if accepted {
		t.Fatal("panicking hook accepted frame")
	}
	event, err := s.Events().Next(context.Background())
	if err != nil || event.Kind != CallbackPanic {
		t.Fatal(event, err)
	}
}
