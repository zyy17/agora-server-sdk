package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zyy17/agora-server-sdk/messaging"
)

const (
	rtmFeatureLoginTimeout   = 10 * time.Second
	rtmFeatureStepTimeout    = 3 * time.Second
	rtmFeatureCleanupTimeout = 2 * time.Second
	rtmFeatureCustomType     = "agora-live-verification-v1"
	rtmFeatureStateKey       = "agora_verify"
)

// Durable mutations are deliberately absent. Reading an empty private channel
// can establish API access, but cannot certify metadata/lock writes or history
// retention. There is no history deletion API, so publications never request it.
type rtmFeatureClient interface {
	rtmClient
	RenewToken(context.Context, string) error
	Subscribe(context.Context, string, messaging.SubscribeOptions) error
	Unsubscribe(context.Context, string) error
	Publish(context.Context, string, []byte, messaging.PublishOptions) error
	Next(context.Context) (messaging.Event, error)
	SetState(context.Context, messaging.Channel, map[string]string) error
	State(context.Context, messaging.Channel, string) (messaging.UserState, error)
	RemoveState(context.Context, messaging.Channel, ...string) error
	ChannelMetadata(context.Context, messaging.Channel) (messaging.Metadata, error)
	Locks(context.Context, messaging.Channel) ([]messaging.Lock, error)
	History(context.Context, messaging.Channel, messaging.HistoryOptions) (messaging.HistoryPage, error)
}

var _ rtmFeatureClient = (*messaging.Client)(nil)

type rtmFeatureFactory func(user, dir string) (rtmFeatureClient, error)

func runRTMFeatures(ctx context.Context, c credentials, dir, identity string, note record) {
	create := func(user, clientDir string) (rtmFeatureClient, error) {
		// Each native client has a private log directory, even with logging off.
		if err := os.Mkdir(clientDir, 0o700); err != nil {
			return nil, err
		}
		client, err := newRTM(c, user, clientDir)
		if err != nil {
			return nil, err
		}
		features, ok := client.(rtmFeatureClient)
		if !ok {
			_ = client.Close()
			return nil, errors.New("RTM feature interface unavailable")
		}
		return features, nil
	}
	checkRTMFeatures(ctx, create, func(user string) (string, error) {
		return rtmToken(c, user)
	}, dir, identity, note)
}

