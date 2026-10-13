# Examples

[简体中文](README.zh-CN.md) · [Home](../README.md)

Build all examples without connecting to Agora:

```sh
make download-agora-libs
make build-examples
```

## RTC setup

Set `AGORA_APP_ID`, a unique `AGORA_USER_ID`, and `AGORA_TOKEN` if your project requires authentication. `AGORA_CHANNEL` defaults to `go-sdk-example`. Use a separate identity and matching token for each participant. Obtain tokens from your server; no App Certificate is needed by these examples.

Run from the repository root. The environment helper supplies native library paths. Ctrl+C closes the service and its sessions.

```sh
./scripts/with_agora_env.sh go run ./examples/recv_pcm -output recordings
./scripts/with_agora_env.sh go run ./examples/send_pcm -input examples/testdata/send_audio_16k_1ch.pcm
```

Run sender and receiver in separate terminals with different `AGORA_USER_ID` values. PCM is signed 16-bit little-endian, 16 kHz mono by default; both examples accept `-rate` and `-channels`. Received files use hex-encoded user IDs and append to existing files. Choose a new output directory for a fresh recording.

## Scenarios

| Directory | Scenario |
| --- | --- |
| [send_pcm](send_pcm) | Pace a PCM file and estimate send completion |
| [duplex_pcm](duplex_pcm/README.md) | Numeric UID, mono receive/stereo send, variable PCM chunks and caller PTS |
| [recv_pcm](recv_pcm) | Save per-user PCM without audio hardware |
| [loopback](loopback) | Return received PCM to the channel; avoid feedback loops |
| [ai_audio](ai_audio) | VAD, interruption, volume and token/connection events |
| [voice_agent](voice_agent/README.md) | Selected-peer streaming voice conversation with ElevenLabs STT/TTS and OpenAI |
| [messages](messages) | Send and receive RTC data-stream messages |
| [multi_sessions](multi_sessions) | Multiple receive-only sessions sharing one service |
| [send_media](send_media) | Decode a local MP4/media file to paced PCM and raw video with a shared timestamp base |
| [media](media) | Raw video, encoded audio/video and optional decode-and-publish |
| [process_audio](process_audio) | External audio processing without joining a channel |
| [external_vad](external_vad/README.md) | Standalone mono/stereo VAD on local 10 ms PCM frames |
| [messaging](messaging/README.md) | RTM pub/sub, presence, metadata, locks, history and topics |
| [rtc_rtm](rtc_rtm/README.md) | Numeric RTC and independent RTM identities with opaque binary notifications |

```sh
./scripts/with_agora_env.sh go run ./examples/ai_audio -reply response.pcm
./scripts/with_agora_env.sh go run ./examples/messages -send 'hello from Go'
./scripts/with_agora_env.sh go run ./examples/multi_sessions -count 2
```

`multi_sessions` uses IDs `${AGORA_USER_ID}-0`, `${AGORA_USER_ID}-1`, etc., and reads the corresponding `AGORA_TOKEN_0`, `AGORA_TOKEN_1`, etc. It does not reuse `AGORA_TOKEN` for different identities.

`ai_audio` demonstrates event handling; replace its token-renewal notice with your own token-server integration. `process_audio` requires the requested native APM extensions; the pinned Linux archive lacks AINS, so the default processor may fail initialization. Do not treat it as a portable noise-suppression example.

## Media formats

```sh
./scripts/with_agora_env.sh go run ./examples/media -mode send-yuv -input input.yuv -width 640 -height 360 -fps 15
./scripts/with_agora_env.sh go run ./examples/media -mode recv-yuv -user remote-user -output received.yuv
./scripts/with_agora_env.sh go run ./examples/media -mode send-aac -input input.aac
./scripts/with_agora_env.sh go run ./examples/media -mode send-h264 -input frames.packets -width 640 -height 360 -fps 15
```

- `send-yuv` / `recv-yuv`: raw I420 (YUV420p), no container header
- `send-aac`: AAC-LC ADTS frames
- `send-opus`: length-prefixed 20 ms Opus packets; set actual `-rate` and `-channels`
- `send-h264` / `recv-h264`: length-prefixed Annex-B encoded access units
- `recv-encoded-audio`: length-prefixed packets
- `transcode-h264` / `transcode-opus`: asynchronous decode-and-publish with `-tags avcodec`; Opus output is 48 kHz mono. Close does not guarantee worker drain; see [RTC transcoding limits](../docs/rtc.md#external-processing-and-optional-decoding) and [development](../docs/development.md)

All receive modes require `-user` to avoid combining unrelated bitstreams and overwrite `-output`. The packet format is a 4-byte big-endian unsigned size followed by that packet’s bytes, limited to 16 MiB. Use `mediaio.WritePacket` when preparing input. It preserves packet boundaries but does not store a full codec/timestamp/container description; it is not MP4, Ogg, or a general recording format. The Opus demo assumes exactly 20 ms per packet. H.264 payloads must use Annex-B start codes and access units must already be correctly framed; splitting on arbitrary read chunks is invalid.

## Safety and verification

Running a channel-based RTC example joins a live channel and may send or record media; `process_audio` and `external_vad` instead process audio locally without joining. Use dedicated test resources and record only with the participants’ authorization. Build and offline test success do not establish live interoperability. RTM scenarios can mutate remote state; read their [specific operations](messaging/README.md) first. `duplex_pcm` and `rtc_rtm` explicitly opt into numeric RTC identities and require matching numeric-UID RTC tokens; see their READMEs before running.

`voice_agent` sends the selected participant's audio to ElevenLabs and transcripts to OpenAI, logs transcript text, and can incur provider charges; obtain participant consent and read its [setup, identity and privacy limits](voice_agent/README.md) before running.

## Send a media file

Install the `ffmpeg` command-line tool, then run:

```sh
./scripts/with_agora_env.sh go run ./examples/send_media -input clip.mp4 -width 640 -height 360 -fps 15
```

The input must contain audio and video. This example decodes through separate FFmpeg processes, normalizes audio to 16 kHz mono PCM and video to fixed-rate I420, and assigns a shared timestamp base. It does not preserve arbitrary source-container timing or guarantee frame-accurate A/V synchronization. It requires the executable at runtime, not FFmpeg development headers or the `avcodec` build tag.
