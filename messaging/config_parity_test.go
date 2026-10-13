package messaging

import (
	"errors"
	"reflect"
	"testing"
	"time"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

func TestIdentityModes(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode IdentityMode
		base *native.RtmConfig
		want bool
	}{
		{"default-string", IdentityDefault, nil, true},
		{"explicit-string", IdentityString, &native.RtmConfig{UseStringUserId: false}, true},
		{"explicit-numeric", IdentityNumeric, &native.RtmConfig{UseStringUserId: true}, false},
		{"native-numeric", IdentityDefault, native.NewRtmConfig(), false},
		{"native-string", IdentityDefault, &native.RtmConfig{UseStringUserId: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := (Config{AppID: "app", UserID: "42", Identity: tc.mode, Native: tc.base}).nativeConfig()
			if err != nil || n.UseStringUserId != tc.want {
				t.Fatalf("config=%+v, err=%v", n, err)
			}
		})
	}
}

func TestTypedConfigPreservesAllNativeOptions(t *testing.T) {
	cfg := Config{AppID: "app", UserID: "42", Identity: IdentityNumeric,
		Connection: &ConnectionConfig{Area: AreaJapan | AreaEurope, Protocol: ProtocolTCPOnly, PresenceTimeout: 42 * time.Second, HeartbeatInterval: 7 * time.Second, ReconnectTimeout: 60 * time.Second, Multipath: true, ISPPolicyEnabled: true},
		Log:        &LogConfig{FilePath: "test.log", FileSizeKB: 2048, Level: LogWarning},
		Proxy:      &ProxyConfig{Type: ProxyHTTP, Server: "proxy.test", Port: 1234, Account: "user", Password: "test"},
		Encryption: &EncryptionConfig{Mode: EncryptionAES128GCM, Key: "test", Salt: [32]byte{1, 2, 3}},
		Private:    &PrivateConfig{Service: ServiceStream, AccessPointHosts: []string{"one.test", "two.test"}},
		Parameters: `{"feature":true}`,
	}
	n, err := cfg.nativeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if n.UseStringUserId || n.AppId != "app" || n.UserId != "42" || n.AreaCode != cfg.Connection.Area || n.ProtocolType != 1 || n.PresenceTimeout != 42 || n.HeartbeatInterval != 7 || n.ReconnectTimeout != 60 || !n.Multipath || !n.ISPPolicyEnabled || n.Parameters != cfg.Parameters {
		t.Fatalf("scalar mapping: %+v", n)
	}
	if !reflect.DeepEqual(n.LogConfig, &native.RtmLogConfig{FilePath: "test.log", FileSizeInKB: 2048, Level: native.RtmLogLevelWARN}) || !reflect.DeepEqual(n.ProxyConfig, &native.RtmProxyConfig{ProxyType: native.RtmProxyTypeHTTP, Server: "proxy.test", Port: 1234, Account: "user", Password: "test"}) {
		t.Fatal("log/proxy mapping")
	}
	if n.EncryptionConfig.EncryptionMode != native.RtmEncryptionModeAES128GCM || n.EncryptionConfig.EncryptionKey != "test" || !reflect.DeepEqual(n.EncryptionConfig.EncryptionSalt, cfg.Encryption.Salt[:]) || n.PrivateConfig.ServiceType != uint32(ServiceStream) || !reflect.DeepEqual(n.PrivateConfig.AccessPointHosts, cfg.Private.AccessPointHosts) {
		t.Fatal("encryption/private mapping")
	}
	cfg.Encryption.Salt[0] = 99
	cfg.Private.AccessPointHosts[0] = "changed"
	cfg.Log.FilePath = "changed"
	cfg.Proxy.Server = "changed"
	if n.EncryptionConfig.EncryptionSalt[0] != 1 || n.PrivateConfig.AccessPointHosts[0] != "one.test" || n.LogConfig.FilePath != "test.log" || n.ProxyConfig.Server != "proxy.test" {
		t.Fatal("configuration retained mutable input")
	}
}

