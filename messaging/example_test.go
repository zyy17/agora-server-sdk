package messaging_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zyy17/agora-server-sdk/messaging"
)

func ExampleNew_invalidConfig() {
	// Local validation fails before native client creation or any network use.
	client, err := messaging.New(messaging.Config{})
	fmt.Println(client == nil)
	fmt.Println(errors.Is(err, messaging.ErrInvalidArgument))
	// Output:
	// true
	// true
}

func ExampleConfig() {
	cfg := messaging.Config{
		AppID:          "project-app-id",
		UserID:         "worker-1",
		EventBuffer:    256,
		RequestTimeout: 5 * time.Second,
	}
	fmt.Println(cfg.UserID, cfg.EventBuffer, cfg.RequestTimeout)
	// Output: worker-1 256 5s
}

func ExampleError() {
	// Errors from the native SDK retain their numeric code and operation.
	err := fmt.Errorf("send update: %w", &messaging.Error{
		Op:      "publish",
		Code:    messaging.ErrorCode(-7),
		Details: "example rejection",
	})
	var rejection *messaging.Error
	if errors.As(err, &rejection) {
		fmt.Println(rejection.Op, rejection.Code, rejection.Details)
	}
	// Output: publish -7 example rejection
}

func ExampleMetadata_unconditional() {
	// Both revision levels must be explicit; zero does not mean unconditional.
	metadata := messaging.Metadata{
		Revision: -1,
		Items: []messaging.MetadataItem{
			{Key: "status", Value: "ready", Revision: -1},
		},
	}
	fmt.Println(metadata.Revision, metadata.Items[0].Revision)
	// Output: -1 -1
}

func ExampleClient_Publish() {
	// Supply an already logged-in client. This example is compiled, not run.
	publish := func(ctx context.Context, client *messaging.Client) error {
		return client.Publish(ctx, "updates", []byte("ready"), messaging.PublishOptions{
			Text:           true,
			CustomType:     "worker-status",
			StoreInHistory: true,
		})
	}
	_ = publish
}

func ExampleClient_Next() {
	// Run one event consumer for an existing client, then fan out if needed.
	consume := func(ctx context.Context, client *messaging.Client) error {
		for {
			event, err := client.Next(ctx)
			if errors.Is(err, messaging.ErrClosed) {
				return nil
			}
			if err != nil {
				return err
			}
			switch event.Kind {
			case messaging.EventMessage:
				fmt.Printf("%s: %s\n", event.Message.Publisher, event.Message.Data)
			case messaging.EventTokenExpiring:
				// Obtain and renew the token for this client or channel.
				fmt.Println("renew token for", event.Channel)
			case messaging.EventToken:
				fmt.Println("token event", event.Token.Type)
			case messaging.EventLink:
				fmt.Println("link state", event.Link.CurrentState)
			}
			// Monitor this cumulative count and reconcile affected state on drops.
			fmt.Println("events dropped:", client.DroppedEvents())
		}
	}
	_ = consume
}

func ExampleClient_OnlineUsers() {
	query := func(ctx context.Context, client *messaging.Client) error {
		channel := messaging.Channel{Name: "updates", Type: messaging.MessageChannel}
		options := messaging.PresenceOptions{IncludeState: true}
		for {
			page, err := client.OnlineUsers(ctx, channel, options)
			if err != nil {
				return err
			}
			for _, user := range page.Users {
				fmt.Println(user.UserID, user.State["status"])
			}
			if page.NextPage == "" {
				return nil
			}
			options.Page = page.NextPage
		}
	}
	// Pages can change during the query; they are not one atomic snapshot.
	_ = query
}

func ExampleClient_History() {
	query := func(ctx context.Context, client *messaging.Client) error {
		channel := messaging.Channel{Name: "updates", Type: messaging.MessageChannel}
		options := messaging.HistoryOptions{Count: 50}
		for {
			page, err := client.History(ctx, channel, options)
			if err != nil {
				return err
			}
			for _, message := range page.Messages {
				fmt.Printf("%s: %s\n", message.Publisher, message.Data)
			}
			if page.NextStart == 0 {
				return nil
			}
			// Preserve native units and reject an overflowing int64 conversion.
			if page.NextStart > uint64(1<<63-1) {
				return fmt.Errorf("history cursor does not fit int64: %d", page.NextStart)
			}
			options.Start = int64(page.NextStart)
		}
	}
	_ = query
}

func ExampleStream_SubscribeTopic() {
	// Supply a stream that is already joined using its channel's RTC token.
	subscribe := func(ctx context.Context, stream *messaging.Stream) error {
		result, err := stream.SubscribeTopic(ctx, "positions", "publisher-1", "publisher-2")
		// An error can accompany partial success; inspect both lists first.
		fmt.Println("subscribed:", result.Succeeded)
		fmt.Println("failed:", result.Failed)
		return err
	}
	_ = subscribe
}

func ExampleClient_Close() {
	shutdown := func(client *messaging.Client) error {
		// Do not reuse a canceled application context for graceful shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		logoutErr := client.Logout(ctx)
		// Always release resources, even if logout times out or fails.
		return errors.Join(logoutErr, client.Close())
	}
	_ = shutdown
}

func ExampleConfig_numericIdentity() {
	cfg := messaging.Config{
		AppID: "project-app-id", UserID: "42", Identity: messaging.IdentityNumeric,
		ResultEvents: true,
		Connection: &messaging.ConnectionConfig{
			Area: messaging.AreaEurope, Protocol: messaging.ProtocolTCPOnly,
			PresenceTimeout: 30 * time.Second, HeartbeatInterval: 5 * time.Second,
			ReconnectTimeout: time.Minute,
		},
	}
	fmt.Println(cfg.UserID, cfg.Identity == messaging.IdentityNumeric, cfg.ResultEvents)
	// Output: 42 true true
}

func ExampleClient_AcquireLockWithOptions() {
	// Supply a dedicated logged-in client CREATED WITH Config{ResultEvents: true}.
	// This function takes ownership and closes it on EVERY exit to stop retry.
	// It has one event consumer and one acquisition for this target, requests
	// ownership only as a probe, and releases the first successful result.
	probe := func(app context.Context, client *messaging.Client) (err error) {
		defer func() { err = errors.Join(err, client.Close()) }()
		channel := messaging.Channel{Name: "locks", Type: messaging.MessageChannel}
		attempt, cancel := context.WithTimeout(app, time.Second)
		err = client.AcquireLockWithOptions(attempt, channel, "mutex", messaging.AcquireLockOptions{Retry: true})
		cancel()
		if err != nil {
			// Neither a context error nor a failed acquisition callback proves
			// the native retry has stopped. Continue until success or shutdown.
			fmt.Println("initial acquisition result:", err)
		}
		// A context error has NOT canceled native retry. Keep handling events
		// using the application context, not the expired acquisition context.
		for {
			event, err := client.Next(app)
			if err != nil {
				return err
			} // Deferred Close stops this dedicated client and its retries.
			if client.DroppedEvents() != 0 {
				// Query Locks with a fresh context and reconcile ownership. A
				// single snapshot cannot prove an in-flight retry will not win later.
				return fmt.Errorf("result events were dropped; reconcile locks before continuing")
			}
			r := event.RequestResult
			if r == nil || r.Operation != messaging.RequestAcquireLock || r.Channel != channel || r.LockName != "mutex" {
				continue
			}
			if r.Error != nil {
				continue // A failed attempt is not proof native retry stopped.
			}
			cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
			defer stop()
			return client.ReleaseLock(cleanup, channel, "mutex")
		}
	}
	_ = probe
}
