package agorartm

import (
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// Infer cgo-only types from converter signatures without importing C in tests.
// The input structs are read by Go; only C-owned strings cross native calls.
func testNativeValues[T, U any](_ func(*T) *U, count int) []T { return make([]T, count) }

func TestMetadataRoundTripContiguous(t *testing.T) {
	want := []MetadataItem{{Key: "first", Value: "one", AuthorUserId: "alice", Revision: 3, UpdateTs: 101}, {Key: "second", Value: "two", AuthorUserId: "bob", Revision: 7, UpdateTs: 202}}
	original := &IMetadata{majorRevision: 9, items: want, itemCount: 100} // stale count must never read past the slice
	data, free := newCMetadata(original)
	got := CMetadataToIMetadata(data)
	free()
	if got.GetMajorRevision() != 9 || !reflect.DeepEqual(got.Items(), want) {
		t.Fatalf("metadata mismatch: %#v", got)
	}
	copied := got.Items()
	copied[0].Value = "changed"
	if got.Items()[0].Value != "one" {
		t.Fatal("Items aliases internal slice")
	}
}

func TestStateItemsRoundTripContiguous(t *testing.T) {
	want := []StateItem{{Key: "first", Value: "one"}, {Key: "second", Value: "two"}}
	ptr, free := newCStateItems([]*StateItem{&want[0], &want[1]})
	got := copyStateItems(ptr, 2)
	free()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("states mismatch: %#v", got)
	}
}

func TestNativeArraysDeepCopy(t *testing.T) {
	stringsPtr, freeStrings := newCStringArray([]string{"first", "second", "owner", "value"})
	raw := nativeSlice(stringsPtr, 4)
	states, freeStates := newCStateItems([]*StateItem{{Key: "status", Value: "online"}, {Key: "mood", Value: "happy"}})
	users := testNativeValues(CUserStateToUserState, 2)
	users[0].userId = raw[0]
	users[0].states = states
	users[0].statesCount = 2
	users[1].userId = raw[1]
	userCopy := copyUserStates(&users[0], 2)
	locks := testNativeValues(CLockDetailToLockDetail, 2)
	locks[0].lockName = raw[0]
	locks[0].owner = raw[2]
	locks[0].ttl = 10
	locks[1].lockName = raw[1]
	locks[1].owner = raw[2]
	locks[1].ttl = 20
	lockCopy := copyLockDetails(&locks[0], 2)
	channels := testNativeValues(CChannelInfoToChannelInfo, 2)
	channels[0].channelName = raw[0]
	channels[0].channelType = 1
	channels[1].channelName = raw[1]
	channels[1].channelType = 2
	channelCopy := copyChannelInfos(&channels[0], 2)
	freeStates()
	freeStrings()
	if len(userCopy) != 2 || userCopy[0].UserId != "first" || userCopy[1].UserId != "second" || len(userCopy[0].States) != 2 || userCopy[0].States[1].Value != "happy" {
		t.Fatalf("bad user copy: %#v", userCopy)
	}
	if len(lockCopy) != 2 || lockCopy[1].LockName != "second" || lockCopy[1].Owner != "owner" || lockCopy[1].Ttl != 20 {
		t.Fatalf("bad lock copy: %#v", lockCopy)
	}
	if len(channelCopy) != 2 || channelCopy[1].ChannelName != "second" || channelCopy[1].ChannelType != 2 {
		t.Fatalf("bad channel copy: %#v", channelCopy)
	}
	if got := copyUserStates(nil, 9); len(got) != 0 {
		t.Fatalf("nil pointer should produce empty array: %#v", got)
	}
}

func TestHistoryPreservesBinaryNULAndLength(t *testing.T) {
	ptr, free := newCStringArray([]string{"a\x00b\xfftail", "publisher", "custom"})
	raw := nativeSlice(ptr, 3)
	messages := testNativeValues(CHistoryMessageToHistoryMessage, 2)
	messages[0].message = raw[0]
	messages[0].messageLength = 4
	messages[0].publisher = raw[1]
	messages[0].customType = raw[2]
	messages[1].message = raw[0]
	messages[1].messageLength = 8
	got := copyHistoryMessages(&messages[0], 2)
	free()
	if len(got) != 2 || got[0].Message != "a\x00b\xff" || got[0].MessageLength != 4 || got[1].Message != "a\x00b\xfftail" {
		t.Fatalf("binary history altered: %#v", got)
	}
}

