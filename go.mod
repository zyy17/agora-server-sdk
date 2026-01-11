module github.com/zyy17/agora-server-sdk

go 1.25.5

require (
	github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2 v2.4.4
	github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src v0.0.0-20240807100336-95d820182fef
)

replace github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2 => ./third_party/Agora-Golang-Server-SDK
