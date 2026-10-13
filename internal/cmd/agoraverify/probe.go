package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sync"
	"time"

	agorasdk "github.com/zyy17/agora-server-sdk"
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	nativeRTM "github.com/zyy17/agora-server-sdk/agora/rtm"
	"github.com/zyy17/agora-server-sdk/messaging"
)

const operationTimeout = 25 * time.Second

// The small interfaces make authentication and cleanup independently testable
// without credentials, native initialization, or network requests.
type rtcSession interface{ Close() error }
type rtcService interface {
	join(context.Context, string, string, string) (rtcSession, error)
	Close() error
}
type rtmClient interface {
	Login(context.Context, string) error
	Logout(context.Context) error
	Close() error
}

type rtcAdapter struct{ *agorasdk.Service }

func (s rtcAdapter) join(ctx context.Context, value, channel, user string) (rtcSession, error) {
	return s.Join(ctx, agorasdk.ChannelConfig{
		Channel: channel, UserID: user, Token: value, Role: agorasdk.Audience,
		Audio:   agorasdk.AudioConfig{Mode: agorasdk.NoAudio, Scenario: agorasdk.ScenarioDefault},
		Video:   agorasdk.VideoConfig{Mode: agorasdk.NoVideo},
		Receive: agorasdk.ReceiveConfig{Disabled: true}, JoinTimeout: operationTimeout,
	})
}

func newRTC(c credentials, dir string) (rtcService, error) {
	s, err := agorasdk.NewService(agorasdk.Config{
		AppID: c.appID, IdentityMode: agorasdk.StringIdentity,
		LogPath: filepath.Join(dir, "rtc.log"), ConfigDir: dir, DataDir: dir,
		ConfigureNative: func(c *rtc.AgoraServiceConfig) error {
			c.EnableAudioProcessor = false
			c.EnableAudioDevice = false
			c.EnableVideo = false
			c.AudioScenario = rtc.AudioScenarioDefault
			c.LogLevel = 0
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return rtcAdapter{s}, nil
}

func newRTM(c credentials, user, dir string) (rtmClient, error) {
	// Use the native compatibility config so this harness also builds against
	// the API already merged on main, before the expanded typed RTM options.
	cfg := nativeRTM.NewRtmConfig()
	cfg.UseStringUserId = true
	cfg.PresenceTimeout = 30
	cfg.ReconnectTimeout = 10
	cfg.LogConfig = &nativeRTM.RtmLogConfig{
		FilePath: filepath.Join(dir, "rtm.log"), Level: nativeRTM.RtmLogLevelNONE,
	}
	return messaging.New(messaging.Config{
		AppID: c.appID, UserID: user, RequestTimeout: operationTimeout, Native: cfg,
	})
}

type record func(string, error)

func checkRTC(ctx context.Context, create func() (rtcService, error), value, channel, user string, note record) {
	s, err := create()
	note("initialize", err)
	if err != nil {
		return
	}
	defer func() { note("close", s.Close()) }()
	session, err := s.join(ctx, value, channel, user)
	note("join", err)
	if err != nil {
		return
	}
	// Close invokes disconnect followed by release. This checks the local
	// leave/release result; it is not a remote leave acknowledgement.
	defer func() { note("leave", session.Close()) }()
}

func checkRTM(ctx context.Context, create func() (rtmClient, error), value string, note record) {
	c, err := create()
	note("initialize", err)
	if err != nil {
		return
	}
	defer func() { note("close", c.Close()) }()
	// Even a timed-out login may have reached Agora: always attempt logout,
	// using a fresh bounded context, and then close regardless of its result.
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		note("logout", c.Logout(cleanup))
	}()
	note("login", c.Login(ctx, value))
}

func probe(ctx context.Context, c credentials, component, dir string) int {
	var r report
	var mu sync.Mutex
	writeFailed := false
	note := func(op string, err error) {
		mu.Lock()
		defer mu.Unlock()
		r.add(op, err)
		if r.save(dir) != nil {
			writeFailed = true
		}
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		note("prepare", errors.New("random identity unavailable"))
		return 1
	}
	identity := "verify-" + hex.EncodeToString(random[:])
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	switch component {
	case "rtc":
		value, err := rtcToken(c, identity, identity)
		note("prepare", err)
		if err != nil || writeFailed {
			return 1
		}
		checkRTC(ctx, func() (rtcService, error) { return newRTC(c, dir) }, value, identity, identity, note)
	case "rtm":
		value, err := rtmToken(c, identity)
		note("prepare", err)
		if err != nil || writeFailed {
			return 1
		}
		checkRTM(ctx, func() (rtmClient, error) { return newRTM(c, identity, dir) }, value, note)
	case "rtc_features":
		runRTCFeatures(ctx, c, dir, identity, note)
	case "rtm_features":
		runRTMFeatures(ctx, c, dir, identity, note)
	default:
		return 2
	}
	r.fillMissing(component)
	if r.save(dir) != nil {
		writeFailed = true
	}
	if writeFailed || !r.passed(component) {
		return 1
	}
	return 0
}
