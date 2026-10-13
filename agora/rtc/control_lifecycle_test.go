package agoraservice

import (
	"os"
	"testing"
)

func TestNativeAudiencePromotion(t *testing.T) {
	if os.Getenv("AGORA_NATIVE_SMOKE") != "1" {
		t.Skip("set AGORA_NATIVE_SMOKE=1 to exercise native lifecycle without connecting")
	}
	t.Chdir(t.TempDir())
	cfg := NewAgoraServiceConfig()
	cfg.AppId = "00000000000000000000000000000000"
	cfg.LogPath = t.TempDir() + "/agora.log"
	cfg.ConfigDir, cfg.DataDir = t.TempDir(), t.TempDir()
	cfg.EnableVideo, cfg.IdleMode = true, true
	if code := Initialize(cfg); code != 0 {
		t.Fatal(code)
	}
	defer Release()
	for _, audioType := range []AudioPublishType{AudioPublishTypePcm, AudioPublishTypeEncodedPcm, AudioPublishTypeNoPublish} {
		pub := NewRtcConPublishConfig()
		pub.IsPublishAudio, pub.IsPublishVideo = false, false
		pub.AudioPublishType, pub.VideoPublishType = audioType, VideoPublishTypeYuv
		conn := NewRtcConnection(&RtcConnectionConfig{ClientRole: ClientRoleAudience, ChannelProfile: ChannelProfileLiveBroadcasting}, pub)
		if conn == nil {
			t.Fatal("create audience")
		}
		if conn.audioTrack != nil || conn.videoTrack != nil {
			t.Fatal("eager audience tracks")
		}
		encoder := &VideoEncoderConfiguration{CodecType: VideoCodecTypeH264, Width: 640, Height: 360, Framerate: 15, Bitrate: 400}
		if code := conn.SetVideoEncoderConfiguration(encoder); code != 0 {
			t.Fatal("defer encoder:", code)
		}
		encoder.Width = 999
		if code := conn.SetUserRole(int(ClientRoleBroadcaster)); code != 0 {
			t.Fatal("promote:", code)
		}
		if conn.videoTrack == nil || (conn.audioTrack == nil) != (audioType == AudioPublishTypeNoPublish) {
			t.Fatal("missing lazy publication resources")
		}
		if conn.videoEncoderConfiguration.Width != 640 {
			t.Fatal("retained caller encoder")
		}
		audio, video := conn.audioTrack, conn.videoTrack
		if code := conn.SetUserRole(int(ClientRoleAudience)); code != 0 {
			t.Fatal("demote:", code)
		}
		if conn.localUser.GetUserRole() != int(ClientRoleAudience) {
			t.Fatal("role snapshot")
		}
		if code := conn.SetUserRole(int(ClientRoleBroadcaster)); code != 0 {
			t.Fatal("repromote:", code)
		}
		if conn.audioTrack != audio || conn.videoTrack != video {
			t.Fatal("recreated owned resources")
		}
		if code := conn.SetUserRole(123); code == 0 {
			t.Fatal("accepted unknown role")
		}
		conn.Release()
	}
}

func TestPublicationStateReflectsPartialOperations(t *testing.T) {
	conn := &RtcConnection{localUser: &LocalUser{publishFlag: false, videoPublishFlag: true}}
	conn.localUser.role.Store(int32(ClientRoleBroadcaster))
	audio, video := conn.GetPublicationState()
	if audio || !video || conn.GetUserRole() != int(ClientRoleBroadcaster) {
		t.Fatal("partial native publication state lost")
	}
	conn.localUser = nil
	if audio, video = conn.GetPublicationState(); audio || video || conn.GetUserRole() != 0 {
		t.Fatal("released native state retained")
	}
}

func TestNativeExternalAudioRuntimeOptions(t *testing.T) {
	if os.Getenv("AGORA_NATIVE_SMOKE") != "1" {
		t.Skip("set AGORA_NATIVE_SMOKE=1 for credential-free native configuration")
	}
	t.Chdir(t.TempDir())
	for round := range 2 {
		cfg := NewAgoraServiceConfig()
		cfg.AppId, cfg.LogPath = "00000000000000000000000000000000", t.TempDir()+"/agora.log"
		cfg.ConfigDir, cfg.DataDir, cfg.IdleMode = t.TempDir(), t.TempDir(), false
		if code := Initialize(cfg); code != 0 {
			t.Fatal(code)
		}
		create := func(value bool, speed int) *RtcConnection {
			pub := NewRtcConPublishConfig()
			pub.SendExternalAudioParameters = &SendExternalAudioParameters{Enabled: true, SendMs: 500, SendSpeed: speed, DeliverMuteDataForFakeAdm: value}
			return NewRtcConnection(&RtcConnectionConfig{ClientRole: ClientRoleBroadcaster, ChannelProfile: ChannelProfileLiveBroadcasting}, pub)
		}
		if conn := create(round != 0, 1); conn != nil {
			conn.Release()
			Release()
			t.Fatal("invalid enabled speed accepted")
		}
		conn := create(round != 0, 2)
		if conn == nil {
			Release()
			t.Fatal("valid external-audio engine configuration failed")
		}
		if code := conn.setTotalExtraSendMs(); code != 0 {
			conn.Release()
			Release()
			t.Fatalf("native per-track extra-send budget rejected: %d", code)
		}
		matching := create(round != 0, 2)
		if matching == nil {
			conn.Release()
			Release()
			t.Fatal("identical global option rejected")
		}
		if conflict := create(round == 0, 2); conflict != nil {
			conflict.Release()
			matching.Release()
			conn.Release()
			Release()
			t.Fatal("conflicting global option silently applied")
		}
		if conflict := create(round != 0, 3); conflict != nil {
			conflict.Release()
			matching.Release()
			conn.Release()
			Release()
			t.Fatal("conflicting runtime send speed silently applied")
		}
		matching.Release()
		conn.Release()
		if code := Release(); code != 0 {
			t.Fatal(code)
		}
		if agoraService.deliverMuteData.configured || agoraService.extraSendSpeed.configured {
			t.Fatal("successful Release retained global option")
		}
	}
}
