# RTC usage

[简体中文](rtc.zh-CN.md) · [Home](../README.md)

## Ownership and defaults

Create one `Service` per process. Call `Service.Join` for each channel or identity; tokens belong to individual sessions. `Service.Close` closes its sessions and audio processors before releasing the native runtime. Both service and session cleanup are idempotent.

`Join(ctx, config)` waits for connection success or an error, with a default 30-second timeout. Cancelling that context after a successful join does not close the session: call `session.Close()` explicitly. Use separate contexts for individual reads and operations.

Defaults are broadcaster mode, 16 kHz mono PCM16, the AI-server audio scenario, automatic audio subscriptions, and per-user before-mixing audio observation. Video reception is opt-in.

## Send and receive audio

```go
// File or other promptly returning reader; PCM16, 16 kHz, mono by default.
if err := session.PlayPCM(ctx, reader); err != nil {
    return err
}

// Live audio already paced by the producer: one or more complete 10 ms frames.
if err := session.SendPCM(pcm); err != nil {
    return err
}

frame, err := session.Audio().Next(ctx)
if err != nil {
    return err
}
// Use frame.UserID, frame.Data, frame.SampleRate and frame.Channels.
```

- `PlayPCM` uses a bounded buffer and paces at 10 ms intervals. It pads a partial final frame with silence; an incomplete PCM sample is an error. Context cancellation cannot interrupt a blocked `io.Reader`.
- `SendPCM` does not pace or resample input. At the default format, a 10 ms frame is 320 bytes. Configure `ChannelConfig.Audio` for another supported format.
- `SendPCMAt` accepts a presentation timestamp; `PlayPCMAt` paces a reader from an explicit initial presentation timestamp. Keep related audio and video timestamps in a consistent timebase.
- `InterruptAudio` requests native audio interruption. `WaitPCM` estimates completion from the native duration clock and margin; it is not a remote playback acknowledgment or native queue-drained guarantee.
- `AudioFrame.Data` and `Speech` are copied out of native memory and may be retained by the reader.

### VAD and observation points

Set `AudioConfig{VAD: true}` for native VAD v2 on 16 kHz mono received audio. Optional `VADConfig` uses native configuration types. Read `AudioFrame.VAD` and `Speech`; VAD output is not speech transcription.

`Receive.AudioPositions` is a bitmask of `AudioPlayback`, `AudioRecord`, `AudioMixed`, `AudioBeforeMixing`, and `AudioEarMonitoring`. Zero selects before-mixing, which includes the remote user ID. Other positions may not identify an individual user. Availability follows native SDK semantics and the active media path.

### Encoded audio

Select `AudioConfig{Mode: agorasdk.EncodedAudio}` and send `EncodedAudioFrame` through `SendEncodedAudio`. Provide a complete codec packet plus its actual format and timing. This API accepts Opus and AAC; it is not a container demuxer. To receive encoded packets, enable `Receive.EncodedAudio` and read `session.EncodedAudio().Next(ctx)`.

## Video

Enable `Config.Video` before creating the service. Select a sending mode in `ChannelConfig.Video` and enable the desired receive streams separately:

```go
cfg := agorasdk.ChannelConfig{
    Channel: channel, UserID: userID, Token: token,
    Video: agorasdk.VideoConfig{Mode: agorasdk.RawVideo},
    Receive: agorasdk.ReceiveConfig{Video: true},
}
```

- `SendVideo(VideoFrame)` accepts tightly packed I420, NV12, RGBA or BGRA CPU frames with explicit dimensions. I420/NV12 dimensions must be even.
- Received raw frames expose Y/U/V planes with individual strides; do not assume tightly packed output.
- For H.264/H.265 packets, select `EncodedVideo`, call `SendEncodedVideo`, and provide frame boundaries, dimensions, timestamps and key-frame information. A whole MP4 file is not one encoded frame.
- Enable `Receive.EncodedVideo` to receive encoded frames. `RequestKeyFrame`, `SetVideoEncoder` and `SetSimulcast` expose common controls.
- SEI metadata has a 1,024-byte payload limit in the high-level send API. Codec capabilities depend on the installed native distribution.

