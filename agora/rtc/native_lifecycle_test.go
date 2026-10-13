package agoraservice

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

// This opt-in smoke test creates native objects without joining a channel.
func TestNativeLifecycle(t *testing.T) {
	if os.Getenv("AGORA_NATIVE_SMOKE") != "1" {
		t.Skip("set AGORA_NATIVE_SMOKE=1 to exercise the installed native SDK")
	}
	t.Chdir(t.TempDir())
	for round := 0; round < 2; round++ {
		cfg := NewAgoraServiceConfig()
		cfg.AppId = "00000000000000000000000000000000"
		cfg.LogPath = t.TempDir() + "/agora.log"
		cfg.ConfigDir = t.TempDir()
		cfg.DataDir = t.TempDir()
		cfg.EnableVideo = true
		cfg.IdleMode = round == 0
		if code := Initialize(cfg); code != 0 {
			t.Fatalf("initialize: %d", code)
		}
		publish := NewRtcConPublishConfig()
		publish.IsPublishVideo = true
		publish.VideoPublishType = VideoPublishTypeYuv
		conn := NewRtcConnection(&RtcConnectionConfig{ClientRole: ClientRoleBroadcaster, ChannelProfile: ChannelProfileLiveBroadcasting}, publish)
		if conn == nil {
			Release()
			t.Fatal("create connection")
		}
		if code := conn.RegisterObserver(&RtcConnectionObserver{}); code != 0 {
			t.Fatalf("register connection: %d", code)
		}
		if code := conn.RegisterLocalUserObserver(&LocalUserObserver{}); code != 0 {
			t.Fatalf("register user: %d", code)
		}
		if code := conn.RegisterAudioFrameObserver(&AudioFrameObserver{}, 1, nil); code != 0 {
			t.Fatalf("register audio: %d", code)
		}
		if code := conn.RegisterVideoFrameObserver(&VideoFrameObserver{}); code != 0 {
			t.Fatalf("register video: %d", code)
		}
		if code := conn.RegisterVideoEncodedFrameObserver(&VideoEncodedFrameObserver{}); code != 0 {
			t.Fatalf("register encoded video: %d", code)
		}
		if code := conn.PushAudioPcmData(nil, 0, 0, 0); code == 0 {
			t.Fatal("invalid PCM accepted")
		}
		// Exercise callback arrival during observer replacement and final release.
		handle := conn.cConnection
		var callbackCount atomic.Int64
		var callbacks sync.WaitGroup
		callbacks.Add(1)
		go func() {
			defer callbacks.Done()
			for range 1000 {
				goOnUserJoined(handle, nil)
			}
		}()
		for range 10 {
			if code := conn.RegisterObserver(&RtcConnectionObserver{OnUserJoined: func(*RtcConnection, string) { callbackCount.Add(1) }}); code != 0 {
				t.Fatalf("replace observer: %d", code)
			}
		}
		if code := conn.UpdateAudioSenario(AudioScenarioChorus); code != 0 {
			t.Fatalf("PCM scenario: %d", code)
		}
		noAudio := NewRtcConPublishConfig()
		noAudio.IsPublishAudio = false
		receiveOnly := NewRtcConnection(&RtcConnectionConfig{}, noAudio)
		if receiveOnly == nil {
			t.Fatal("create receive-only")
		}
		if code := receiveOnly.UpdateAudioSenario(AudioScenarioChorus); code != 0 {
			t.Fatalf("receive-only scenario: %d", code)
		}
		receiveOnly.Release()
		encoded := NewRtcConPublishConfig()
		encoded.AudioPublishType = AudioPublishTypeEncodedPcm
		encodedConnection := NewRtcConnection(&RtcConnectionConfig{}, encoded)
		if encodedConnection == nil {
			t.Fatal("create encoded-audio")
		}
		track := encodedConnection.audioTrack
		if code := encodedConnection.UpdateAudioSenario(AudioScenarioChorus); code != 0 {
			t.Fatalf("encoded scenario: %d", code)
		}
		if encodedConnection.audioTrack != track {
			t.Fatal("encoded track was replaced by a PCM track")
		}
		encodedConnection.Release()
		conn.Disconnect()
		conn.Release()
		callbacks.Wait()
		assertConnectionReleased(t, conn)
		conn.Release()
		if code := Release(); code != 0 {
			t.Fatalf("release: %d", code)
		}
		if code := Release(); code != 0 {
			t.Fatalf("repeat release: %d", code)
		}
	}
}

func assertConnectionReleased(t *testing.T, conn *RtcConnection) {
	t.Helper()
	if conn.cConnection != nil || conn.cAudioObserver != nil || conn.cVideoObserver != nil || conn.cEncodedVideoObserver != nil ||
		conn.cLocalUserObserver != nil || conn.cHandler != nil || conn.cCapabilitiesObserver != nil || conn.cCapObserverHandle != nil {
		t.Fatal("released connection retained native handles")
	}
	if conn.localUser != nil || conn.parameter != nil || conn.audioVadManager != nil || conn.retiredObservers != nil {
		t.Fatal("released connection retained local state")
	}
	if conn.audioTrack != nil || conn.videoTrack != nil || conn.audioSender != nil || conn.videoSender != nil ||
		conn.encodedAudioSender != nil || conn.encodedVideoSender != nil {
		t.Fatal("released connection retained publishing resources")
	}
	if conn.callbacks.begin() {
		conn.callbacks.end()
		t.Fatal("released connection admitted a callback")
	}
}
