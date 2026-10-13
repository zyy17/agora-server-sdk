package messaging

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

func testClient(t *testing.T, buffer int) *Client {
	t.Helper()
	c := newClient(buffer, time.Second)
	t.Cleanup(func() { _ = c.Close() })
	return c
}
func TestRequestCallbackBeforeReturn(t *testing.T) {
	c := testClient(t, 1)
	got, err := c.request(context.Background(), "early", nil, func() (int, uint64) { c.complete(9, result{value: "done"}); return 0, 9 })
	if err != nil || got != "done" {
		t.Fatalf("got %v, %v", got, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) != 0 || c.early != nil {
		t.Fatal("retained completed request")
	}
}
func TestImmediateFailure(t *testing.T) {
	c := testClient(t, 1)
	_, err := c.request(context.Background(), "publish", nil, func() (int, uint64) { return -7, 0 })
	var sdk *Error
	if !errors.As(err, &sdk) || sdk.Code != -7 || sdk.Op != "publish" {
		t.Fatalf("wrong error: %v", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) != 0 {
		t.Fatal("immediate failure left waiter")
	}
}
func TestAsynchronousFailure(t *testing.T) {
	c := testClient(t, 1)
	_, err := c.request(context.Background(), "acquire", nil, func() (int, uint64) { c.complete(11, result{code: -23, details: "locked"}); return 0, 11 })
	var sdk *Error
	if !errors.As(err, &sdk) || sdk.Code != -23 || sdk.Details != "locked" {
		t.Fatalf("wrong error: %v", err)
	}
}
func TestContextCancellationRemovesWaiter(t *testing.T) {
	c := testClient(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := c.request(ctx, "pending", nil, func() (int, uint64) { close(started); return 0, 17 })
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	c.complete(17, result{value: "late"})
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) != 0 || len(c.early) != 0 {
		t.Fatal("late result leaked")
	}
}
func TestDefaultTimeout(t *testing.T) {
	c := testClient(t, 1)
	c.timeout = 5 * time.Millisecond
	_, err := c.request(context.Background(), "missing callback", nil, func() (int, uint64) { return 0, 1 })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}
func TestCanceledRequestDoesNotStart(t *testing.T) {
	c := testClient(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.request(ctx, "canceled", nil, func() (int, uint64) { t.Fatal("native action started"); return 0, 1 })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
func TestWaitingForGateHonorsContext(t *testing.T) {
	c := testClient(t, 1)
	<-c.gate
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, err := c.request(ctx, "blocked", nil, func() (int, uint64) { t.Fatal("native action started"); return 0, 1 })
	c.unlock()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}
func TestConcurrentCorrelation(t *testing.T) {
	c := testClient(t, 1)
	const n = 200
	ids := make(chan uint64, n)
	var serial atomic.Uint64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			want := serial.Add(1)
			v, err := c.request(context.Background(), "concurrent", nil, func() (int, uint64) { ids <- want; return 0, want })
			if err != nil || v != want {
				t.Errorf("got %v, %v; want %d", v, err, want)
			}
		}()
	}
	all := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		all = append(all, <-ids)
	}
	for i := len(all) - 1; i >= 0; i-- {
		c.complete(all[i], result{value: all[i]})
	}
	wg.Wait()
}
func TestCloseUnblocksRequestsAndIsIdempotent(t *testing.T) {
	c := testClient(t, 1)
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := c.request(context.Background(), "pending", nil, func() (int, uint64) { close(started); return 0, 88 })
		done <- err
	}()
	<-started
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := c.request(context.Background(), "closed", nil, func() (int, uint64) { t.Fatal("native action after close"); return 0, 1 })
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v", err)
	}
	if _, ok := <-c.Events(); ok {
		t.Fatal("events still open")
	}
	c.complete(88, result{})
	c.emit(Event{Kind: EventMessage})
}
func TestEventQueueNeverBlocksAndCopies(t *testing.T) {
	c := testClient(t, 1)
	h := c.handler()
	data := []byte{0, 1, 2, 0, 255}
	h.OnMessageEvent(&native.MessageEvent{ChannelName: "room", Message: data})
	data[1] = 9
	for i := 0; i < 10000; i++ {
		h.OnMessageEvent(&native.MessageEvent{Message: data})
	}
	if got := c.DroppedEvents(); got != 10000 {
		t.Fatalf("dropped=%d", got)
	}
	event := <-c.Events()
	if !bytes.Equal(event.Message.Data, []byte{0, 1, 2, 0, 255}) {
		t.Fatalf("aliased payload: %v", event.Message.Data)
	}
	// Completion must not be lost when the event queue is saturated.
	c.emit(Event{Kind: EventTokenExpiring})
	_, err := c.request(context.Background(), "login", nil, func() (int, uint64) { h.OnLoginResult(1, 0); return 0, 1 })
	if err != nil {
		t.Fatal(err)
	}
}
func TestPresenceAndMetadataCopy(t *testing.T) {
	c := testClient(t, 2)
	h := c.handler()
	state := &native.UserState{UserId: "alice", States: []native.StateItem{{Key: "status", Value: "online"}}}
	h.OnPresenceEvent(&native.PresenceEvent{Snapshot: &native.SnapshotInfo{UserStateList: []*native.UserState{state}}})
	state.States[0].Value = "changed"
	ev := <-c.Events()
	if ev.Presence.Snapshot[0].State["status"] != "online" {
		t.Fatal("aliased presence")
	}
	m := &native.IMetadata{}
	m.SetMajorRevision(4)
	m.SetMetadataItem(&native.MetadataItem{Key: "first", Value: "one"})
	m.SetMetadataItem(&native.MetadataItem{Key: "second", Value: "two"})
	h.OnStorageEvent(&native.StorageEvent{Data: m})
	m.SetMajorRevision(8)
	m.SetMetadataItem(&native.MetadataItem{Key: "third", Value: "three"})
	ev = <-c.Events()
	if ev.Storage.Metadata.Revision != 4 || len(ev.Storage.Metadata.Items) != 2 {
		t.Fatalf("aliased metadata: %+v", ev.Storage.Metadata)
	}
}
func TestHistoryAndMultiElementResult(t *testing.T) {
	c := testClient(t, 1)
	h := c.handler()
	got, err := c.request(context.Background(), "history", nil, func() (int, uint64) {
		h.OnGetHistoryMessagesResult(7, []native.HistoryMessage{{Message: "a\x00b", MessageLength: 3}, {Message: "second", MessageLength: 6}}, 99, 0)
		return 0, 7
	})
	if err != nil {
		t.Fatal(err)
	}
	page := got.(HistoryPage)
	if len(page.Messages) != 2 || string(page.Messages[0].Data) != "a\x00b" || page.NextStart != 99 {
		t.Fatalf("wrong history: %+v", page)
	}
	got, err = c.request(context.Background(), "presence", nil, func() (int, uint64) {
		h.OnGetOnlineUsersResult(8, []native.UserState{{UserId: "first"}, {UserId: "second"}}, "next", 0)
		return 0, 8
	})
	if err != nil {
		t.Fatal(err)
	}
	p := got.(PresencePage)
	if len(p.Users) != 2 || p.Users[1].UserID != "second" || p.NextPage != "next" {
		t.Fatalf("wrong presence: %+v", p)
	}
}
func TestClosedStreamPreventsStart(t *testing.T) {
	c := testClient(t, 1)
	s := &Stream{client: c, closed: true}
	_, err := s.request(context.Background(), "closed", func() (int, uint64) { t.Fatal("closed stream executed"); return 0, 1 })
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v", err)
	}
}
func TestCloseConcurrentWithEventsAndRequests(t *testing.T) {
	c := testClient(t, 2)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.emit(Event{})
				c.complete(uint64(j), result{})
				_ = c.DroppedEvents()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = c.Close() }()
	}
	wg.Wait()
}
func TestValidation(t *testing.T) {
	cases := []Config{{}, {AppID: "a"}, {AppID: "a\x00b", UserID: "u"}, {AppID: "a", UserID: "u", EventBuffer: -1}, {AppID: "a", UserID: "u", RequestTimeout: -1}}
	for _, cfg := range cases {
		if _, err := New(cfg); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("config %+v: %v", cfg, err)
		}
	}
	c := testClient(t, 1)
	ctx := context.Background()
	for _, err := range []error{
		//lint:ignore SA1012 This negative test verifies that nil contexts are rejected.
		c.Login(nil, ""), c.Publish(ctx, "", []byte{1}, PublishOptions{}), c.Publish(ctx, "room", nil, PublishOptions{}),
		c.Publish(ctx, "room", []byte{255}, PublishOptions{Text: true}), c.SetState(ctx, Channel{Name: "room", Type: MessageChannel}, map[string]string{"bad\x00key": "value"}),
		c.SetLock(ctx, Channel{Name: "room", Type: MessageChannel}, "lock", time.Millisecond), c.SetParameters(ctx, "[]"), c.WithNative(ctx, nil),
	} {
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("wrong validation: %v", err)
		}
	}
}
func TestNextCancellationAndClose(t *testing.T) {
	c := testClient(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	c.emit(Event{Kind: EventMessage})
	_ = c.Close()
	if e, err := c.Next(context.Background()); err != nil || e.Kind != EventMessage {
		t.Fatalf("lost buffered event: %v %v", e, err)
	}
	if _, err := c.Next(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func TestPartialTopicSubscriptionPreservesResult(t *testing.T) {
	c := testClient(t, 1)
	h := c.handler()
	value, err := c.request(context.Background(), "subscribe topic", nil, func() (int, uint64) {
		h.OnSubscribeTopicResult(44, "room", "self", "topic", native.UserList{Users: []string{"accepted"}}, native.UserList{Users: []string{"rejected"}}, -99)
		return 0, 44
	})
	var sdk *Error
	if !errors.As(err, &sdk) || sdk.Code != -99 {
		t.Fatalf("got %v", err)
	}
	result := value.(TopicSubscription)
	if len(result.Succeeded) != 1 || result.Succeeded[0] != "accepted" || result.Failed[0] != "rejected" {
		t.Fatalf("lost partial result: %+v", result)
	}
}
func TestRequestNativeUnavailable(t *testing.T) {
	c := testClient(t, 1)
	_, err := c.request(context.Background(), "missing interface", nil, func() (int, uint64) { return unavailableCode, 0 })
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
}
func TestStreamCloseStateOnlyCancelsItsRequests(t *testing.T) {
	c := testClient(t, 1)
	owner := &Stream{client: c}
	other := &Stream{client: c}
	done := make(chan error, 1)
	started := make(chan struct{})
	go func() {
		_, err := c.request(context.Background(), "owned", owner, func() (int, uint64) { close(started); return 0, 55 })
		done <- err
	}()
	<-started
	// Acquire the same gate used by Stream.Close so the request is registered.
	<-c.gate
	c.closeState(other)
	c.closeState(owner)
	c.unlock()
	if err := <-done; !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v", err)
	}
	_, err := c.request(context.Background(), "client still open", nil, func() (int, uint64) { c.complete(56, result{}); return 0, 56 })
	if err != nil {
		t.Fatal(err)
	}
}
