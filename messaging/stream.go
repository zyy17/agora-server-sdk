package messaging

import (
	"context"
	"unicode/utf8"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

// Stream owns a native stream channel on behalf of its Client.
// Create one with Client.NewStream; its zero value is not usable. Methods are
// safe for concurrent callers, but a Stream must not be copied after creation.
// All events are delivered through the parent's event queue. Close releases the
// stream, and Client.Close releases any streams still open.
type Stream struct {
	client *Client
	native *native.IStreamChannel
	name   string
	closed bool
}

// JoinOptions controls joining a stream channel.
// The zero value enables none of the optional event families.
type JoinOptions struct {
	// Token is an RTC token for this stream channel and identity, distinct
	// from the RTM token passed to Client.Login. Empty tokens pass local
	// validation but may be rejected by project authentication settings.
	Token string
	// Presence requests channel presence events.
	Presence bool
	// Metadata requests channel metadata events.
	Metadata bool
	// Locks requests channel lock events.
	Locks bool
	// Quiet requests joining without notifying other users of this client's
	// join, leave, or presence-state changes.
	Quiet bool
}

// Priority selects native scheduling priority for a topic publisher.
// The zero value is PriorityNormal. Other unspecified values are rejected by
// Stream.JoinTopic; the Go values are mapped to native priority constants.
type Priority uint8

const (
	// PriorityNormal selects the default topic-message scheduling priority.
	PriorityNormal Priority = iota
	// PriorityHighest selects the highest topic-message scheduling priority.
	PriorityHighest
	// PriorityHigh selects a higher-than-normal scheduling priority.
	PriorityHigh
	// PriorityLow selects a lower-than-normal scheduling priority.
	PriorityLow
)

// TopicOptions configures joining a topic as a publisher.
// The zero value requests unordered messages at normal priority without media
// synchronization or publisher metadata.
type TopicOptions struct {
	// Ordered requests the native ordered-message QoS when true.
	Ordered bool
	// Metadata is optional publisher metadata advertised with the topic.
	// It must be UTF-8 without NUL.
	Metadata string
	// SyncWithMedia enables the SDK's media-synchronization behavior.
	// TopicMessageOptions.SendTimestamp supplies the associated native timestamp.
	SyncWithMedia bool
	// Priority selects topic-message scheduling priority.
	// Only the Priority constants declared by this package are accepted.
	Priority Priority
}

// TopicMessageOptions controls a stream-topic publication.
// The zero value sends binary data without a custom type or media timestamp.
type TopicMessageOptions struct {
	// Text requires valid UTF-8 and selects TextMessage. False selects
	// BinaryMessage. Embedded NUL bytes are preserved in either case.
	Text bool
	// CustomType is optional application-defined message metadata.
	// It must be UTF-8 without NUL; native size limits still apply.
	CustomType string
	// SendTimestamp is passed unchanged to the native media-synchronization
	// API and applies when TopicOptions.SyncWithMedia is enabled. It is not
	// converted from wall-clock time or a history cursor.
	SendTimestamp uint64
}

// NewStream creates an owned stream-channel handle without joining it.
// The name must be nonempty UTF-8 without NUL. Call Stream.Join before topic
// operations and Stream.Close when finished; Client.Close also closes it.
//
// The context bounds waiting for serialized access but cannot interrupt native
// creation after it begins. Config.RequestTimeout does not apply. Native
// creation failures return *Error or ErrUnavailable.
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

// Name returns the stream-channel name supplied to Client.NewStream.
// The name remains available after the stream has been closed.
func (s *Stream) Name() string { return s.name }

func (s *Stream) request(ctx context.Context, op string, fn func() (int, uint64)) (any, error) {
	return s.client.request(ctx, op, s, fn)
}

func (s *Stream) call(ctx context.Context, op string, fn func() (int, uint64)) error {
	_, err := s.request(ctx, op, fn)
	return err
}

// Close releases the stream and unblocks its pending requests with ErrClosed.
// It does not close the parent client or remove events already queued there.
// Subsequent calls, including after Client.Close, return nil without releasing
// again; a native release failure is returned only by the call performing it.
//
// Close does not wait for Leave. Call Leave first when graceful departure is
// required, then close even if it fails. A blocking native operation or a parent
// WithNative callback can delay Close.
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

// Join joins the stream channel and waits for the SDK result.
// Call Client.Login first and supply an RTC token for the channel and identity
// through o.Token. The token must be UTF-8 without NUL; empty is locally allowed.
// This token is separate from the RTM login token. Cancellation stops waiting
// without undoing a submitted join.
func (s *Stream) Join(ctx context.Context, o JoinOptions) error {
	if err := validText("stream token", o.Token, false); err != nil {
		return err
	}
	options := &native.JoinChannelOptions{Token: o.Token, WithPresence: o.Presence, WithMetadata: o.Metadata, WithLock: o.Locks, BeQuiet: o.Quiet}
	return s.call(ctx, "join stream", func() (int, uint64) { var id uint64; code := s.native.Join(options, &id); return code, id })
}

// Leave requests departure from the stream channel and waits for the SDK
// result. It does not release the native handle; call Close when finished.
// Cancellation does not undo an already submitted leave.
func (s *Stream) Leave(ctx context.Context) error {
	return s.call(ctx, "leave stream", func() (int, uint64) { var id uint64; code := s.native.Leave(&id); return code, id })
}

// RenewToken replaces this stream's RTC token and waits for the SDK result.
// Use a token for the stream channel and identity, not the RTM login token.
// Tokens must be UTF-8 without NUL; an empty token passes local validation.
// There is no automatic renewal.
func (s *Stream) RenewToken(ctx context.Context, token string) error {
	if err := validText("stream token", token, false); err != nil {
		return err
	}
	return s.call(ctx, "renew stream token", func() (int, uint64) { return s.native.RenewToken(token) })
}

// JoinTopic joins topic as a publisher and waits for the SDK result.
// The stream must first be joined. The topic must be nonempty UTF-8 without NUL;
// publisher metadata must be UTF-8 without NUL, and Priority must be a declared
// constant. Receiving other publishers' messages separately requires
// SubscribeTopic; joining does not create those subscriptions.
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

// LeaveTopic leaves topic as a publisher and waits for the SDK result.
// The topic must be nonempty UTF-8 without NUL. It does not release the stream
// handle; use UnsubscribeTopic to remove subscriptions to other publishers.
func (s *Stream) LeaveTopic(ctx context.Context, topic string) error {
	if err := validText("topic", topic, true); err != nil {
		return err
	}
	return s.call(ctx, "leave topic", func() (int, uint64) { var id uint64; code := s.native.LeaveTopic(topic, &id); return code, id })
}

// Publish sends a message to a topic joined by this stream and waits for the
// SDK result. The topic must be nonempty UTF-8 without NUL, and data must be
// nonempty. Text mode requires valid UTF-8 but allows embedded NUL bytes.
//
// The payload is copied before submission; do not modify it concurrently with
// the call. Native size and rate limits still apply. A nil error does not prove
// recipient application processing. Cancellation or timeout does not undo a
// submitted publication.
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

// SubscribeTopic subscribes to the specified publishers in topic and waits
// for the SDK result. The topic and supplied user IDs must be nonempty UTF-8
// without NUL. Use explicit publisher IDs; an empty list is passed to the native
// SDK unchanged rather than expanded by this wrapper.
//
// Inspect both Succeeded and Failed even when err is non-nil: a callback can
// report partial success, and successful subscriptions are not rolled back.
// If no result was received, including on timeout, the returned lists are empty;
// that does not prove a submitted subscription failed.
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

// UnsubscribeTopic removes subscriptions to the specified publishers in topic
// and waits for the SDK result. Topic and supplied user IDs must be nonempty
// UTF-8 without NUL. An empty user list is passed to the native SDK unchanged.
// This does not leave the topic as a publisher.
func (s *Stream) UnsubscribeTopic(ctx context.Context, topic string, users ...string) error {
	options, err := topicUsers(topic, users)
	if err != nil {
		return err
	}
	return s.call(ctx, "unsubscribe topic", func() (int, uint64) { return s.native.UnsubscribeTopic(topic, options) })
}

// Subscribers returns the publisher IDs this client subscribes to in topic.
// It does not list the audience subscribed to this client's publications.
// The topic must be nonempty UTF-8 without NUL. The result is a Go-owned slice;
// on any error it returns nil.
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