func TestConfigNativeCopyAndExplicitOverrides(t *testing.T) {
	base := native.NewRtmConfig()
	base.ISPPolicyEnabled = true
	base.Parameters = `{"native":true}`
	base.EventHandler = &native.RtmEventHandler{}
	base.EncryptionConfig = &native.RtmEncryptionConfig{EncryptionSalt: []byte{1, 2}}
	base.PrivateConfig = &native.RtmPrivateConfig{AccessPointHosts: []string{"host"}}
	n, err := (Config{AppID: "app", UserID: "42", Native: base, Parameters: `{"override":true}`}).nativeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !n.ISPPolicyEnabled || n.Parameters != `{"override":true}` || n.EventHandler != nil || base.Parameters != `{"native":true}` || base.EventHandler == nil {
		t.Fatal("native precedence or mutation")
	}
	base.EncryptionConfig.EncryptionSalt[0] = 99
	base.PrivateConfig.AccessPointHosts[0] = "changed"
	if n.EncryptionConfig.EncryptionSalt[0] != 1 || n.PrivateConfig.AccessPointHosts[0] != "host" {
		t.Fatal("native nested input retained")
	}
}

func TestConfigParityValidation(t *testing.T) {
	cases := []Config{
		{Identity: IdentityMode(255)}, {Connection: &ConnectionConfig{Protocol: ProtocolType(3)}},
		{Connection: &ConnectionConfig{PresenceTimeout: -time.Second}}, {Connection: &ConnectionConfig{HeartbeatInterval: time.Millisecond}},
		{Connection: &ConnectionConfig{ReconnectTimeout: (1 << 32) * time.Second}},
		{Parameters: "null"}, {Parameters: "[]"}, {Parameters: "{}\x00"},
		{Log: &LogConfig{FilePath: "bad\x00"}}, {Proxy: &ProxyConfig{Password: "bad\x00"}},
		{Encryption: &EncryptionConfig{Key: "bad\x00"}}, {Private: &PrivateConfig{AccessPointHosts: []string{""}}},
		{Native: &native.RtmConfig{EncryptionConfig: &native.RtmEncryptionConfig{EncryptionSalt: make([]byte, 33)}}},
	}
	for i, cfg := range cases {
		cfg.AppID = "app"
		cfg.UserID = "42"
		if _, err := cfg.nativeConfig(); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func TestSubscriptionAndPresenceBooleanParity(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s := (SubscribeOptions{DisableMessages: enabled, Presence: enabled, Metadata: enabled, Locks: enabled, Quiet: enabled}).nativeOptions()
		if s.WithMessage == enabled || s.WithPresence != enabled || s.WithMetadata != enabled || s.WithLock != enabled || s.BeQuiet != enabled {
			t.Fatalf("subscription: %+v", s)
		}
		p := (PresenceOptions{ExcludeUserID: enabled, IncludeState: enabled, Page: "next"}).nativeOptions()
		if p.IncludeUserId == enabled || p.IncludeState != enabled || p.Page != "next" {
			t.Fatalf("presence: %+v", p)
		}
	}
}

type recordingLockAcquirer struct {
	retry   bool
	channel string
	kind    ChannelType
	name    string
}

func (n *recordingLockAcquirer) AcquireLock(channel string, kind ChannelType, name string, retry bool, id *uint64) int {
	n.channel, n.kind, n.name, n.retry = channel, kind, name, retry
	*id = 123
	return -7
}

func TestAcquireLockOptionsForwardRetryAndRequestID(t *testing.T) {
	for _, retry := range []bool{false, true} {
		n := &recordingLockAcquirer{}
		code, id := submitLockAcquisition(n, Channel{Name: "room", Type: StreamChannel}, "lock", AcquireLockOptions{Retry: retry})
		if code != -7 || id != 123 || n.retry != retry || n.channel != "room" || n.kind != StreamChannel || n.name != "lock" {
			t.Fatalf("submission=%+v, code=%d, id=%d", n, code, id)
		}
	}
}
