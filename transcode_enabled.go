//go:build avcodec

package agorasdk

// TranscodeVideo decodes H.264/H.265 to the configured raw-video track using optional FFmpeg.
func (s *Session) TranscodeVideo(frame EncodedVideoFrame) error {
	if err := frame.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.cfg.Video.Mode != RawVideo {
		return ErrNotPublishing
	}
	if err := s.publishVideoLocked(); err != nil {
		return err
	}
	return check("transcode video", s.native.PushVideoEncodedDataForTranscode(frame.Data, frame.nativeInfo()))
}
func (s *Session) TranscodeOpus(packet []byte, channels int) error {
	if len(packet) == 0 || (channels != 1 && channels != 2) {
		return invalid("Opus packet", "requires nonempty data and 1–2 channels")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.cfg.Audio.Mode != PCM {
		return ErrNotPublishing
	}
	if err := s.publishAudioLocked(); err != nil {
		return err
	}
	return check("transcode Opus", s.native.PushAudioEncodedDataForTranscode(packet, channels, int(Opus)))
}
