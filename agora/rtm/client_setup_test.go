package agorartm

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestRtmClientRejectsNilConfig(t *testing.T) {
	if client, code := NewRtmClientWithError(nil); client != nil || code != -10014 {
		t.Fatalf("nil config: client=%v code=%d", client, code)
	}
	if client := NewRtmClient(nil); client != nil {
		t.Fatalf("nil config created a client: %v", client)
	}
}

func TestRtmClientEventHandlerSnapshot(t *testing.T) {
	calls := 0
	handler := &RtmEventHandler{OnUnsubscribeResult: func(uint64, string, int) { calls++ }}
	client := newRtmClientWithEventHandler(handler)
	if client == nil {
		t.Fatal("native event handler allocation failed")
	}
	defer client.Release()
	if client.handler == handler {
		t.Fatal("client retained the mutable caller-owned handler")
	}
	handler.OnUnsubscribeResult = func(uint64, string, int) { t.Error("callback used the changed handler") }
	registerRtmClient(client)
	cgo_RtmEventHandlerBridge_onUnsubscribeResult(client.cEventHandler, 1, nil, 0)
	if calls != 1 {
		t.Fatalf("original callback count=%d", calls)
	}
}

func TestRtmClientDefaultEventHandler(t *testing.T) {
	client := newRtmClientWithEventHandler(nil)
	if client == nil {
		t.Fatal("native event handler allocation failed")
	}
	defer client.Release()
	if client.handler == nil || client.cEventHandler == nil {
		t.Fatal("client needs both Go and native handlers without a caller-supplied handler")
	}
	registerRtmClient(client)
	cgo_RtmEventHandlerBridge_onUnsubscribeResult(client.cEventHandler, 1, nil, 0)
}

func TestRtmClientConfigResourcesPreserveFields(t *testing.T) {
	resources := newRtmClientConfigResources()
	if resources == nil {
		t.Fatal("native configuration allocation failed")
	}
	defer resources.free()
	config := &RtmConfig{
		AppId: "test-app", UserId: "test-user", AreaCode: RtmAreaCodeGLOB,
		ProtocolType: 3, PresenceTimeout: 41, HeartbeatInterval: 7, ReconnectTimeout: 11,
		Context: unsafe.Pointer(resources.config), UseStringUserId: true, Multipath: true,
		LogConfig:        &RtmLogConfig{FilePath: "rtm.log", FileSizeInKB: 2048, Level: RtmLogLevelNONE},
		EncryptionConfig: &RtmEncryptionConfig{EncryptionMode: RtmEncryptionModeNONE, EncryptionKey: "key", EncryptionSalt: []byte{1, 2, 3}},
		ProxyConfig:      &RtmProxyConfig{Server: "proxy.test", Port: 9000, Account: "account", Password: "password"},
		PrivateConfig:    &RtmPrivateConfig{ServiceType: uint32(RtmServiceTypeMESSAGE), AccessPointHosts: []string{"one.test", "two.test"}},
	}
	resources.populate(config)
	native := resources.config
	t.Run("identity", func(t *testing.T) {
		if got := copyStrings(&native.appId, 1); !reflect.DeepEqual(got, []string{config.AppId}) {
			t.Fatalf("app ID=%v", got)
		}
		if got := copyStrings(&native.userId, 1); !reflect.DeepEqual(got, []string{config.UserId}) {
			t.Fatalf("user ID=%v", got)
		}
		if native.appId != resources.appID || native.userId != resources.userID {
			t.Fatal("native config must borrow the owned identity strings")
		}
	})
	t.Run("options", func(t *testing.T) {
		if RtmAreaCode(native.areaCode) != config.AreaCode || uint32(native.protocolType) != config.ProtocolType ||
			uint32(native.presenceTimeout) != config.PresenceTimeout || uint32(native.heartbeatInterval) != config.HeartbeatInterval ||
			uint32(native.reconnectTimeout) != config.ReconnectTimeout || native.context != config.Context ||
			bool(native.useStringUserId) != config.UseStringUserId || bool(native.multipath) != config.Multipath {
			t.Fatal("native scalar configuration does not match the input")
		}
	})
	t.Run("log", func(t *testing.T) {
		if native.logConfig != *resources.log || copyStrings(&native.logConfig.filePath, 1)[0] != config.LogConfig.FilePath {
			t.Fatal("native log configuration does not match the input")
		}
	})
	t.Run("encryption", func(t *testing.T) {
		if native.encryptionConfig != *resources.encryption || copyStrings(&native.encryptionConfig.encryptionKey, 1)[0] != config.EncryptionConfig.EncryptionKey {
			t.Fatal("native encryption configuration does not match the input")
		}
		for i, want := range config.EncryptionConfig.EncryptionSalt {
			if byte(native.encryptionConfig.encryptionSalt[i]) != want {
				t.Fatalf("encryption salt[%d]=%d, want %d", i, native.encryptionConfig.encryptionSalt[i], want)
			}
		}
	})
	t.Run("proxy", func(t *testing.T) {
		if native.proxyConfig != *resources.proxy || copyStrings(&native.proxyConfig.server, 1)[0] != config.ProxyConfig.Server {
			t.Fatal("native proxy configuration does not match the input")
		}
	})
	t.Run("private", func(t *testing.T) {
		if native.privateConfig != *resources.private {
			t.Fatal("native private configuration does not match the input")
		}
		if got := copyStrings(native.privateConfig.accessPointHosts, native.privateConfig.accessPointHostsCount); !reflect.DeepEqual(got, config.PrivateConfig.AccessPointHosts) {
			t.Fatalf("private hosts=%v", got)
		}
	})
}
