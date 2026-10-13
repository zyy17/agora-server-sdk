package messaging

import (
	"context"
	"encoding/json"
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

// Client owns its native client and all streams created from it. Its methods
// are safe to call concurrently. Close must be called to release native memory.
type Client struct {
	native    *native.IRtmClient
	gate      chan struct{}
	mu        sync.Mutex
	closed    bool
	pending   map[uint64]waiter
	early     map[uint64]result
	events    chan Event
	dropped   uint64
	timeout   time.Duration
	streams   map[*Stream]struct{}
	closeOnce sync.Once
	closeErr  error
}

func newClient(buffer int, timeout time.Duration) *Client {
	c := &Client{gate: make(chan struct{}, 1), pending: make(map[uint64]waiter), events: make(chan Event, buffer), timeout: timeout, streams: make(map[*Stream]struct{})}
	c.gate <- struct{}{}
	return c
}

// New creates a native client but does not log in or make an application-level
// network request. RTM support is supplied by the SDK pinned by this repository.
func New(cfg Config) (*Client, error) {
	if err := validText("app ID", cfg.AppID, true); err != nil {
		return nil, err
	}
	if err := validText("user ID", cfg.UserID, true); err != nil {
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
	nc := native.NewRtmConfig()
	if cfg.Native != nil {
		*nc = *cfg.Native
	}
	nc.AppId = cfg.AppID
	nc.UserId = cfg.UserID
	nc.UseStringUserId = true
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

// Version returns the linked RTM native SDK version.
func Version() string { return native.GetVersion() }

// Events returns a bounded, receive-only event queue. Events own their data and
// remain valid after Close. When full, the newest event is dropped; request
// completions never use this queue. Close closes the queue after queued events.
func (c *Client) Events() <-chan Event { return c.events }

func (c *Client) DroppedEvents() uint64 { c.mu.Lock(); defer c.mu.Unlock(); return c.dropped }

// Next waits for an event or cancellation. Consumers can instead range Events.
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
		c.early[id] = r
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

// Close releases streams before their client and unblocks all requests. It is
// idempotent. It does not wait for Logout; call Logout first when needed.
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

func (c *Client) Login(ctx context.Context, token string) error {
	if err := validText("token", token, false); err != nil {
		return err
	}
	return c.call(ctx, "login", func() (int, uint64) { return c.native.Login(token) })
}

func (c *Client) Logout(ctx context.Context) error {
	return c.call(ctx, "logout", func() (int, uint64) { return c.native.Logout() })
}

func (c *Client) RenewToken(ctx context.Context, token string) error {
	if err := validText("token", token, false); err != nil {
		return err
	}
	return c.call(ctx, "renew token", func() (int, uint64) { return c.native.RenewToken(token) })
}

func (c *Client) Publish(ctx context.Context, channel string, data []byte, options PublishOptions) error {
	return c.publish(ctx, channel, MessageChannel, data, options)
}

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

func (c *Client) Subscribe(ctx context.Context, channel string, o SubscribeOptions) error {
	if err := validText("channel", channel, true); err != nil {
		return err
	}
	options := &native.SubscribeOptions{WithMessage: true, WithPresence: o.Presence, WithMetadata: o.Metadata, WithLock: o.Locks, BeQuiet: o.Quiet}
	return c.call(ctx, "subscribe", func() (int, uint64) { return c.native.Subscribe(channel, options) })
}

func (c *Client) Unsubscribe(ctx context.Context, channel string) error {
	if err := validText("channel", channel, true); err != nil {
		return err
	}
	return c.call(ctx, "unsubscribe", func() (int, uint64) { return c.native.Unsubscribe(channel) })
}

// SetParameters accepts an SDK parameter JSON object for advanced configuration.
func (c *Client) SetParameters(ctx context.Context, parameters string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(parameters), &object); err != nil || object == nil {
		return invalid("parameters must be a JSON object")
	}
	return c.WithNative(ctx, func(n *native.IRtmClient) error { return sdkError("set parameters", n.SetParameters(parameters)) })
}

// WithNative provides serialized access to the entire native API. The function
// must return promptly, must not reenter this Client, and must not release or
// retain native objects. Do not replace the event handler. Asynchronous native
// results are owned by this package; prefer the context-aware public methods.
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
