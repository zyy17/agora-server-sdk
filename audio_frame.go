package agorasdk

import rtc "github.com/zyy17/agora-server-sdk/agora/rtc"

func nativeSendAudioFrame(frame AudioFrame) rtc.AudioFrame {
	return rtc.AudioFrame{Type: rtc.AudioFrameTypePCM16, Buffer: frame.Data, BytesPerSample: frame.BytesPerSample,
		SamplesPerChannel: frame.SamplesPerChannel, Channels: frame.Channels, SamplesPerSec: frame.SampleRate,
		RenderTimeMs: frame.TimestampMillis, PresentTimeMs: frame.PresentationMillis}
}

// SendPCMFrame sends explicit PCM16 metadata, including the native uint32 capture
// timestamp and signed presentation value. SampleRate and Channels must match
// AudioConfig.Send; BytesPerSample must be 2 and SamplesPerChannel must exactly
// match nonempty Data. Unlike SendPCM, integral-millisecond duration is not
// required. Native codec support for short frames remains SDK-dependent.
// Only Data, format/sample counts, TimestampMillis and PresentationMillis are
// sent; source identity, observation position, VAD and telemetry are ignored.
// It shares publication/lifecycle locking with SendPCM. Keep Data unchanged
// during this unpaced call. Success is local acceptance, not remote playback.
func (s *Session) SendPCMFrame(frame AudioFrame) error {
	native := nativeSendAudioFrame(frame)
	if err := native.ValidatePCM(); err != nil {
		return invalid("PCM frame", err.Error())
	}
	if frame.SampleRate != s.cfg.Audio.Send.SampleRate || frame.Channels != s.cfg.Audio.Send.Channels {
		return invalid("PCM frame format", "must match AudioConfig.Send")
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
	return check("send PCM frame", s.native.PushAudioFrame(&native))
}
