package agorasdk

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

type sessionTransport interface {
	Connect(string, string, string, string) int
	Disconnect() int
	Release()
	PublishAudio() int
	UnpublishAudio() int
	PushAudioPcmData([]byte, int, int, int64) int
}

// Session owns one RTC connection and its receive streams. Obtain it through
// Service.Join; its zero value is not usable. Close it explicitly when finished.
//
// Native operations are serialized against each other and Close. Stream readers
// may run concurrently, but compete for items rather than receiving broadcasts.
// The wrapper does not serialize whole PlayPCM calls: concurrent producers can
// interleave audio, so applications should use one producer per track.
// Callers remain responsible for synchronization of their own buffers.
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
	role                           atomic.Uint32
	fallback                       atomic.Int32
	events                         *Stream[Event]
	audio                          *Stream[AudioFrame]
	video                          *Stream[ReceivedVideoFrame]
	encodedAudio                   *Stream[ReceivedEncodedAudio]
	encodedVideo                   *Stream[ReceivedEncodedVideo]
	messages                       *Stream[Message]
}

// ConnectionInfo is a copied connection snapshot, safe to retain across callbacks
// and Close. Native state and reason values are passed through unchanged.
type ConnectionInfo struct {
	// Channel and UserID identify the current native connection.
	// UserID reflects an allocated numeric UID after a successful join with "0".
	Channel, UserID string
	// State and Reason are the latest native connection state and reason codes.
	State, Reason int
	// ConnectionID is the native connection identifier, not the remote user identity.
	ConnectionID uint
}

func emptySession(service *Service, cfg ChannelConfig) *Session {
	s := &Session{service: service, cfg: cfg, done: make(chan struct{}), ready: make(chan error, 1),
		events: newStream[Event](cfg.QueueCapacity), audio: newStream[AudioFrame](cfg.QueueCapacity),
		video: newStream[ReceivedVideoFrame](cfg.QueueCapacity), encodedAudio: newStream[ReceivedEncodedAudio](cfg.QueueCapacity),
		encodedVideo: newStream[ReceivedEncodedVideo](cfg.QueueCapacity), messages: newStream[Message](cfg.QueueCapacity)}
	s.info.Store(&ConnectionInfo{Channel: cfg.Channel, UserID: cfg.UserID})
	s.role.Store(uint32(cfg.Role))
	s.storeAudioFallback(cfg.Audio.Fallback)
	return s
}

func newSession(service *Service, cfg ChannelConfig) (*Session, error) {
	s := emptySession(service, cfg)
	connCfg, pub, err := sessionNativeConfig(cfg)
	if err != nil {
		return nil, err
	}
	s.native = rtc.NewRtcConnection(connCfg, pub)
	if s.native == nil {
		return nil, &Error{Op: "create connection", Code: -1}
	}
	s.transport = s.native
	s.cfg.Mode = ChannelMode(connCfg.ChannelType)
	// A failed setup must release every native object before returning.
	if err := s.configure(); err != nil {
		s.transport.Disconnect()
		s.transport.Release()
		s.closeStreams()
		return nil, err
	}
	return s, nil
}

func sessionNativeConfig(cfg ChannelConfig) (*rtc.RtcConnectionConfig, *rtc.RtcConnectionPublishConfig, error) {
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
			return nil, nil, err
		}
	}
	if err := validateNativeOwnership(cfg, connCfg, pub); err != nil {
		return nil, nil, err
	}
	if connCfg.ChannelType == rtc.ChannelTypeLargeScale && (connCfg.AutoSubscribeAudio || connCfg.AutoSubscribeVideo) {
		return nil, nil, invalid("large-scale channel", "cannot auto-subscribe")
	}
	return connCfg, pub, nil
}

func (s *Session) configure() error {
	if err := check("register connection observer", s.native.RegisterObserver(s.connectionObserver())); err != nil {
		return err
	}
	if err := check("register local user observer", s.native.RegisterLocalUserObserver(s.localObserver())); err != nil {
		return err
	}
	if err := s.configurePCMReception(); err != nil {
		return err
	}
	if s.cfg.Receive.Video {
		if err := check("register video observer", s.native.RegisterVideoFrameObserver(s.videoObserver())); err != nil {
			return err
		}
	}
	if s.cfg.Receive.EncodedVideo {
		if err := check("register encoded video observer", s.native.RegisterVideoEncodedFrameObserver(s.encodedVideoObserver())); err != nil {
			return err
		}
	}
	if s.cfg.Receive.EncodedAudio {
		if err := check("register encoded audio observer", s.native.RegisterAudioEncodedFrameObserver(s.encodedAudioObserver())); err != nil {
			return err
		}
	}
	if s.cfg.Video.Encoder != nil {
		if err := check("configure video encoder", s.native.SetVideoEncoderConfiguration(s.cfg.Video.Encoder)); err != nil {
			return err
		}
	}
	if s.cfg.Encryption != nil {
		return check("enable encryption", s.native.EnableEncryption(1, s.cfg.Encryption))
	}
	return nil
}