// checkRTMFeatures uses only synthetic data and two identities derived from the
// run's cryptographically random identity. The supplied collector is called on
// this goroutine only, including for concurrently executing independent probes.
func checkRTMFeatures(ctx context.Context, create rtmFeatureFactory, tokenFor func(string) (string, error), dir, identity string, note record) {
	senderID, peerID := identity+"-a", identity+"-b"
	channel := messaging.Channel{Name: identity + "-messages", Type: messaging.MessageChannel}
	payload := append([]byte("agora-sdk-verifier:"+identity+":"), 0, 1, 127, 255)
	senderToken, err := tokenFor(senderID)
	if err != nil {
		note("prepare", err)
		return
	}
	peerToken, err := tokenFor(peerID)
	note("prepare", err)
	if err != nil {
		return
	}

	sender, senderErr := create(senderID, filepath.Join(dir, "sender"))
	note("initialize", senderErr)
	peer, peerErr := create(peerID, filepath.Join(dir, "peer"))
	note("peer_initialize", peerErr)
	var senderSubscribeAttempted, peerSubscribeAttempted, stateAttempted bool
	defer func() {
		// A timed-out login, subscription or state change may still have been
		// accepted. Cleanup is attempted once, using fresh bounded contexts.
		// Close has no context API; the outer subprocess deadline bounds native
		// hangs. The two clients clean up independently so one cannot stall both.
		rtmFeatureGroup(note, func(emit record) {
			if senderErr != nil {
				return
			}
			if stateAttempted {
				emit("presence_remove", rtmFeatureCleanup(func(cleanup context.Context) error {
					return sender.RemoveState(cleanup, channel, rtmFeatureStateKey)
				}))
			}
			if senderSubscribeAttempted {
				emit("unsubscribe", rtmFeatureCleanup(func(cleanup context.Context) error {
					return sender.Unsubscribe(cleanup, channel.Name)
				}))
			}
			emit("logout", rtmFeatureCleanup(sender.Logout))
			emit("close", sender.Close())
		}, func(emit record) {
			if peerErr != nil {
				return
			}
			if peerSubscribeAttempted {
				emit("peer_unsubscribe", rtmFeatureCleanup(func(cleanup context.Context) error {
					return peer.Unsubscribe(cleanup, channel.Name)
				}))
			}
			emit("peer_logout", rtmFeatureCleanup(peer.Logout))
			emit("peer_close", peer.Close())
		})
	}()
	if senderErr != nil || peerErr != nil {
		return
	}

	var loginErr, peerLoginErr error
	rtmFeatureGroup(note, func(emit record) {
		loginErr = rtmFeatureLogin(ctx, sender, senderToken)
		emit("login", loginErr)
	}, func(emit record) {
		peerLoginErr = rtmFeatureLogin(ctx, peer, peerToken)
		emit("peer_login", peerLoginErr)
	})
	if loginErr != nil || peerLoginErr != nil {
		return
	}

	var subscribeErr, peerSubscribeErr error
	senderSubscribeAttempted, peerSubscribeAttempted = true, true
	rtmFeatureGroup(note, func(emit record) {
		subscribeErr = rtmFeatureStep(ctx, func(step context.Context) error {
			return sender.Subscribe(step, channel.Name, messaging.SubscribeOptions{Presence: true})
		})
		emit("subscribe", subscribeErr)
	}, func(emit record) {
		peerSubscribeErr = rtmFeatureStep(ctx, func(step context.Context) error {
			return peer.Subscribe(step, channel.Name, messaging.SubscribeOptions{Presence: true})
		})
		emit("peer_subscribe", peerSubscribeErr)
	}, func(emit record) {
		renewed, err := tokenFor(senderID)
		if err == nil {
			err = rtmFeatureStep(ctx, func(step context.Context) error { return sender.RenewToken(step, renewed) })
		}
		emit("renew_token", err)
	})

	stateAttempted = subscribeErr == nil
	// Capability reads follow subscription completion so the random channel
	// exists first. They still run independently of message and presence checks.
	rtmFeatureGroup(note, func(emit record) {
		if peerSubscribeErr != nil {
			return
		}
		// A sender need not subscribe to publish. Never retry this write after
		// a timeout: instead still inspect the peer's delivery queue once.
		emit("publish", rtmFeatureStep(ctx, func(step context.Context) error {
			return sender.Publish(step, channel.Name, payload, messaging.PublishOptions{CustomType: rtmFeatureCustomType, StoreInHistory: false})
		}))
		emit("receive", rtmFeatureStep(ctx, func(step context.Context) error {
			return receiveRTMFeatureMessage(step, peer, channel, senderID, payload)
		}))
	}, func(emit record) {
		if !stateAttempted {
			return
		}
		err := rtmFeatureStep(ctx, func(step context.Context) error {
			return sender.SetState(step, channel, map[string]string{rtmFeatureStateKey: identity})
		})
		emit("presence_set", err)
		if err != nil {
			return
		}
		emit("presence_get", rtmFeatureStep(ctx, func(step context.Context) error {
			state, err := peer.State(step, channel, senderID)
			if err != nil {
				return err
			}
			if state.UserID != senderID || state.State[rtmFeatureStateKey] != identity {
				return errors.New("presence roundtrip mismatch")
			}
			return nil
		}))
	}, func(emit record) {
		emit("metadata_read", rtmFeatureStep(ctx, func(step context.Context) error {
			_, err := sender.ChannelMetadata(step, channel)
			return err
		}))
	}, func(emit record) {
		emit("locks_read", rtmFeatureStep(ctx, func(step context.Context) error {
			_, err := sender.Locks(step, channel)
			return err
		}))
	}, func(emit record) {
		emit("history_read", rtmFeatureStep(ctx, func(step context.Context) error {
			_, err := sender.History(step, channel, messaging.HistoryOptions{Count: 1})
			return err
		}))
	})
}

func receiveRTMFeatureMessage(ctx context.Context, peer rtmFeatureClient, channel messaging.Channel, publisher string, payload []byte) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		event, err := peer.Next(ctx)
		if err != nil {
			return err
		}
		if event.Kind != messaging.EventMessage || event.Message == nil {
			continue
		}
		message := event.Message
		if message.Channel != channel || message.Publisher != publisher {
			continue
		}
		if message.Type != messaging.BinaryMessage || message.CustomType != rtmFeatureCustomType || !bytes.Equal(message.Data, payload) {
			return errors.New("message delivery mismatch")
		}
		return nil
	}
}

func rtmFeatureStep(ctx context.Context, call func(context.Context) error) error {
	step, cancel := context.WithTimeout(ctx, rtmFeatureStepTimeout)
	defer cancel()
	return call(step)
}

func rtmFeatureLogin(ctx context.Context, client rtmFeatureClient, value string) error {
	// Initial service discovery and transport setup need more time than the
	// subsequent feature requests; both clients still connect concurrently.
	step, cancel := context.WithTimeout(ctx, rtmFeatureLoginTimeout)
	defer cancel()
	return client.Login(step, value)
}

func rtmFeatureCleanup(call func(context.Context) error) error {
	cleanup, cancel := context.WithTimeout(context.Background(), rtmFeatureCleanupTimeout)
	defer cancel()
	return call(cleanup)
}

func rtmFeatureGroup(note record, tasks ...func(record)) {
	type outcome struct {
		operation string
		err       error
	}
	results := make(chan outcome)
	var pending sync.WaitGroup
	for _, task := range tasks {
		pending.Add(1)
		go func(task func(record)) {
			defer pending.Done()
			task(func(operation string, err error) { results <- outcome{operation, err} })
		}(task)
	}
	go func() {
		pending.Wait()
		close(results)
	}()
	for result := range results {
		note(result.operation, result.err)
	}
}
