# Messaging (RTM)

[English](README.md) | [简体中文](README.zh-CN.md)

`messaging` wraps the pinned native SDK's supported RTM client interfaces with
owned resources, context-aware request completion, ordinary Go values, and copied
queued events. It is independent of RTC sessions. No callback functions are needed
for application code.

```go
client, err := messaging.New(messaging.Config{
    AppID: os.Getenv("AGORA_APP_ID"),
    UserID: os.Getenv("AGORA_RTM_USER_ID"),
})
if err != nil { return err }
defer client.Close()

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := client.Login(ctx, os.Getenv("AGORA_RTM_TOKEN")); err != nil { return err }
if err := client.Subscribe(ctx, "room", messaging.SubscribeOptions{Presence: true}); err != nil { return err }
if err := client.Publish(ctx, "room", []byte("hello"), messaging.PublishOptions{Text: true}); err != nil { return err }

event, err := client.Next(ctx)
if err != nil { return err }
if event.Message != nil { fmt.Printf("%s: %s\n", event.Message.Publisher, event.Message.Data) }
```

## Supported interfaces

- Client: login, logout, token renewal, publish to channels/users, subscribe and
  unsubscribe, JSON SDK parameters, SDK version
- Presence: online-user pagination, user channels, set/remove/get state
- Storage: set/update/remove/get channel and user metadata; user metadata
  subscriptions and unsubscriptions
- Locks: create/query/remove/acquire/release/revoke
- History: page queries including copied binary payloads and the next cursor
- Streams: owned channel creation, join/leave/renew token, join/leave topics,
  topic publish, subscribe/unsubscribe publishers, subscribed-publisher queries
- Events: messages, presence, topics, locks, storage, link state and token events

`Config.Native` exposes advanced native configuration, including region, logging,
proxy, encryption and private configuration. Start with `agorartm.NewRtmConfig()`
when setting this field. `WithNative` exposes serialized native access for remaining
configuration or specialized APIs. The callback must return promptly; do not retain
objects, replace event handlers, release the client, or reenter its methods.
Native pointers are never needed for the common operations above.

## Lifecycle and concurrency

All `Client` and `Stream` methods support concurrent callers. Native request
initiation is serialized; asynchronous waits are independent and correlated by
request ID, including callbacks arriving before the native method returns.
`Close` is idempotent and releases streams before the client. Pending requests
unblock with `ErrClosed`; queued events remain available before the queue ends.
A closed stream cannot issue more native requests.

Every asynchronous request has the caller's context and a safety deadline from
`Config.RequestTimeout` (default 15 seconds). Canceling or timing out stops waiting,
not an already submitted server operation, and cannot interrupt a synchronous native
SDK call already in progress. A canceled publish, metadata operation
or lock acquisition may still take effect. Never retry consequential writes blindly.
The facade disables automatic native lock-acquisition retry.

`Events()` is a buffered receive-only channel (default 128), also exposed through
`Next(ctx)`. Payloads are independent Go copies and can be kept after the callback
or client ends. When a queue is full, the newest event is dropped and
`DroppedEvents()` increases. Request completions use a separate path and are never
dropped. Size and drain the event queue for your workload; this is not a durable
message log. Multiple consumers share the queue, they do not each get every event.

`Error` contains the operation and typed `ErrorCode` with the exact native numeric
code. Use `errors.As` to inspect it; use `errors.Is` for context errors,
`ErrClosed`, `ErrInvalidArgument`, and `ErrUnavailable`. Some native APIs have a
void submission method: the facade still waits for their authoritative result
callback rather than reporting submission as success. A partial topic subscription
returns successful/failed publisher lists alongside its SDK error when available.

Metadata revision `-1` requests the native SDK's unconditional revision behavior;
zero is an actual revision value. State and metadata keys are validated against
empty values, NULs and invalid UTF-8; service-specific limits remain authoritative
SDK errors. An empty `RemoveState` key list removes all keys.

## Verification and limits

The offline tests cover callback-before-return correlation, concurrent requests,
immediate/asynchronous errors, context cancellation, timeout, close races, queue
saturation, multi-element results, deep copies and binary payloads. Native binding
tests separately exercise flat C arrays and callback lifetime. The examples compile
without credentials. Live network behavior requires an Agora project, suitable
tokens, enabled RTM features and service quotas; it is not exercised by these tests.

The SDK release pin and the RTM core's reported version are distinct. Use
`messaging.Version()` for the loaded RTM core. No cloud REST services, server-side
token issuance or legacy RTM service API are implied by this client package.
