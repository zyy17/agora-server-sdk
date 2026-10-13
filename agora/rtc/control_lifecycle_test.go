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
