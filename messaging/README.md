# Messaging (RTM) reference

[简体中文](README.zh-CN.md) · [Home](../README.md) · [Runnable scenarios](../examples/messaging/README.md)

`messaging` owns native RTM clients and Stream Channels. Its asynchronous methods correlate SDK result callbacks to requests; incoming events are copied into a bounded Go queue. It is independent of RTC `Service` and `Session`. Project feature access, quotas, and native errors still apply.

## Minimal example

Install the pinned SDK as described in the [root guide](../README.md). Run this program with `AGORA_APP_ID`, `AGORA_RTM_USER_ID`, and the matching `AGORA_RTM_TOKEN`. It subscribes, publishes one text message, and listens until Ctrl+C. Run another logged-in client with a different user ID in `demo` to test reception; do not rely on receiving your own publication.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/zyy17/agora-server-sdk/messaging"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    if err := run(ctx); err != nil {
        log.Print(err)
    }
}

func run(ctx context.Context) (err error) {
    client, err := messaging.New(messaging.Config{
        AppID:  os.Getenv("AGORA_APP_ID"),
        UserID: os.Getenv("AGORA_RTM_USER_ID"),
    })
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, client.Close()) }()
    if err = client.Login(ctx, os.Getenv("AGORA_RTM_TOKEN")); err != nil {
        return err
    }
    defer func() {
        shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        err = errors.Join(err, client.Logout(shutdown))
    }()
    if err = client.Subscribe(ctx, "demo", messaging.SubscribeOptions{}); err != nil {
        return err
    }
    if err = client.Publish(ctx, "demo", []byte("hello"), messaging.PublishOptions{Text: true}); err != nil {
        return err
    }
    for {
        event, nextErr := client.Next(ctx)
        if nextErr != nil {
            return nextErr
        }
        if event.Kind == messaging.EventMessage && event.Message != nil {
            fmt.Printf("%s: %s\n", event.Message.Publisher, event.Message.Data)
        }
    }
}
```

The loop filters message events because login and subscription also generate other event types. Production applications must also handle link/token events and monitor `DroppedEvents()`. Each asynchronous request has its own default 15-second deadline; the receive loop’s context remains active until canceled.

## Configuration and naming

| `Config` field | Wrapper default / validation |
| --- | --- |
| `AppID`, `UserID` | Required, valid UTF-8, no NUL; further format/length checks are native |
| `EventBuffer` | 128 events; zero selects default; accepted explicit range 1–65536 |
| `RequestTimeout` | 15 seconds; zero selects default; negative rejected |
| `Identity` | `IdentityDefault`: string IDs without `Native`, otherwise preserve `Native.UseStringUserId`; explicit `IdentityString` / `IdentityNumeric` override it |
| `Connection` | Optional typed region, transport, presence/heartbeat/reconnection timers, multipath, ISP policy |
| `Log`, `Proxy`, `Encryption`, `Private` | Optional typed native configuration; non-nil values replace corresponding `Native` settings |
| `Parameters` | Optional JSON object applied during creation; empty retains `Native.Parameters` |
| `ResultEvents` | `false`; enable to observe all asynchronous result callbacks, including late results |
| `Native` | `nil` uses the pinned binding’s defaults; retained for compatibility and platform `Context` |

Use stable, distinct user IDs within one App ID. Official setup guidance limits RTM IDs to 64 ASCII characters; logging in elsewhere with the same ID can displace the existing login. For compatible names, prefer ASCII letters/digits/underscores/hyphens, beginning with a letter or digit; check current service restrictions before using other characters. Wrapper UTF-8 validation alone does not enforce the native naming rules.

Prefer typed configuration: `ConnectionConfig`, `LogConfig`, `ProxyConfig`, `EncryptionConfig`, and `PrivateConfig` expose the native fields without importing the raw binding. Connection durations must be nonnegative whole seconds fitting `uint32`; when a `ConnectionConfig` is supplied, zero durations pass through unchanged, and zero area means global. A zero reconnect timeout asks the native SDK to retry indefinitely, independently of the Go request timeout. A nil connection preserves the pinned Go binding’s presence/heartbeat/reconnect defaults of 30/0/0 seconds. `EncryptionConfig.Salt` is a fixed 32-byte array; referenced configuration is copied during creation. Do not mutate inputs concurrently with `New`.

`IdentityNumeric` interprets `UserID` as a numeric RTC-compatible ID; supply its decimal string and matching token identity. `IdentityString` preserves a numeric-looking ID as a string identity. If supplying `Config.Native`, start from `agorartm.NewRtmConfig()` and adjust only needed fields. The wrapper overwrites `AppId`, `UserId`, and `EventHandler`; explicitly supplied typed settings take precedence. Unlike earlier wrapper versions, `IdentityDefault` now respects `Native.UseStringUserId`. Set `IdentityString` explicitly to retain the earlier forced-string behavior. Region, proxy, encryption, and private settings still require native project support. Platform-specific `Context` remains available through `Native` and must obey cgo pointer-lifetime rules.

## Lifecycle, concurrency, and cancellation

```mermaid
flowchart TD
    A[New] --> B[Login with RTM Token]
    B --> C[Message Channel operations]
    B --> D[NewStream]
    D --> E[Join with RTC Token]
    E --> F[Topic operations]
    F --> G[Leave and Close stream]
    C --> H[Logout with fresh shutdown context]
    G --> H
    H --> I[Client.Close]
    I --> J[Release remaining streams and client]