func (s *Session) configurePCMReception() error {
	if s.cfg.Receive.Disabled {
		return nil
	}
	format := s.cfg.Audio.Receive
	if err := check("configure received PCM", s.native.GetLocalUser().SetPlaybackAudioFrameBeforeMixingParameters(format.Channels, format.SampleRate)); err != nil {
		return err
	}
	enabled := 0
	if s.cfg.Audio.VAD {
		enabled = 1
	}
	return check("register audio observer", s.native.RegisterAudioFrameObserver(s.audioObserver(), enabled, s.cfg.Audio.VADConfig))
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

func (s *Session) signalReady(err error) { s.readyOnce.Do(func() { s.ready <- err }) }

// Events returns the session's bounded asynchronous event queue.
// Consume it promptly for token renewal and failures; overflow drops oldest
// events. Native callbacks never wait for consumers.
func (s *Session) Events() *Stream[Event] { return s.events }

// Audio returns copied PCM frames from the observation points enabled by Receive.
// Filter AudioFrame.Position and UserID when processing individual speakers.
func (s *Session) Audio() *Stream[AudioFrame] { return s.audio }

// Video returns copied decoded video frames when Receive.Video is enabled.
func (s *Session) Video() *Stream[ReceivedVideoFrame] { return s.video }

// EncodedAudio returns copied encoded packets when Receive.EncodedAudio is enabled.
func (s *Session) EncodedAudio() *Stream[ReceivedEncodedAudio] { return s.encodedAudio }

// EncodedVideo returns copied encoded frames when Receive.EncodedVideo is enabled.
func (s *Session) EncodedVideo() *Stream[ReceivedEncodedVideo] { return s.encodedVideo }

// Messages returns received RTC data-stream messages. This is distinct from RTM.
func (s *Session) Messages() *Stream[Message] { return s.messages }

// Done returns a channel closed when Close begins. It does not signal completion
// of native cleanup; call Close and inspect its result to wait for cleanup.
func (s *Session) Done() <-chan struct{} { return s.done }

// Info returns a point-in-time copy of the latest connection information.
// It remains readable after Close and does not require access to native handles.
func (s *Session) Info() ConnectionInfo { return *s.info.Load() }

func (s *Session) closeStreams() {
	s.events.close()
	s.audio.close()
	s.video.close()
	s.encodedAudio.close()
	s.encodedVideo.close()
	s.messages.close()
}

// Close disconnects and releases the native connection, closes all receive
// streams, and removes the session from its Service. It is idempotent and safe
// for concurrent calls. Every call returns the same disconnect error, if any.
//
// Done closes before cleanup finishes. Buffered receive values remain available
// through Stream.Next with a live context until io.EOF. With Config.IdleMode,
// native handle destruction is briefly deferred; Service.Close drains it.
// Close is not a remote
// playback acknowledgement and does not guarantee pending sends were delivered.
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

// WithNative calls fn while holding the session's native-operation lock.
// fn must not be nil. The borrowed connection and its child handles must not
// escape fn. Do not reenter Session methods, release resources, or replace
// observers owned by the wrapper; these actions can deadlock or break ownership.
//
// The role and local publication state are reconciled after fn returns, even
// when it returns an error after a partially successful native operation.
// fn's error is returned unchanged; a closed session returns ErrClosed.
// Raw operations retain their native validation, result-code, and lifetime
// contracts, without additional high-level safety guarantees.
func (s *Session) WithNative(fn func(*rtc.RtcConnection) error) error {
	if fn == nil {
		return invalid("native function", "must not be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	err := fn(s.native)
	s.syncNativeState()
	return err
}

// RenewToken submits a nonempty replacement RTC token for this session's project,
// channel, identity namespace, and privileges. It does not generate tokens or
// renew RTM credentials. Consume TokenExpiring and TokenExpired events and obtain
// replacement credentials from a trusted server. Native acceptance is not proof
// that the token will authorize every later operation.
func (s *Session) RenewToken(token string) error {
	if token == "" || strings.ContainsRune(token, 0) {
		return invalid("token", "must be nonempty without NUL")
	}
	return s.withNative("renew token", func(c *rtc.RtcConnection) int { return c.RenewToken(token) })
}

// SetParameters submits an official native JSON parameter object for this
// connection. Only JSON object syntax is validated; supported keys and timing
// are native-version-specific. It returns ErrClosed after Close or *Error for
// a nonzero native result.
func (s *Session) SetParameters(parameters string) error {
	if err := validateParameters(parameters); err != nil {
		return err
	}
	return s.withNative("set parameters", func(c *rtc.RtcConnection) int { return c.GetAgoraParameter().SetParameters(parameters) })
}

func (s *Session) publishAudioLocked() error {
	s.syncNativeState()
	if s.Role() == Audience || s.cfg.Audio.Mode == NoAudio {
		return ErrNotPublishing
	}
	if s.audioPublished {
		return nil
	}
	if err := check("publish audio", s.transport.PublishAudio()); err != nil {
		return err
	}
	s.audioPublished = true
	return nil
}

func (s *Session) publishVideoLocked() error {
	s.syncNativeState()
	if s.Role() == Audience || s.cfg.Video.Mode == NoVideo {
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

// PublishAudio explicitly publishes the configured audio track.
// It is idempotent while published; SendPCM and SendEncodedAudio also publish
// on first use. The current Audience role and NoAudio mode return ErrNotPublishing.
func (s *Session) PublishAudio() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	return s.publishAudioLocked()
}

// PublishVideo explicitly publishes the configured video track.
// It is idempotent while published; video sends also publish on first use.
// The current Audience role and NoVideo mode return ErrNotPublishing.
func (s *Session) PublishVideo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	return s.publishVideoLocked()
}

// UnpublishAudio stops publishing the audio track. It is a no-op if already
// unpublished. A later send can publish the track again.
func (s *Session) UnpublishAudio() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if !s.audioPublished && (s.Role() == Audience || s.cfg.Audio.Mode == NoAudio) {
		return nil
	}
	// The raw call also cancels a pending retry after failed scenario rollback,
	// even if the latest snapshot reports that no track is currently published.
	if err := check("unpublish audio", s.transport.UnpublishAudio()); err != nil {
		return err
	}
	s.audioPublished = false
	return nil
}

// UnpublishVideo stops publishing the video track. It is a no-op if already
// unpublished. A later send can publish the track again.
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
