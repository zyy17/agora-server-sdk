package agoraservice

import (
	"reflect"
	"testing"
)

type recordingTrackControls struct {
	calls                   []string
	enabled                 bool
	volume, delay, capacity int
	volumeCode, delayCode   int
}

func (t *recordingTrackControls) SetEnabled(enabled bool) {
	t.enabled = enabled
	t.calls = append(t.calls, "enabled")
}

func (t *recordingTrackControls) AdjustPublishVolume(volume int) int {
	t.volume = volume
	t.calls = append(t.calls, "volume")
	return t.volumeCode
}

func (t *recordingTrackControls) SetSendDelayMs(delay int) int {
	t.delay = delay
	t.calls = append(t.calls, "delay")
	return t.delayCode
}

func (t *recordingTrackControls) SetMaxBufferedAudioFrameNumber(capacity int) {
	t.capacity = capacity
	t.calls = append(t.calls, "capacity")
}

func TestAudioTrackSettingsReplayBeforePublication(t *testing.T) {
	settings := audioTrackSettings{enabledSet: true, enabled: false, volumeSet: true, volume: 37, delaySet: true, delay: 20, capacitySet: true, capacity: 7}
	for _, fail := range []string{"", "volume", "delay"} {
		track := &recordingTrackControls{enabled: true}
		if fail == "volume" {
			track.volumeCode = -9
		}
		if fail == "delay" {
			track.delayCode = -10
		}
		code := settings.apply(track)
		if track.enabled || len(track.calls) == 0 || track.calls[0] != "enabled" {
			t.Fatal("mute was not the first restored setting")
		}
		switch fail {
		case "":
			if code != 0 || track.volume != 37 || track.delay != 20 || track.capacity != 7 || !reflect.DeepEqual(track.calls, []string{"enabled", "volume", "delay", "capacity"}) {
				t.Fatal(code, track)
			}
		case "volume":
			if code != -9 || len(track.calls) != 2 {
				t.Fatal(code, track)
			}
		case "delay":
			if code != -10 || len(track.calls) != 3 {
				t.Fatal(code, track)
			}
		}
	}
	track := &recordingTrackControls{}
	if code := (audioTrackSettings{}).apply(track); code != 0 || len(track.calls) != 0 {
		t.Fatal("unspecified settings replaced native defaults")
	}
}

func TestLostPublicationRecoveryRetriesOnlyRequestedMedia(t *testing.T) {
	conn := &RtcConnection{}
	calls := 0
	publish := func() int { calls++; return -17 }
	if code := conn.recoverAudioPublication(publish); code != 0 || calls != 0 {
		t.Fatal("started deliberately unpublished media")
	}
	conn.audioPublicationRecovery = true
	if code := conn.recoverAudioPublication(publish); code != -17 || !conn.audioPublicationRecovery || calls != 1 {
		t.Fatal("failed retry was lost")
	}
	if code := conn.recoverAudioPublication(func() int { calls++; return 0 }); code != 0 || conn.audioPublicationRecovery || calls != 2 {
		t.Fatal("recovery did not complete")
	}
	if code := conn.recoverAudioPublication(publish); code != 0 || calls != 2 {
		t.Fatal("successful recovery repeated publication")
	}
}

func TestScenarioRollbackTracksIncompleteRestoration(t *testing.T) {
	for _, scenarioCode := range []int{0, -8} {
		for _, publishCode := range []int{0, -9} {
			conn := &RtcConnection{pcmConsumeStats: &PcmConsumeStats{duration: 1000, totalLength: 32000}}
			var sequence []string
			conn.rollbackAudioScenario(AudioScenarioChorus, true, func(scenario AudioScenario) int {
				if scenario != AudioScenarioChorus {
					t.Fatal(scenario)
				}
				sequence = append(sequence, "scenario")
				return scenarioCode
			}, func() int { sequence = append(sequence, "publish"); return publishCode })
			if conn.audioScenarioDirty != (scenarioCode != 0) || conn.audioPublicationRecovery != (publishCode != 0) || !reflect.DeepEqual(sequence, []string{"scenario", "publish"}) {
				t.Fatalf("scenario=%d publish=%d dirty=%v recovery=%v sequence=%v", scenarioCode, publishCode, conn.audioScenarioDirty, conn.audioPublicationRecovery, sequence)
			}
			if publishCode != 0 && conn.pcmConsumeStats.duration != 0 {
				t.Fatal("failed restoration kept old send estimate")
			}
		}
	}
	conn := &RtcConnection{}
	conn.rollbackAudioScenario(AudioScenarioChorus, false, func(AudioScenario) int { return 0 }, func() int { t.Fatal("published a deliberately inactive track"); return 0 })
	if conn.audioPublicationRecovery {
		t.Fatal("created unintended recovery")
	}
}
