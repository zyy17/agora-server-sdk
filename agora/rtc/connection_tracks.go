package agoraservice

// SetAudioTrackEnabled enables or disables the connection-owned local audio
// track without changing publication state. Serialize raw lifecycle operations
// externally. Returns -2000 if absent; native SetEnabled has no result code.
func (conn *RtcConnection) SetAudioTrackEnabled(enabled bool) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.audioTrack == nil || conn.audioTrack.cTrack == nil {
		return -2000
	}
	conn.audioTrack.SetEnabled(enabled)
	return 0
}

// SetVideoTrackEnabled controls the connection-owned local video track without
// changing publication. Serialize all video/lifecycle operations externally.
// Native SetEnabled is void; zero confirms invocation only.
func (conn *RtcConnection) SetVideoTrackEnabled(enabled bool) int {
	if conn == nil || conn.cConnection == nil || conn.videoTrack == nil || conn.videoTrack.cTrack == nil {
		return -2000
	}
	conn.videoTrack.SetEnabled(enabled)
	return 0
}

// AdjustAudioPublishVolume forwards native volume for the connection-owned track.
// The native SDK validates the value. Serialize raw lifecycle operations externally.
func (conn *RtcConnection) AdjustAudioPublishVolume(volume int) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.audioTrack == nil {
		return -2000
	}
	return conn.audioTrack.AdjustPublishVolume(volume)
}

// SetAudioSendDelayMs forwards the local audio track's send delay in milliseconds.
// Native configuration is void; zero confirms invocation, not a timing guarantee.
func (conn *RtcConnection) SetAudioSendDelayMs(delay int) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.audioTrack == nil {
		return -2000
	}
	return conn.audioTrack.SetSendDelayMs(delay)
}

// SetAudioBufferCapacity sets the local sender capacity in 10 ms frames.
// This temporary upstream interface may change in future native releases.
// Native configuration is void; zero confirms invocation only.
func (conn *RtcConnection) SetAudioBufferCapacity(frames int) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.audioTrack == nil || conn.audioTrack.cTrack == nil {
		return -2000
	}
	conn.audioTrack.SetMaxBufferedAudioFrameNumber(frames)
	return 0
}

// ClearAudioBuffer clears the local native sender queue and resets its completion
// estimate on success. It does not unpublish, stop producers or clear peer audio.
func (conn *RtcConnection) ClearAudioBuffer() int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.audioTrack == nil {
		return -2000
	}
	result := conn.audioTrack.ClearSenderBuffer()
	if result == 0 && conn.pcmConsumeStats != nil {
		conn.pcmConsumeStats.reset()
	}
	return result
}

// PushAudioFrame submits an explicit PCM frame through the connection-owned
// sender, preserving uint32 capture time and signed presentation time. It uses
// the same send-round and duration-estimate bookkeeping as PushAudioPcmData.
// Keep a consistent format and serialize raw lifecycle operations externally.
func (conn *RtcConnection) PushAudioFrame(frame *AudioFrame) int {
	if conn == nil {
		return -2000
	}
	if frame.ValidatePCM() != nil {
		return -2
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.audioSender == nil {
		return -2000
	}
	if conn.pcmConsumeStats != nil && conn.pcmConsumeStats.isNewRound(frame.SamplesPerSec, frame.Channels) {
		conn.setTotalExtraSendMs()
	}
	result := conn.audioSender.SendAudioPcmData(frame)
	if result == 0 && conn.pcmConsumeStats != nil {
		conn.pcmConsumeStats.addPcmData(len(frame.Buffer), frame.SamplesPerSec, frame.Channels)
	}
	return result
}