func TestCallbackRegistryAndUnsubscribeBridge(t *testing.T) {
	calls := 0
	client := &IRtmClient{cEventHandler: CRtmEventHandler(), handler: &RtmEventHandler{OnUnsubscribeResult: func(id uint64, channel string, code int) {
		if id != 42 || code != 0 {
			t.Errorf("wrong callback: %d %d", id, code)
		}
		calls++
	}}}
	registerRtmClient(client)
	handler := client.cEventHandler
	cgo_RtmEventHandlerBridge_onUnsubscribeResult(handler, 42, nil, 0)
	unregisterRtmClient(client)
	cgo_RtmEventHandlerBridge_onUnsubscribeResult(handler, 42, nil, 0)
	if calls != 1 {
		t.Fatalf("callback count=%d", calls)
	}
	if client.Release() != 0 || client.Release() != 0 {
		t.Fatal("release was not idempotent")
	}
	// A stale address is used only as a registry key; it is never dereferenced.
	cgo_RtmEventHandlerBridge_onUnsubscribeResult(handler, 42, nil, 0)
	cgo_RtmEventHandlerBridge_onUnsubscribeResult(nil, 42, nil, 0)
}

func TestMetadataCallbackOwnsData(t *testing.T) {
	var got *IMetadata
	client := &IRtmClient{cEventHandler: CRtmEventHandler(), handler: &RtmEventHandler{OnGetUserMetadataResult: func(_ uint64, _ string, data *IMetadata, _ int) { got = data }}}
	registerRtmClient(client)
	defer client.Release()
	ptr, free := newCMetadata(&IMetadata{items: []MetadataItem{{Key: "first", Value: "one"}, {Key: "second", Value: "two"}}})
	cgo_RtmEventHandlerBridge_onGetUserMetadataResult(client.cEventHandler, 1, nil, ptr, 0)
	free()
	if got == nil || len(got.Items()) != 2 || got.Items()[1].Value != "two" {
		t.Fatalf("invalid metadata callback: %#v", got)
	}
}

func TestCallbackRegistryConcurrentLookup(t *testing.T) {
	client := &IRtmClient{cEventHandler: CRtmEventHandler(), handler: &RtmEventHandler{}}
	registerRtmClient(client)
	handler := client.cEventHandler
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 1000; j++ {
				_ = lookupRtmClient(handler)
			}
		}()
	}
	unregisterRtmClient(client)
	workers.Wait()
	client.Release()
}

func TestNilHandlesAndLocalValidation(t *testing.T) {
	var client *IRtmClient
	if client.Release() != 0 {
		t.Fatal("nil release failed")
	}
	if code, _ := client.Login(""); code != -10001 {
		t.Fatalf("nil login error=%d", code)
	}
	var id uint64
	if code := (&IRtmStorage{}).GetUserMetadata("", &id); code != -10001 {
		t.Fatalf("nil storage error=%d", code)
	}
	if code := (&IRtmLock{}).GetLocks("", 0, &id); code != -10001 {
		t.Fatalf("nil lock error=%d", code)
	}
	if code := (&IRtmPresence{}).SetState("", 0, nil, 0, &id); code != -10001 {
		t.Fatalf("nil presence error=%d", code)
	}
	if code, _ := (&IStreamChannel{}).PublishTopicMessage("", "", 1, nil); code != -10001 {
		t.Fatalf("nil stream error=%d", code)
	}
	if len(copyStrings(nil, 42)) != 0 {
		t.Fatal("nil string array should be empty")
	}
}

func TestNativeCreateReleaseOffline(t *testing.T) {
	if os.Getenv("AGORA_NATIVE_SMOKE") != "1" {
		t.Skip("set AGORA_NATIVE_SMOKE=1 for native construction smoke")
	}
	t.Chdir(t.TempDir())
	// Creation and local teardown only: no login, network operation, or credential.
	t.Logf("native RTM SDK version: %s", GetVersion())
	for _, withHandler := range []bool{false, true} {
		config := NewRtmConfig()
		config.AppId = strings.Repeat("0", 32)
		config.UserId = "offline-binding-test"
		config.UseStringUserId = true
		config.LogConfig = &RtmLogConfig{FilePath: t.TempDir() + "/rtm.log", FileSizeInKB: 1024, Level: RtmLogLevelNONE}
		if withHandler {
			config.EventHandler = &RtmEventHandler{}
		}
		client, code := NewRtmClientWithError(config)
		if client == nil {
			t.Errorf("offline construction failed (handler=%t): %d %s", withHandler, code, GetErrorReason(code))
			continue
		}
		if client.GetStorage() == nil || client.GetPresence() == nil || client.GetLock() == nil || client.GetHistory() == nil {
			t.Fatal("missing native interface")
		}
		channel, code := client.CreateStreamChannelWithError("offline-stream")
		if channel == nil || code != 0 {
			client.Release()
			t.Fatalf("native stream creation failed: %d", code)
		}
		if channel.GetChannelName() != "offline-stream" {
			t.Error("native stream name was not preserved")
		}
		if code, _ := channel.PublishTopicMessage("topic", "short", 100, nil); code != -10014 {
			t.Errorf("invalid payload length was not rejected: %d", code)
		}
		if code := channel.Release(); code != 0 {
			t.Errorf("stream release=%d", code)
		}
		if code := channel.Release(); code != 0 {
			t.Errorf("second stream release=%d", code)
		}
		if code := client.Release(); code != 0 {
			t.Fatalf("release=%d", code)
		}
		if code := client.Release(); code != 0 {
			t.Fatalf("repeated release=%d", code)
		}
	}
}

