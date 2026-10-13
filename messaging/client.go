package messaging

import (
	"context"
	"sync"
	"time"
	"unicode/utf8"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

type result struct {
	value   any
	code    int
	details string
	err     error
}

type waiter struct {
	result chan result
	owner  *Stream
}

// Client owns a native RTM client and all streams created from it.
// Create one with New; its zero value is not usable. Methods are safe to call
// concurrently, but a Client must not be copied after creation. Native initiation
// is serialized, while asynchronous requests wait independently for their own
// results. Call Close to release native resources even if login or another
// operation fails.
type Client struct {
	native       *native.IRtmClient
	gate         chan struct{}
	mu           sync.Mutex
	closed       bool
	pending      map[uint64]waiter
	early        map[uint64]result
	events       chan Event
	dropped      uint64
	timeout      time.Duration
	resultEvents bool
	streams      map[*Stream]struct{}
	closeOnce    sync.Once
	closeErr     error
}

func newClient(buffer int, timeout time.Duration) *Client {
	c := &Client{gate: make(chan struct{}, 1), pending: make(map[uint64]waiter), events: make(chan Event, buffer), timeout: timeout, streams: make(map[*Stream]struct{})}
	c.gate <- struct{}{}
	return c
}

// New creates an owned native client without logging in or making an
// application-level network request. Call Client.Login before service operations
// and Client.Close when finished. Creation does not establish connectivity or
// project feature access.
//
// New validates Config, returning an error wrapping ErrInvalidArgument for
// invalid inputs. Native initialization failures return *Error or ErrUnavailable.
// The linked native SDK and its shared libraries must be available.
func New(cfg Config) (*Client, error) {
	nc, err := cfg.nativeConfig()
	if err != nil {
		return nil, err
	}
	if cfg.EventBuffer < 0 || cfg.EventBuffer > 65536 {
		return nil, invalid("event buffer must be between 0 and 65536")
	}
	if cfg.RequestTimeout < 0 {
		return nil, invalid("request timeout")
	}
	if cfg.EventBuffer == 0 {
		cfg.EventBuffer = 128
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 15 * time.Second
	}
	c := newClient(cfg.EventBuffer, cfg.RequestTimeout)
	c.resultEvents = cfg.ResultEvents
	nc.EventHandler = c.handler()
	var code int
	c.native, code = native.NewRtmClientWithError(nc)
	if c.native == nil {
		c.closeState(nil)
		if code != 0 {
			return nil, sdkError("create", code)
		}
		return nil, ErrUnavailable
	}
	return c, nil
}

// Version returns the linked native RTM SDK version.
// It is the RTM core version, which can differ from the distribution's RTC version.
func Version() string { return native.GetVersion() }

// Events returns the client's bounded, receive-only event queue.
// Events own their payloads and remain valid after Close. When the queue is full,
// the incoming (newest) event is dropped and DroppedEvents increases. Request
// completions use a separate path and are not dropped by event-queue overflow.
//
// Close closes this channel; buffered events remain available before a receive
// reports closure. Events and Next consume the same queue. Multiple consumers
// compete for events rather than receiving a broadcast.
func (c *Client) Events() <-chan Event { return c.events }

// DroppedEvents returns the cumulative count of incoming events discarded
// because the event queue was full. It includes all event kinds and is not reset
// when the queue is drained. It does not count notifications ignored after Close.
func (c *Client) DroppedEvents() uint64 { c.mu.Lock(); defer c.mu.Unlock(); return c.dropped }

// Next waits for one event from the same queue returned by Events.
// It returns ErrClosed once the closed queue has been drained, or the context's
// error when cancellation wins. A context already canceled on entry is rejected
// before consuming an event. A nil context returns ErrInvalidArgument.
// Config.RequestTimeout does not apply to event waits.
func (c *Client) Next(ctx context.Context) (Event, error) {
	if ctx == nil {
		return Event{}, invalid("nil context")
	}
	select {
	case <-ctx.Done():
		return Event{}, ctx.Err()
	default:
	}
	select {
	case ev, ok := <-c.events:
		if !ok {
			return Event{}, ErrClosed
		}
		return ev, nil
	case <-ctx.Done():
		return Event{}, ctx.Err()
	}
}

func (c *Client) emit(e Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.events <- e:
	default:
		c.dropped++
	}
}

