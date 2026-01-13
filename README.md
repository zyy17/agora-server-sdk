# Agora Server SDK

## Overview

[agora-server-sdk](https://pkg.go.dev/github.com/zyy17/agora-server-sdk) is a further encapsulation based on the [Agora-Golang-Server-SDK](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK), designed to improve the developer experience when using Agora services.

The project is still in an **early stage of development** and the API is subject to change.

## Design

agora-server-sdk provides the clean APIs based on the original SDK to help you quickly create an Agora RTC connection:

- **`agorasdk.Service`**: Represents an Agora RTC service instance.
- **`agorasdk.RTCConnection`**: Represents an Agora RTC connection instance.

You can quickly create an Agora RTC connection as follows:

```go
...
// Create an Agora RTC service instance.
svc, err := agorasdk.NewService(
	agorasdk.WithAppID(appID),
	agorasdk.WithAppCert(appCert),
	agorasdk.WithChannelName(channelName),
	agorasdk.WithUserID(userID),
	agorasdk.WithLogPath(logPath),
	agorasdk.WithConfigDir(configDir),
	agorasdk.WithDataDir(dataDir),
)
if err != nil {
	log.Fatalf("failed to create service: %v", err)
}
defer svc.Release()

// Create an Agora RTC connection instance.
conn, err := svc.NewRTCConnection(
	agorasdk.WithOnConnected(onConnected),
	agorasdk.WithSampleRate(defaultSampleRate),
	agorasdk.WithAudioChannelType(defaultAudioChannelType),
)
if err != nil {
	log.Fatalf("failed to create RTC connection: %v", err)
}
defer conn.Release()

// Publish audio to the RTC connection.
if err := conn.PublishAudio(); err != nil {
	log.Fatalf("failed to publish audio: %v", err)
}
...
```

You can refer to the [examples](./examples) for more details.

## Prerequisites

Go >= 1.23 or higher.

## Quick Start

1. Use the `agora-server-sdk` module in your project:

   ```console
   go get github.com/zyy17/agora-server-sdk
   ```

2. Download the Agora libraries. It will download the Agora libraries to the current directory as `agora_libs/`:

   ```console
   curl -fsSL \
     https://raw.githubusercontent.com/zyy17/agora-server-sdk/refs/heads/main/scripts/download_agora_libs.sh | sh
   ```

3. Build with Agora libraries, for example:

   ```console
   CGO_LDFLAGS="-L$(PWD)/agora_libs" go build -o my_app main.go
   ```

## Run the Examples

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

2. Download the `agora-server-sdk` repository:

   ```console
   git clone https://github.com/zyy17/agora-server-sdk.git
   ```

3. Build the examples:

   ```console
   make build-examples
   ```

4. Run the examples:

   You can log in to the [Agora Console](https://console.agora.io/), create a project, and obtain the App ID and App Certificate. Then, you can run the examples as follows:

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

## License

The project is licensed under the MIT License.
