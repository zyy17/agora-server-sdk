package agorasdk

import (
	"context"
	"errors"
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"sync"
	"sync/atomic"
)

type sessionTransport interface {
	Connect(string, string, string, string) int
	Disconnect() int
	Release()
}

// Session owns a connected RTC channel and its media streams. Use Close, not
// native Release. Methods may run concurrently; native operations are serialized.
type Session struct {
	service                        *Service
	cfg                            ChannelConfig
	native                         *rtc.RtcConnection
	transport                      sessionTransport
	mu                             sync.Mutex
	closed                         bool
	audioPublished, videoPublished bool
	closeOnce                      sync.Once
	closeErr                       error
	done                           chan struct{}
	ready                          chan error
	readyOnce                      sync.Once
	info                           atomic.Pointer[ConnectionInfo]
	events                         *Stream[Event]
	audio                          *Stream[AudioFrame]
	video                          *Stream[ReceivedVideoFrame]
	encodedAudio                   *Stream[ReceivedEncodedAudio]
	encodedVideo                   *Stream[ReceivedEncodedVideo]
	messages                       *Stream[Message]
}

// ConnectionInfo is a copied snapshot, safe to retain across callbacks and Close.
type ConnectionInfo struct {
	Channel, UserID string
	State, Reason   int
	ConnectionID    uint
}

func emptySession(service *Service, cfg ChannelConfig) *Session {
	s := &Session{service: service, cfg: cfg, done: make(chan struct{}), ready: make(chan error, 1),
		events: newStream[Event](cfg.QueueCapacity), audio: newStream[AudioFrame](cfg.QueueCapacity),
		video: newStream[ReceivedVideoFrame](cfg.QueueCapacity), encodedAudio: newStream[ReceivedEncodedAudio](cfg.QueueCapacity),
		encodedVideo: newStream[ReceivedEncodedVideo](cfg.QueueCapacity), messages: newStream[Message](cfg.QueueCapacity)}
	s.info.Store(&ConnectionInfo{Channel: cfg.Channel, UserID: cfg.UserID})
	return s
}
func newSession(service *Service, cfg ChannelConfig) (*Session, error) {
	s := emptySession(service, cfg)
	connCfg := &rtc.RtcConnectionConfig{
		AutoSubscribeAudio: !cfg.Receive.Disabled && !cfg.Receive.ManualSubscriptions,
		AutoSubscribeVideo: (cfg.Receive.Video || cfg.Receive.EncodedVideo) && !cfg.Receive.ManualSubscriptions,
		ClientRole:         rtc.ClientRoleBroadcaster, ChannelProfile: rtc.ChannelProfileLiveBroadcasting,
		ChannelType: rtc.ChannelType(cfg.Mode), AudioRecvEncodedFrame: cfg.Receive.EncodedAudio,
	}
	if cfg.Role == Audience {
		connCfg.ClientRole = rtc.ClientRoleAudience
	}
	pub := rtc.NewRtcConPublishConfig()
	pub.IsPublishAudio = cfg.Audio.Mode != NoAudio && cfg.Role == Broadcaster
	pub.IsPublishVideo = cfg.Video.Mode != NoVideo && cfg.Role == Broadcaster
	pub.AudioPublishType = rtc.AudioPublishTypePcm
	if cfg.Audio.Mode == EncodedAudio {
		pub.AudioPublishType = rtc.AudioPublishTypeEncodedPcm
	}
	if cfg.Audio.Mode == NoAudio {
		pub.AudioPublishType = rtc.AudioPublishTypeNoPublish
	}
	pub.VideoPublishType = rtc.VideoPublishType(cfg.Video.Mode)
	pub.AudioScenario = cfg.Audio.Scenario.native()
	pub.AudioProfile = cfg.Audio.Profile
	if cfg.Video.EncodedSender != nil {
		pub.VideoEncodedImageSenderOptions = cfg.Video.EncodedSender
	}
	if cfg.ConfigureNative != nil {
		if err := cfg.ConfigureNative(connCfg, pub); err != nil {
			return nil, err
		}
	}
	if connCfg.ChannelType == rtc.ChannelTypeLargeScale && (connCfg.AutoSubscribeAudio || connCfg.AutoSubscribeVideo) {
		return nil, invalid("large-scale channel", "cannot auto-subscribe")
	}
	s.native = rtc.NewRtcConnection(connCfg, pub)
	if s.native == nil {
		return nil, &Error{Op: "create connection", Code: -1}
	}
	s.transport = s.native
	// A failed setup must release every native object before returning.
	if err := s.configure(); err != nil {
		s.transport.Disconnect()
		s.transport.Release()
		s.closeStreams()
		return nil, err
	}
	return s, nil
}
func (s *Session) configure() error {
	steps := []struct {
		op  string
		run func() int
	}{
		{"register connection observer", func() int { return s.native.RegisterObserver(s.connectionObserver()) }},
		{"register local user observer", func() int { return s.native.RegisterLocalUserObserver(s.localObserver()) }},
	}
	if !s.cfg.Receive.Disabled {
		steps = append(steps, struct {
			op  string
			run func() int
		}{"configure received PCM", func() int {
			return s.native.GetLocalUser().SetPlaybackAudioFrameBeforeMixingParameters(s.cfg.Audio.Channels, s.cfg.Audio.SampleRate)
		}})
		steps = append(steps, struct {
			op  string
			run func() int
		}{"register audio observer", func() int {
			enabled := 0
			if s.cfg.Audio.VAD {
				enabled = 1
			}
			return s.native.RegisterAudioFrameObserver(s.audioObserver(), enabled, s.cfg.Audio.VADConfig)
		}})
	}
	if s.cfg.Receive.Video {
		steps = append(steps, struct {
			op  string
			run func() int
		}{"register video observer", func() int { return s.native.RegisterVideoFrameObserver(s.videoObserver()) }})
	}
	if s.cfg.Receive.EncodedVideo {
		steps = append(steps, struct {
			op  string
			run func() int
		}{"register encoded video observer", func() int { return s.native.RegisterVideoEncodedFrameObserver(s.encodedVideoObserver()) }})
	}
	if s.cfg.Receive.EncodedAudio {
		steps = append(steps, struct {
			op  string
			run func() int
		}{"register encoded audio observer", func() int { return s.native.RegisterAudioEncodedFrameObserver(s.encodedAudioObserver()) }})
	}
	if s.cfg.Video.Encoder != nil {
		steps = append(steps, struct {
			op  string
			run func() int
		}{"configure video encoder", func() int { return s.native.SetVideoEncoderConfiguration(s.cfg.Video.Encoder) }})
	}
	if s.cfg.Encryption != nil {
		steps = append(steps, struct {
			op  string
			run func() int
		}{"enable encryption", func() int { return s.native.EnableEncryption(1, s.cfg.Encryption) }})
	}
	for _, step := range steps {
		if err := check(step.op, step.run()); err != nil {
			return err
		}
	}
	return nil
}
func (s *Session) connect(ctx context.Context) error {
	if ctx == nil {
		return invalid("context", "must not be nil")
	}
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return err
	}
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	err := check("connect", s.transport.Connect(s.cfg.Token, s.cfg.Channel, s.cfg.UserID, s.cfg.Info))
	// Avoid retaining credentials for the lifetime of the connection.
	s.cfg.Token = ""
	s.cfg.Encryption = nil
	s.mu.Unlock()
	if err != nil {
		return err
	}
	select {
	case err := <-s.ready:
		select {
		case <-s.done:
			return ErrClosed
		default:
		}
		return err
	case <-s.done:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Session) signalReady(err error)                       { s.readyOnce.Do(func() { s.ready <- err }) }
