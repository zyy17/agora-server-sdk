package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zyy17/agora-server-sdk/messaging"
)

type fakeRTMFeatureBus struct {
	mu            sync.Mutex
	states        map[string]map[string]string
	events        chan messaging.Event
	subscriptions map[string]bool
}

type rtmFeatureCall struct {
	method    string
	ctxErr    error
	remaining time.Duration
	bounded   bool
}

type fakeRTMFeatureClient struct {
	mu                sync.Mutex
	user              string
	bus               *fakeRTMFeatureBus
	calls             []rtmFeatureCall
	fail              map[string]error
	wait              map[string]bool
	skipDelivery      bool
	deliverOnError    bool
	changeMessage     func(*messaging.Message)
	changeState       func(*messaging.UserState)
	publication       messaging.PublishOptions
	publicationBytes  []byte
	publicationTarget string
	loginToken        string
	renewedToken      string
	historyOptions    messaging.HistoryOptions
	queryTargets      []messaging.Channel
	removedKeys       []string
}

func newFakeRTMFeaturePair() (*fakeRTMFeatureClient, *fakeRTMFeatureClient) {
	bus := &fakeRTMFeatureBus{states: make(map[string]map[string]string), events: make(chan messaging.Event, 16), subscriptions: make(map[string]bool)}
	newClient := func(user string) *fakeRTMFeatureClient {
		return &fakeRTMFeatureClient{user: user, bus: bus, fail: make(map[string]error), wait: make(map[string]bool)}
	}
	return newClient("verify-fake-a"), newClient("verify-fake-b")
}

