package agorasdk

import (
	"errors"
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"testing"
)

func validChannel() ChannelConfig {
	return ChannelConfig{Channel: "room", UserID: "user", Token: "opaque"}
}
func TestConfigDefaults(t *testing.T) {
	c, err := normalized(validChannel())
	if err != nil {
		t.Fatal(err)
	}
	if c.Audio.SampleRate != 16000 || c.Audio.Channels != 1 || c.QueueCapacity != 64 || c.JoinTimeout <= 0 {
		t.Fatalf("%+v", c)
	}
	if c.Audio.Scenario.native() != rtc.AudioScenarioAiServer || ScenarioDefault.native() != rtc.AudioScenarioDefault {
		t.Fatal("scenario zero is ambiguous")
	}
}
func TestInvalidConfig(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ChannelConfig)
	}{
		{"channel", func(c *ChannelConfig) { c.Channel = "" }}, {"NUL channel", func(c *ChannelConfig) { c.Channel = "a\x00b" }},
		{"user", func(c *ChannelConfig) { c.UserID = "" }}, {"NUL token", func(c *ChannelConfig) { c.Token = "a\x00b" }},
		{"rate", func(c *ChannelConfig) { c.Audio.SampleRate = -1 }}, {"channels", func(c *ChannelConfig) { c.Audio.Channels = 3 }},
		{"queue", func(c *ChannelConfig) { c.QueueCapacity = -2 }}, {"timeout", func(c *ChannelConfig) { c.JoinTimeout = -1 }},
		{"role", func(c *ChannelConfig) { c.Role = 99 }}, {"scenario", func(c *ChannelConfig) { c.Audio.Scenario = 99 }},
		{"large scale auto", func(c *ChannelConfig) { c.Mode = LargeScale }}, {"VAD format", func(c *ChannelConfig) { c.Audio.VAD = true; c.Audio.SampleRate = 48000 }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := validChannel()
			tt.mutate(&c)
			if _, err := normalized(c); !errors.Is(err, ErrInvalidConfig) {
				t.Fatal(err)
			}
		})
	}
	c := validChannel()
	c.Mode = LargeScale
	c.Receive.ManualSubscriptions = true
	if _, err := normalized(c); err != nil {
		t.Fatal(err)
	}
	c.Token = ""
	if _, err := normalized(c); err != nil {
		t.Fatal("empty token is valid for unsecured projects:", err)
	}
}
func TestConfigCopiesPointers(t *testing.T) {
	c := validChannel()
	c.Encryption = &rtc.EncryptionConfig{EncryptionKdfSalt: []byte{1, 2}}
	c.Audio.VADConfig = &rtc.AudioVadConfigV2{StartRecognizeCount: 7}
	got, err := normalized(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Encryption.EncryptionKdfSalt[0] = 9
	c.Audio.VADConfig.StartRecognizeCount = 99
	if got.Encryption.EncryptionKdfSalt[0] != 1 || got.Audio.VADConfig.StartRecognizeCount != 7 {
		t.Fatal("config retained mutable caller memory")
	}
}
func TestPCMValidation(t *testing.T) {
	for _, rate := range []int{8000, 16000, 32000, 44100, 48000} {
		for _, channels := range []int{1, 2} {
			size := rate / 100 * channels * 2
			if err := validatePCM(make([]byte, size), rate, channels); err != nil {
				t.Fatal(rate, channels, err)
			}
			if err := validatePCM(make([]byte, size-1), rate, channels); err == nil {
				t.Fatal("accepted partial frame")
			}
		}
	}
	for _, rate := range []int{0, -1, 1, 192000} {
		if err := validatePCM(nil, rate, 0); err == nil {
			t.Fatal("accepted bad format")
		}
	}
}
func TestVideoValidation(t *testing.T) {
	for _, f := range []VideoFrame{{Width: 2, Height: 2, Format: I420, Data: make([]byte, 6)}, {Width: 2, Height: 2, Format: NV12, Data: make([]byte, 6)}, {Width: 2, Height: 2, Format: RGBA, Data: make([]byte, 16)}, {Width: 2, Height: 2, Format: BGRA, Data: make([]byte, 16)}} {
		if err := f.validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []VideoFrame{{}, {Width: 3, Height: 2, Format: I420, Data: make([]byte, 9)}, {Width: 2, Height: 2, Format: I420, Data: make([]byte, 5)}, {Width: 2, Height: 2, Format: I420, Data: make([]byte, 6), Rotation: 13}, {Width: 2, Height: 2, Format: I420, Data: make([]byte, 6), Alpha: []byte{1}}, {Width: 1 << 30, Height: 1 << 30, Format: RGBA}} {
		if err := f.validate(); err == nil {
			t.Fatal("accepted invalid video")
		}
	}
}
func TestParameterValidation(t *testing.T) {
	for _, v := range []string{"", "null", "[]", "{bad}"} {
		if validateParameters(v) == nil {
			t.Fatal(v)
		}
	}
	if err := validateParameters(`{"rtc.test":1}`); err != nil {
		t.Fatal(err)
	}
}
func TestNativeError(t *testing.T) {
	err := check("publish", -7)
	var sdkErr *Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != -7 || sdkErr.Op != "publish" {
		t.Fatal(err)
	}
	if check("ok", 0) != nil {
		t.Fatal("success failed")
	}
}
