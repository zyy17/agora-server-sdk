# Duplex PCM

[简体中文](README.zh-CN.md) · [Examples](../README.md)

## Purpose

Join with a numeric RTC UID, send a short synthetic stereo tone, and consume incoming mono PCM concurrently. No audio hardware or input file is required.

## Run

Install the native SDK as described in the [main README](../../README.md). From the repository root, set `AGORA_APP_ID`, `AGORA_CHANNEL`, `AGORA_USER_ID` (for example, `12345`), and the matching `AGORA_TOKEN` when authentication is enabled. Obtain a numeric-UID RTC token from your trusted server using Agora's official token builder; the `token.RTC` helper uses string accounts.

```sh
./scripts/with_agora_env.sh go run ./examples/duplex_pcm -duration 2s -pts 123
```

Use a second, authorized participant with a different numeric UID in the same channel to hear the tone and supply return audio. Start both participants together; the example exits after sending and estimating completion. Ctrl+C cancels pacing and closes the session. Running joins a live channel and publishes audio; building and testing do not.

## Contracts

- `Config.IdentityMode = sdk.NumericIdentity` applies to the whole service. UIDs are canonical decimal `uint32` strings; choose a positive UID with a matching token. An all-digit string in the default identity mode is still a string account.
- `Audio.Send` is 16 kHz stereo; `Audio.Receive` is independently 16 kHz mono. PCM is signed, interleaved PCM16LE. Incoming audio is counted and discarded, not recorded or echoed.
- Chunks cycle through 7, 13, and 20 ms; the last chunk is shortened to the remaining integral milliseconds. `SendPCMAt` accepts complete samples with integral-ms duration. At 44.1 kHz, the smallest integral-ms chunk is 10 ms.
- Every chunk forwards the exact `-pts` value unchanged. It is an opaque caller value here, not a generated timeline. Pacing uses cancellable waits; slow sends do not cause catch-up bursts. For ordinary 10 ms paced files, use `PlayPCM` instead.
- Audio and event consumers run outside native callbacks. Shutdown waits for them and reports queue drops. This short demo discards pending receive items on cancellation; it is not a lossless recorder. `WaitPCM` estimates local completion, not remote playback acknowledgment.

## Offline checks

```sh
./scripts/with_agora_env.sh go test ./examples/duplex_pcm
./scripts/with_agora_env.sh go build -o ./bin/duplex_pcm ./examples/duplex_pcm
```

Tests use synthetic data and fake sends/waits to check chunk boundaries, sample continuity, exact PTS forwarding, cancellation, and validation. They do not prove live RTC interoperability.
