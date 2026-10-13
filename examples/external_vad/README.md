# Standalone VAD

[简体中文](README.zh-CN.md) · [Examples](../README.md)

## Purpose

Process mono or stereo PCM with standalone native VAD. This example creates no RTC service, joins no channel, and requires no App ID or tokens. It is separate from external APM/noise suppression and the received-audio VAD v2 option.

## Run

Install the native SDK as described in the [main README](../../README.md). From the repository root:

```sh
./scripts/with_agora_env.sh go run ./examples/external_vad -channels 1
./scripts/with_agora_env.sh go run ./examples/external_vad -channels 2
./scripts/with_agora_env.sh go run ./examples/external_vad -channels 2 -input /absolute/path/to/stereo.pcm
```

Without `-input`, process one second of synthetic silence. A supplied file is read only and must be raw 16 kHz PCM16LE in the selected channel layout, without a WAV header. Output shows state changes and returned byte counts; no output files are written. Ctrl+C stops between frames. A blocked file read itself is not interruptible.

## Contracts

- Mono uses `rtc.NewAudioVad`; stereo uses the compatibility spelling `rtc.NewSteroVad`. Check constructors for `nil` and always release the detector.
- Each input is exactly 10 ms: 160 samples per channel, 2 bytes per sample, or 320/640 bytes for mono/stereo. `AudioFrame` explicitly sets the PCM16 type, sample rate, channel count, sample count, and byte width. Partial final frames are rejected.
- Stereo is interleaved left/right input. The two independent detectors return separate mono speech segments; their lengths and speech boundaries can differ. Do not concatenate them as if they were synchronized stereo output.
- Processing is synchronous and unpaced. Native VAD states are 0 (not speaking), 1 (start), 2 (speaking), and 3 (stop); negative values indicate failure. Synthetic silence demonstrates API use, not recognition accuracy.
- VAD may buffer speech. Only returned segments are counted; EOF does not flush buffered tail audio, and `Release` is not a flush operation. This example does not promise a complete speech recording.

## Offline checks

```sh
./scripts/with_agora_env.sh go test ./examples/external_vad
./scripts/with_agora_env.sh go build -o ./bin/external_vad ./examples/external_vad
```

Tests check frame construction, PCM preservation, partial-input rejection, cancellation, and consumer errors using synthetic data and a fake detector callback. Tests do not initialize native VAD or measure recognition quality. Running the commands in “Run” does initialize native VAD locally.
