// Package agoraservice exposes the native Agora RTC server SDK.
//
// These bindings follow Agora-Golang-Server-SDK v2.6.5, commit
// d51ff1b5baf85dd53155b31543bceba8ec304dc7, with local ownership and input
// validation fixes. Native headers retain their original Agora copyright notices.
//
// The native service is process-global. Serialize connection API calls, release
// connections before releasing the service, and never call lifecycle or observer
// registration methods synchronously from a native callback. Dispatch that work
// to another goroutine instead. Callback media payloads are copied into Go memory;
// native track handles are valid only for the lifetime documented by Agora.
//
// FFmpeg decoding and transcoding are optional and require the avcodec build tag.
// Ordinary PCM and encoded media transport do not require FFmpeg.
package agoraservice
