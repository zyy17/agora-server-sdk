package agorasdk

import (
	"errors"
	"strings"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func validChannel() ChannelConfig {
	return ChannelConfig{Channel: "room", UserID: "user", Token: "opaque"}
}

func TestConfigDefaults(t *testing.T) {
	c, err := normalized(validChannel())
	if err != nil {
		t.Fatal(err)
	}
	if c.Audio.Send.SampleRate != 16000 || c.Audio.Send.Channels != 1 || c.QueueCapacity != 64 || c.JoinTimeout <= 0 {
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
		{"rate", func(c *ChannelConfig) { c.Audio.Send.SampleRate = -1 }}, {"channels", func(c *ChannelConfig) { c.Audio.Send.Channels = 3 }},
		{"queue", func(c *ChannelConfig) { c.QueueCapacity = -2 }}, {"timeout", func(c *ChannelConfig) { c.JoinTimeout = -1 }},
		{"role", func(c *ChannelConfig) { c.Role = 99 }}, {"scenario", func(c *ChannelConfig) { c.Audio.Scenario = 99 }},
		{"large scale auto", func(c *ChannelConfig) { c.Mode = LargeScale }}, {"VAD format", func(c *ChannelConfig) { c.Audio.VAD = true; c.Audio.Receive.SampleRate = 48000 }},
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
	c.Video.Encoder = &rtc.VideoEncoderConfiguration{Width: 640}
	c.Video.EncodedSender = &rtc.VideoEncodedImageSenderOptions{}
	got, err := normalized(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Encryption.EncryptionKdfSalt[0] = 9
	c.Audio.VADConfig.StartRecognizeCount = 99
	c.Video.Encoder.Width = 1280
	if got.Encryption.EncryptionKdfSalt[0] != 1 || got.Audio.VADConfig.StartRecognizeCount != 7 || got.Video.Encoder.Width != 640 || got.Video.EncodedSender == c.Video.EncodedSender {
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

func TestIndependentAudioDefaultsAndVAD(t *testing.T) {
	c := validChannel()
	c.Audio.Send = PCMFormat{48000, 2}
	c.Audio.VAD = true
	got, err := normalized(c)
	if err != nil || got.Audio.Receive != (PCMFormat{16000, 1}) || got.Audio.Send != (PCMFormat{48000, 2}) {
		t.Fatal(got, err)
	}
	c.Receive.Disabled = true
	if _, err := normalized(c); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	c.Receive.Disabled = false
	c.Receive.AudioPositions = AudioMixed
	if _, err := normalized(c); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}

func TestConfigValidationOrder(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ChannelConfig)
		want   string
	}{
		{"identity before mode", func(c *ChannelConfig) { c.Channel = ""; c.Role = 99 }, "channel must"},
		{"mode before format", func(c *ChannelConfig) { c.Role = 99; c.Audio.Send.SampleRate = -1 }, "mode is unknown"},
		{"send before receive", func(c *ChannelConfig) { c.Audio.Send.Channels = 3; c.Audio.Receive.SampleRate = -1 }, "channels must"},
		{"positions before queue", func(c *ChannelConfig) { c.Receive.AudioPositions = 32; c.QueueCapacity = -1 }, "audio positions"},
		{"queue before timeout", func(c *ChannelConfig) { c.QueueCapacity = -1; c.JoinTimeout = -1 }, "queue capacity"},
		{"timeout before mode constraints", func(c *ChannelConfig) { c.JoinTimeout = -1; c.Mode = LargeScale }, "join timeout"},
		{"mode constraints before scenario", func(c *ChannelConfig) { c.Mode = LargeScale; c.Audio.Scenario = 99 }, "large-scale channel"},
		{"scenario before VAD", func(c *ChannelConfig) { c.Audio.Scenario = 99; c.Audio.VAD = true; c.Receive.Disabled = true }, "audio scenario"},
		{"VAD format before position", func(c *ChannelConfig) {
			c.Audio.VAD = true
			c.Audio.Receive.SampleRate = 48000
			c.Receive.Disabled = true
		}, "requires 16 kHz"},
		{"VAD position before thresholds", func(c *ChannelConfig) {
			c.Audio.VAD = true
			c.Receive.Disabled = true
			c.Audio.VADConfig = &rtc.AudioVadConfigV2{StartRecognizeCount: -1}
		}, "requires received PCM"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validChannel()
			tt.mutate(&cfg)
			if _, err := normalized(cfg); !errors.Is(err, ErrInvalidConfig) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want invalid %s", err, tt.want)
			}
		})
	}
}

func TestConfigFailurePreservesPartialDefaults(t *testing.T) {
	cfg := validChannel()
	cfg.QueueCapacity = -1
	cfg.Audio.VADConfig = &rtc.AudioVadConfigV2{StartRecognizeCount: 7}
	got, err := normalized(cfg)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if got.Audio.Send != (PCMFormat{16000, 1}) || got.Audio.Receive != (PCMFormat{16000, 1}) || got.Receive.AudioPositions != AudioBeforeMixing {
		t.Fatalf("earlier defaults were lost: %+v", got)
	}
	if got.JoinTimeout != 0 || got.Audio.VADConfig != cfg.Audio.VADConfig {
		t.Fatal("normalization continued past the validation error")
	}
}
