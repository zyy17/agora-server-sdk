//go:build avcodec

package agorasdk

const avcodecAvailable = true

// TranscodeVideo asynchronously decodes H.264/H.265 input and publishes through
// the configured RawVideo track using optional FFmpeg workers.
// It requires Broadcaster, RawVideo, the avcodec build tag, and FFmpeg libraries.
// Supported input is 8-bit 4:2:0 Annex-B video; metadata validation alone does
// not prove decodability. Success means acceptance into the worker path.
// Later decode/send failures are not returned through this call.
//
// The worker sends its latest decoded frame on a timer, so frames may be skipped
// or repeated and arbitrary source timestamps are not preserved. Close can
// discard pending input. Avoid another producer on the same track.
func (s *Session) TranscodeVideo(frame EncodedVideoFrame) error {
	if err := validateTranscodeVideo(frame); err != nil {
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

// TranscodeOpus asynchronously decodes a nonempty Opus packet with 1 or 2 input
// channels and publishes 48 kHz mono PCM, independently of AudioConfig.Send.
// It requires Broadcaster, PCM publication, the avcodec build tag, and FFmpeg
// libraries. Success is worker acceptance, not decode or remote-playback success.
// Later worker errors are not returned through this call; Close can discard
// pending input. Avoid mixing this output with another producer on the track.
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
