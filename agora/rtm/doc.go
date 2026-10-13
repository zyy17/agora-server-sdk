// Package agorartm exposes the low-level Agora RTM binding.
//
// Applications that want context-aware operations and managed ownership should
// use github.com/zyy17/agora-server-sdk/messaging. This package follows the
// linked native SDK's data types, numeric errors, and callback model rather than
// providing an independent Go service state machine.
//
// # Ownership and concurrency
//
// A successfully created [IRtmClient] must be released. Stream-channel handles
// must be released before their client. Interfaces returned by the client are
// borrowed from it and must not be used after release. Direct callers must
// serialize SDK method calls with Release; the callback registry itself is safe
// for concurrent native callbacks. Never invoke a blocking SDK method or Release
// from inside a native event handler.
//
// Callback payloads are deep-copied into Go-owned values and may be retained
// after the callback. [RtmConfig.EventHandler] is copied when creating a client;
// changing its function fields afterward does not change registered callbacks.
// Keep handlers short and arrange application work outside native callbacks.
//
// # Submission and completion
//
// Some native lock, storage, and stream APIs return void. Their Go wrappers
// return zero only when a request was submitted locally, or a negative local
// validation error. The corresponding asynchronous callback is the authoritative
// operation result. Do not interpret successful submission as successful
// server-side completion, delivery, or recipient processing.
//
// This package does not add context cancellation, request deadlines, event
// buffering, or automatic token renewal. Use the messaging facade for bounded
// waits and an owned event queue. Even there, cancellation of a wait does not
// undo an operation already submitted to the native SDK.
//
// # Native SDK requirements
//
// Build and runtime environments must provide the pinned SDK's headers and
// shared libraries. [GetVersion] reports the linked RTM core version, which can
// differ from the distribution's RTC version. Service permissions, supported
// features, naming restrictions, and limits are validated by that SDK and the
// Agora service.
package agorartm
