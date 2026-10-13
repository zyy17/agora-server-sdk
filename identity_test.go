package agorasdk

import (
	"context"
	"errors"
	"strings"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestServiceIdentityConfig(t *testing.T) {
	for _, mode := range []IdentityMode{StringIdentity, NumericIdentity} {
		called := false
		config, err := serviceConfig(Config{AppID: strings.Repeat("0", 32), IdentityMode: mode, ConfigureNative: func(c *rtc.AgoraServiceConfig) error {
			called = true
			if c.UseStringUid != (mode == StringIdentity) {
				t.Fatal("wrong native identity")
			}
			return nil
		}})
		if err != nil || !called || config.UseStringUid != (mode == StringIdentity) {
			t.Fatal(config, err)
		}
		_, err = serviceConfig(Config{AppID: strings.Repeat("0", 32), IdentityMode: mode, ConfigureNative: func(c *rtc.AgoraServiceConfig) error { c.UseStringUid = !c.UseStringUid; return nil }})
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatal("accepted identity override", err)
		}
	}
	if _, err := serviceConfig(Config{AppID: strings.Repeat("0", 32), IdentityMode: 99}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}

func TestNumericIdentityValidation(t *testing.T) {
	for _, uid := range []string{"0", "1", "12345", "4294967295"} {
		if err := NumericIdentity.validateUserID(uid, true); err != nil {
			t.Fatal(uid, err)
		}
	}
	for _, uid := range []string{"", "00", "01", "+1", "-1", " 1", "1 ", "1.0", "4294967296", "alice"} {
		if err := NumericIdentity.validateUserID(uid, true); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(uid, err)
		}
	}
	if NumericIdentity.validateUserID("0", false) == nil {
		t.Fatal("remote auto UID accepted")
	}
	if err := StringIdentity.validateUserID("001", false); err != nil {
		t.Fatal("string account reinterpreted", err)
	}
	service := &Service{identityMode: NumericIdentity}
	if _, err := service.Join(context.Background(), ChannelConfig{Channel: "room", UserID: "alice"}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	s, _ := testSession(t)
	defer s.Close()
	s.service = service
	if err := s.SubscribeAudio("alice"); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if err := s.SubscribeVideo("0", VideoSubscription{}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	s.connectionObserver().OnUserJoined(nil, "12345")
	event, err := s.Events().Next(context.Background())
	if err != nil || event.UserID != "12345" {
		t.Fatal(event, err)
	}
}
