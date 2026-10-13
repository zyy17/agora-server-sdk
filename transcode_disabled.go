//go:build !avcodec

package agorasdk

// TranscodeVideo requires a build with -tags avcodec and FFmpeg development libraries.
func (s *Session) TranscodeVideo(EncodedVideoFrame) error { return ErrUnsupported }

// TranscodeOpus requires a build with -tags avcodec and FFmpeg development libraries.
func (s *Session) TranscodeOpus([]byte, int) error { return ErrUnsupported }
