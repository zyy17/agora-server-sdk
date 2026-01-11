# Agora Server SDK

## Overview

[agora-server-sdk](https://pkg.go.dev/github.com/zyy17/agora-server-sdk) is a further encapsulation of [Agora-Golang-Server-SDK](https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK) to improve the developer experience when using Agora services.

The project is still in an **early stage of development** and the API is subject to change.

## Design

agora-server-sdk provides the clean APIs based on the original SDK to help you quickly create an Agora RTC connection:

- **`agorasdk.Service`**: Represents an Agora RTC service instance.
- **`agorasdk.RTCConnection`**: Represents an Agora RTC connection instance.

You can quickly create an Agora RTC connection as follows:

```go
// Create an Agora RTC service instance.
svc, err := agorasdk.NewService(
	agorasdk.WithAppID(*appID),
	agorasdk.WithAppCert(*appCert),
	agorasdk.WithChannelName(*channelName),
	agorasdk.WithUserID(*userID),
	agorasdk.WithLogPath(*logPath),
	agorasdk.WithConfigDir(*configDir),
	agorasdk.WithDataDir(*dataDir),
)
if err != nil {
	logFatalf("failed to create service: %v", err)
}
defer svc.Release()

// Create an Agora RTC connection instance.
conn, err := svc.NewRTCConnection(
	agorasdk.WithOnConnected(onConnected),
	agorasdk.WithSampleRate(defaultSampleRate),
	agorasdk.WithAudioChannelType(defaultAudioChannelType),
)
if err != nil {
	logFatalf("failed to create RTC connection: %v", err)
}
defer conn.Release()

// Publish audio to the RTC connection.
if err := conn.PublishAudio(); err != nil {
	logFatalf("failed to publish audio: %v", err)
}
```

You can refer to the [examples](./examples) for more details.

## Prerequisites

1. Go >= 1.24 or higher

2. Log in to the [Agora Console](https://console.agora.io/), create a project, and obtain the App ID and App Certificate.

## Quick Start

1. Install agora-server-sdk

   ```console
   git clone https://github.com/zyy17/agora-server-sdk.git
   ```

2. Build the examples

   ```console
   make build-examples
   ```

3. Run the examples

   ```console
   # Send PCM audio to the Agora RTC channel
   AGORA_APP_ID=<your-app-id> AGORA_APP_CERT=<your-app-certificate> ./bin/send_pcm | grep "send_pcm"

   # Receive PCM audio from the Agora RTC channel
   AGORA_APP_ID=<your-app-id> AGORA_APP_CERT=<your-app-certificate> ./bin/recv_pcm | grep "recv_pcm"
   ```

## License

The project is licensed under the MIT License.
