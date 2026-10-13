package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zyy17/agora-server-sdk/messaging"
)

func (o scenarioOptions) subscribe(ctx context.Context, client *messaging.Client) error {
	if o.name == "direct" || o.name == "stream" {
		return nil
	}
	options := messaging.SubscribeOptions{
		Presence: o.name == "presence",
		Metadata: o.name == "metadata",
		Locks:    o.name == "locks",
	}
	return client.Subscribe(ctx, o.channel, options)
}

func (o scenarioOptions) execute(ctx context.Context, client *messaging.Client, userID string) error {
	channel := messaging.Channel{Name: o.channel, Type: messaging.MessageChannel}
	switch o.name {
	case "pubsub":
		return client.Publish(ctx, o.channel, []byte("hello from Go"), messaging.PublishOptions{Text: true})
	case "direct":
		return client.PublishToUser(ctx, o.peer, []byte("hello from Go"), messaging.PublishOptions{Text: true})
	case "presence":
		return presence(ctx, client, channel, userID)
	case "metadata":
		return metadata(ctx, client, channel, userID)
	case "locks":
		return locks(ctx, client, channel)
	case "history":
		page, err := client.History(ctx, channel, messaging.HistoryOptions{Count: 20})
		if err == nil {
			log.Printf("history messages=%d next=%d", len(page.Messages), page.NextStart)
		}
		return err
	case "stream":
		peer := o.peer
		if peer == "" {
			peer = userID
		}
		return stream(ctx, client, o.channel, peer)
	default:
		return fmt.Errorf("unknown scenario %q", o.name)
	}
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
