package agorasdk

import (
	"errors"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestSessionNativeConfigModes(t *testing.T) {
	cases := []struct {
		name                                 string
		cfg                                  ChannelConfig
		audioSubscription, videoSubscription bool
		audioPublication, videoPublication   bool
		audioType                            rtc.AudioPublishType
		role                                 rtc.ClientRole
	}{
		{"defaults", ChannelConfig{}, true, false, true, false, rtc.AudioPublishTypePcm, rtc.ClientRoleBroadcaster},
		{"audience", ChannelConfig{Role: Audience, Video: VideoConfig{Mode: RawVideo}, Receive: ReceiveConfig{Video: true}}, true, true, false, false, rtc.AudioPublishTypePcm, rtc.ClientRoleAudience},
		{"encoded", ChannelConfig{Audio: AudioConfig{Mode: EncodedAudio}, Video: VideoConfig{Mode: EncodedVideo}, Receive: ReceiveConfig{EncodedVideo: true}}, true, true, true, true, rtc.AudioPublishTypeEncodedPcm, rtc.ClientRoleBroadcaster},
		{"disabled", ChannelConfig{Audio: AudioConfig{Mode: NoAudio}, Receive: ReceiveConfig{Disabled: true}}, false, false, false, false, rtc.AudioPublishTypeNoPublish, rtc.ClientRoleBroadcaster},
		{"manual", ChannelConfig{Mode: LargeScale, Receive: ReceiveConfig{Video: true, ManualSubscriptions: true}}, false, false, true, false, rtc.AudioPublishTypePcm, rtc.ClientRoleBroadcaster},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conn, pub, err := sessionNativeConfig(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}
			if conn.AutoSubscribeAudio != tt.audioSubscription || conn.AutoSubscribeVideo != tt.videoSubscription || conn.ClientRole != tt.role || conn.ChannelType != rtc.ChannelType(tt.cfg.Mode) {
				t.Fatalf("unexpected connection config: %+v", conn)
			}
			if pub.IsPublishAudio != tt.audioPublication || pub.IsPublishVideo != tt.videoPublication || pub.AudioPublishType != tt.audioType || pub.VideoPublishType != rtc.VideoPublishType(tt.cfg.Video.Mode) {
				t.Fatalf("unexpected publish config: %+v", pub)
			}
		})
	}
}

func TestSessionNativeConfigOverride(t *testing.T) {
	sender := &rtc.VideoEncodedImageSenderOptions{}
	cfg := ChannelConfig{Video: VideoConfig{EncodedSender: sender}, Receive: ReceiveConfig{EncodedAudio: true}}
	called := false
	cfg.ConfigureNative = func(conn *rtc.RtcConnectionConfig, pub *rtc.RtcConnectionPublishConfig) error {
		called = true
		if !conn.AudioRecvEncodedFrame || pub.VideoEncodedImageSenderOptions != sender || pub.AudioScenario != rtc.AudioScenarioAiServer {
			t.Fatal("callback did not receive mapped defaults")
		}
		conn.AutoSubscribeAudio = false
		conn.ChannelType = rtc.ChannelTypeLargeScale
		pub.AudioScenario = rtc.AudioScenarioMeeting
		return nil
	}
	conn, pub, err := sessionNativeConfig(cfg)
	if err != nil || !called || conn.ChannelType != rtc.ChannelTypeLargeScale || pub.AudioScenario != rtc.AudioScenarioMeeting {
		t.Fatalf("override was lost: %v, %+v, %+v", err, conn, pub)
	}
}

func TestSessionNativeConfigValidatesAfterOverride(t *testing.T) {
	want := errors.New("configuration rejected")
	for _, callbackErr := range []error{nil, want} {
		cfg := ChannelConfig{ConfigureNative: func(conn *rtc.RtcConnectionConfig, _ *rtc.RtcConnectionPublishConfig) error {
			conn.ChannelType = rtc.ChannelTypeLargeScale
			return callbackErr
		}}
		conn, pub, err := sessionNativeConfig(cfg)
		if callbackErr == nil && !errors.Is(err, ErrInvalidConfig) || callbackErr != nil && !errors.Is(err, callbackErr) {
			t.Fatalf("unexpected validation result: %v", err)
		}
		if conn != nil || pub != nil {
			t.Fatal("returned native configuration on failure")
		}
	}
}
