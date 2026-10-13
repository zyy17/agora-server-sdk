// Package agorasdk provides owned, context-aware RTC sessions on the Agora
// Server SDK. It supports server-side PCM and encoded audio, raw and encoded
// video, subscription control, RTC data streams, metadata, and copied events.
//
// # Runtime ownership
//
// Create one Service with NewService and close it when finished. A Service owns
// every Session and AudioProcessor created through it. Join waits for success,
// failure, cancellation, or ChannelConfig.JoinTimeout, and cleans up failed
// connections. After Join succeeds, its context no longer controls the session:
// call Session.Close explicitly. Service.Close also closes remaining children.
// Do not combine this ownership model with direct native initialization/release.
//
// Session native calls are serialized against Close. This does not make a whole
// playback loop atomic: use one producer per media track, and synchronize your
// own input buffers, readers, and shared received values. Close waits for native
// cleanup and is not bounded by a context. Session.Done closes when cleanup
// begins, not when it has finished.
//
// # Identity and authentication
//
// Config.IdentityMode applies to the whole Service. StringIdentity, the default,
// uses string accounts even when they contain only digits. NumericIdentity uses
// canonical decimal uint32 UIDs; local "0" requests allocation, while remote
// user IDs must be nonzero. These namespaces are not interchangeable.
// ConnectionInfo reports the assigned identity after joining.
//
// Supply a token matching the project, channel, identity namespace, and required
// privileges. Empty tokens work only for unsecured projects. Joining does not
// need an App Certificate. Consume TokenExpiring and TokenExpired events and
// obtain replacements from a trusted server. The token subpackage optionally
// builds RTC-only string-account tokens; it is not an RTM or combined-token
// generator. Never embed App Certificates in untrusted clients or log tokens.
//
// # Audio and timing
//
// PCM is signed interleaved PCM16 little-endian. AudioConfig.Send and Receive are
// independent and each defaults to 16000 Hz mono. Supported PCM rates are 8000,
// 16000, 32000, 44100, and 48000 Hz with one or two channels. Encoded-audio rates
// follow EncodedAudioFrame instead. The wrapper does not resample PCM for you.
//
// SendPCM and SendPCMAt publish on first use and submit complete, nonempty,
// integral-millisecond chunks (10 ms multiples at 44100 Hz). They do not pace
// calls. PlayPCM paces an io.Reader in 10 ms frames and pads a partial final frame
// containing complete samples. A blocked reader must be unblocked by the caller;
// context cancellation cannot interrupt an arbitrary Read.
//
// SendPCMAt preserves its presentation value exactly, including opaque tags.
// PlayPCMAt increments a nonzero presentation value by 10 per frame and keeps
// zero unchanged. Native render/capture timestamps need not be Unix time.
// WaitPCM is a duration estimate, not a remote-playback acknowledgement.
// To interrupt playback, cancel producers, discard application queues, and call
// InterruptAudio; audio already delivered to peers cannot be recalled.
//
// AudioConfig.VAD enables the SDK VAD v2 segmenter at AudioBeforeMixing and
// requires 16 kHz mono reception. AudioFrame.Speech may contain pre-roll and
// continued speech, so it is not solely a speech-onset signal. The standalone
// native detector in agora/rtc is a separate API that accepts exactly one
// 10 ms, 16 kHz mono frame per call. AudioProcessor instead accepts nonempty
// multiples of 10 ms and requires installed native APM extensions.
//
// # Receiving and backpressure
//
// Audio, Video, EncodedAudio, EncodedVideo, Messages, and Events each return an
// independent bounded Stream. QueueCapacity defaults to 64 items, not bytes.
// Native callback payloads are copied into Go memory. A full RTC queue drops
// its oldest item rather than blocking the callback; inspect Stream.Stats.
// Multiple Next readers divide items. Fan out in the application when every
// consumer needs a copy. These queues are not lossless logs.
//
// With a live context, Stream.Next drains buffered values after closure and
// then returns io.EOF. An already canceled context returns its context error.
// Event.Data uses the types documented on Event; use checked type assertions.
// RTC messages and metadata follow native limits. The messaging subpackage
// provides independent RTM clients with different request and overflow semantics.
//
// # Errors and native access
//
// Use errors.Is for ErrClosed, ErrInvalidConfig, ErrUnsupported,
// ErrNotPublishing, and context errors. Use errors.As to inspect *Error and
// its native code. A successful send normally reports local native acceptance,
// not delivery, decoding, or recipient processing; monitor Events as well.
//
// Config.ConfigureNative and ChannelConfig.ConfigureNative run synchronously
// before allocation/initialization and must not retain their pointers.
// Session.WithNative lends a connection under its operation lock. Do not retain
// handles, reenter Session methods, destroy resources, or replace owned observers.
// Raw APIs keep their native validation, lifetime, and result-code contracts.
//
// # Build and deployment
//
// This package requires cgo, a C/C++ toolchain, the repository's selected Go
// toolchain, and the pinned Agora headers and shared libraries. Linux amd64 is
// the primary supported deployment target; the distributed macOS libraries are
// for development/testing. Deployments must provide shared libraries and loader
// paths; a built binary is not standalone. Optional TranscodeVideo and
// TranscodeOpus decoding requires the avcodec build tag and FFmpeg libraries;
// otherwise those methods return ErrUnsupported.
//
// Package comments, exported declaration comments, and Example functions are
// the API reference. Run go doc from the intended checkout to read that version;
// a public package index may not yet contain the current development branch.
// Offline tests and examples do not prove live authentication, media
// interoperability, or feature availability on a particular native platform.
package agorasdk
