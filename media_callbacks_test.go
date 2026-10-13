package agorasdk

import (
	"context"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestAudioObserverIndependentParameters(t *testing.T) {
	s := &Session{cfg: ChannelConfig{Audio: AudioConfig{Receive: PCMFormat{SampleRate: 16000, Channels: 1}, ReceiveParameters: &AudioReceiveParameters{
		Playback:      &AudioFrameParameters{Format: PCMFormat{SampleRate: 48000, Channels: 2}, SamplesPerCall: 1920, ReadWrite: true},
		Record:        &AudioFrameParameters{Format: PCMFormat{SampleRate: 32000, Channels: 1}},
		Mixed:         &AudioFrameParameters{Format: PCMFormat{SampleRate: 44100, Channels: 2}},
		EarMonitoring: &AudioFrameParameters{Format: PCMFormat{SampleRate: 8000, Channels: 1}},
	}}}}
	o := s.audioObserver()
	for _, test := range []struct {
		getter                func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams
		rate, channels, count int
		write                 bool
	}{
		{o.OnGetPlaybackAudioFrameParam, 48000, 2, 1920, true},
		{o.OnGetRecordAudioFrameParam, 32000, 1, 320, false},
		{o.OnGetMixedAudioFrameParam, 44100, 2, 882, false},
		{o.OnGetEarMonitoringAudioFrameParam, 8000, 1, 80, false},
	} {
		p := test.getter(nil)
		if p.SampleRate != test.rate || p.Channels != test.channels || p.SamplesPerCall != test.count || (p.Mode == rtc.RawAudioFrameOpModeReadWrite) != test.write {
			t.Fatal(p)
		}
	}
}

func TestAudioObserverHookIsolationAndWriteback(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		for _, writable := range []bool{false, true} {
			s := &Session{audio: newStream[AudioFrame](1), events: newStream[Event](1), cfg: ChannelConfig{Audio: AudioConfig{Receive: PCMFormat{SampleRate: 16000, Channels: 1}, ReceiveParameters: &AudioReceiveParameters{Playback: &AudioFrameParameters{ReadWrite: writable}}}}}
			var retained []byte
			s.cfg.Audio.Callbacks = &AudioCallbacks{OnFrame: func(f AudioFrame) bool { retained = f.Data; f.Data[0] = 7; f.SampleRate = 48000; return accepted }}
			f := &rtc.AudioFrame{Buffer: []byte{1, 2}, SamplesPerSec: 16000, Channels: 1}
			if got := s.audioObserver().OnPlaybackAudioFrame(nil, "room", f); got != accepted {
				t.Fatal(got)
			}
			want := byte(1)
			if accepted && writable {
				want = 7
			}
			if f.Buffer[0] != want {
				t.Fatal("wrong native writeback", f.Buffer)
			}
			retained[0] = 9 // Even a contract-breaking retained slice cannot mutate the queue.
			out, err := s.audio.Next(context.Background())
			if err != nil || out.Data[0] != want || out.SampleRate != 16000 {
				t.Fatal(out, err)
			}
			s.audio.close()
			s.events.close()
		}
	}
}

func TestBeforeMixingHookIsReadOnlyAndPanicsReject(t *testing.T) {
	s := &Session{audio: newStream[AudioFrame](2), events: newStream[Event](1), cfg: ChannelConfig{Audio: AudioConfig{Callbacks: &AudioCallbacks{OnFrame: func(f AudioFrame) bool { f.Data[0] = 7; f.Speech[0] = 8; return true }}}}}
	defer s.audio.close()
	defer s.events.close()
	f := &rtc.AudioFrame{Buffer: []byte{1, 2}}
	speech := &rtc.AudioFrame{Buffer: []byte{3, 4}}
	if !s.audioObserver().OnPlaybackAudioFrameBeforeMixing(nil, "room", "alice", f, rtc.VadStateSpeeking, speech) {
		t.Fatal("rejected")
	}
	out, err := s.audio.Next(context.Background())
	if err != nil || out.Data[0] != 1 || out.Speech[0] != 3 || f.Buffer[0] != 1 || speech.Buffer[0] != 3 {
		t.Fatal(out, err)
	}
	s.cfg.Audio.Callbacks.OnFrame = func(AudioFrame) bool { panic("bad application") }
	if s.audioObserver().OnPlaybackAudioFrameBeforeMixing(nil, "room", "alice", f, 0, nil) {
		t.Fatal("panic accepted")
	}
	e, err := s.events.Next(context.Background())
	if err != nil || e.Kind != CallbackPanic {
		t.Fatal(e, err)
	}
}

func TestEncodedVideoHookDecisionsAndIsolation(t *testing.T) {
	s := &Session{encodedVideo: newStream[ReceivedEncodedVideo](2), events: newStream[Event](1)}
	defer s.encodedVideo.close()
	defer s.events.close()
	var retained, retainedSEI []byte
	s.cfg.Video.OnEncodedFrame = func(frame ReceivedEncodedVideo) bool {
		retained, retainedSEI = frame.Frame.Data, frame.Frame.SEI
		retained[0], retainedSEI[0] = 9, 9
		frame.Frame.UID = 123
		return false
	}
	data := []byte{1}
	info := &rtc.EncodedVideoFrameInfo{Uid: 42, SeiData: []byte{2}}
	if s.encodedVideoObserver().OnEncodedVideoFrame("room", "alice", data, info) {
		t.Fatal("decision ignored")
	}
	retained[0], retainedSEI[0] = 8, 8
	out, err := s.encodedVideo.Next(context.Background())
	if err != nil || out.Frame.Data[0] != 1 || out.Frame.SEI[0] != 2 || out.Frame.UID != 42 || data[0] != 1 || info.SeiData[0] != 2 {
		t.Fatal(out, err)
	}
	s.cfg.Video.OnEncodedFrame = func(ReceivedEncodedVideo) bool { panic("bad application") }
	if s.encodedVideoObserver().OnEncodedVideoFrame("room", "alice", data, info) {
		t.Fatal("panic accepted")
	}
	e, err := s.events.Next(context.Background())
	if err != nil || e.Kind != CallbackPanic {
		t.Fatal(e, err)
	}
}
