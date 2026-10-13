package messaging

import (
	"context"
	"unicode/utf8"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

// Stream is owned by its Client. Close releases it, and Client.Close also closes
// any remaining streams. Events are delivered through the parent Client.
type Stream struct {
	client *Client
	native *native.IStreamChannel
	name   string
	closed bool
}
type JoinOptions struct {
	Token                            string
	Presence, Metadata, Locks, Quiet bool
}

// Priority controls topic message scheduling. The zero value is normal.
type Priority uint8

const (
	PriorityNormal Priority = iota
	PriorityHighest
	PriorityHigh
	PriorityLow
)

type TopicOptions struct {
	Ordered       bool
	Metadata      string
	SyncWithMedia bool
	Priority      Priority
}
type TopicMessageOptions struct {
	Text          bool
	CustomType    string
	SendTimestamp uint64
}

func (c *Client) NewStream(ctx context.Context, name string) (*Stream, error) {
	if err := validText("stream name", name, true); err != nil {
		return nil, err
	}
	if err := c.lock(ctx); err != nil {
		return nil, err
	}
	defer c.unlock()
	n, code := c.native.CreateStreamChannelWithError(name)
	if n == nil {
		if code != 0 {
			return nil, sdkError("create stream", code)
		}
		return nil, ErrUnavailable
	}
	s := &Stream{client: c, native: n, name: name}
	c.streams[s] = struct{}{}
	return s, nil
}
func (s *Stream) Name() string { return s.name }
func (s *Stream) request(ctx context.Context, op string, fn func() (int, uint64)) (any, error) {
	return s.client.request(ctx, op, s, fn)
}
func (s *Stream) call(ctx context.Context, op string, fn func() (int, uint64)) error {
	_, err := s.request(ctx, op, fn)
	return err
}
func (s *Stream) Close() error {
	c := s.client
	<-c.gate
	defer c.unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	delete(c.streams, s)
	c.closeState(s)
	return sdkError("close stream", s.native.Release())
}
func (s *Stream) Join(ctx context.Context, o JoinOptions) error {
	if err := validText("stream token", o.Token, false); err != nil {
		return err
	}
	options := &native.JoinChannelOptions{Token: o.Token, WithPresence: o.Presence, WithMetadata: o.Metadata, WithLock: o.Locks, BeQuiet: o.Quiet}
	return s.call(ctx, "join stream", func() (int, uint64) { var id uint64; code := s.native.Join(options, &id); return code, id })
}
func (s *Stream) Leave(ctx context.Context) error {
	return s.call(ctx, "leave stream", func() (int, uint64) { var id uint64; code := s.native.Leave(&id); return code, id })
}
func (s *Stream) RenewToken(ctx context.Context, token string) error {
	if err := validText("stream token", token, false); err != nil {
		return err
	}
	return s.call(ctx, "renew stream token", func() (int, uint64) { return s.native.RenewToken(token) })
}
func (s *Stream) JoinTopic(ctx context.Context, topic string, o TopicOptions) error {
	if err := validText("topic", topic, true); err != nil {
		return err
	}
	if err := validText("topic metadata", o.Metadata, false); err != nil {
		return err
	}
	options := native.NewJoinTopicOptions()
	if o.Ordered {
		options.SetQos(native.RtmMessageQosORDERED)
	}
	options.SetMeta(o.Metadata)
	options.SetSyncWithMedia(o.SyncWithMedia)
	switch o.Priority {
	case PriorityNormal:
		options.SetPriority(native.RtmMessagePriorityNORMAL)
	case PriorityHighest:
		options.SetPriority(native.RtmMessagePriorityHIGHEST)
	case PriorityHigh:
		options.SetPriority(native.RtmMessagePriorityHIGH)
	case PriorityLow:
		options.SetPriority(native.RtmMessagePriorityLOW)
	default:
		return invalid("topic priority")
	}
	return s.call(ctx, "join topic", func() (int, uint64) { var id uint64; code := s.native.JoinTopic(topic, options, &id); return code, id })
}
func (s *Stream) LeaveTopic(ctx context.Context, topic string) error {
	if err := validText("topic", topic, true); err != nil {
		return err
	}
	return s.call(ctx, "leave topic", func() (int, uint64) { var id uint64; code := s.native.LeaveTopic(topic, &id); return code, id })
}
func (s *Stream) Publish(ctx context.Context, topic string, data []byte, o TopicMessageOptions) error {
	if err := validText("topic", topic, true); err != nil {
		return err
	}
	if len(data) == 0 {
		return invalid("empty topic message")
	}
	if err := validText("custom type", o.CustomType, false); err != nil {
		return err
	}
	if o.Text && !utf8.Valid(data) {
		return invalid("text message must be UTF-8")
	}
	options := &native.TopicMessageOptions{MessageType: BinaryMessage, CustomType: o.CustomType, SendTs: o.SendTimestamp}
	if o.Text {
		options.MessageType = TextMessage
	}
	payload := string(data)
	return s.call(ctx, "publish topic", func() (int, uint64) { return s.native.PublishTopicMessage(topic, payload, uint(len(payload)), options) })
}
func topicUsers(topic string, users []string) (*native.TopicOptions, error) {
	if err := validText("topic", topic, true); err != nil {
		return nil, err
	}
	for _, user := range users {
		if err := validText("topic user ID", user, true); err != nil {
			return nil, err
		}
	}
	options := native.NewTopicOptions()
	options.SetUsers(users)
	return options, nil
}

// SubscribeTopic reports successful and failed publishers independently.
func (s *Stream) SubscribeTopic(ctx context.Context, topic string, users ...string) (TopicSubscription, error) {
	options, err := topicUsers(topic, users)
	if err != nil {
		return TopicSubscription{}, err
	}
	v, err := s.request(ctx, "subscribe topic", func() (int, uint64) {
		var id uint64
		code := s.native.SubscribeTopic(topic, options, &id)
		return code, id
	})
	if v != nil {
		return v.(TopicSubscription), err
	}
	return TopicSubscription{}, err
}
func (s *Stream) UnsubscribeTopic(ctx context.Context, topic string, users ...string) error {
	options, err := topicUsers(topic, users)
	if err != nil {
		return err
	}
	return s.call(ctx, "unsubscribe topic", func() (int, uint64) { return s.native.UnsubscribeTopic(topic, options) })
}
func (s *Stream) Subscribers(ctx context.Context, topic string) ([]string, error) {
	if err := validText("topic", topic, true); err != nil {
		return nil, err
	}
	v, err := s.request(ctx, "topic subscribers", func() (int, uint64) { return s.native.GetSubscribedUserList(topic) })
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}
