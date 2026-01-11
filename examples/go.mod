module github.com/zyy17/agora-server-sdk/examples

go 1.25.5

require (
	github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2 v2.4.4
	github.com/gordonklaus/portaudio v0.0.0-20250206071425-98a94950218b
	github.com/zyy17/agora-server-sdk v0.0.0
)

require github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src v0.0.0-20240807100336-95d820182fef // indirect

replace github.com/zyy17/agora-server-sdk => ../

replace github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2 => ../third_party/Agora-Golang-Server-SDK
