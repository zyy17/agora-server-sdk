# Media API contracts

[中文](media-api.zh-CN.md) · [Coverage](upstream-comparison.md)

The native baseline is official Go SDK v2.6.5, commit `d51ff1b5baf85dd53155b31543bceba8ec304dc7`. These APIs expose its usable media options with explicit validation and ownership. Enum support does not guarantee a codec, extension, GPU path, or peer supports the feature.

## Sending and observing

- `SendPCM` / `SendPCMAt`: configured PCM16 format, integral-millisecond duration, optional signed presentation value.
- `SendPCMFrame`: explicit sample count, PCM16 width, unsigned 32-bit capture timestamp and signed presentation value. Its rate/channels must match `AudioConfig.Send`; observer telemetry is not transmitted. Complete sub-millisecond samples are accepted structurally, subject to native codec support.
- `SendEncodedAudio`: all named native codecs, `SendEvenIfEmpty`, and 3–8 channels for `OpusMultiChannel`. Other codecs accept mono/stereo. Zero capture time is rewritten to -1 by the connection binding. Compressed payloads are not parsed.
- `SendVideo`: compact CPU layouts. `SendCPUVideo` adds pixel stride, crop, color-space IDs, alpha placement/extraction and raw/array buffer type. Both use the raw sender's authoritative layout validator.
- `SendEncodedVideo`: VP8, H264, H265, GenericVideo, GenericH264, AV1, VP9 and JPEG metadata. `FrameType` preserves blank/key/delta/B/droppable/unknown; `TrackID` and numeric `UID` are forwarded. `FrameType` defaults to `VideoFrameAuto`: `KeyFrame=true` selects key, otherwise delta. Use `VideoFrameBlank` for an actual blank frame. Explicit classifications are unaffected by `KeyFrame=false`; true conflicts with every explicit non-key type, including blank. High-level frame-type values are not raw enum values; receive maps each pinned native type explicitly so retransmission preserves it. Optional SEI insertion supports H264/H265 only.
- `ReceivedVideoFrame` preserves native Type, AVSyncType, AlphaMode, texture ID and transformation matrix along with copied planes. Texture ID/matrix are descriptors only: the wrapper does not own GPU resources or expose a borrowed shared context. Encoded receive preserves the original frame type, track ID and UID.

Sends are synchronous, unpaced, and only report local native acceptance. Keep input slices unchanged until return. Received slices are Go-owned and survive callbacks and Close. A bounded Stream can drop old frames; inspect its statistics. Use one producer per track to preserve ordering.

### Padded CPU pixels

```go
frame := agora.CPUVideoFrame{
    Data: make([]byte, 644*480*4), Stride: 644, Height: 480,
    Format: agora.RGBA, CropRight: 4, FillAlphaBuffer: true,
}
err := session.SendCPUVideo(frame) // visible image: 640×480
```

Stride counts pixels, not bytes. Data includes every padded row; crop never reduces required allocation size. I420/NV12/NV21 use 1.5 bytes/pixel, I422 uses 2, I010 uses 3, RGBA/BGRA use 4. Planar chroma uses half the pixel stride. 4:2:0 dimensions/crops must be even; I422 requires even horizontal values. Explicit alpha occupies stride×height bytes, including padding; alpha extraction requires RGBA/BGRA and no explicit alpha slice.

The pinned Go sender does not marshal texture/EGL/CVPixelBuffer/D3D handles. Those paths are rejected, rather than appearing to work while dropping handles. The C header describes metadata for texture paths; forwarding CPU metadata is not a guarantee it reaches peers. I010 remains a native preview feature. Linux/macOS binary availability and native codec support still apply.

## Track controls

`SetAudioEnabled` / `SetVideoEnabled` toggle an existing track without publishing or unpublishing it. `SetAudioPublishVolume`, `SetAudioSendDelay`, `SetAudioBufferCapacity` and `ClearAudioBuffer` expose the connection-owned audio track. Calls share Session's lifecycle lock; missing native tracks return errors.

Delay is a duration truncated to milliseconds. Buffer capacity counts 10 ms frames. Enabled/delay/capacity native setters are void, so success confirms invocation only. Clearing only affects the local native queue. Stop producers first; received audio cannot be recalled. `InterruptAudio` retains its distinct AI-server unpublish/republish behavior.

## Processing and voice activity