func TestNativeConstructionFailureCleansRegistry(t *testing.T) {
	if os.Getenv("AGORA_NATIVE_SMOKE") != "1" {
		t.Skip("set AGORA_NATIVE_SMOKE=1 for native construction smoke")
	}
	t.Chdir(t.TempDir())
	rtmClients.RLock()
	before := len(rtmClients.clients)
	rtmClients.RUnlock()
	if client, code := NewRtmClientWithError(nil); client != nil || code != -10014 {
		t.Fatalf("nil config: %v %d", client, code)
	}
	config := NewRtmConfig()
	config.AppId = "bad-app-id"
	config.UserId = "bad-numeric-id"
	config.LogConfig = &RtmLogConfig{FilePath: t.TempDir() + "/rtm.log", FileSizeInKB: 1024, Level: RtmLogLevelNONE}
	config.EventHandler = &RtmEventHandler{}
	if client, code := NewRtmClientWithError(config); client != nil || code == 0 {
		if client != nil {
			client.Release()
		}
		t.Fatalf("invalid native config accepted: %v %d", client, code)
	}
	rtmClients.RLock()
	after := len(rtmClients.clients)
	rtmClients.RUnlock()
	if before != after {
		t.Fatalf("failed create leaked callback registry: %d -> %d", before, after)
	}
}

func TestArrayResultCallbacksIncludeEveryElement(t *testing.T) {
	ptr, free := newCStringArray([]string{"one", "two"})
	defer free()
	raw := nativeSlice(ptr, 2)
	users := testNativeValues(CUserStateToUserState, 2)
	users[0].userId = raw[0]
	users[1].userId = raw[1]
	locks := testNativeValues(CLockDetailToLockDetail, 2)
	locks[0].lockName = raw[0]
	locks[1].lockName = raw[1]
	channels := testNativeValues(CChannelInfoToChannelInfo, 2)
	channels[0].channelName = raw[0]
	channels[1].channelName = raw[1]
	calls := 0
	checkUsers := func(_ uint64, got []UserState, _ string, _ int) {
		calls++
		if len(got) != 2 || got[1].UserId != "two" {
			t.Fatalf("bad callback users: %#v", got)
		}
	}
	checkChannels := func(_ uint64, got []ChannelInfo, _ int) {
		calls++
		if len(got) != 2 || got[1].ChannelName != "two" {
			t.Fatalf("bad callback channels: %#v", got)
		}
	}
	client := &IRtmClient{cEventHandler: CRtmEventHandler(), handler: &RtmEventHandler{
		OnWhoNowResult: checkUsers, OnGetOnlineUsersResult: checkUsers,
		OnWhereNowResult: checkChannels, OnGetUserChannelsResult: checkChannels,
		OnGetLocksResult: func(_ uint64, _ string, _ RtmChannelType, got []LockDetail, _ int) {
			calls++
			if len(got) != 2 || got[1].LockName != "two" {
				t.Fatalf("bad callback locks: %#v", got)
			}
		},
	}}
	registerRtmClient(client)
	defer client.Release()
	cgo_RtmEventHandlerBridge_onWhoNowResult(client.cEventHandler, 1, &users[0], 2, nil, 0)
	cgo_RtmEventHandlerBridge_onGetOnlineUsersResult(client.cEventHandler, 2, &users[0], 2, nil, 0)
	cgo_RtmEventHandlerBridge_onWhereNowResult(client.cEventHandler, 3, &channels[0], 2, 0)
	cgo_RtmEventHandlerBridge_onGetUserChannelsResult(client.cEventHandler, 4, &channels[0], 2, 0)
	cgo_RtmEventHandlerBridge_onGetLocksResult(client.cEventHandler, 5, nil, 0, &locks[0], 2, 0)
	if calls != 5 {
		t.Fatalf("callbacks=%d", calls)
	}
}