## Subscriptions

Default subscriptions are automatic. Set `Receive.ManualSubscriptions` to control individual users with `SubscribeAudio`, `UnsubscribeAudio`, `SubscribeVideo` and `UnsubscribeVideo`. Video subscriptions accept `VideoSubscription{LowQuality, EncodedOnly}`.

Large-scale channels require `Mode: agorasdk.LargeScale` and manual subscriptions. Subscribe to individual user IDs; the subscribe-all helpers are not supported in that mode. An audience session can receive but cannot publish.

## Events, tokens and messages

Consume `session.Events().Next(ctx)` in a dedicated application loop. Event kinds cover connection state, users, publication/subscription, errors, volume, statistics, metadata and token expiration. Enable volume events with `SetVolumeIndication`.

On `TokenExpiring` or `TokenExpired`, obtain a fresh channel token from your server and call `RenewToken`. The session does not mint tokens or retrieve secrets. The separate pure-Go `token.RTC` helper can generate RTC AccessToken2 values on a trusted server; keep its App Certificate server-side. Monitor connection state as well as event queue drop counts.

`SendMessage` sends a 1–1,024-byte RTC data-stream message; read it through `Messages().Next(ctx)`. This stream is configured as unordered and unreliable. Use the separate [RTM API](../messaging/README.md) when you need application messaging semantics beyond that transport.

`SendAudioMetadata` sends 1–1,024 bytes; incoming audio metadata appears as an `AudioMetadata` event.

## Queue behavior

Each RTC receive stream has a fixed capacity, default 64. Set `ChannelConfig.QueueCapacity` to change it. A full queue drops its **oldest** item to keep the native callback nonblocking. Inspect `stream.Stats()` for capacity, buffered count, dropped count and closed state.

`Next(ctx)` waits for one item. After closure it drains buffered items, then returns `io.EOF`. Multiple readers compete for items; a stream is not a broadcast bus. Fan out explicitly in your application when several consumers need the same frames. RTM has its own [queue contract](../messaging/README.md).

## External audio processing and decoding

`Config.AudioProcessing: true` enables creation of owned processors with `Service.NewAudioProcessor`. Feed PCM with `processor.SendPCM` and consume `processor.Output().Next(ctx)`. Set `ProcessorConfig.APM` and optional VAD settings as needed.

The native package must contain the requested APM extensions. The pinned Linux distribution does not include the AINS extension; do not assume AI noise suppression is available. Missing extensions may cause a native initialization error. Validate the actual processor on your target distribution before deployment.

`TranscodeVideo` (H.264/H.265, 8-bit 4:2:0 Annex-B) and `TranscodeOpus` are optional local decoding helpers. Default builds return `ErrUnsupported`. Enable `-tags avcodec` and install FFmpeg development dependencies to use them; this does not change native RTC codec availability.

## Advanced configuration

Use `Config.ConfigureNative` for process initialization settings and `ChannelConfig.ConfigureNative` for connection/publishing settings before allocation. Native configuration pointers must not escape those callbacks.

`session.WithNative(func(*rtc.RtcConnection) error)` serializes advanced access with session operations. Do not call session methods recursively from that callback, close/release the borrowed object, retain it for later use, or replace observers owned by the session.

The `agora/rtc` package exposes lower-level bindings. Do not mix raw global initialization/release with an active high-level `Service`. Native-only APIs may require extra sequencing and resources beyond the high-level ownership guarantees.

## Errors

Use `errors.Is` for sentinel errors such as `ErrClosed` and `ErrUnsupported`, and `errors.As` with `*agorasdk.Error` to inspect operation and native error code. A successful send means local SDK acceptance, not remote receipt or playback.
