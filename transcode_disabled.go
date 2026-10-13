//go:build !avcodec

package agorasdk

const avcodecAvailable = false

// TranscodeVideo always returns ErrUnsupported in builds without avcodec.
// With that tag and FFmpeg development libraries, it asynchronously decodes
// H.264/H.265 into a Broadcaster RawVideo track. Worker acceptance does not
// acknowledge decoding, native sending, or remote playback; pending input can
// be discarded on Close.
func (s *Session) TranscodeVideo(EncodedVideoFrame) error { return ErrUnsupported }

// TranscodeOpus always returns ErrUnsupported in builds without avcodec.
// With that tag and FFmpeg development libraries, it asynchronously decodes
// Opus with 1 or 2 input channels into a Broadcaster PCM track as 48 kHz mono.
// This output format is independent of AudioConfig.Send; worker success does
// not acknowledge remote playback.
func (s *Session) TranscodeOpus([]byte, int) error { return ErrUnsupported }
