package agoraservice

// SetAudioTrackEnabled enables or disables the connection-owned local audio
// track without changing publication state. Serialize raw lifecycle operations
// externally. An audience with configured audio stores the setting until
// promotion. Native SetEnabled has no result code; NoPublish returns -2000.
func (conn *RtcConnection) SetAudioTrackEnabled(enabled bool) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if !conn.canConfigureAudioTrack() {
		return -2000
	}
	if conn.audioTrack != nil {
		conn.audioTrack.SetEnabled(enabled)
	}
	conn.audioSettings.enabledSet, conn.audioSettings.enabled = true, enabled
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
	if !conn.canConfigureAudioTrack() {
		return -2000
	}
	if conn.audioTrack != nil {
		if code := conn.audioTrack.AdjustPublishVolume(volume); code != 0 {
			return code
		}
	}
	conn.audioSettings.volumeSet, conn.audioSettings.volume = true, volume
	return 0
}

// SetAudioSendDelayMs forwards the local audio track's send delay in milliseconds.
// Native configuration is void; zero confirms invocation, not a timing guarantee.
func (conn *RtcConnection) SetAudioSendDelayMs(delay int) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if !conn.canConfigureAudioTrack() {
		return -2000
	}
	if conn.audioTrack != nil {
		if code := conn.audioTrack.SetSendDelayMs(delay); code != 0 {
			return code
		}
	}
	conn.audioSettings.delaySet, conn.audioSettings.delay = true, delay
	return 0
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
	if !conn.canConfigureAudioTrack() {
		return -2000
	}
	if conn.audioTrack != nil {
		conn.audioTrack.SetMaxBufferedAudioFrameNumber(frames)
	}
	conn.audioSettings.capacitySet, conn.audioSettings.capacity = true, frames
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
	if code := conn.recoverAudioPublication(func() int { return conn.localUser.publishAudio(conn.audioTrack) }); code != 0 {
		return code
	}
	if conn.pcmConsumeStats != nil && conn.pcmConsumeStats.isNewRound(frame.SamplesPerSec, frame.Channels) && conn.sendExternalAudioParameters != nil && conn.sendExternalAudioParameters.Enabled {
		if code := conn.setTotalExtraSendMs(); code != 0 {
			return code
		}
	}
	result := conn.audioSender.SendAudioPcmData(frame)
	if result == 0 && conn.pcmConsumeStats != nil {
		conn.pcmConsumeStats.addPcmData(len(frame.Buffer), frame.SamplesPerSec, frame.Channels)
	}
	return result
}
