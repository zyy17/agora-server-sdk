// Package messaging provides context-aware Agora RTM messaging with explicit
// native-resource ownership.
//
// It is independent of the root SDK's RTC Service and Session. The package
// supports message channels, direct-user publications, stream-channel topics,
// presence, metadata, locks, and stored-message history. Native SDK libraries
// must be installed and available to the linker and runtime; creating a client
// does not establish project feature access, quotas, or network connectivity.
//
// # Client and stream lifecycle
//
// Create a [Client] with [New], then call [Client.Login] before service
// operations. The caller owns the client and must call [Client.Close], including
// after login or another operation fails. Create a [Stream] with
// [Client.NewStream], call [Stream.Join] before topic operations, and close it
// when finished. Closing a client also releases all of its remaining streams.
//
// Client and Stream methods support concurrent callers. Native method initiation
// is serialized, while asynchronous requests wait independently for their
// matching result callbacks. Do not copy a Client or Stream, and do not mutate
// input maps, slices, or referenced configuration while a method reads them.
// Returned event and query data is copied into Go-owned values.
//
// Close is resource cleanup, not a graceful logout or leave request. If graceful
// departure matters, call [Stream.Leave] and [Client.Logout] with a fresh bounded
// shutdown context before closing. Close must still run if those operations
// fail. A synchronous native call or [Client.WithNative] callback can delay Close.
//
// # Contexts, completion, and errors
//
// Every context-taking method rejects a nil context. Asynchronous requests use
// the earlier of the caller's deadline and [Config.RequestTimeout], which
// defaults to 15 seconds. This timeout includes waiting for serialized native
// access. It does not apply to [Client.Next], [Client.WithNative],
// [Client.SetParameters], [Client.NewStream], or Close. A context cannot interrupt
// a synchronous native call or a WithNative callback already running.
//
// Canceling a request stops waiting; it does not cancel or undo an operation
// already submitted to the SDK. A timed-out publish, metadata update, or lock
// acquisition may still take effect. Check remote state or use an application
// idempotency strategy before retrying operations with side effects.
//
// A nil result from an asynchronous method means its SDK completion callback
// reported success, not merely that native submission succeeded. A successful
// publish is not an acknowledgment that a recipient application processed it.
// Add application-level acknowledgments when that distinction matters.
//
// Use errors.Is for [ErrInvalidArgument], [ErrClosed], [ErrUnavailable],
// context.Canceled, and context.DeadlineExceeded. Use errors.As to inspect
// [Error], whose [ErrorCode] preserves the native numeric code. Native project
// permissions, size limits, quotas, and unsupported-feature errors remain the
// responsibility of the linked SDK and service.
//
// # Events and overflow
//
// Consume unsolicited notifications through [Client.Events] or [Client.Next].
// Both read the same queue; multiple consumers compete rather than receiving a
// broadcast. [Event.Kind] identifies the populated payload. All payload data,
// including nested maps, slices, and binary messages, survives the native
// callback and can outlive the client.
//
// The default queue holds 128 events. When full, the incoming newest event is
// discarded and [Client.DroppedEvents] increases. This includes token, link,
// presence, metadata, and lock events. Request completions use a separate path
// and are not lost because of queue overflow. Monitor drops and query state
// again when correctness depends on an uninterrupted sequence of notifications.
//
// Closing the client closes its queue without discarding buffered events.
// A live context can continue draining them through Next, which returns
// ErrClosed after the queue empties. Ranging Events ends after the same drain.
// Event waits use their own contexts, not the request timeout.
//
// # Authentication
//
// [Client.Login] and [Client.RenewToken] use an RTM token matching the configured
// App ID and user ID. [Stream.Join] and [Stream.RenewToken] use an RTC token for
// the stream channel and identity. These tokens are not interchangeable. Obtain
// them from a trusted application server; never embed an App Certificate in an
// end-user application. Empty tokens pass local validation but may be rejected
// by the project's authentication configuration.
//
// Token renewal is not automatic. Handle [EventToken] and [EventTokenExpiring],
// and renew the appropriate client or stream token. Keep track of expiry outside
// the event queue because notification overflow can hide an expiry notice.
//
// # Presence, metadata, locks, and history
//
// [Client.OnlineUsers] returns one page at a time. Pass [PresencePage.NextPage]
// unchanged into the next [PresenceOptions.Page] until it is empty. Presence
// notifications can contain partial changes; neither pages nor queued events
// form an atomic snapshot across a changing channel. Presence state belongs to
// a user in a channel and is separate from durable metadata.
//
// Metadata has both collection and item revisions. For unconditional writes,
// explicitly set [Metadata.Revision] and each [MetadataItem.Revision] to -1.
// Go zero values are forwarded unchanged. Conditional writes should use queried
// revisions and handle native conflicts. Metadata writes require a nonempty
// item list with unique keys; an empty list does not mean delete everything.
// In contrast, [Client.RemoveState] with no keys removes all presence state.
//
// [Client.SetLock] defines a lock; [Client.AcquireLock] separately requests
// ownership. Automatic native acquisition retry is disabled, but cancellation
// can still race with a successful submitted acquisition. Lock event delivery
// alone is not proof of ownership because events can be dropped.
//
// [Client.History] supports [MessageChannel] and [UserChannel], not stream
// topics. Stored history requires project support and publications made with
// [PublishOptions.StoreInHistory]. It is not a bidirectional conversation log
// or a replacement for delivery acknowledgments. Preserve native timestamps and
// pagination cursors without changing units; check a [HistoryPage.NextStart]
// value fits int64 before assigning it to [HistoryOptions.Start].
//
// [Stream.SubscribeTopic] can return both a nonempty [TopicSubscription] and an
// error. Always inspect the successful and failed publisher lists; successful
// subscriptions are not rolled back by a partial failure. Other query methods
// return zero or nil results on error rather than partial callback data.
//
// # Native escape hatch
//
// [Client.WithNative] provides borrowed, serialized native access for advanced
// use. Its callback must return promptly and must not reenter the client or its
// streams, retain or release borrowed objects, or replace the event handler.
// Asynchronous native result callbacks are owned by this package; prefer the
// public context-aware methods for supported operations.
package messaging