```go
processor, err := service.NewAudioProcessor(agora.ProcessorConfig{
    SampleRate: 16000, Channels: 1,
    Input: agora.PCMFormat{SampleRate: 48000, Channels: 2},
    DisableAPM: true, VAD: &rtc.AudioVadConfigV2{},
})
```

The Service must enable `AudioProcessing`. `Input` sets SendPCM/SendPCMAt defaults; each zero field inherits output. `SendPCMFormat` accepts a per-call format and signed presentation value; `SendPCMFrame` forwards the equivalent fields from AudioFrame. Processor input requires complete 10 ms PCM16 multiples, 8000–48000 Hz divisible by 100, and one or two channels. Output format is independent. Zero per-call format selects 16 kHz mono, not the processor's configured input.

APM configuration is deeply copied. `DisableAPM` requires VAD and forbids explicit APM; it supports VAD-only processing without optional APM filters. Full APM requires the appropriate installed extensions and can fail initialization. `Stats` gives native submitted/processed durations, not a flush or playback acknowledgement. Close can discard pending work; Service.Close also closes its processors.

- `AudioVAD` / `StereoVAD`: standalone native classification, no Service/channel required. Process exactly 10 ms of 16 kHz PCM16 (320 mono or 640 stereo bytes). Both own their detectors, copy selected speech and serialize Process/Close. Stereo preserves the other channel's successful result when one channel fails.
- `TelemetryVAD`: Go SDK v2 segmenter driven by AudioFrame voice/RMS telemetry, not an independent PCM classifier. One instance per consistent stream; it copies retained input and output. Nil config uses upstream defaults; a zero config is different.
- Speech states: silent, start, speaking, end. Selected speech may include pre-roll or be empty. Standalone speech timestamps count selected output duration. Forward either full PCM or selected speech to a recognizer to avoid duplication.

## Optional decoding and transcoding

`NewVideoDecoder` and `NewOpusDecoder` require `-tags avcodec` and FFmpeg development/runtime libraries. Without that build feature they return `ErrUnsupported`. Decoders own resources, serialize Decode/Close, and return copied output without opening an RTC channel.

Video accepts H264/H265 Annex-B access units and produces compact 8-bit I420; optional even dimensions resize output. `ErrDecoderNeedsInput` means submit another unit. Only the latest picture from an access unit is returned; upstream exposes no flush/timestamp API. Codec switches discard buffered pictures.

Opus accepts raw mono/stereo packets, not Ogg. Its configuration supports independent output PCM rate/channels. Output can be empty while buffering; no resampler flush is exposed upstream. Both decoders discard pending output on Close.

Session `TranscodeVideo` and `TranscodeOpus` retain asynchronous worker behavior: acceptance does not report later decode/send failures. Video is H264/H265-only even though encoded transport accepts more codec IDs. Workers may skip/repeat video on a timer; Opus worker output is 48 kHz mono. Avoid mixing producers. Use synchronous decoders when the application needs direct results and control of sending/pacing.

## Offline verification

Unit tests cover format/overflow validation, metadata preservation, copied receive data, decoder feature gates, VAD ownership and Close races. `-tags agora_test` tests round-trip media structures through the pinned C ABI. Native smoke tests create processors/detectors and verify 48 kHz stereo→16 kHz mono output without credentials or channel joins. Optional decoder fixtures run in the avcodec CI job. These checks do not prove live two-peer interoperability or all platform/codec combinations.

## Synchronous observation decisions

`AudioConfig.ReceiveParameters` independently selects Playback, Record, Mixed and EarMonitoring PCM format/sample counts. Unspecified entries inherit the normal 10 ms read-only receive format; before-mixing stays read-only. `AudioConfig.Callbacks.OnFrame` can accept/reject synchronously. Only accepted in-place Data edits at a `ReadWrite` point affect native PCM and the queued snapshot. Rejected/read-only edits and changes to metadata/slice headers are ignored. Queue buffers are independent of callback buffers.

`VideoConfig.OnEncodedFrame` synchronously accepts/rejects encoded video; its edits never change native packets or queued snapshots. Both hook types still enqueue observations when rejected, contain panics as `CallbackPanic` plus rejection, and must return promptly. Do not perform blocking I/O, wait for events or call lifecycle-locking Session/Service methods (including Close/Send/WithNative/SetRole) from them. Hooks may run concurrently; synchronize application state.
