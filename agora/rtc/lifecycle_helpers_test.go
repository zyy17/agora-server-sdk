package agoraservice

import (
	"reflect"
	"testing"
)

func TestServiceConfigLocalAPMDefaults(t *testing.T) {
	custom := NewAPMConfig()
	custom.AgcConfig.Enabled = false
	custom.AiNsConfig.NsEnabled = true
	custom.AiAecConfig.Enabled = true
	for _, test := range []struct {
		name    string
		enabled bool
		config  *APMConfig
	}{
		{name: "disabled"},
		{name: "defaults", enabled: true},
		{name: "custom", enabled: true, config: custom},
		{name: "disabled custom", config: custom},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := NewAgoraServiceConfig()
			cfg.EnableLocalAudioTrackWithAPM = test.enabled
			cfg.LocalAudioTrackAPMConfig = test.config
			serviceCopy := *cfg
			service := newAgoraService()
			service.storeConfig(&serviceCopy)
			if service.serviceConfig != &serviceCopy {
				t.Fatal("service did not keep its configuration copy")
			}
			if cfg.LocalAudioTrackAPMConfig != test.config {
				t.Fatal("caller configuration was changed")
			}
			got := service.serviceConfig.LocalAudioTrackAPMConfig
			if !test.enabled || test.config != nil {
				if got != test.config {
					t.Fatal("explicit configuration or disabled APM was replaced")
				}
				return
			}
			if got == nil || !got.AgcConfig.Enabled || got.AiNsConfig.NsEnabled || got.AiAecConfig.Enabled {
				t.Fatal("unexpected local APM defaults")
			}
		})
	}
	if custom.AgcConfig.Enabled || !custom.AiNsConfig.NsEnabled || !custom.AiAecConfig.Enabled {
		t.Fatal("caller-owned APM settings were changed")
	}
}

func TestNativeObserverCleanupRetainsRetiredCallbacks(t *testing.T) {
	var calls []int
	conn := &RtcConnection{
		retiredObservers: []func(){
			func() { calls = append(calls, 1) },
			func() { calls = append(calls, 2) },
		},
	}
	cleanup := conn.nativeObserverCleanup()
	if len(calls) != 0 {
		t.Fatal("capturing cleanup released observers early")
	}
	// Release clears these fields before the idle queue destroys the connection.
	conn.retiredObservers = nil
	cleanup()
	if !reflect.DeepEqual(calls, []int{1, 2}) {
		t.Fatalf("retired cleanup order = %v, want [1 2]", calls)
	}
}
