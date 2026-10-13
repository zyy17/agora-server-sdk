package agorasdk

import (
	"context"
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

type roleTestTransport struct {
	fakeTransport
	roleCode, unpublishCode int
	roles                   []int
}

func (t *roleTestTransport) SetUserRole(role int) int {
	t.roles = append(t.roles, role)
	return t.roleCode
}

func (t *roleTestTransport) UnpublishAudio() int { return t.unpublishCode }

func TestRoleChangesAndPublication(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	transport := &roleTestTransport{}
	s.transport = transport
	if err := s.PublishAudio(); err != nil {
		t.Fatal(err)
	}
	if err := s.SetRole(Audience); err != nil || s.Role() != Audience || s.audioPublished {
		t.Fatalf("demotion = %v, role %v, published %v", err, s.Role(), s.audioPublished)
	}
	if err := s.PublishAudio(); !errors.Is(err, ErrNotPublishing) {
		t.Fatal(err)
	}
	if err := s.SetRole(Broadcaster); err != nil || s.Role() != Broadcaster || s.audioPublished {
		t.Fatalf("promotion = %v, role %v", err, s.Role())
	}
	if err := s.PublishAudio(); err != nil || !s.audioPublished {
		t.Fatal(err)
	}
	if err := s.SetRole(Broadcaster); err != nil || len(transport.roles) != 2 {
		t.Fatalf("idempotent role change = %v, calls %v", err, transport.roles)
	}
	if !reflect.DeepEqual(transport.roles, []int{int(rtc.ClientRoleAudience), int(rtc.ClientRoleBroadcaster)}) {
		t.Fatal(transport.roles)
	}
	if err := s.SetRole(Role(99)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	s.Close()
	if err := s.SetRole(Audience); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func TestRoleFailureLeavesRequestedState(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	transport := &roleTestTransport{unpublishCode: -7}
	s.transport = transport
	s.audioPublished = true
	if err := s.SetRole(Audience); err == nil || s.Role() != Broadcaster || !s.audioPublished || len(transport.roles) != 0 {
		t.Fatalf("failed demotion = %v, role %v", err, s.Role())
	}
	transport.unpublishCode, transport.roleCode = 0, -8
	if err := s.SetRole(Audience); err == nil || s.Role() != Broadcaster || s.audioPublished {
		t.Fatalf("failed role setter = %v, role %v", err, s.Role())
	}
}

func TestRoleConcurrentClose(t *testing.T) {
	s, _ := testSession(t)
	s.transport = &roleTestTransport{}
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 30 {
				if err := s.SetRole(Role(i % 2)); err != nil && !errors.Is(err, ErrClosed) {
					t.Error(err)
				}
				_ = s.Role()
			}
		}()
	}
	s.Close()
	wg.Wait()
}

func TestNativeOwnershipOverridesRejected(t *testing.T) {
	for _, mutate := range []func(*rtc.RtcConnectionConfig, *rtc.RtcConnectionPublishConfig){
		func(c *rtc.RtcConnectionConfig, _ *rtc.RtcConnectionPublishConfig) {
			c.ClientRole = rtc.ClientRoleAudience
		},
		func(_ *rtc.RtcConnectionConfig, p *rtc.RtcConnectionPublishConfig) { p.IsPublishAudio = false },
		func(_ *rtc.RtcConnectionConfig, p *rtc.RtcConnectionPublishConfig) { p.IsPublishVideo = true },
		func(_ *rtc.RtcConnectionConfig, p *rtc.RtcConnectionPublishConfig) {
			p.AudioPublishType = rtc.AudioPublishTypeEncodedPcm
		},
		func(_ *rtc.RtcConnectionConfig, p *rtc.RtcConnectionPublishConfig) {
			p.VideoPublishType = rtc.VideoPublishTypeYuv
		},
		func(c *rtc.RtcConnectionConfig, _ *rtc.RtcConnectionPublishConfig) {
			c.ChannelType = rtc.ChannelType(123)
		},
	} {
		cfg := ChannelConfig{ConfigureNative: func(c *rtc.RtcConnectionConfig, p *rtc.RtcConnectionPublishConfig) error { mutate(c, p); return nil }}
		if _, _, err := sessionNativeConfig(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
}

func TestAudioFallbackDecisions(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	for _, test := range []struct {
		policy *AudioFallbackPolicy
		want   int
	}{{nil, -1}, {&AudioFallbackPolicy{Scenario: ScenarioMeeting}, int(rtc.AudioScenarioMeeting)}, {&AudioFallbackPolicy{UseRecommended: true}, 77}} {
		if err := s.SetAudioFallback(test.policy); err != nil {
			t.Fatal(err)
		}
		got := s.connectionObserver().OnAIQoSCapabilityMissing(nil, 77)
		event, err := s.Events().Next(context.Background())
		decision, ok := event.Data.(AudioFallbackDecision)
		if got != test.want || err != nil || !ok || decision.Selected != test.want || decision.Recommended != 77 {
			t.Fatalf("fallback = %d, %+v, %v", got, event, err)
		}
	}
	if err := s.SetAudioFallback(&AudioFallbackPolicy{Scenario: 99}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}

func TestAudioFallbackConcurrentCallback(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	observer := s.connectionObserver()
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = s.SetAudioFallback(&AudioFallbackPolicy{Scenario: ScenarioMeeting})
				observer.OnAIQoSCapabilityMissing(nil, 5)
			}
		}()
	}
	wg.Wait()
}

func TestCompleteEventPayloads(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	observer := s.localObserver()
	info := &rtc.VideoTrackInfo{OwnerUid: 17, TrackId: 8, ChannelId: "room", CodecType: 2}
	observer.OnUserVideoTrackSubscribed(nil, "speaker", info, nil)
	info.TrackId = 99
	event, _ := s.Events().Next(context.Background())
	if event.Data.(rtc.VideoTrackInfo).TrackId != 8 {
		t.Fatal(event)
	}
	observer.OnUserAudioTrackStateChanged(nil, "a", nil, 3, 4, 51)
	observer.OnUserVideoTrackStateChanged(nil, "v", nil, 5, 6, 52)
	observer.OnAudioPublishStateChanged(nil, "room", 7, 8, 53)
	for _, elapsed := range []int{51, 52, 53} {
		event, _ = s.Events().Next(context.Background())
		if event.ElapsedMillis != elapsed {
			t.Fatal(event)
		}
	}
}

func TestIdleAndLocalAccessPointConfiguration(t *testing.T) {
	for _, idle := range []bool{false, true} {
		cfg, err := serviceConfig(Config{AppID: "00000000000000000000000000000000", IdleMode: idle})
		if err != nil || cfg.IdleMode != idle {
			t.Fatal(cfg, err)
		}
	}
	cfg := &rtc.LocalAccessPointConfiguration{IPList: []string{"127.0.0.1"}, DomainList: []string{"example.test"}}
	if err := validateLocalAccessPoint(*cfg); err != nil {
		t.Fatal(err)
	}
	copy := copyLocalAccessPoint(cfg)
	cfg.IPList[0], cfg.DomainList[0] = "bad", "bad"
	if copy.IPList[0] != "127.0.0.1" || copy.DomainList[0] != "example.test" {
		t.Fatal(copy)
	}
	for _, invalid := range []rtc.LocalAccessPointConfiguration{{}, {IPList: []string{""}}, {DomainList: []string{"a\x00b"}}, {IPList: []string{"127.0.0.1"}, Mode: 9}} {
		if err := validateLocalAccessPoint(invalid); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
}

func TestServiceExtensionLifecycleGuards(t *testing.T) {
	s := &Service{}
	s.Close()
	for _, call := range []func() error{
		func() error { return s.LoadExtensionProvider("provider.so", true) },
		func() error { return s.EnableExtension("provider", "extension", "", true) },
		func() error { return s.DisableExtension("provider", "extension", "") },
	} {
		if err := call(); !errors.Is(err, ErrClosed) {
			t.Fatal(err)
		}
	}
	if err := s.LoadExtensionProvider("a\x00b", true); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if err := s.EnableExtension("", "extension", "", false); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}

func TestVADRejectsNaN(t *testing.T) {
	for _, cfg := range []rtc.AudioVadConfigV2{{ActivePercent: float32(math.NaN())}, {InactivePercent: float32(math.NaN())}, {AdaptiveRmsThresholdFactor: float32(math.NaN())}} {
		if err := validateVAD(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
}

func TestAllOfficialObserversHaveHandlers(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	for _, observer := range []any{s.connectionObserver(), s.localObserver()} {
		value := reflect.ValueOf(observer).Elem()
		for i := 0; i < value.NumField(); i++ {
			if value.Field(i).Kind() == reflect.Func && value.Field(i).IsNil() {
				t.Errorf("%s.%s is not surfaced", value.Type(), value.Type().Field(i).Name)
			}
		}
	}
}

func TestDiagnosticAndEncoderControlGuards(t *testing.T) {
	s, _ := testSession(t)
	if err := s.SetAudioProfile(rtc.AudioProfile(-1)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if _, err := s.SessionID(); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	s.Close()
	if _, err := s.SessionID(); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err := s.SetAudioProfile(rtc.AudioProfileDefault); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

type nativeStateTransport struct {
	fakeTransport
	role         int
	audio, video bool
	publishes    int
}

func (t *nativeStateTransport) GetUserRole() int { return t.role }

func (t *nativeStateTransport) GetPublicationState() (bool, bool) { return t.audio, t.video }

func (t *nativeStateTransport) PublishAudio() int {
	t.publishes++
	t.audio = true
	return 0
}

func TestWithNativeReconcilesPartialDemotion(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	transport := &nativeStateTransport{role: int(rtc.ClientRoleBroadcaster), audio: true, video: true}
	s.transport = transport
	s.audioPublished, s.videoPublished = true, true
	failure := errors.New("video unpublish failed")
	err := s.WithNative(func(*rtc.RtcConnection) error {
		// Model SetUserRole(Audience): audio unpublish succeeds, video unpublish
		// fails, so the native requested role remains Broadcaster.
		transport.audio = false
		return failure
	})
	if !errors.Is(err, failure) || s.Role() != Broadcaster || s.audioPublished || !s.videoPublished {
		t.Fatalf("partial transition not reconciled: %v, role=%v audio=%v video=%v", err, s.Role(), s.audioPublished, s.videoPublished)
	}
	if err := s.SendPCM(make([]byte, 320)); err != nil || transport.publishes != 1 {
		t.Fatalf("send did not republish audio: %v, calls=%d", err, transport.publishes)
	}
}

func TestSendRefreshesAsynchronousPublicationState(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	transport := &nativeStateTransport{role: int(rtc.ClientRoleBroadcaster), audio: false}
	s.transport = transport
	// An automatic native fallback can fail to republish without a managed call.
	s.audioPublished = true
	if err := s.SendPCM(make([]byte, 320)); err != nil || transport.publishes != 1 || !s.audioPublished {
		t.Fatalf("stale callback publication state was used: %v, calls=%d", err, transport.publishes)
	}
}

type recoveryCancelTransport struct {
	fakeTransport
	unpublishes int
}

func (t *recoveryCancelTransport) UnpublishAudio() int { t.unpublishes++; return 0 }

func TestUnpublishCancelsRecoveryAfterFailedScenario(t *testing.T) {
	s, _ := testSession(t)
	defer s.Close()
	transport := &recoveryCancelTransport{}
	s.transport = transport
	// SetAudioScenario has reconciled a failed rollback as currently unpublished.
	s.audioPublished = false
	if err := s.UnpublishAudio(); err != nil || transport.unpublishes != 1 {
		t.Fatalf("pending native recovery was not cancelled: %v calls=%d", err, transport.unpublishes)
	}
}
