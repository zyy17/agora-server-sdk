package messaging

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

func TestEveryNativeCallbackHasManagedHandler(t *testing.T) {
	h := reflect.ValueOf(testClient(t, 128).handler()).Elem()
	for i := 0; i < h.NumField(); i++ {
		if h.Field(i).Kind() == reflect.Func && h.Field(i).IsNil() {
			t.Errorf("missing handler %s", h.Type().Field(i).Name)
		}
	}
}

func TestEveryResultProducesTypedNotification(t *testing.T) {
	c := testClient(t, 128)
	c.resultEvents = true
	h := reflect.ValueOf(c.handler()).Elem()
	operations := map[RequestOperation]bool{}
	for i := 0; i < h.NumField(); i++ {
		name := h.Type().Field(i).Name
		if !strings.HasSuffix(name, "Result") {
			continue
		}
		fn := h.Field(i)
		args := make([]reflect.Value, fn.Type().NumIn())
		for j := range args {
			args[j] = reflect.Zero(fn.Type().In(j))
		}
		args[0] = reflect.ValueOf(uint64(42))
		fn.Call(args)
		ev, err := c.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if ev.Kind != EventRequestResult || ev.RequestResult == nil || ev.RequestResult.RequestID != 42 || ev.RequestResult.Error != nil || ev.RequestResult.Operation == "" {
			t.Fatalf("%s: %+v", name, ev)
		}
		if operations[ev.RequestResult.Operation] {
			t.Errorf("duplicate operation %s", ev.RequestResult.Operation)
		}
		operations[ev.RequestResult.Operation] = true
	}
	if len(operations) != 38 {
		t.Fatalf("got %d result handlers; update exhaustive test for pinned ABI", len(operations))
	}
}

func TestRequestResultDisabledByDefault(t *testing.T) {
	c := testClient(t, 1)
	c.handler().OnLoginResult(1, 0)
	select {
	case e := <-c.Events():
		t.Fatalf("unexpected event: %+v", e)
	default:
	}
}

func TestCanceledRetryRemainsObservable(t *testing.T) {
	c := testClient(t, 8)
	c.resultEvents = true
	h := c.handler()
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- c.call(ctx, "acquire lock", func() (int, uint64) { close(started); return 0, 17 }) }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	// Native retry has no cancellation API. A late success and repeated result
	// must remain observable after the caller has abandoned its waiter.
	for i := 0; i < 2; i++ {
		h.OnAcquireLockResult(17, "room", MessageChannel, "mutex", 0, "")
		e, err := c.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		r := e.RequestResult
		if r == nil || r.Operation != RequestAcquireLock || r.RequestID != 17 || r.Channel != (Channel{Name: "room", Type: MessageChannel}) || r.LockName != "mutex" || r.Error != nil {
			t.Fatalf("late result: %+v", r)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) != 0 || len(c.early) != 0 {
		t.Fatal("late result recreated waiter")
	}
}

func TestRepeatedEarlyResultDoesNotReplaceFirst(t *testing.T) {
	c := testClient(t, 4)
	c.resultEvents = true
	h := c.handler()
	_, err := c.request(context.Background(), "acquire lock", nil, func() (int, uint64) {
		h.OnAcquireLockResult(5, "room", MessageChannel, "mutex", 0, "")
		h.OnAcquireLockResult(5, "room", MessageChannel, "mutex", int(CodeLockAcquireFailed), "second")
		return 0, 5
	})
	if err != nil {
		t.Fatalf("first result lost: %v", err)
	}
	first := <-c.Events()
	second := <-c.Events()
	if first.RequestResult.Error != nil || second.RequestResult.Error == nil || second.RequestResult.Error.Details != "second" {
		t.Fatal("repeated result event lost")
	}
}

func TestResultOverflowDoesNotDropRequestCompletion(t *testing.T) {
	c := testClient(t, 1)
	c.resultEvents = true
	c.emit(Event{Kind: EventTokenExpiring})
	if err := c.call(context.Background(), "login", func() (int, uint64) { c.handler().OnLoginResult(3, 0); return 0, 3 }); err != nil {
		t.Fatal(err)
	}
	if c.DroppedEvents() != 1 {
		t.Fatalf("dropped=%d", c.DroppedEvents())
	}
}

