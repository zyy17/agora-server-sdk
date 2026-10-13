//go:build agora_test

package agoraservice

import (
	"os"
	"testing"
)

func TestNativeAudioSettingsSurvivePromotionAndFallback(t *testing.T) {
	if os.Getenv("AGORA_NATIVE_SMOKE") != "1" {
		t.Skip("set AGORA_NATIVE_SMOKE=1 for credential-free native track lifecycle")
	}
	t.Chdir(t.TempDir())
	cfg := NewAgoraServiceConfig()
	cfg.AppId, cfg.LogPath = "00000000000000000000000000000000", t.TempDir()+"/agora.log"
	cfg.ConfigDir, cfg.DataDir, cfg.IdleMode = t.TempDir(), t.TempDir(), false
	if code := Initialize(cfg); code != 0 {
		t.Fatal(code)
	}
	defer Release()
	pub := NewRtcConPublishConfig()
	pub.IsPublishAudio = false
	conn := NewRtcConnection(&RtcConnectionConfig{ClientRole: ClientRoleAudience, ChannelProfile: ChannelProfileLiveBroadcasting}, pub)
	if conn == nil {
		t.Fatal("create audience")
	}
	defer conn.Release()
	for _, code := range []int{conn.SetAudioTrackEnabled(false), conn.AdjustAudioPublishVolume(37), conn.SetAudioSendDelayMs(20), conn.SetAudioBufferCapacity(7)} {
		if code != 0 {
			t.Fatal("defer setting:", code)
		}
	}
	if conn.audioTrack != nil {
		t.Fatal("configuration eagerly created an audience track")
	}
	if code := conn.SetUserRole(int(ClientRoleBroadcaster)); code != 0 {
		t.Fatal("promote:", code)
	}
	assertSettings := func() {
		t.Helper()
		enabled, volume, code := inspectNativeAudioTrack(conn.audioTrack)
		if code != 0 || enabled || volume != 37 {
			t.Fatalf("native settings: enabled=%v volume=%d code=%d", enabled, volume, code)
		}
		if conn.audioSettings.delay != 20 || conn.audioSettings.capacity != 7 {
			t.Fatal("timing/buffer settings lost")
		}
	}
	assertSettings()
	previous := conn.audioTrack
	if code := conn.RegisterObserver(&RtcConnectionObserver{OnAIQoSCapabilityMissing: func(*RtcConnection, int) int { return int(AudioScenarioChorus) }}); code != 0 {
		t.Fatal(code)
	}
	invokeMissingAIQoSCapability(conn)
	if conn.audioScenario != AudioScenarioChorus || conn.audioTrack == previous {
		t.Fatal("automatic fallback did not replace track")
	}
	assertSettings()
	if code := conn.UpdateAudioSenario(AudioScenarioMeeting); code != 0 {
		t.Fatal(code)
	}
	assertSettings()
}
