package agorasdk

import (
	"context"
	"errors"
	"strings"
	"sync"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

var runtimeOwner struct {
	sync.Mutex
	active bool
}

// Service owns the process-wide native RTC runtime, its Sessions, and its
// AudioProcessors. Construct it with NewService; the zero value is not usable.
// Service methods serialize lifecycle changes, but callers must synchronize
// their own buffers and configuration values. Do not mix this ownership model
// with direct rtc.Initialize or rtc.Release calls.
type Service struct {
	mu                      sync.Mutex
	closed                  bool
	video                   bool
	sessions                map[*Session]struct{}
	closeOnce               sync.Once
	closeErr                error
	release                 func() error
	processors              map[*AudioProcessor]struct{}
	audioProcessing         bool
	identityMode            IdentityMode
	setParameters           func(string) int
	localAccessPoint        *rtc.LocalAccessPointConfiguration
	localAccessPointApplied bool
}

func serviceConfig(cfg Config) (*rtc.AgoraServiceConfig, error) {
	if len(cfg.AppID) != 32 || strings.ContainsRune(cfg.AppID, 0) {
		return nil, invalid("app ID", "must be 32 characters")
	}
	native := rtc.NewAgoraServiceConfig()
	native.IdleMode = cfg.IdleMode
	if cfg.AudioProcessing {
		native.APMModel = 1
		native.APMConfig = rtc.NewAPMConfig()
	}
	native.AppId = cfg.AppID
	if cfg.IdentityMode > NumericIdentity {
		return nil, invalid("identity mode", "is unknown")
	}
	native.UseStringUid = cfg.IdentityMode == StringIdentity
	native.EnableVideo = cfg.Video
	native.LogPath = cfg.LogPath
	native.ConfigDir = cfg.ConfigDir
	native.DataDir = cfg.DataDir
	if cfg.ConfigureNative != nil {
		if err := cfg.ConfigureNative(native); err != nil {
			return nil, err
		}
	}
	if native.UseStringUid != (cfg.IdentityMode == StringIdentity) {
		return nil, invalid("UseStringUid", "must match Config.IdentityMode")
	}
	if cfg.LocalAccessPoint != nil {
		if err := validateLocalAccessPoint(*cfg.LocalAccessPoint); err != nil {
			return nil, err
		}
	}
	return native, nil
}

// NewService initializes the native RTC runtime with cfg.
// It returns ErrServiceActive if another high-level Service owns the runtime,
// an error wrapping ErrInvalidConfig for invalid settings, the configuration
// hook's error, or an *Error for native initialization failure.
// A successful caller must call Service.Close, even if no channel is joined.
func NewService(cfg Config) (*Service, error) {
	native, err := serviceConfig(cfg)
	if err != nil {
		return nil, err
	}
	runtimeOwner.Lock()
	defer runtimeOwner.Unlock()
	if runtimeOwner.active {
		return nil, ErrServiceActive
	}
	if err := check("initialize service", rtc.Initialize(native)); err != nil {
		return nil, err
	}
	runtimeOwner.active = true
	return &Service{localAccessPoint: copyLocalAccessPoint(cfg.LocalAccessPoint), setParameters: rtc.GetAgoraParameter().SetParameters, identityMode: cfg.IdentityMode, video: native.EnableVideo, sessions: make(map[*Session]struct{}), processors: make(map[*AudioProcessor]struct{}), audioProcessing: native.APMModel > 0, release: func() error {
		runtimeOwner.Lock()
		defer runtimeOwner.Unlock()
		err := check("release service", rtc.Release())
		if err == nil {
			runtimeOwner.active = false
		}
		return err
	}}, nil
}

// Join creates a session and waits for connection success, failure, the earlier
// of ctx's deadline and ChannelConfig.JoinTimeout, or context cancellation.
// ctx must not be nil. Failed joins release the partially created session;
// the returned error may join the connection and cleanup errors.
//
// A successfully returned Session remains open until Session.Close or
// Service.Close; subsequent cancellation of ctx does not disconnect it.
// Empty tokens are valid only for projects that do not require authentication.
// IdentityMode, channel, user identity, token privileges, and peer configuration
// must agree. Native connection success does not prove media interoperability.
func (s *Service) Join(ctx context.Context, cfg ChannelConfig) (*Session, error) {
	if ctx == nil {
		return nil, invalid("context", "must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.identityMode.validateUserID(cfg.UserID, true); err != nil {
		return nil, err
	}
	cfg, err := normalized(cfg)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	if !s.video && (cfg.Video.Mode != NoVideo || cfg.Receive.Video || cfg.Receive.EncodedVideo) {
		s.mu.Unlock()
		return nil, invalid("video", "requires Config.Video=true")
	}
	session, err := newSession(s, cfg)
	if err == nil && s.localAccessPoint != nil && !s.localAccessPointApplied {
		err = check("set local access point", rtc.SetLocalAccessPoint(s.localAccessPoint))
		if err == nil {
			s.localAccessPointApplied = true
		}
		if err != nil {
			// Not yet registered with the Service; avoid reentering its lock.
			session.service = nil
			err = errors.Join(err, session.Close())
		}
	}
	if err == nil {
		s.sessions[session] = struct{}{}
	}
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.JoinTimeout)
	defer cancel()
	if err = session.connect(ctx); err != nil {
		return nil, errors.Join(err, session.Close())
	}
	return session, nil
}

// Close closes all owned processors and sessions, then releases the native runtime.
// It is safe to call repeatedly or concurrently; every call returns the same
// combined cleanup error. Release failure may leave process ownership reserved,
// so do not assume a new Service can be created after a failed Close.
// Close waits for native cleanup and is not bounded by a context.
func (s *Service) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		sessions := make([]*Session, 0, len(s.sessions))
		for c := range s.sessions {
			sessions = append(sessions, c)
		}
		processors := make([]*AudioProcessor, 0, len(s.processors))
		for p := range s.processors {
			processors = append(processors, p)
		}
		s.mu.Unlock()
		for _, p := range processors {
			s.closeErr = errors.Join(s.closeErr, p.Close())
		}
		for _, c := range sessions {
			s.closeErr = errors.Join(s.closeErr, c.Close())
		}
		if s.release != nil {
			s.closeErr = errors.Join(s.closeErr, s.release())
		}
	})
	return s.closeErr
}

func (s *Service) remove(c *Session) { s.mu.Lock(); delete(s.sessions, c); s.mu.Unlock() }

// SetParameters applies a native process-level JSON object under the service
// lifecycle lock. Call it before Join for keys that affect new connections.
// The wrapper validates JSON object syntax only; supported keys and timing
// requirements belong to the pinned native SDK. It returns ErrClosed after
// Close, ErrUnsupported when the parameter interface is unavailable, or *Error
// for a nonzero native result.
func (s *Service) SetParameters(parameters string) error {
	if err := validateParameters(parameters); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.setParameters == nil {
		return ErrUnsupported
	}
	return check("set service parameters", s.setParameters(parameters))
}
