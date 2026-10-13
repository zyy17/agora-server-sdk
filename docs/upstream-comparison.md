# Comparison with the official Agora Go Server SDK

[简体中文](upstream-comparison.zh-CN.md) · [Home](../README.md) · [RTC guide](rtc.md) · [RTM guide](../messaging/README.md)

This repository builds an owned, context-aware Go API on modified Agora Go bindings. Its main differences are application-facing API contracts, lifecycle management, validation, callback delivery, binding fixes, and development tooling. Audio/video transport, codecs, network behavior, and RTM services still come from Agora. This is a separate module, not an import-compatible replacement for the official SDK.

## Comparison baseline

“Official” below means **AgoraIO-Extensions/Agora-Golang-Server-SDK v2.6.5**, commit [`d51ff1b5baf85dd53155b31543bceba8ec304dc7`](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/d51ff1b5baf85dd53155b31543bceba8ec304dc7), rather than the latest moving branch. Local source links describe the checkout containing this guide.

The native archives match that [upstream installer's][up-install] selections: Linux amd64 **4.6.31.134** and macOS amd64/arm64 **4.6.3.133-20260618**. URLs and SHA-256 values live in [`scripts/sdk_versions.sh`](../scripts/sdk_versions.sh). The Go binding release, RTC archive version, and RTM runtime version are separate; use `messaging.Version()` to inspect the linked RTM runtime.

## At a glance

| Area | Official v2.6.5 | This repository |
| --- | --- | --- |
| RTC entry point | Process-level `Initialize`/`Release`, `RtcConnection`, observers | `Service`, `Session`, and typed receive `Stream[T]` |
| Completion and errors | Native return codes and result callbacks | Bounded join/request waits, contexts, sentinel and typed Go errors |
| Receive delivery | Application observer callbacks | Copied values in bounded queues with explicit drop counters |
| Media | PCM/encoded audio, raw/encoded video, VAD, interruption, APM, optional decoding | Validated media methods, separate send/receive configuration, paced PCM reader helper |
| RTM | Client, Presence, storage, locks, history, Stream Channel bindings | Owned clients/streams, request correlation, Go results and event queue |
| Native access | Binding-level API | Modified raw bindings plus constrained configuration/access hooks |
| Integration | Upstream installer, examples and README | Checksum-pinned installer, environment helper, offline checks, GoDoc and bilingual guides |

Multiple connections, per-user audio, VAD, interruption, encoded-media observation, and RTM features already exist upstream. Even receive copying is not entirely new: [upstream's PCM/video conversions][up-convert] already use `C.GoBytes`. This repository adds a consistent high-level ownership and queue contract around those capabilities.

## Lifecycle and concrete API flow

The official [connection implementation][up-connection] already creates and releases configured tracks and senders. Applications still arrange initialization, observers, asynchronous connection completion, disconnection, and final release. Its [Chinese README][up-zh] specifies one service per process and prohibits SDK calls or expensive work inside callbacks.

For the same receive-audio task, the source-level flow changes as follows. These are API outlines, not complete programs:

| Step | Official binding | High-level wrapper |
| --- | --- | --- |
| Initialize | `NewAgoraServiceConfig`, `Initialize` | `NewService(Config)` |
| Configure/connect | `NewRtcConnection`, `RegisterObserver`, `RegisterAudioFrameObserver`, `Connect` | `service.Join(ctx, ChannelConfig)` configures owned observers and waits for connection |
| Consume PCM | `OnPlaybackAudioFrameBeforeMixing` callback | `session.Audio().Next(ctx)` in application code |
| Finish | `Disconnect`, connection `Release`, process `Release` | `session.Close()`, `service.Close()` |

See the [complete wrapper example](../README.md#quick-start) and the official implementation for configuration details; `Connect` returning zero alone is not an `OnConnected` notification.

[`Service`](../service.go) enforces a single active high-level RTC runtime with `ErrServiceActive`, owns its sessions and audio processors, and closes children before releasing the runtime. [`Session`](../session.go) owns its connection, tracks, observers, and queues. Close methods are idempotent. `IdleMode` must remain false so the wrapper does not select upstream's delayed connection destruction. This ownership enforcement covers the high-level API; mixing raw `Initialize`/`Release` with a live service violates the contract.

## Contexts, errors, and receive queues

- `Join` waits for success, failure, caller cancellation, or `JoinTimeout` (default 30 seconds), and closes an unsuccessful session. Canceling the context after a successful join does not close that session.
- `Next(ctx)` and `PlayPCM(ctx, reader)` provide cancellable waits. A context cannot interrupt a blocking reader or a synchronous native call already running. Session operations serialize native access; media producer ordering remains the application's responsibility.
- Use `errors.Is` for sentinel/context errors and `errors.As` for `*agorasdk.Error` or `*messaging.Error`; native numeric codes remain available. Local validation and successful native submission do not prove remote delivery or playback.

The six RTC queues (`Audio`, `Video`, `EncodedAudio`, `EncodedVideo`, `Messages`, `Events`) default to **64 items each**. A full queue drops the **oldest** item; `Stats()` reports drops. With an active context, closing drains buffered values before `io.EOF`. Multiple readers compete for items rather than receiving a broadcast copy.

RTM defaults to **128 events**, drops the **newest** event, and reports `DroppedEvents()`. Its `Next` returns `messaging.ErrClosed` after the buffer drains. RTM request completions use a separate correlation path, not the droppable event queue. Both APIs copy high-level receive payloads; neither is a lossless log or a byte-bounded memory budget. Applications must consume token/network events and plan for overload. See [`stream.go`](../stream.go) and [`messaging/client.go`](../messaging/client.go).

## Media and pacing

The wrapper exposes combinations of existing native capabilities: independent `Audio.Send` and `Audio.Receive` PCM formats, independent publication/reception choices, per-user PCM/VAD, encoded reception, manual subscriptions, and separate audio processors. For example, 16 kHz mono receive with 16 kHz stereo send is configuration of existing media paths, not a new native codec or resampler.

`SendPCM`/`SendPCMAt` accept complete integral-millisecond PCM16 chunks; at 44.1 kHz that means multiples of 10 ms. They do not pace, pad, or resample. Upstream already accepts variable-length PCM; this binding fixes its integer-kHz sample calculation for 44.1 kHz and preserves the supplied presentation value. `PlayPCM` adds a cancellable 10 ms software schedule over an `io.Reader`, with silence padding for a final complete-but-short frame. `WaitPCM` wraps a sent-byte/wall-clock estimate, not a native drain or remote-playback acknowledgment. Cancel the producer before interruption, otherwise later sends can refill the sender.

Encoded audio/video methods expect codec packets or frames, not arbitrary container bytes, and do not pace input. High-level encoded video accepts H.264/H.265; this does not cover every native codec or imply H.265 encoding support. Upstream lists H.265 as decode-only. Raw video has explicit layout/size validation, while native codec/platform limits still apply.

Received-audio VAD v2, standalone native mono/stereo VAD, and external APM are distinct paths. VAD is not speech recognition. External APM needs its native extensions; the pinned Linux archive lacks AINS, so a default processor can fail initialization. Optional `avcodec` helpers decode and publish asynchronously; successful input acceptance is not completion, and close does not guarantee draining. Full contracts are in the [RTC guide](rtc.md).

## Identities and tokens

The [upstream service configuration][up-service] exposes `UseStringUid` and defaults it to false. The wrapper defaults to **string accounts**, with explicit service-wide `NumericIdentity` as an alternative. Numeric IDs are canonical decimal strings representing uint32 values; `"0"` requests native allocation. An all-digit string account does not automatically become a numeric UID. Tokens and peers must use the selected identity mode.

Joining accepts an externally supplied RTC token and needs no App Certificate. Applications handle expiry events and call `RenewToken`; the wrapper does not fetch or automatically refresh tokens. The optional [`token.RTC`](../token/token.go) helper uses the official AccessToken2 builder for **string-account RTC tokens only**, on trusted servers. Keep the official UID or combined RTC+RTM token flow when needed; this helper does not replace it. RTM identity/token configuration remains separate. String-account length compatibility also differs across documented versions; see the [identity contract](rtc.md#configuration-and-validation) before migrating IDs.

## RTM facilities and changed defaults

The official [RTM bindings][up-rtm] already expose messaging, Presence, metadata storage, locks, history, and Stream Topics. [`messaging`](../messaging/README.md) adds independently owned clients, stream ownership, typed Go results, and request-ID correlation, including callbacks that arrive before initiation returns. Asynchronous methods wait for their result or the earlier of the caller deadline and a default **15-second** request timeout.

Cancellation stops waiting; it does not roll back a submitted publish, metadata write, or lock acquisition. Query state or use application idempotency before retrying. `Close` fails pending waits and releases streams before the client, but does not await graceful `Logout` or `Leave`.

Review the wrapper's deliberate defaults: `SubscribeOptions{}` enables messages but not Presence/metadata/lock events; automatic native lock-acquisition retry is disabled; metadata revision zero is forwarded unchanged rather than converted to unconditional `-1`. History requires project access and publication with history enabled. RTM Stream Channel join/renewal uses an RTC token. Here `messaging.Stream` means an owned RTM Stream Channel; RTC `Stream[T]` means a local receive queue. RTC data streams and RTM are separate transports, and neither promises recipient application processing.

## Advanced access and coverage limits

`Config.ConfigureNative` and `ChannelConfig.ConfigureNative` expose pre-initialization/allocation configuration. `Service.SetParameters`, `Session.WithNative`, and RTM `WithNative` provide controlled advanced access. These hooks do not supply new native capabilities or guarantee that every option composes with high-level defaults.

Do not retain borrowed pointers, release owned objects, replace owned observers/event handlers, or reenter the high-level object from its `WithNative` callback. Direct `agora/rtc` and `agora/rtm` usage retains raw return codes and requires manual lifecycle/concurrency discipline. The high-level API is intentionally narrower than the full binding surface and is not a drop-in compatibility layer.

## Binding fixes in this repository

These are changes relative to the pinned source, not claims that every upstream version has the same issues:

- **Callback ABI:** align remote-statistics callbacks with opaque native track handles and the correct statistics structures; resolve track handles to public user IDs; make raw-video callback return types match the headers. Typed C observer registration turns signature drift into a compilation error. See [`observer_bindings.c`](../agora/rtc/observer_bindings.c), [`remote_track_registry.go`](../agora/rtc/remote_track_registry.go), and [synthetic C callback tests](../agora/rtc/callback_abi_test.go), compared with [upstream callback declarations][up-callbacks] and [video declarations][up-video-callbacks].
- **Callback and resource lifetime:** drain active RTC callbacks before observer/lifecycle teardown; synchronize remote-track identity bookkeeping; use an RTM handler registry so native code does not retain pointers to Go pointer-containing objects. See [`callback_gate.go`](../agora/rtc/callback_gate.go) and [`callback_registry.go`](../agora/rtm/callback_registry.go).
- **PCM and VAD:** validate whole samples without 44.1 kHz truncation; copy VAD v2 configuration before normalization so one user's setup does not mutate another's; validate standalone detector input as exactly 10 ms of 16 kHz PCM16. Standalone VAD processing/release is serialized, and a C++ exception is contained as `-2002` with the affected detector retired. See [PCM tests](../agora/rtc/pcm_frame_test.go), [VAD v2 tests](../agora/rtc/audio_vad2_test.go), [standalone VAD tests](../agora/rtc/audio_vad_test.go), and the [exception bridge](../agora/rtc/audio_vad_bridge.cc).

These are targeted safeguards with regression tests, not proof of absence of native bugs, a complete security audit, or performance superiority.

## Dependencies, validation, documentation, and licensing

Both implementations' RTC/RTM integration uses cgo and native libraries; this repository's `mediaio` and `token` helpers are pure Go. This repository pins **Go 1.27.2**; [upstream's README][up-en] requires **Go 1.21+** even though its [module file][up-module] says `go 1.20`. The wrapper installer validates SHA-256 and archive contents before replacing an installation, and `with_agora_env.sh` sets build/runtime paths for the selected installation. Deploy the shared libraries with RTC/RTM applications too; a successful Go build does not produce a standalone RTC/RTM binary.

Linux amd64 is the primary build/CI target. The installer also handles macOS amd64/arm64, but upstream restricts macOS to development/testing. Linux arm64 and Windows are unsupported by this distribution. Neither installation support nor Linux tests establish macOS production readiness.

Upstream already makes decoder support optional with `avcodec`. This repository also keeps default builds free of **system FFmpeg development-library** requirements and verifies that in CI; optional decoder tests run separately. The `send_media` example instead uses the FFmpeg executable at runtime. This does not mean the Agora native archive contains no FFmpeg-related components.

[`make check`](../Makefile) covers formatting, tooling fixtures, first-party Staticcheck, all-package vet/tests/race checks, example builds, credential-free native lifecycle, and synthetic registered C callbacks with strict cgo checks. The Go race detector does not instrument native internals. These checks do not establish live migration success, remote media quality, authentication correctness, or feature entitlement; run the [live integration checklist](development.md#live-integration) for the deployment environment.

Standard package/symbol GoDoc documents API contracts; bilingual [RTC](rtc.md), [RTM](../messaging/README.md), [example](../examples/README.md), and [development](development.md) guides explain workflows. The official [English][up-en] and [Chinese][up-zh] READMEs remain useful for upstream behavior. Online [Chinese Go API documentation](https://doc.shengwang.cn/api-ref/rtc-server-sdk/go/go-api/overview) is supporting context, not a substitute for the pinned source and headers.

The repository's Go wrapper is distributed under its [MIT license](../License). That does not relicense Agora's proprietary native SDK, vendored materials, or optional third-party codecs; retain their notices and comply with their separate terms. This repository does not claim official Agora maintenance or endorsement.

## Before migrating

1. Replace imports and callback wiring deliberately; map ownership and shutdown to `Service`, `Session`, or independent RTM clients.
2. Set identity mode, role, media formats, subscriptions, and token privileges explicitly instead of assuming upstream defaults are identical.
3. Account for queue drops, asynchronous errors, cancellation without rollback, and application acknowledgments where needed.
4. Verify every required raw option, codec/extension, RTM feature, and platform against the pinned distribution; compile and run two-party integration tests.

Use the high-level API when its lifecycle, bounded queues, and context-based waits fit the application. Keep direct official/raw bindings when the application needs a different callback model or native surface, and accept the associated ownership and concurrency responsibilities.

[up-connection]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/rtc_connection.go
[up-rtm]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtm
[up-en]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/README.md
[up-zh]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/README.zh.md
[up-callbacks]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/rtc_callbacks_cgo.h
[up-video-callbacks]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/video_observer_cgo.h
[up-install]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/scripts/install_agora_sdk.sh
[up-convert]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/type_convert.go
[up-service]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go_sdk/rtc/agora_service.go
[up-module]: https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/blob/d51ff1b5baf85dd53155b31543bceba8ec304dc7/go.mod
