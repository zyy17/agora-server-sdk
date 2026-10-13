# Official Go API coverage

[简体中文](api-coverage.zh-CN.md) · [Home](../README.md) · [Comparison](upstream-comparison.md)

## Baseline and meaning

The reference is official **Go v2.6.5**, commit [`d51ff1b5baf85dd53155b31543bceba8ec304dc7`](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/tree/d51ff1b5baf85dd53155b31543bceba8ec304dc7). This is a separate module with deliberate source-compatibility changes, not a drop-in replacement or a claim of “100% functionality.”

The checked-in [reference](api-reference/official-go.tsv) contains **1,159 declarations from 40 SHA-verified official Go files**:

| Official surface | RTC | RTM |
| --- | ---: | ---: |
| Functions | 65 | 46 |
| Methods | 118 | 97 |
| Exported struct fields, including callbacks | 336 | 172 |
| Types | 104 | 51 |
| Constants | 130 | 37 |
| Variables | 3 | 0 |

All **326 function/method names** remain. This says nothing by itself about correct parameters, callback delivery, native marshalling, feature entitlement, or high-level access. The gate separately checks signatures and all exported fields, with **33 exact compatibility exceptions** described below. Parameter names and private struct layouts are excluded; parameter types/order/count, pointer receivers, callback results, exported field types, aliases, and constant expressions are retained. All build variants are inspected; duplicate equivalent declarations count once.

## Coverage classes

The machine-readable [coverage map](api-reference/coverage.json) links each capability to official declarations, current public APIs/configuration/events, named regression tests, and limitations. Every official callable has a classification; unwrapped constructors/converters/resource methods remain explicitly `raw-only`.

| Class | Contract |
| --- | --- |
| `high-level` | A named owned API/configuration/event covers the bounded behavior described in the record |
| `managed-advanced` | Configuration or borrowed native access, subject to ownership and combination restrictions |
| `raw-only` | Retained low-level binding; manual lifetime/concurrency and native return codes |
| `safe-compatibility` | An intentional, tested difference from unsafe or incomplete upstream signatures |
| `native-unavailable` | Missing from this pinned Go/C boundary or conditional on an unavailable extension/platform |
| `planned` | Not integrated; never counted as implemented coverage |

### Current application-facing paths

| Capability | Public entry points | Evidence / limit |
| --- | --- | --- |
| RTC lifetime and identity | `NewService`, `Service.Join`, `Session.Close`, `Config.IdentityMode` | [Join/close tests](../session_test.go), [identity tests](../identity_test.go) |
| RTC advanced configuration | `ConfigureNative`, `Session.WithNative`, service/session `SetParameters` | [Configuration tests](../session_config_test.go); borrowed handles cannot be retained or released |
| PCM, encoded audio, pacing and interruption | `AudioConfig.Send/Receive`, `SendPCMAt`, `PlayPCM`, `SendEncodedAudio`, `InterruptAudio`, `WaitPCM` | [Audio tests](../audio_test.go); `WaitPCM` is an estimate, not remote playback acknowledgement |
| CPU/encoded video and subscriptions | `SendVideo`, `SendEncodedVideo`, receive streams, publish/subscribe methods | [Validation](../config_test.go), [copied callbacks](../session_test.go); no hardware-frame guarantee |
| RTC data and events | `SendMessage`, `Messages`, `Events` | [Callback tests](../session_test.go), [bounded queue tests](../stream_test.go) |
| External audio processing | `Service.NewAudioProcessor`, `SendPCM`, `Output` | [Processor tests](../processor_test.go); extension availability still applies |
| RTM lifecycle, messages and result correlation | `messaging.New`, login/logout/renew, publish/subscribe | [Request tests](../messaging/client_test.go); cancellation does not roll back native actions |
| RTM Presence, metadata, locks, history | `OnlineUsers`, `UserChannels`, state/metadata/lock methods, `History` | [Result/copy tests](../messaging/client_test.go), [history validation](../messaging/history_validation_test.go) |
| RTM Topics | `NewStream`, join/leave/renew, topic publish/subscribe | [Partial-result and ownership tests](../messaging/client_test.go); Stream Channel tokens are RTC tokens |
| RTM events and native access | `Events`, `Next`, `Config.Native`, `WithNative` | [Event/copy tests](../messaging/client_test.go); owned handlers cannot be replaced |

