# Examples

This directory contains the examples for the `agora-server-sdk`.

## Agora App ID and App Certificate

You can login the [Agora Console](https://console.agora.io/), create a project, and obtain the App ID and App Certificate. You can set the environment variables `AGORA_APP_ID` and `AGORA_APP_CERT` to the App ID and App Certificate, respectively.

```console
export AGORA_APP_ID=<your-app-id>
export AGORA_APP_CERT=<your-app-certificate>
```

## Build the Examples

1. Install [pkg-config](https://pkg-config.freedesktop.org/wiki/) and [portaudio](https://www.portaudio.com/):

   For macOS, you can use the following command to install portaudio:

   ```console
   brew install portaudio
   ```

   For Ubuntu, you can use the following command to install portaudio:

   ```console
   sudo apt-get update
   sudo apt-get install portaudio19-dev
   ```

2. Build the examples:

   ```console
   make build-examples
   ```

## Send and Receive PCM Audio

The `send_pcm` and `recv_pcm` examples are used to send and receive PCM audio to the Agora RTC channel. The `send_pcm` example sends a test PCM audio to the Agora RTC channel, and the `recv_pcm` example receives the PCM audio from the Agora RTC channel and plays it through the speaker on your computer.

You can run the examples as follows:

```shell
# Export the Agora SDK library path(only for Linux).
# If you are using macOS, you should export DYLD_LIBRARY_PATH instead of LD_LIBRARY_PATH.
export LD_LIBRARY_PATH=$(PWD)/agora_libs

# Send PCM audio to the Agora RTC channel
AGORA_APP_ID=<your-app-id> AGORA_APP_CERT=<your-app-certificate> ./bin/send_pcm | grep "send_pcm"

# Receive PCM audio from the Agora RTC channel
AGORA_APP_ID=<your-app-id> AGORA_APP_CERT=<your-app-certificate> ./bin/recv_pcm | grep "recv_pcm"
```

Notice that your computer should have a microphone to play the audio.

If you want to play the test PCM audio, you can use the following command(requires [ffmpeg](https://ffmpeg.org/)):

```console
ffplay -f s16le -ar 16000 -ch_layout mono -autoexit ./examples/testdata/send_audio_16k_1ch.pcm
```
