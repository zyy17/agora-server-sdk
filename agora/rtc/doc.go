// Package agoraservice exposes the raw Agora RTC server SDK bindings.
// Its import path is github.com/zyy17/agora-server-sdk/agora/rtc; importing it
// as rtc is convenient because the historical package name differs from the
// final path element.
//
// These bindings follow Agora-Golang-Server-SDK v2.6.5, commit
// d51ff1b5baf85dd53155b31543bceba8ec304dc7, with local ownership and validation
// fixes. Native headers retain their original Agora copyright notices.
// The root agorasdk package is the preferred owned, context-aware API.
// Raw bindings expose native options and result codes without promising the
// root package's validation, cancellation, or queue contracts.
//
// # Lifetime and callbacks
//
// The native RTC service is process-global. Initialize it before constructing
// connections, media factories, tracks, senders, or processors, and release all
// dependent objects before releasing the service. Do not initialize or release
// it directly while a high-level agorasdk.Service owns the runtime.
// Raw resource types are not generally safe for arbitrary concurrent lifecycle
// changes. Serialize connection operations and coordinate each object's owner.
//
// Never synchronously release resources, replace observers, or perform other
// lifecycle operations from a native callback. Dispatch that work to another
// goroutine; release waits for in-flight callbacks and can otherwise deadlock.
// Callback media payloads are copied into Go memory, but native track handles
// and other borrowed handles retain their native lifetimes. Do not assume that
// all pointer-bearing callback arguments are independently owned.
//
// If a high-level Session lends a connection through WithNative, keep all
// borrowed objects inside that call. Do not release them, replace owned
// observers, or reenter the high-level session. Native calls do not acquire
// additional context cancellation or remote-delivery guarantees.
//
// # Audio and VAD
//
// PCM frame layouts, sample counts, and units must agree with the native call.
// AudioPcmDataSender sends synchronously without application pacing. Native
// render/capture timestamps are media clock values, not necessarily Unix time;
// presentation values may carry application-defined tags.
//
// AudioVad is a standalone native detector. ProcessPcmFrame accepts exactly one
// 10 ms, 16 kHz, mono PCM16 frame (160 samples, 320 bytes). SteroAudioVad uses two
// independent detectors for exactly 10 ms interleaved stereo (640 bytes).
// AudioVadV2 is a different, Go-side segmenter driven by SDK telemetry. Its
// Process method retains input frames and may return the original input pointer;
// it does not enforce the standalone detector's exact frame shape. Do not reuse
// those buffers while retained, and serialize Process with Release.
// The high-level SDK VAD path validates 16 kHz mono received PCM before using it.
//
// # Build and native limits
//
// cgo, compatible headers, and the pinned shared libraries are required.
// Linux amd64 is the primary deployment target; the distributed macOS libraries
// are intended for development/testing. Native feature support depends on the
// actual platform, extensions, peer SDK, and project settings, not just whether
// a Go method is present. Consult the matching pinned headers for result codes,
// units, ownership, and native rate limits.
//
// Optional FFmpeg decoding requires the avcodec build tag and development
// libraries. Ordinary PCM and encoded-media transport do not require FFmpeg.
// A successful asynchronous worker submission does not prove later decode,
// native send, or remote playback success.
package agoraservice