func (c *Client) complete(id uint64, r result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	if w, ok := c.pending[id]; ok {
		delete(c.pending, id)
		w.result <- r
		return
	}
	// Initiation is serialized. A callback may run before the SDK returns its ID.
	// Keep only this initiation window; late canceled callbacks cannot accumulate.
	if c.early != nil && len(c.early) < 1024 {
		if _, alreadyCompleted := c.early[id]; !alreadyCompleted {
			c.early[id] = r
		}
	}
}

func (c *Client) lock(ctx context.Context) error {
	if ctx == nil {
		return invalid("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.gate:
	}
	if err := ctx.Err(); err != nil {
		c.unlock()
		return err
	}
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		c.unlock()
		return ErrClosed
	}
	return nil
}

func (c *Client) unlock() { c.gate <- struct{}{} }

func (c *Client) request(ctx context.Context, op string, owner *Stream, start func() (int, uint64)) (any, error) {
	if ctx == nil {
		return nil, invalid("nil context")
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if err := c.lock(ctx); err != nil {
		return nil, err
	}
	if owner != nil && owner.closed {
		c.unlock()
		return nil, ErrClosed
	}
	c.mu.Lock()
	c.early = make(map[uint64]result)
	c.mu.Unlock()
	code, id := start()
	ch := make(chan result, 1)
	c.mu.Lock()
	early, ready := c.early[id]
	c.early = nil
	if code == 0 && !ready {
		c.pending[id] = waiter{result: ch, owner: owner}
	}
	c.mu.Unlock()
	c.unlock()
	if code == unavailableCode {
		return nil, ErrUnavailable
	}
	if code != 0 {
		return nil, sdkError(op, code)
	}
	if ready {
		return unpack(op, early)
	}
	select {
	case r := <-ch:
		return unpack(op, r)
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

func unpack(op string, r result) (any, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.code != 0 {
		return r.value, &Error{Op: op, Code: ErrorCode(r.code), Details: r.details}
	}
	return r.value, nil
}

func (c *Client) call(ctx context.Context, op string, start func() (int, uint64)) error {
	_, err := c.request(ctx, op, nil, start)
	return err
}

func (c *Client) closeState(owner *Stream) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if owner == nil {
		c.closed = true
		close(c.events)
	}
	for id, w := range c.pending {
		if owner == nil || w.owner == owner {
			delete(c.pending, id)
			w.result <- result{err: ErrClosed}
		}
	}
}

// Close releases all remaining streams before the native client, closes the
// event queue, and unblocks pending requests with ErrClosed. It is idempotent:
// subsequent calls return the first close result. If native releases fail, Close
// returns the first such *Error and still attempts the remaining releases.
//
// Close does not perform or wait for Logout or Stream.Leave. Call them first
// with a fresh bounded context when graceful departure is required, then close
// even if they fail. Close waits for serialized native access; a blocking native
// call or WithNative callback can delay it.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		<-c.gate
		defer c.unlock()
		c.closeState(nil)
		for s := range c.streams {
			s.closed = true
			if e := sdkError("close stream", s.native.Release()); c.closeErr == nil {
				c.closeErr = e
			}
		}
		clear(c.streams)
		if c.native != nil {
			if e := sdkError("close", c.native.Release()); c.closeErr == nil {
				c.closeErr = e
			}
			c.native = nil
		}
	})
	return c.closeErr
}

// Login authenticates the configured user and waits for the SDK login result.
// Use an RTM token issued for Config.AppID and Config.UserID, not the RTC token
// used to join a stream channel. The wrapper allows an empty token, but project
// authentication settings may reject it. Tokens must be UTF-8 without NUL.
//
// The wait is bounded by ctx and Config.RequestTimeout. Cancellation does not
// undo a submitted login or stop native reconnection attempts.
func (c *Client) Login(ctx context.Context, token string) error {
	if err := validText("token", token, false); err != nil {
		return err
	}
	return c.call(ctx, "login", func() (int, uint64) { return c.native.Login(token) })
}

// Logout requests logout and waits for the SDK result within ctx and
// Config.RequestTimeout. It does not release the client or its streams; call
// Close to free them. Cancellation does not undo an already submitted logout.
func (c *Client) Logout(ctx context.Context) error {
	return c.call(ctx, "logout", func() (int, uint64) { return c.native.Logout() })
}

// RenewToken submits a replacement RTM login token and waits for the SDK result.
// Obtain a token for the configured project and user from a trusted token service.
// Stream-channel RTC tokens must instead be renewed through Stream.RenewToken.
// Empty tokens pass local validation; all tokens must be UTF-8 without NUL.
// There is no automatic token renewal.
func (c *Client) RenewToken(ctx context.Context, token string) error {
	if err := validText("token", token, false); err != nil {
		return err
	}
	return c.call(ctx, "renew token", func() (int, uint64) { return c.native.RenewToken(token) })
}

