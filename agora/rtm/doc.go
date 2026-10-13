// Package agorartm exposes the low-level Agora RTM binding.
//
// Use the top-level SDK's messaging facade for context-aware calls and managed
// lifecycle. Direct low-level callers must serialize SDK methods and Release;
// the callback registry itself is safe for concurrent native callbacks. Never
// invoke a blocking SDK method or Release from inside a native event handler.
//
// Callback payloads are deep-copied into Go-owned values and may be retained
// after the callback. RtmConfig.EventHandler is copied when creating a client;
// changing its function fields afterward does not change registered callbacks.
//
// Some native lock, storage and stream APIs return void. Their Go wrappers return
// zero only when a request was submitted locally, or a negative local validation
// error. The corresponding asynchronous callback is the authoritative operation
// result. Do not interpret submission as successful server-side completion.
package agorartm
