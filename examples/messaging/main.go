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

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
func run() error {
	scenario := flag.String("scenario", "pubsub", "pubsub, direct, presence, metadata, locks, history, or stream")
	channel := flag.String("channel", "go-sdk-example", "RTM channel")
	peer := flag.String("peer", "", "user ID for direct messages or stream subscriptions")
	listen := flag.Duration("listen", 0, "listen after the scenario until this duration or Ctrl+C")
	flag.Parse()
	appID, userID := os.Getenv("AGORA_APP_ID"), os.Getenv("AGORA_RTM_USER_ID")
	if appID == "" || userID == "" {
		return errors.New("set AGORA_APP_ID and AGORA_RTM_USER_ID; set AGORA_RTM_TOKEN if your project requires tokens")
	}
	switch *scenario {
	case "pubsub", "presence", "metadata", "locks", "history", "stream":
	case "direct":
		if *peer == "" {
			return errors.New("direct requires -peer")
		}
	default:
		return fmt.Errorf("unknown scenario %q", *scenario)
	}
	if *listen < 0 {
		return errors.New("-listen cannot be negative")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c, err := messaging.New(messaging.Config{AppID: appID, UserID: userID})
	if err != nil {
		return err
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Printf("close: %v", err)
		}
		log.Printf("dropped events=%d", c.DroppedEvents())
	}()
	log.Printf("RTM native version: %s", messaging.Version())
	if err := c.Login(ctx, os.Getenv("AGORA_RTM_TOKEN")); err != nil {
		return err
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := c.Logout(shutdown); err != nil {
			log.Printf("logout: %v", err)
		}
	}()
	go func() {
		for e := range c.Events() {
			switch e.Kind {
			case messaging.EventMessage:
				log.Printf("message channel=%s publisher=%s data=%q", e.Message.Channel.Name, e.Message.Publisher, e.Message.Data)
			case messaging.EventLink:
				log.Printf("link state=%d reason=%s", e.Link.CurrentState, e.Link.Reason)
			case messaging.EventTokenExpiring, messaging.EventToken:
				log.Printf("token event: obtain a fresh server-generated token and call RenewToken")
			default:
				log.Printf("event=%s", e.Kind)
			}
		}
	}()
	ch := messaging.Channel{Name: *channel, Type: messaging.MessageChannel}
	if *scenario != "direct" && *scenario != "stream" {
		options := messaging.SubscribeOptions{}
		switch *scenario {
		case "presence":
			options.Presence = true
		case "metadata":
			options.Metadata = true
		case "locks":
			options.Locks = true
		}
		if err := c.Subscribe(ctx, *channel, options); err != nil {
			return err
		}
	}
	switch *scenario {
	case "pubsub":
		err = c.Publish(ctx, *channel, []byte("hello from Go"), messaging.PublishOptions{Text: true})
	case "direct":
		err = c.PublishToUser(ctx, *peer, []byte("hello from Go"), messaging.PublishOptions{Text: true})
	case "presence":
		err = presence(ctx, c, ch, userID)
	case "metadata":
		err = metadata(ctx, c, ch, userID)
	case "locks":
		err = locks(ctx, c, ch)
	case "history":
		var page messaging.HistoryPage
		page, err = c.History(ctx, ch, messaging.HistoryOptions{Count: 20})
		if err == nil {
			log.Printf("history messages=%d next=%d", len(page.Messages), page.NextStart)
		}
	case "stream":
		target := *peer
		if target == "" {
			target = userID
		}
		err = stream(ctx, c, *channel, target)
	}
	if err != nil {
		return err
	}
	if *listen > 0 {
		timer := time.NewTimer(*listen)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}
	log.Print("finished")
	return nil
}
func presence(ctx context.Context, c *messaging.Client, ch messaging.Channel, userID string) error {
	if err := c.SetState(ctx, ch, map[string]string{"go_example_status": "online", "go_example_language": "go"}); err != nil {
		return err
	}
	page, err := c.OnlineUsers(ctx, ch, messaging.PresenceOptions{IncludeState: true})
	if err != nil {
		return err
	}
	log.Printf("online users=%d next page=%q", len(page.Users), page.NextPage)
	state, err := c.State(ctx, ch, userID)
	if err != nil {
		return err
	}
	log.Printf("state=%v", state.State)
	channels, err := c.UserChannels(ctx, userID)
	if err != nil {
		return err
	}
	log.Printf("channels=%v", channels)
	return c.RemoveState(ctx, ch, "go_example_status", "go_example_language")
}
func metadata(ctx context.Context, c *messaging.Client, ch messaging.Channel, userID string) error {
	data := messaging.Metadata{Revision: -1, Items: []messaging.MetadataItem{{Key: "go_example", Value: "hello", Revision: -1}}}
	options := messaging.MetadataOptions{RecordTimestamp: true, RecordUserID: true}
	if err := c.SetChannelMetadata(ctx, ch, data, options); err != nil {
		return err
	}
	data.Items[0].Value = "updated"
	if err := c.UpdateChannelMetadata(ctx, ch, data, options); err != nil {
		return err
	}
	got, err := c.ChannelMetadata(ctx, ch)
	if err != nil {
		return err
	}
	log.Printf("channel metadata revision=%d items=%d", got.Revision, len(got.Items))
	if err := c.RemoveChannelMetadata(ctx, ch, data, options); err != nil {
		return err
	}
	if err := c.SubscribeUserMetadata(ctx, userID); err != nil {
		return err
	}
	if err := c.SetUserMetadata(ctx, userID, data, options); err != nil {
		return err
	}
	data.Items[0].Value = "user update"
	if err := c.UpdateUserMetadata(ctx, userID, data, options); err != nil {
		return err
	}
	got, err = c.UserMetadata(ctx, userID)
	if err != nil {
		return err
	}
	log.Printf("user metadata revision=%d items=%d", got.Revision, len(got.Items))
	if err := c.RemoveUserMetadata(ctx, userID, data, options); err != nil {
		return err
	}
	return c.UnsubscribeUserMetadata(ctx, userID)
}
func locks(ctx context.Context, c *messaging.Client, ch messaging.Channel) error {
	name := "go_example_lock"
	if err := c.SetLock(ctx, ch, name, 30*time.Second); err != nil {
		return err
	}
	if err := c.AcquireLock(ctx, ch, name); err != nil {
		return err
	}
	got, err := c.Locks(ctx, ch)
	if err != nil {
		return err
	}
	log.Printf("locks=%v", got)
	if err := c.ReleaseLock(ctx, ch, name); err != nil {
		return err
	}
	return c.RemoveLock(ctx, ch, name)
}
func stream(ctx context.Context, c *messaging.Client, channel, peer string) error {
	s, err := c.NewStream(ctx, channel)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := s.Join(ctx, messaging.JoinOptions{Token: os.Getenv("AGORA_RTM_STREAM_TOKEN"), Presence: true}); err != nil {
		return err
	}
	const topic = "go_example_topic"
	if err := s.JoinTopic(ctx, topic, messaging.TopicOptions{Ordered: true}); err != nil {
		return err
	}
	result, err := s.SubscribeTopic(ctx, topic, peer)
	if err != nil {
		return err
	}
	log.Printf("topic subscriptions succeeded=%v failed=%v", result.Succeeded, result.Failed)
	if err := s.Publish(ctx, topic, []byte("hello topic"), messaging.TopicMessageOptions{Text: true}); err != nil {
		return err
	}
	users, err := s.Subscribers(ctx, topic)
	if err != nil {
		return err
	}
	log.Printf("topic publishers=%v", users)
	if err := s.UnsubscribeTopic(ctx, topic, peer); err != nil {
		return err
	}
	if err := s.LeaveTopic(ctx, topic); err != nil {
		return err
	}
	return s.Leave(ctx)
}
