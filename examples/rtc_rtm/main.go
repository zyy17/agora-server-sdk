// rtc_rtm keeps RTC numeric UIDs and RTM string identities independent.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"sync"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
	"github.com/zyy17/agora-server-sdk/messaging"
)

func main() { example.Main(run) }

func run() (err error) {
	rtmChannel := flag.String("rtm-channel", "go-sdk-notifications", "independent RTM message channel")
	listen := flag.Duration("listen", 2*time.Second, "listen after sending, between zero and 1 minute")
	flag.Parse()
	if *listen < 0 || *listen > time.Minute {
		return errors.New("-listen must be between zero and 1 minute")
	}
	cfg, channel, err := example.Config()
	if err != nil {
		return err
	}
	rtmUser := os.Getenv("AGORA_RTM_USER_ID")
	if rtmUser == "" {
		return errors.New("set a separate AGORA_RTM_USER_ID and, when required, AGORA_RTM_TOKEN")
	}
	cfg.IdentityMode = sdk.NumericIdentity
	channel.Receive.Disabled = true // This demo only receives RTC messages/events.
	ctx, stop := example.Context()
	defer stop()
	service, err := sdk.NewService(cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, service.Close()) }()
	session, err := service.Join(ctx, channel)
	if err != nil {
		return err
	}
	client, err := messaging.New(messaging.Config{AppID: cfg.AppID, UserID: rtmUser})
	if err != nil {
		return errors.Join(err, session.Close())
	}
	wait := observe(session, client)
	defer func() {
		// Close producers first. The observers then drain their finite queues
		// to EOF rather than waiting for a timer or a guessed event count.
		err = errors.Join(err, session.Close(), client.Close())
		wait()
		log.Printf("dropped: RTC events=%d messages=%d, RTM events=%d", session.Events().Stats().Dropped, session.Messages().Stats().Dropped, client.DroppedEvents())
	}()
	if err := client.Login(ctx, os.Getenv("AGORA_RTM_TOKEN")); err != nil {
		return err
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err = errors.Join(err, client.Logout(shutdown))
	}()
	if err := client.Subscribe(ctx, *rtmChannel, messaging.SubscribeOptions{}); err != nil {
		return err
	}
	log.Printf("RTC UID=%s channel=%s; RTM user=%s channel=%s", session.Info().UserID, channel.Channel, rtmUser, *rtmChannel)
	if err := sendNotifications(ctx, session, client, *rtmChannel); err != nil {
		return err
	}
	return waitForNotifications(ctx, *listen)
}

func sendNotifications(ctx context.Context, session *sdk.Session, client *messaging.Client, channel string) error {
	// Arbitrary bytes, including NUL and non-UTF-8: no application protocol.
	payload := []byte{0x00, 0x01, 0x02, 0xff}
	if err := session.SendMessage(payload); err != nil {
		return err
	}
	// Publish PCM before adding audio metadata, then send another second of
	// silence. Audio metadata is associated with audio, not an RTM message.
	if err := session.SendPCM(make([]byte, 320)); err != nil {
		return err
	}
	if err := session.SendAudioMetadata(payload); err != nil {
		return err
	}
	if err := client.Publish(ctx, channel, payload, messaging.PublishOptions{Text: false}); err != nil {
		return err
	}
	if err := session.PlayPCM(ctx, bytes.NewReader(make([]byte, 16000*2))); err != nil {
		return err
	}
	finish, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := session.WaitPCM(finish); err != nil {
		return err
	}
	return nil
}

func waitForNotifications(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func observe(session *sdk.Session, client *messaging.Client) func() {
	var workers sync.WaitGroup
	start := func(fn func()) {
		workers.Add(1)
		go func() { defer workers.Done(); fn() }()
	}
	start(func() { drain(session.Events().Next, logRTCEvent) })
	start(func() {
		drain(session.Messages().Next, func(message sdk.Message) {
			log.Printf("RTC data user=%s data=%x", message.UserID, message.Data)
		})
	})
	start(func() {
		for event := range client.Events() {
			logRTMEvent(event)
		}
	})
	return workers.Wait
}

func logRTCEvent(event sdk.Event) {
	switch event.Kind {
	case sdk.AudioMetadata:
		log.Printf("RTC audio metadata user=%s data=%x", event.UserID, event.Data)
	case sdk.TokenExpiring, sdk.TokenExpired:
		log.Print("renew RTC with a numeric-UID RTC token through Session.RenewToken")
	case sdk.SDKError, sdk.ConnectionFailed:
		log.Printf("RTC event=%s code=%d message=%s", event.Kind, event.Code, event.Message)
	}
}

func logRTMEvent(event messaging.Event) {
	switch event.Kind {
	case messaging.EventMessage:
		if event.Message != nil {
			log.Printf("RTM message user=%s type=%d data=%x", event.Message.Publisher, event.Message.Type, event.Message.Data)
		}
	case messaging.EventToken, messaging.EventTokenExpiring:
		log.Print("renew RTM with an RTM user token through Client.RenewToken")
	}
}

// Close owns termination for these read-only consumers. Callers must close all
// producers before waiting; using a cancelled context would skip buffered data.
func drain[T any](next func(context.Context) (T, error), consume func(T)) {
	for {
		value, err := next(context.Background())
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("receive: %v", err)
			}
			return
		}
		consume(value)
	}
}