// Publish sends data to a message channel and waits for the SDK result.
// Login is required; subscribing to the channel is not required to publish.
// The channel must be nonempty UTF-8 without NUL, and data must be nonempty.
// Text messages require valid UTF-8 but may contain embedded NUL bytes.
//
// Publish copies data before native submission; do not modify data concurrently
// with the call. Native size, permission, and rate limits still apply. A nil error
// reports SDK completion, not recipient application processing. Cancellation or
// timeout stops waiting without undoing a submitted publication.
func (c *Client) Publish(ctx context.Context, channel string, data []byte, options PublishOptions) error {
	return c.publish(ctx, channel, MessageChannel, data, options)
}

// PublishToUser sends a direct message to userID and waits for the SDK result.
// The recipient is a user identity, not a message-channel subscription. userID
// must be nonempty UTF-8 without NUL. Payload copying, validation, completion,
// and cancellation follow Client.Publish.
func (c *Client) PublishToUser(ctx context.Context, userID string, data []byte, options PublishOptions) error {
	return c.publish(ctx, userID, UserChannel, data, options)
}

func (c *Client) publish(ctx context.Context, target string, kind ChannelType, data []byte, o PublishOptions) error {
	if err := validText("publish target", target, true); err != nil {
		return err
	}
	if len(data) == 0 {
		return invalid("empty message")
	}
	if err := validText("custom type", o.CustomType, false); err != nil {
		return err
	}
	// A text payload may contain NUL; it is a length-delimited message, not a C string.
	if o.Text && !utf8.Valid(data) {
		return invalid("text message must be UTF-8")
	}
	options := &native.PublishOptions{ChannelType: kind, MessageType: BinaryMessage, CustomType: o.CustomType, StoreInHistory: o.StoreInHistory}
	if o.Text {
		options.MessageType = TextMessage
	}
	payload := append([]byte(nil), data...)
	return c.call(ctx, "publish", func() (int, uint64) { return c.native.Publish(target, payload, options) })
}

// Subscribe subscribes to a message channel and waits for the SDK result.
// Messages are enabled unless o.DisableMessages; presence, metadata, and lock
// events are opt-in through o. The channel must be nonempty UTF-8 without NUL. Received events
// arrive through Events or Next and are subject to the queue's overflow policy.
func (c *Client) Subscribe(ctx context.Context, channel string, o SubscribeOptions) error {
	if err := validText("channel", channel, true); err != nil {
		return err
	}
	options := o.nativeOptions()
	return c.call(ctx, "subscribe", func() (int, uint64) { return c.native.Subscribe(channel, options) })
}

// Unsubscribe removes a message-channel subscription and waits for the SDK
// result. The channel must be nonempty UTF-8 without NUL. Events already queued
// are not removed by this operation.
func (c *Client) Unsubscribe(ctx context.Context, channel string) error {
	if err := validText("channel", channel, true); err != nil {
		return err
	}
	return c.call(ctx, "unsubscribe", func() (int, uint64) { return c.native.Unsubscribe(channel) })
}

// SetParameters passes a JSON object to the native SDK for advanced configuration.
// Invalid JSON, null, and non-object values return ErrInvalidArgument; the native
// SDK validates supported parameter names and values. It uses WithNative, so ctx
// bounds waiting for access but cannot interrupt the synchronous native call.
// Config.RequestTimeout does not apply.
func (c *Client) SetParameters(ctx context.Context, parameters string) error {
	if err := validateParameters(parameters); err != nil {
		return err
	}
	return c.WithNative(ctx, func(n *native.IRtmClient) error { return sdkError("set parameters", n.SetParameters(parameters)) })
}

// WithNative lends fn serialized access to the native client and returns fn's
// error unchanged. A nil fn or ctx returns ErrInvalidArgument. The context bounds
// waiting for access; Config.RequestTimeout does not apply, and cancellation
// cannot interrupt fn once it is running.
//
// The function must return promptly and must not reenter this Client or any of
// its Streams, retain or release borrowed native objects, or replace the event
// handler. Such reentry can deadlock, including when calling Close. Asynchronous
// native result callbacks belong to this package; use the public context-aware
// methods for supported operations.
func (c *Client) WithNative(ctx context.Context, fn func(*native.IRtmClient) error) error {
	if fn == nil {
		return invalid("nil native function")
	}
	if err := c.lock(ctx); err != nil {
		return err
	}
	defer c.unlock()
	return fn(c.native)
}
