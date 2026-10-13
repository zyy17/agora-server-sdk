// Run with explicitly supplied credentials. Merely building this example never
// logs in. See README.md for the operation performed by each scenario.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zyy17/agora-server-sdk/messaging"
)

type scenarioOptions struct {
	name, channel, peer string
	listen              time.Duration
}

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	var options scenarioOptions
	flag.StringVar(&options.name, "scenario", "pubsub", "pubsub, direct, presence, metadata, locks, history, or stream")
	flag.StringVar(&options.channel, "channel", "go-sdk-example", "RTM channel")
	flag.StringVar(&options.peer, "peer", "", "user ID for direct messages or stream subscriptions")
	flag.DurationVar(&options.listen, "listen", 0, "listen after the scenario until this duration or Ctrl+C")
	flag.Parse()
	appID, userID := os.Getenv("AGORA_APP_ID"), os.Getenv("AGORA_RTM_USER_ID")
	if appID == "" || userID == "" {
		return errors.New("set AGORA_APP_ID and AGORA_RTM_USER_ID; set AGORA_RTM_TOKEN if your project requires tokens")
	}
	if err := options.validate(); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, err := messaging.New(messaging.Config{AppID: appID, UserID: userID})
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close: %v", err)
		}
		log.Printf("dropped events=%d", client.DroppedEvents())
	}()
	log.Printf("RTM native version: %s", messaging.Version())
	if err := client.Login(ctx, os.Getenv("AGORA_RTM_TOKEN")); err != nil {
		return err
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := client.Logout(shutdown); err != nil {
			log.Printf("logout: %v", err)
		}
	}()
	go logEvents(client.Events())
	if err := options.subscribe(ctx, client); err != nil {
		return err
	}
	if err := options.execute(ctx, client, userID); err != nil {
		return err
	}
	listenForEvents(ctx, options.listen)
	log.Print("finished")
	return nil
}

func (o scenarioOptions) validate() error {
	switch o.name {
	case "pubsub", "presence", "metadata", "locks", "history", "stream":
	case "direct":
		if o.peer == "" {
			return errors.New("direct requires -peer")
		}
	default:
		return fmt.Errorf("unknown scenario %q", o.name)
	}
	if o.listen < 0 {
		return errors.New("-listen cannot be negative")
	}
	return nil
}

func logEvents(events <-chan messaging.Event) {
	for event := range events {
		switch event.Kind {
		case messaging.EventMessage:
			log.Printf("message channel=%s publisher=%s data=%q", event.Message.Channel.Name, event.Message.Publisher, event.Message.Data)
		case messaging.EventLink:
			log.Printf("link state=%d reason=%s", event.Link.CurrentState, event.Link.Reason)
		case messaging.EventTokenExpiring, messaging.EventToken:
			log.Print("token event: obtain a fresh server-generated token and call RenewToken")
		default:
			log.Printf("event=%s", event.Kind)
		}
	}
}

func listenForEvents(ctx context.Context, duration time.Duration) {
	if duration <= 0 {
		return
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