The exact symbol/test links and narrower semantics live in the coverage map, rather than inferring feature coverage from a table row. The map currently keeps expanded RTC control/media and RTM option/result contracts `planned` until their implementation and tests are integrated. Update those records only after checking the final source.

## Deliberate compatibility changes

The gate permits only these exact departures; further changes fail instead of inheriting a broad exclusion:

- RTC: remove three debug file globals; remove `RemoteAudioTrackStats.MosValue`, `PublishDuration`, and `TotalActiveTime`, which do not exist in the pinned statistics struct; use the correct C struct in `GoRemoteAudioStats`; make `VideoFrameObserver.OnFrame` return void to match the C ABI. See [typed observer registration](../agora/rtc/observer_bindings.c), [synthetic ABI tests](../agora/rtc/callback_abi_test.go), and [VAD tests](../agora/rtc/audio_vad_test.go).
- RTM: remove legacy `OnConnectionStateChanged`, absent from pinned `C_IRtmEventHandler`; `OnLinkStateEvent` remains available. Five result callbacks now return copied slices so every native array element survives. Nineteen lock/storage/topic methods preserve immediate status or request IDs needed for error reporting and correlation. See [binding tests](../agora/rtm/bindings_test.go) and the exact replacements in the coverage map.

These are intentional API changes. Applications using the official signatures must migrate; retained names do not promise source compatibility.

## Boundaries and remaining gaps

- Native media-player, RTMP publishing, and device APIs are **outside this official Go baseline**. Their existence in another Agora SDK does not make them missing official-Go functions here.
- GPU textures, shared/native hardware buffers, and platform image handles need supported native marshalling and lifetime rules. A raw Go field or enum value is not evidence that the high-level sender supports them.
- The pinned RTM C bridge lacks C++-only stream parameters and several event timestamp/occupancy fields. Do not fabricate these values or call them lost Go functionality.
- Native codecs/extensions, platform support, peer compatibility, project entitlements, network behavior, and live media quality remain separate checks. The pinned Linux archive lacks AINS; optional FFmpeg paths require their own build/runtime setup.
- Lifecycle, callbacks, and copied queues intentionally differ from direct observer ownership. Do not retain borrowed native handles, replace owned observers, release owned objects, or reenter the wrapper from an advanced-access callback.

## Offline regression gate

```sh
make check-api-coverage
```

This standard-library-only Go AST check runs with `CGO_ENABLED=0`, no native libraries, no credentials, and no network. It is also part of `make check`. It checks the pinned signatures, exact compatibility replacements/removals, complete callable classification, public evidence links, and named test existence. Its own mutation tests cover callback/result changes, pointer/arity preservation, missing evidence, conflicting build variants, removed unsafe fields, and moving provenance.

The coverage-map keys are `package:kind:name`; an empty exception replacement means deliberate removal. AST fingerprints do not evaluate C enum values, defaults or method bodies. A green gate does **not** run the linked tests or prove native marshalling or behavior. Run `make check`, `make test-callbacks`, optional decoder checks where relevant, and the [live integration checklist](development.md#live-integration). Go's race detector does not instrument native internals.

To regenerate the reference intentionally, use an unmodified checkout at the pinned official commit:

```sh
go run ./internal/cmd/apicoverage -snapshot /path/to/official-checkout
```

The generator verifies each source against its Git blob ID, records source paths/hashes, and refuses a different commit or altered source. A connector-materialized source tree can instead provide `source-manifest.json` with the same repository/ref/tag and per-file Git IDs; generation verifies the local bytes, while the manifest itself must come from a trusted retrieval of that exact commit. Ordinary checks use only the small checked-in reference, not an upstream checkout or a copy of upstream implementation bodies. Never regenerate from this repository to make a regression pass.