```

- `New` allocates a client; it does not log in. Call `Login` before service operations. Creation is not proof of network or feature access.
- `Client` and `Stream` methods support concurrent callers. Native initiation is serialized; request waits are independent and keyed by request ID, including callbacks received before initiation returns. Do not mutate caller-owned input maps/slices while an operation is reading them.
- Each asynchronous wait uses the earlier of the caller deadline and `RequestTimeout`. Nil contexts are invalid. Cancellation stops waiting, **not an already submitted operation**, and cannot interrupt a synchronous native call already running.
- A canceled publish, metadata write, or lock acquisition may still take effect. Inspect remote state or use an application idempotency scheme before retrying. `AcquireLock` disables native retry; `AcquireLockWithOptions` can enable it, including retries that outlive the caller’s context.
- `Close` is idempotent and unblocks pending requests with `ErrClosed`. `Client.Close` releases remaining streams before the client, then no further operations may be initiated. `Stream.Close` also fails its pending requests.
- `Close` does not await `Logout` or a graceful stream `Leave`. When required, call those first with a fresh bounded context, then close even if they fail. A blocked native call can delay close.

## Channels and publication options

| API | Destination and prerequisite |
| --- | --- |
| `Publish` | Message Channel; publisher must log in but need not subscribe |
| `Subscribe` / `Unsubscribe` | Message Channel reception; messages enabled unless `DisableMessages` is true |
| `PublishToUser` | Direct message to a logged-in user ID; no channel subscription required |
| `NewStream` / `Stream.Join` | Owned Stream Channel; log in, create, then join before topic operations |
| `Channel{Name, Type}` | Presence, metadata, locks: `MessageChannel` or `StreamChannel`; history: `MessageChannel` or `UserChannel` |

| Option | Zero value / behavior |
| --- | --- |
| `PublishOptions.Text` | `false`: binary; `true`: valid UTF-8 required; a text payload may contain NUL |
| `PublishOptions.CustomType` | Empty; otherwise UTF-8 without NUL |
| `PublishOptions.StoreInHistory` | `false`; opt in at publish time and enable project history access |
| `SubscribeOptions.Presence` / `Metadata` / `Locks` | All `false`; opt in to each event family and provision required project features |
| `SubscribeOptions.DisableMessages` | `false`; set `true` for presence/metadata/lock-only subscriptions |
| `SubscribeOptions.Quiet` | `false`; `true` requests native quiet subscription behavior |

The wrapper’s `SubscribeOptions{}` differs from native defaults: messages are enabled, Presence/Metadata/Locks are disabled. Publish payloads must be nonempty. Server message-size and rate quotas are not prechecked; inspect native errors. A nil publish error means the SDK operation succeeded, not that a recipient application processed or persisted the message. Add application acknowledgments when that distinction matters.

## Presence

| API / option | Contract |
| --- | --- |
| `OnlineUsers(ctx, channel, PresenceOptions)` | One page; user IDs included unless `ExcludeUserID`; `IncludeState` defaults to `false` |
| `PresenceOptions.ExcludeUserID` | `false`; `true` forwards native `includeUserId=false`. The pinned C callback does not supply total occupancy, so this is not a count-only API |
| `PresenceOptions.Page` | Empty for first page; pass returned `NextPage` unchanged; empty `NextPage` ends paging |
| `UserChannels(ctx, userID)` | Queries channels for a user |
| `SetState(ctx, channel, map)` | Nonempty map; nonempty keys; keys/values must be UTF-8 without NUL |
| `State(ctx, channel, userID)` | Queries a user’s channel state |
| `RemoveState(ctx, channel, keys...)` | Removes specified keys; **an empty key list removes all state keys** |

State is channel-scoped Presence data, not durable application storage. Querying online users does not require joining the channel; receiving Presence events requires subscribing or joining with Presence enabled. `SetState` only sets the current user’s state. Project permissions remain native requirements. Paginated results and queued events are not an atomic snapshot across a changing channel.

## Metadata and revision checks

Use `SetChannelMetadata`, `UpdateChannelMetadata`, `RemoveChannelMetadata`, and `ChannelMetadata` for channel items; corresponding `*UserMetadata` methods address user items. `Set` creates or replaces supplied items, `Update` modifies existing items, and `Remove` deletes supplied keys; native errors define missing-item/revision conflicts. User metadata events require `SubscribeUserMetadata`; channel metadata events require the subscription/join `Metadata` option.

For unconditional writes, explicitly set **both** `Metadata.Revision` and each `MetadataItem.Revision` to `-1`. Go zero values are forwarded unchanged; they are not converted to `-1`. For conditional writes, use revisions returned by a query and handle conflicts. Do not confuse whole-set revision with per-item revision.

This is a value-construction example; pass it to the appropriate metadata method with a valid channel/user and context:

```go
metadata := messaging.Metadata{
    Revision: -1,
    Items: []messaging.MetadataItem{
        {Key: "status", Value: "ready", Revision: -1},
    },
}
```

The wrapper requires at least one item, unique nonempty keys, and valid UTF-8 without NUL in keys, values, and author IDs. `MetadataOptions.RecordTimestamp` / `RecordUserID` default to `false`. `LockName` optionally protects channel metadata writes and is rejected for user metadata writes. Storage feature access, item limits, and lock ownership are checked by the service.

## Locks

The lifecycle is `SetLock` → `AcquireLock` → `ReleaseLock` → `RemoveLock`; `Locks` queries definitions/owners, and `RevokeLock` targets a specified owner. `SetLock` defines a lock but does not acquire it. TTL describes how long ownership is retained after its owner disconnects/leaves, not the lifetime of the lock definition. The official range is 10–300 seconds; the wrapper only checks positive whole seconds fitting `uint32`, so server validation can still reject a locally valid duration.

Enable lock feature access and `Locks` events where needed. `AcquireLock(ctx, channel, name)` remains the no-retry convenience method. Use `AcquireLockWithOptions(ctx, channel, name, AcquireLockOptions{Retry: true})` to enable native retry.

**Canceling or timing out the Go call does not cancel native retry.** The pinned SDK exposes no per-request cancellation API; it can acquire the lock after the Go call has returned, including after an acquisition-failed callback. A failed attempt is not proof that retry stopped. Enable `Config.ResultEvents` before creation, consume `RequestAcquireLock` results and lock notifications, and release unwanted successful acquisitions using a fresh context. See the [compiled cleanup example](example_test.go). The example takes ownership of a dedicated client and closes it on every exit to stop further retry. It assumes one acquisition for that target and one queue consumer; dispatch deliberately when multiple operations share a target.

A single `Locks` query is only a snapshot and cannot prove a still-pending retry will never acquire the lock. Reconcile on dropped events and retain ownership/cleanup handling for the lifetime of the retrying client. Closing releases native resources but is not a successful `ReleaseLock`; disconnected-owner retention still follows the native TTL. Lock events can be dropped, so the queue alone is not proof of continued ownership for a consequential write.

## History

`History(ctx, channel, options)` queries **Message Channel or User Channel** history; Stream Channels are unsupported. User Channel history addresses a user channel; do not treat it as a bidirectional conversation log. History must be enabled, and the message must have been published with `StoreInHistory: true`; ordinary pub/sub examples do not create stored history.

| Field | Contract |
| --- | --- |
| `HistoryOptions.Count` | Zero selects 100; explicit range 1–100 |
| `Start` | Exclusive newer bound / cursor; zero delegates to native default |
| `End` | Inclusive older bound; zero delegates to native default |
| Nonzero bounds | `Start >= End`; results are queried from newer to older |
| `HistoryPage.NextStart` | Zero ends paging; otherwise pass as the next `Start`, retaining other query options |
| `HistoryMessage.Data` | Copied binary-safe payload; timestamps retain native units |

Do not convert history cursors to RTC media timestamps or assume a wall-clock unit from the field name. Preserve SDK-returned cursors. The official API also limits an explicit query interval to 24 hours; the wrapper leaves that check to the service because it preserves native timestamp units. Service retention, availability, and rate limits still apply; an event queue is not a replacement for history storage.

## Stream Channels and Topics

A Stream is owned by its Client; its events arrive on the Client queue. Call `NewStream`, `Join`, then topic operations. `JoinOptions.Token` is an **RTC Token for that Stream Channel**, separate from the RTM login token. `Presence`, `Metadata`, `Locks`, and `Quiet` default to `false`; enable only the needed event families.

| API / option | Contract |
| --- | --- |
| `JoinTopic` / `LeaveTopic` | Join/leave as a publisher; reception separately uses `SubscribeTopic` |
| `TopicOptions.Ordered` | `false`; `true` requests native ordered QoS |
| `TopicOptions.Metadata` | Empty; UTF-8 without NUL |
| `TopicOptions.SyncWithMedia` | `false`; native media synchronization behavior when enabled |
| `TopicOptions.Priority` | `PriorityNormal`; alternatives: `PriorityHighest`, `PriorityHigh`, `PriorityLow` |
| `Stream.Publish` | Nonempty payload; `TopicMessageOptions.Text` defaults to binary, `CustomType` empty, `SendTimestamp` zero; timestamp semantics are native |
| `SubscribeTopic(topic, users...)` | Subscribes to specified publishers; inspect `Succeeded` / `Failed` even when an error is returned |
| `UnsubscribeTopic` | Removes publisher subscriptions |
| `Subscribers(topic)` | Returns publisher IDs this client subscribes to; not the audience subscribing to this client |
| `Leave` / `Close` | Leave the native channel, then release it; Client close releases any remaining streams |

Ordering, priority, and media synchronization do not replace application delivery acknowledgments. Use explicit publisher IDs and inspect partial subscription results. The [stream scenario](../examples/messaging/README.md) demonstrates setup and teardown; it is not a long-running topic receiver.

## Authentication and renewal

| Operation | Token |
| --- | --- |
| `Client.Login`, `Client.RenewToken` | RTM Token for the configured App ID and RTM user ID |
| `Stream.Join`, `Stream.RenewToken` | RTC Token for the Stream Channel and identity |
| Root `token.RTC` helper | RTC AccessToken2 only; not RTM token generation |

Obtain tokens from a trusted application server. Empty tokens are accepted by the wrapper but may be rejected by project authentication settings. Consume `EventTokenExpiring` / `EventToken`, inspect their channel/type information, and renew the matching client or stream; there is no automatic token service. Keep known expiry times and link state under application control because event overflow can drop token notifications. Never embed App Certificates in end-user applications.

## Events and queue behavior

`Events()` returns a buffered receive-only channel; `Next(ctx)` consumes the same queue. Event kinds are `EventMessage`, `EventPresence`, `EventTopic`, `EventLock`, `EventStorage`, `EventLink`, `EventToken`, and `EventTokenExpiring`; enabling `ResultEvents` also adds `EventRequestResult`. Read only the populated field indicated by `Kind`. Data, nested slices/maps, and binary payloads are independent Go copies and can outlive the callback/client.

A full RTM queue drops the **newest** event and increments `DroppedEvents()`. This includes token and state events. Request results use a separate correlation path and are not dropped by event-queue overflow. A live context drains buffered events after close; once empty, `Next` returns `ErrClosed`, and ranging `Events()` ends. Canceled contexts return their context error. Multiple consumers compete for events; fan out explicitly if needed. This differs from [RTC’s oldest-drop / `io.EOF` contract](../docs/rtc.md#queues-and-data-lifetime).

### Typed results and late completions

`Config.ResultEvents: true` copies each of the 38 native asynchronous result callbacks into `Event.RequestResult`. `Operation` uses a named `RequestOperation` constant; `RequestID` preserves the SDK ID. `Channel`, `UserID`, `Topic`, `TopicMetadata`, `LockName`, and `Service` expose whichever context the callback supplies. Query payloads have typed fields: `Presence`, `State`, `Channels`, `Metadata`, `Locks`, `Subscription`, `Subscribers`, and `History`. Error callbacks retain code, details, and any partial payload.

These copies are independent of the synchronous method’s returned slices/maps. Results are observable even after timeout/cancellation or when the SDK repeats a callback. A waiting operation completes on its first matching callback; repeated callbacks do not overwrite it. Request events share the bounded queue and can be dropped, while operation completion uses a separate path. Local validation/submission/context errors do not synthesize a native callback. No asynchronous callbacks are delivered after the client closes.

Named `PresenceEventType`, `TopicEventType`, `LockEventType`, `StorageEventType`, `TokenEventType`, `LinkState`, `LinkOperation`, `LinkReason`, `ServiceType`, and `StorageType` values make event handling possible without magic integers. Values retain the native numbers, including unknown future values. `ErrorCode` exposes the pinned signed `Code…` constants and `Reason()` queries the linked SDK’s explanatory string.

## Native access and errors

`WithNative(ctx, fn)` serializes native access. The callback must return promptly and must not reenter the Client or its Streams, retain/release borrowed objects, or replace event handlers. Asynchronous result callbacks belong to the wrapper; use public context-aware methods for supported requests. `SetParameters` accepts a JSON object and uses this serialized path.

| Error | Meaning |
| --- | --- |
| `ErrInvalidArgument` | Wrapper validation failed |
| `ErrClosed` | Client/stream closed, pending request canceled by close, or event queue drained |
| `ErrUnavailable` | Requested native interface could not be obtained |
| `context.Canceled` / `context.DeadlineExceeded` | Waiting ended; submitted side effects may still occur |
| `*messaging.Error` | Immediate or asynchronous SDK rejection; inspect `Op`, `Code`, and `Details` |

Use `errors.Is` for sentinels/context errors and `errors.As` for `*messaging.Error`. `ErrorCode` preserves the native numeric value. Native methods with void submission signatures still await their result callback; submission alone is not reported as success. For partial topic subscription, inspect the returned publisher lists alongside the error.

## Pinned coverage and native boundaries

The reference is [official Go SDK v2.6.5 at `d51ff1b5…`](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtm), together with the vendored C headers and matching library, not an unrelated current-language SDK.

| Pinned interface | Managed API |
| --- | --- |
| Client creation/release, login/logout, renew token, publish, subscribe/unsubscribe, parameters, version/error reason | `New` / `Close`, `Login` / `Logout`, `RenewToken`, `Publish` / `PublishToUser`, `Subscribe` / `Unsubscribe`, `SetParameters`, `Version`, `ErrorCode.Reason` |
| Presence user/channel queries, state set/remove/get | `OnlineUsers`, `UserChannels`, `SetState`, `RemoveState`, `State`; raw `WhoNow`/`WhereNow` retained as legacy equivalents |
| Metadata create/build, channel and user set/update/remove/get, user subscriptions | `Metadata` values plus the corresponding `*ChannelMetadata` / `*UserMetadata` methods |
| Lock set/query/remove/acquire/release/revoke | `SetLock`, `Locks`, `RemoveLock`, `AcquireLock` / `AcquireLockWithOptions`, `ReleaseLock`, `RevokeLock` |
| History query | `History` and typed cursor/message results |
| Stream create/name/join/leave/release, renew token, topic join/leave/publish/subscribe/unsubscribe/subscriber query | `NewStream`, `Name`, `Join`, `Leave`, `Close`, `RenewToken`, `JoinTopic`, `LeaveTopic`, `Publish`, `SubscribeTopic`, `UnsubscribeTopic`, `Subscribers` |
| Eight unsolicited callbacks and 38 result callbacks | Eight ordinary event kinds plus opt-in `EventRequestResult`; all payloads copied |

Genuine pinned C-ABI boundaries:

- The official Go source’s legacy `OnConnectionStateChanged` field has no slot in the pinned `C_IRtmEventHandler`. Connection notifications use `OnLinkStateEvent` / `EventLink`; no incompatible callback layout is introduced.
- C++ `IStreamChannel::setParameters` has no pinned C wrapper/export and is not an official Go method. Client `SetParameters` remains available.
- The C++ message/presence/topic/lock/storage event timestamps and interval `totalOccupancy` are absent from the pinned C callback structures. They cannot be reconstructed reliably and are not exposed as invented zero-valued data. Link/token/history timestamps are preserved because their ABI carries them.
- No per-request cancel or native retry-cancel operation is available. Go cancellation bounds waiting only.

The raw package retains all 143 official public function/method names. Its deliberate request-ID/error-return corrections and complete-slice callback corrections are preserved; parity does not restore unsafe upstream signatures. `GetAuthorUserId` adds a correctly spelled accessor while keeping the upstream `GetAquthorUserId` alias.

## Sources and verification scope

Wrapper contracts come from [`client.go`](client.go), [`types.go`](types.go), and feature-specific files in this package. The native distribution pin is an RTC/Go release identifier, not an RTM core version; call `messaging.Version()` to inspect the loaded RTM core. The pinned Linux core reports 2.3.0.1, while the Chinese C++ reference is labeled 2.3.0. Online additions are not automatically available in this distribution.

Official Chinese sources:

- [Application setup and identities](https://doc.shengwang.cn/doc/rtm2/cpp/user-guide/setup/application-setup) and [migration/naming guidance](https://doc.shengwang.cn/doc/rtm2/cpp/overview/migration-guide)
- [Message publication](https://doc.shengwang.cn/doc/rtm2/cpp/user-guide/message/send-message), [Channel API](https://doc.shengwang.cn/api-ref/rtm2/cpp/toc-channel/channel), and [Token API](https://doc.shengwang.cn/api-ref/rtm2/cpp/toc-token/token)
- [Storage and revisions](https://doc.shengwang.cn/api-ref/rtm2/cpp/toc-storage/storage) and [Lock semantics](https://doc.shengwang.cn/api-ref/rtm2/cpp/toc-lock/lock)
- [History guide](https://doc.shengwang.cn/doc/rtm2/cpp/user-guide/message/history-message) and [Message/History API](https://doc.shengwang.cn/api-ref/rtm2/cpp/toc-message/message)

Offline tests cover request correlation, cancellation, timeouts, close races, queue overflow, copied multi-element results, callback lifetimes, and binary payloads. They do not establish live delivery, feature entitlement, regional availability, or quotas. Validate those with dedicated project resources. This client package does not include cloud REST services, RTM token issuance, or legacy RTM service APIs.
