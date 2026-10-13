package agoraservice

import (
	"encoding/json"
	"testing"
)

func TestExternalAudioSendSpeedParameters(t *testing.T) {
	for _, speed := range []int{2, 3, 5, 99} {
		params := &SendExternalAudioParameters{Enabled: true, SendMs: 500, SendSpeed: speed}
		calls := 0
		code := configureExtraSendFrameSpeed(params, func(value string) int {
			calls++
			var object map[string]int
			if err := json.Unmarshal([]byte(value), &object); err != nil {
				t.Fatalf("invalid native JSON: %q: %v", value, err)
			}
			if object["che.audio.extra_send_frames_per_interval_for_fake_adm"] != min(speed, 5) {
				t.Fatal(object)
			}
			return -17
		})
		if calls != 1 || code != -17 || params.SendSpeed != speed {
			t.Fatalf("calls=%d code=%d config=%+v", calls, code, params)
		}
	}
	for _, params := range []*SendExternalAudioParameters{nil, {}, {Enabled: true}, {Enabled: true, SendMs: 500, SendSpeed: 1}} {
		if code := configureExtraSendFrameSpeed(params, func(string) int { t.Fatal("invalid config reached native SDK"); return 0 }); code != -2001 {
			t.Fatal(code)
		}
	}
}