func (f *fakeRTMFeatureClient) call(ctx context.Context, method string) error {
	deadline, bounded := ctx.Deadline()
	f.mu.Lock()
	f.calls = append(f.calls, rtmFeatureCall{method: method, ctxErr: ctx.Err(), remaining: time.Until(deadline), bounded: bounded})
	f.mu.Unlock()
	if f.wait[method] {
		<-ctx.Done()
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return f.fail[method]
}

func (f *fakeRTMFeatureClient) Login(ctx context.Context, value string) error {
	f.loginToken = value
	return f.call(ctx, "login")
}

func (f *fakeRTMFeatureClient) RenewToken(ctx context.Context, value string) error {
	f.renewedToken = value
	return f.call(ctx, "renew_token")
}

func (f *fakeRTMFeatureClient) Logout(ctx context.Context) error {
	return f.call(ctx, "logout")
}

func (f *fakeRTMFeatureClient) Close() error {
	return f.call(context.Background(), "close")
}

func (f *fakeRTMFeatureClient) Subscribe(ctx context.Context, channel string, options messaging.SubscribeOptions) error {
	if channel != "verify-fake-messages" || !options.Presence || options.Metadata || options.Locks {
		return errors.New("unexpected subscription scope")
	}
	err := f.call(ctx, "subscribe")
	f.bus.mu.Lock()
	f.bus.subscriptions[f.user] = true
	f.bus.mu.Unlock()
	return err
}

func (f *fakeRTMFeatureClient) Unsubscribe(ctx context.Context, channel string) error {
	if channel != "verify-fake-messages" {
		return errors.New("unexpected unsubscribe scope")
	}
	return f.call(ctx, "unsubscribe")
}

func (f *fakeRTMFeatureClient) Publish(ctx context.Context, channel string, data []byte, options messaging.PublishOptions) error {
	f.publication, f.publicationBytes, f.publicationTarget = options, append([]byte(nil), data...), channel
	err := f.call(ctx, "publish")
	if !f.skipDelivery && (err == nil || f.deliverOnError) {
		message := &messaging.Message{
			Channel: messaging.Channel{Name: channel, Type: messaging.MessageChannel}, Publisher: f.user,
			Type: messaging.BinaryMessage, Data: append([]byte(nil), data...), CustomType: options.CustomType,
		}
		if f.changeMessage != nil {
			f.changeMessage(message)
		}
		f.bus.events <- messaging.Event{Kind: messaging.EventMessage, Message: message}
	}
	return err
}

func (f *fakeRTMFeatureClient) Next(ctx context.Context) (messaging.Event, error) {
	if err := f.call(ctx, "next"); err != nil {
		return messaging.Event{}, err
	}
	select {
	case event := <-f.bus.events:
		return event, nil
	case <-ctx.Done():
		return messaging.Event{}, ctx.Err()
	}
}

func (f *fakeRTMFeatureClient) SetState(ctx context.Context, channel messaging.Channel, state map[string]string) error {
	if channel != (messaging.Channel{Name: "verify-fake-messages", Type: messaging.MessageChannel}) {
		return errors.New("unexpected presence scope")
	}
	if err := f.call(ctx, "presence_set"); err != nil {
		return err
	}
	f.bus.mu.Lock()
	defer f.bus.mu.Unlock()
	if f.bus.states[f.user] == nil {
		f.bus.states[f.user] = make(map[string]string)
	}
	for key, value := range state {
		f.bus.states[f.user][key] = value
	}
	return nil
}

func (f *fakeRTMFeatureClient) State(ctx context.Context, channel messaging.Channel, user string) (messaging.UserState, error) {
	if channel != (messaging.Channel{Name: "verify-fake-messages", Type: messaging.MessageChannel}) {
		return messaging.UserState{}, errors.New("unexpected presence scope")
	}
	if err := f.call(ctx, "presence_get"); err != nil {
		return messaging.UserState{}, err
	}
	f.bus.mu.Lock()
	defer f.bus.mu.Unlock()
	state := messaging.UserState{UserID: user, State: make(map[string]string)}
	for key, value := range f.bus.states[user] {
		state.State[key] = value
	}
	if f.changeState != nil {
		f.changeState(&state)
	}
	return state, nil
}

func (f *fakeRTMFeatureClient) RemoveState(ctx context.Context, channel messaging.Channel, keys ...string) error {
	if channel != (messaging.Channel{Name: "verify-fake-messages", Type: messaging.MessageChannel}) {
		return errors.New("unexpected presence cleanup scope")
	}
	f.removedKeys = append([]string(nil), keys...)
	if err := f.call(ctx, "presence_remove"); err != nil {
		return err
	}
	f.bus.mu.Lock()
	defer f.bus.mu.Unlock()
	for _, key := range keys {
		delete(f.bus.states[f.user], key)
	}
	return nil
}

func (f *fakeRTMFeatureClient) query(ctx context.Context, method string, channel messaging.Channel) error {
	f.bus.mu.Lock()
	subscriptionsCompleted := f.bus.subscriptions["verify-fake-a"] && f.bus.subscriptions["verify-fake-b"]
	f.bus.mu.Unlock()
	if !subscriptionsCompleted {
		return errors.New("capability query preceded subscriptions")
	}
	f.mu.Lock()
	f.queryTargets = append(f.queryTargets, channel)
	f.mu.Unlock()
	return f.call(ctx, method)
}

func (f *fakeRTMFeatureClient) ChannelMetadata(ctx context.Context, channel messaging.Channel) (messaging.Metadata, error) {
	return messaging.Metadata{}, f.query(ctx, "metadata_read", channel)
}

func (f *fakeRTMFeatureClient) Locks(ctx context.Context, channel messaging.Channel) ([]messaging.Lock, error) {
	return nil, f.query(ctx, "locks_read", channel)
}

func (f *fakeRTMFeatureClient) History(ctx context.Context, channel messaging.Channel, options messaging.HistoryOptions) (messaging.HistoryPage, error) {
	f.historyOptions = options
	return messaging.HistoryPage{}, f.query(ctx, "history_read", channel)
}

func (f *fakeRTMFeatureClient) count(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, call := range f.calls {
		if call.method == method {
			count++
		}
	}
	return count
}

func runFakeRTMFeatures(t *testing.T, ctx context.Context, sender, peer *fakeRTMFeatureClient) map[string]error {
	t.Helper()
	results := make(map[string]error)
	create := func(user, dir string) (rtmFeatureClient, error) {
		switch {
		case user == sender.user && dir == filepath.Join("private", "sender"):
			return sender, nil
		case user == peer.user && dir == filepath.Join("private", "peer"):
			return peer, nil
		default:
			return nil, errors.New("unexpected client identity or directory")
		}
	}
	issued := 0
	checkRTMFeatures(ctx, create, func(user string) (string, error) {
		issued++
		return strings.Repeat("x", issued) + ":" + user, nil
	}, "private", "verify-fake", func(operation string, err error) {
		if _, exists := results[operation]; exists {
			t.Error("duplicate operation recorded")
		}
		results[operation] = err
	})
	return results
}

func TestRTMFeaturesTwoClientRoundtrip(t *testing.T) {
	sender, peer := newFakeRTMFeaturePair()
	// Unrelated events must not count as delivery or prevent finding our event.
	sender.bus.events <- messaging.Event{Kind: messaging.EventToken}
	sender.bus.states[sender.user] = map[string]string{"unrelated": "keep"}
	results := runFakeRTMFeatures(t, context.Background(), sender, peer)
	operations := []string{
		"prepare", "initialize", "peer_initialize", "login", "peer_login", "subscribe", "peer_subscribe",
		"publish", "receive", "renew_token", "presence_set", "presence_get", "presence_remove", "metadata_read",
		"locks_read", "history_read", "unsubscribe", "peer_unsubscribe", "logout", "peer_logout", "close", "peer_close",
	}
	if len(results) != len(operations) {
		t.Fatalf("recorded %d operations; want %d", len(results), len(operations))
	}
	for _, operation := range operations {
		err, exists := results[operation]
		if !exists || err != nil {
			t.Errorf("%s did not pass", operation)
		}
	}
	if sender.loginToken != "x:"+sender.user || peer.loginToken != "xx:"+peer.user || sender.renewedToken != "xxx:"+sender.user {
		t.Error("login or renewal used the wrong identity token")
	}
	if sender.publication.Text || sender.publication.StoreInHistory || sender.publication.CustomType != rtmFeatureCustomType || !bytes.Contains(sender.publicationBytes, []byte{0, 1, 127, 255}) {
		t.Error("publication must test binary bytes without persistent history")
	}
	if sender.publicationTarget != "verify-fake-messages" || sender.historyOptions != (messaging.HistoryOptions{Count: 1}) {
		t.Error("publication or history query escaped its bounded scope")
	}
	if len(sender.queryTargets) != 3 {
		t.Error("missing read-only capability queries")
	}
	for _, target := range sender.queryTargets {
		if target != (messaging.Channel{Name: "verify-fake-messages", Type: messaging.MessageChannel}) {
			t.Error("query escaped the synthetic channel")
		}
	}
	if len(sender.removedKeys) != 1 || sender.removedKeys[0] != rtmFeatureStateKey || len(sender.bus.states[sender.user]) != 1 || sender.bus.states[sender.user]["unrelated"] != "keep" {
		t.Error("cleanup must remove only this run's presence key")
	}
	for _, client := range []*fakeRTMFeatureClient{sender, peer} {
		for _, call := range client.calls {
			if call.method == "close" {
				continue
			}
			maximum := rtmFeatureStepTimeout
			if call.method == "login" {
				maximum = rtmFeatureLoginTimeout
			}
			if !call.bounded || call.remaining <= 0 || call.remaining > maximum || call.ctxErr != nil {
				t.Errorf("%s lacked a live bounded context", call.method)
			}
		}
	}
}

func TestRTMFeaturesPublishSuccessRequiresExactPeerDelivery(t *testing.T) {
	for _, test := range []struct {
		name   string
		absent bool
		change func(*messaging.Message)
	}{
		{name: "no callback", absent: true},
		{name: "wrong bytes", change: func(m *messaging.Message) { m.Data = []byte("different") }},
		{name: "truncated binary", change: func(m *messaging.Message) { m.Data = bytes.Split(m.Data, []byte{0})[0] }},
		{name: "wrong message type", change: func(m *messaging.Message) { m.Type = messaging.TextMessage }},
		{name: "wrong custom type", change: func(m *messaging.Message) { m.CustomType = "different" }},
		{name: "wrong channel", change: func(m *messaging.Message) { m.Channel.Name = "another-channel" }},
		{name: "wrong channel type", change: func(m *messaging.Message) { m.Channel.Type = messaging.UserChannel }},
		{name: "wrong publisher", change: func(m *messaging.Message) { m.Publisher = "another-user" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			sender, peer := newFakeRTMFeaturePair()
			sender.skipDelivery, sender.changeMessage = test.absent, test.change
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			results := runFakeRTMFeatures(t, ctx, sender, peer)
			if err, exists := results["publish"]; !exists || err != nil {
				t.Fatal("fake publish did not succeed")
			}
			if err, exists := results["receive"]; !exists || err == nil {
				t.Fatal("publish acknowledgement incorrectly counted as verified delivery")
			}
			if sender.count("publish") != 1 {
				t.Error("publication was retried")
			}
		})
	}
}

func TestRTMFeaturesTimedOutPublishStillChecksDeliveryWithoutRetry(t *testing.T) {
	sender, peer := newFakeRTMFeaturePair()
	sender.fail["publish"], sender.deliverOnError = context.DeadlineExceeded, true
	results := runFakeRTMFeatures(t, context.Background(), sender, peer)
	if !errors.Is(results["publish"], context.DeadlineExceeded) || results["receive"] != nil || sender.count("publish") != 1 {
		t.Fatal("uncertain publication must be checked for receipt without retrying")
	}
}

func TestRTMFeaturesPresenceRequiresPeerStateMatch(t *testing.T) {
	for _, change := range []func(*messaging.UserState){
		func(state *messaging.UserState) { state.UserID = "another-user" },
		func(state *messaging.UserState) { state.State[rtmFeatureStateKey] = "different" },
	} {
		sender, peer := newFakeRTMFeaturePair()
		peer.changeState = change
		results := runFakeRTMFeatures(t, context.Background(), sender, peer)
		if results["presence_set"] != nil || results["presence_get"] == nil || results["presence_remove"] != nil {
			t.Error("presence acknowledgement incorrectly counted as verified state roundtrip")
		}
	}
}

func TestRTMFeaturesCleanupAfterFailures(t *testing.T) {
	for _, method := range []string{"login", "subscribe", "presence_set", "presence_remove", "logout"} {
		t.Run(method, func(t *testing.T) {
			sender, peer := newFakeRTMFeaturePair()
			sender.fail[method] = context.DeadlineExceeded
			results := runFakeRTMFeatures(t, context.Background(), sender, peer)
			for _, client := range []*fakeRTMFeatureClient{sender, peer} {
				if client.count("logout") != 1 || client.count("close") != 1 {
					t.Error("every initialized client must logout and close after failure")
				}
			}
			if method == "subscribe" && sender.count("unsubscribe") != 1 {
				t.Error("timed-out subscription was not cleaned up")
			}
			if method == "presence_set" && sender.count("presence_remove") != 1 {
				t.Error("timed-out presence mutation was not cleaned up")
			}
			if method != "login" {
				for _, operation := range []string{"metadata_read", "locks_read", "history_read", "renew_token"} {
					if err, exists := results[operation]; !exists || err != nil {
						t.Errorf("independent %s was prevented by another operation's failure", operation)
					}
				}
			}
		})
	}
}

func TestRTMFeaturesCanceledWorkUsesFreshCleanupContexts(t *testing.T) {
	sender, peer := newFakeRTMFeaturePair()
	sender.wait["presence_set"] = true
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	results := runFakeRTMFeatures(t, ctx, sender, peer)
	if !errors.Is(results["presence_set"], context.DeadlineExceeded) || ctx.Err() == nil {
		t.Fatal("work context did not expire as expected")
	}
	for _, client := range []*fakeRTMFeatureClient{sender, peer} {
		for _, call := range client.calls {
			if call.method != "logout" && call.method != "unsubscribe" && call.method != "presence_remove" {
				continue
			}
			if call.ctxErr != nil || !call.bounded || call.remaining <= 0 || call.remaining > rtmFeatureCleanupTimeout {
				t.Errorf("%s reused canceled work context", call.method)
			}
		}
		if client.count("close") != 1 || client.count("logout") != 1 {
			t.Error("canceled work prevented cleanup")
		}
	}
}

func TestRTMFeaturesPartialInitializationStillCleansClient(t *testing.T) {
	sender, _ := newFakeRTMFeaturePair()
	results := make(map[string]error)
	checkRTMFeatures(context.Background(), func(user, dir string) (rtmFeatureClient, error) {
		if user == sender.user {
			return sender, nil
		}
		return nil, messaging.ErrUnavailable
	}, func(string) (string, error) { return "synthetic", nil }, "private", "verify-fake", func(operation string, err error) {
		results[operation] = err
	})
	if !errors.Is(results["peer_initialize"], messaging.ErrUnavailable) || sender.count("logout") != 1 || sender.count("close") != 1 {
		t.Error("partial initialization did not release the initialized client")
	}
	if sender.count("login") != 0 || sender.count("publish") != 0 {
		t.Error("feature work started without both clients")
	}
}

func TestRTMFeaturesTokenFailureNeverInitializesClient(t *testing.T) {
	created := 0
	var operations []string
	checkRTMFeatures(context.Background(), func(user, dir string) (rtmFeatureClient, error) {
		created++
		return nil, messaging.ErrUnavailable
	}, func(string) (string, error) { return "", errors.New("synthetic token failure") }, "private", "verify-fake", func(operation string, err error) {
		operations = append(operations, operation)
		if err == nil {
			t.Error("token failure was not recorded")
		}
	})
	if created != 0 || len(operations) != 1 || operations[0] != "prepare" {
		t.Error("failed token preparation must stop before client initialization")
	}
}
