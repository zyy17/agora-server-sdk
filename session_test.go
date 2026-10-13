package agorasdk

import (
	"context"
	"errors"
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeTransport struct {
	connect                         func() int
	disconnectCode                  int
	connects, disconnects, releases atomic.Int32
	mu                              sync.Mutex
	order                           []string
}

func (f *fakeTransport) Connect(_, _, _, _ string) int {
	f.connects.Add(1)
	if f.connect != nil {
		return f.connect()
	}
	return 0
}
func (f *fakeTransport) Disconnect() int {
	f.disconnects.Add(1)
	f.mu.Lock()
	f.order = append(f.order, "disconnect")
	f.mu.Unlock()
	return f.disconnectCode
}
func (f *fakeTransport) Release() {
	f.releases.Add(1)
	f.mu.Lock()
	f.order = append(f.order, "release")
	f.mu.Unlock()
}
func testSession(t *testing.T) (*Session, *fakeTransport) {
	t.Helper()
	cfg, err := normalized(validChannel())
	if err != nil {
		t.Fatal(err)
	}
	s := emptySession(nil, cfg)
	f := new(fakeTransport)
	s.transport = f
	return s, f
}
func TestJoinWait(t *testing.T) {
	t.Run("success synchronous callback", func(t *testing.T) {
		s, f := testSession(t)
		defer s.Close()
		f.connect = func() int { s.signalReady(nil); s.signalReady(errors.New("late")); return 0 }
		if err := s.connect(context.Background()); err != nil {
			t.Fatal(err)
		}
		if s.cfg.Token != "" {
			t.Fatal("token retained")
		}
	})
	t.Run("native failure", func(t *testing.T) {
		s, f := testSession(t)
		defer s.Close()
		f.connect = func() int { return -17 }
		var e *Error
		if err := s.connect(context.Background()); !errors.As(err, &e) || e.Code != -17 {
			t.Fatal(err)
		}
	})
	t.Run("asynchronous failure", func(t *testing.T) {
		s, f := testSession(t)
		defer s.Close()
		f.connect = func() int { s.connectionObserver().OnConnectionFailure(nil, nil, 8); return 0 }
		if err := s.connect(context.Background()); err == nil {
			t.Fatal("failure ignored")
		}
	})
	t.Run("timeout", func(t *testing.T) {
		s, _ := testSession(t)
		defer s.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		if err := s.connect(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	})
	t.Run("concurrent close", func(t *testing.T) {
		s, f := testSession(t)
		started := make(chan struct{})
		f.connect = func() int { close(started); return 0 }
		done := make(chan error, 1)
		go func() { done <- s.connect(context.Background()) }()
		<-started
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if !errors.Is(err, ErrClosed) {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("join blocked")
		}
	})
}
func TestCloseConcurrentIdempotent(t *testing.T) {
	s, f := testSession(t)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if f.disconnects.Load() != 1 || f.releases.Load() != 1 {
		t.Fatalf("disconnect=%d release=%d", f.disconnects.Load(), f.releases.Load())
	}
	if strings.Join(f.order, ",") != "disconnect,release" {
		t.Fatal(f.order)
	}
	if _, err := s.Audio().Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if err := s.WithNative(func(*rtc.RtcConnection) error { t.Fatal("called after close"); return nil }); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
func TestCloseReleasesAfterDisconnectError(t *testing.T) {
	s, f := testSession(t)
	f.disconnectCode = -3
	if err := s.Close(); err == nil {
		t.Fatal("disconnect error lost")
	}
	if f.releases.Load() != 1 {
		t.Fatal("release skipped")
	}
}
func TestServiceOwnsSessions(t *testing.T) {
	svc := &Service{sessions: make(map[*Session]struct{})}
	s1, f1 := testSession(t)
	s2, f2 := testSession(t)
	s1.service = svc
	s2.service = svc
	svc.sessions[s1] = struct{}{}
	svc.sessions[s2] = struct{}{}
	var released atomic.Int32
	svc.release = func() error {
		if f1.releases.Load() != 1 || f2.releases.Load() != 1 {
			t.Error("runtime released before sessions")
		}
		released.Add(1)
		return nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if released.Load() != 1 || len(svc.sessions) != 0 {
		t.Fatal("ownership cleanup failed")
	}
	if _, err := svc.Join(context.Background(), validChannel()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
func TestCopiedAudioCallback(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	f := &rtc.AudioFrame{Buffer: []byte{1, 2}, SamplesPerSec: 16000, Channels: 1, RenderTimeMs: 17}
	speech := &rtc.AudioFrame{Buffer: []byte{3, 4}}
	s.audioObserver().OnPlaybackAudioFrameBeforeMixing(nil, "channel", "alice", f, rtc.VadState(1), speech)
	f.Buffer[0] = 9
	speech.Buffer[0] = 8
	got, err := s.Audio().Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Data[0] != 1 || got.Speech[0] != 3 || got.Channel != "channel" || got.UserID != "alice" || got.TimestampMillis != 17 {
		t.Fatalf("%+v", got)
	}
}
func TestCopiedMessageCallback(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	data := []byte{1, 2}
	s.localObserver().OnStreamMessage(nil, "bob", 9, data, 42)
	data[0] = 8
	got, err := s.Messages().Next(context.Background())
	if err != nil || got.Data[0] != 1 || got.UserID != "bob" || got.SentAtMillis != 42 {
		t.Fatal(got, err)
	}
}
func TestCopiedVideoCallback(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	f := &rtc.VideoFrame{YBuffer: []byte{1}, UBuffer: []byte{2}, VBuffer: []byte{3}, MetadataBuffer: []byte{4}}
	s.videoObserver().OnFrame("room", "alice", f)
	f.YBuffer[0] = 7
	f.MetadataBuffer[0] = 8
	got, err := s.Video().Next(context.Background())
	if err != nil || got.Y[0] != 1 || got.Metadata[0] != 4 {
		t.Fatal(got, err)
	}
}
func TestCallbackQueueNeverBlocksJoin(t *testing.T) {
	s, f := testSession(t)
	defer s.Close()
	for i := 0; i < 1000; i++ {
		s.event(Event{Kind: UserJoined})
	}
	f.connect = func() int { s.connectionObserver().OnConnected(nil, nil, 0); return 0 }
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.connect(ctx); err != nil {
		t.Fatal(err)
	}
	if s.Events().Stats().Dropped == 0 {
		t.Fatal("expected drop metric")
	}
}
func TestCallbacksRaceClose(t *testing.T) {
	s, _ := testSession(t)
	var wg sync.WaitGroup
	obs := s.connectionObserver()
	audio := s.audioObserver()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				obs.OnUserJoined(nil, "bob")
				audio.OnPlaybackAudioFrameBeforeMixing(nil, "room", "bob", &rtc.AudioFrame{Buffer: []byte{1, 2}}, 0, nil)
				_ = s.Info()
			}
		}()
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
func TestCapabilityFallbackDoesNotMutateNative(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	if got := s.connectionObserver().OnAIQoSCapabilityMissing(nil, 7); got != -1 {
		t.Fatal("native callback must not rebuild audio tracks", got)
	}
	e, err := s.Events().Next(context.Background())
	if err != nil || e.Kind != CapabilityFallback || e.Reason != 7 {
		t.Fatal(e, err)
	}
}

// TestNativeSmoke initializes and releases the native runtime but never connects.
func TestNativeSmoke(t *testing.T) {
	t.Chdir(t.TempDir())
	svc, err := NewService(Config{AppID: strings.Repeat("0", 32)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(Config{AppID: strings.Repeat("0", 32)}); !errors.Is(err, ErrServiceActive) {
		t.Fatal("duplicate owner", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.Join(ctx, validChannel()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCanceledConnectDoesNotStartTransport(t *testing.T) {
	s, f := testSession(t)
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.connect(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if f.connects.Load() != 0 {
		t.Fatal("started canceled join")
	}
}
func TestEncodedAudioRejectsInvalidBeforeNative(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	if err := s.SendEncodedAudio(EncodedAudioFrame{Data: []byte{1}, SampleRate: 48000, Channels: 1, Codec: 999}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	s.cfg.Audio.Mode = EncodedAudio
	if err := s.WaitPCM(context.Background()); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
}