func TestResultPayloadDoesNotAliasMethodReturnOrNative(t *testing.T) {
	c := testClient(t, 8)
	c.resultEvents = true
	h := c.handler()
	state := []native.UserState{{UserId: "one", States: []native.StateItem{{Key: "k", Value: "v"}}}, {UserId: "two"}}
	v, err := c.request(context.Background(), "users", nil, func() (int, uint64) { h.OnGetOnlineUsersResult(8, state, "next", 0); return 0, 8 })
	if err != nil {
		t.Fatal(err)
	}
	state[0].States[0].Value = "native changed"
	v.(PresencePage).Users[0].State["k"] = "caller changed"
	r := (<-c.Events()).RequestResult
	if r.Presence.Users[0].State["k"] != "v" || len(r.Presence.Users) != 2 || r.Presence.NextPage != "next" {
		t.Fatalf("payload aliased: %+v", r.Presence)
	}
	h.OnSubscribeTopicResult(9, "stream", "self", "topic", native.UserList{Users: []string{"ok"}}, native.UserList{Users: []string{"failed"}}, int(CodeInvalidParameter))
	r = (<-c.Events()).RequestResult
	if r.Error == nil || r.Error.Code != CodeInvalidParameter || r.Channel.Name != "stream" || r.UserID != "self" || r.Topic != "topic" || r.Subscription.Succeeded[0] != "ok" || r.Subscription.Failed[0] != "failed" {
		t.Fatalf("partial result lost: %+v", r)
	}
	h.OnJoinTopicResult(10, "stream", "resolved-user", "topic", "returned-metadata", 0)
	r = (<-c.Events()).RequestResult
	if r.UserID != "resolved-user" || r.TopicMetadata != "returned-metadata" {
		t.Fatalf("callback metadata lost: %+v", r)
	}
}

func TestCloneRequestResultCopiesEveryMutablePayload(t *testing.T) {
	in := RequestResult{Error: &Error{Details: "details"}, Presence: &PresencePage{Users: []UserState{{State: map[string]string{"k": "v"}}}}, State: &UserState{State: map[string]string{"k": "v"}}, Channels: []Channel{{Name: "room"}}, Metadata: &Metadata{Items: []MetadataItem{{Key: "k"}}}, Locks: []Lock{{Name: "lock"}}, Subscription: &TopicSubscription{Succeeded: []string{"yes"}, Failed: []string{"no"}}, Subscribers: []string{"user"}, History: &HistoryPage{Messages: []HistoryMessage{{Data: []byte{0, 1, 0}}}}}
	out := cloneRequestResult(in)
	in.Error.Details = "changed"
	in.Presence.Users[0].State["k"] = "changed"
	in.State.State["k"] = "changed"
	in.Channels[0].Name = "changed"
	in.Metadata.Items[0].Key = "changed"
	in.Locks[0].Name = "changed"
	in.Subscription.Succeeded[0] = "changed"
	in.Subscription.Failed[0] = "changed"
	in.Subscribers[0] = "changed"
	in.History.Messages[0].Data[0] = 99
	if out.Error.Details != "details" || out.Presence.Users[0].State["k"] != "v" || out.State.State["k"] != "v" || out.Channels[0].Name != "room" || out.Metadata.Items[0].Key != "k" || out.Locks[0].Name != "lock" || out.Subscription.Succeeded[0] != "yes" || out.Subscription.Failed[0] != "no" || out.Subscribers[0] != "user" || out.History.Messages[0].Data[0] != 0 {
		t.Fatal("mutable result payload was not cloned")
	}
}

func TestTimedOutResultsAndResultsAfterClose(t *testing.T) {
	c := testClient(t, 4)
	c.resultEvents = true
	c.timeout = time.Millisecond
	_, err := c.request(context.Background(), "acquire lock", nil, func() (int, uint64) { return 0, 12 })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	h := c.handler()
	h.OnAcquireLockResult(12, "room", MessageChannel, "lock", 0, "")
	if (<-c.Events()).RequestResult.RequestID != 12 {
		t.Fatal("late timeout result lost")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	h.OnAcquireLockResult(12, "room", MessageChannel, "lock", 0, "")
	if _, ok := <-c.Events(); ok {
		t.Fatal("event admitted after close")
	}
}

func TestRetryFailureThenSuccessRemainsObservable(t *testing.T) {
	c := testClient(t, 4)
	c.resultEvents = true
	h := c.handler()
	err := c.call(context.Background(), "acquire lock", func() (int, uint64) {
		h.OnAcquireLockResult(77, "room", MessageChannel, "lock", int(CodeLockAcquireFailed), "occupied")
		return 0, 77
	})
	var rejection *Error
	if !errors.As(err, &rejection) || rejection.Code != CodeLockAcquireFailed {
		t.Fatalf("initial failure: %v", err)
	}
	first := <-c.Events()
	if first.RequestResult.Error == nil {
		t.Fatal("initial failure event lost")
	}
	// A failed attempt does not establish that a retrying native request stopped.
	h.OnAcquireLockResult(77, "room", MessageChannel, "lock", 0, "")
	late := <-c.Events()
	if late.RequestResult.RequestID != 77 || late.RequestResult.Error != nil {
		t.Fatal("late successful retry lost")
	}
}
