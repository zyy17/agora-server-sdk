package agorasdk

import (
	"context"
	"errors"
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"strings"
	"sync"
)

var runtimeOwner struct {
	sync.Mutex
	active bool
}

// Service owns the native runtime and every Session joined through it.
// Close is idempotent, closes all sessions, and then releases the runtime.
type Service struct {
	mu              sync.Mutex
	closed          bool
	video           bool
	sessions        map[*Session]struct{}
	closeOnce       sync.Once
	closeErr        error
	release         func() error
	processors      map[*AudioProcessor]struct{}
	audioProcessing bool
}

func NewService(cfg Config) (*Service, error) {
	if len(cfg.AppID) != 32 || strings.ContainsRune(cfg.AppID, 0) {
		return nil, invalid("app ID", "must be 32 characters")
	}
	native := rtc.NewAgoraServiceConfig()
	native.IdleMode = false
	if cfg.AudioProcessing {
		native.APMModel = 1
		native.APMConfig = rtc.NewAPMConfig()
	}
	native.AppId = cfg.AppID
	native.UseStringUid = true
	native.EnableVideo = cfg.Video
	native.LogPath = cfg.LogPath
	native.ConfigDir = cfg.ConfigDir
	native.DataDir = cfg.DataDir
	if cfg.ConfigureNative != nil {
		if err := cfg.ConfigureNative(native); err != nil {
			return nil, err
		}
	}
	if !native.UseStringUid {
		return nil, invalid("UseStringUid", "must remain true for the high-level API")
	}
	// Idle mode delays destruction beyond Session.Close and violates ownership.
	if native.IdleMode {
		return nil, invalid("IdleMode", "must remain false for deterministic Close")
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
	return &Service{video: native.EnableVideo, sessions: make(map[*Session]struct{}), processors: make(map[*AudioProcessor]struct{}), audioProcessing: native.APMModel > 0, release: func() error {
		runtimeOwner.Lock()
		defer runtimeOwner.Unlock()
		err := check("release service", rtc.Release())
		if err == nil {
			runtimeOwner.active = false
		}
		return err
	}}, nil
}

// Join waits for connection success, failure, timeout, or context cancellation.
// The returned session stays open until Close; cancellation after Join returns
// does not silently disconnect it. An empty token is valid for unsecured projects.
func (s *Service) Join(ctx context.Context, cfg ChannelConfig) (*Session, error) {
	if ctx == nil {
		return nil, invalid("context", "must not be nil")
	}
	if err := ctx.Err(); err != nil {
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