func (s *Session) Events() *Stream[Event]                      { return s.events }
func (s *Session) Audio() *Stream[AudioFrame]                  { return s.audio }
func (s *Session) Video() *Stream[ReceivedVideoFrame]          { return s.video }
func (s *Session) EncodedAudio() *Stream[ReceivedEncodedAudio] { return s.encodedAudio }
func (s *Session) EncodedVideo() *Stream[ReceivedEncodedVideo] { return s.encodedVideo }
func (s *Session) Messages() *Stream[Message]                  { return s.messages }
func (s *Session) Done() <-chan struct{}                       { return s.done }
func (s *Session) Info() ConnectionInfo                        { return *s.info.Load() }
func (s *Session) closeStreams() {
	s.events.close()
	s.audio.close()
	s.video.close()
	s.encodedAudio.close()
	s.encodedVideo.close()
	s.messages.close()
}
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.done)
		s.signalReady(ErrClosed)
		if s.transport != nil {
			s.closeErr = check("disconnect", s.transport.Disconnect())
			s.transport.Release()
		}
		s.native = nil
		s.transport = nil
		s.mu.Unlock()
		s.closeStreams()
		if s.service != nil {
			s.service.remove(s)
		}
	})
	return s.closeErr
}
func (s *Session) withNative(op string, f func(*rtc.RtcConnection) int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	return check(op, f(s.native))
}

// WithNative provides serialized advanced access. Do not retain native handles,
// invoke Session methods inside fn, destroy resources, or replace owned observers.
// Raw APIs use native codes and do not have high-level validation/ownership guarantees.
func (s *Session) WithNative(fn func(*rtc.RtcConnection) error) error {
	if fn == nil {
		return invalid("native function", "must not be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	return fn(s.native)
}
func (s *Session) RenewToken(token string) error {
	if token == "" {
		return invalid("token", "must not be empty")
	}
	return s.withNative("renew token", func(c *rtc.RtcConnection) int { return c.RenewToken(token) })
}

// SetParameters passes an official native JSON parameter object.
func (s *Session) SetParameters(parameters string) error {
	if err := validateParameters(parameters); err != nil {
		return err
	}
	return s.withNative("set parameters", func(c *rtc.RtcConnection) int { return c.GetAgoraParameter().SetParameters(parameters) })
}
func (s *Session) publishAudioLocked() error {
	if s.cfg.Role == Audience || s.cfg.Audio.Mode == NoAudio {
		return ErrNotPublishing
	}
	if s.audioPublished {
		return nil
	}
	if err := check("publish audio", s.native.PublishAudio()); err != nil {
		return err
	}
	s.audioPublished = true
	return nil
}
func (s *Session) publishVideoLocked() error {
	if s.cfg.Role == Audience || s.cfg.Video.Mode == NoVideo {
		return ErrNotPublishing
	}
	if s.videoPublished {
		return nil
	}
	if err := check("publish video", s.native.PublishVideo()); err != nil {
		return err
	}
	s.videoPublished = true
	return nil
}
func (s *Session) PublishAudio() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	return s.publishAudioLocked()
}
func (s *Session) PublishVideo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	return s.publishVideoLocked()
}
func (s *Session) UnpublishAudio() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if !s.audioPublished {
		return nil
	}
	if err := check("unpublish audio", s.native.UnpublishAudio()); err != nil {
		return err
	}
	s.audioPublished = false
	return nil
}
func (s *Session) UnpublishVideo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if !s.videoPublished {
		return nil
	}
	if err := check("unpublish video", s.native.UnpublishVideo()); err != nil {
		return err
	}
	s.videoPublished = false
	return nil
}

// SessionError reports asynchronous failures on Events as well as join failures.
func eventError(op string, code int) error {
	if code == 0 {
		return errors.New("agora: " + op)
	}
	return &Error{Op: op, Code: code}
}
