package agoraservice

import "fmt"

// preparePublishTracks lazily creates the configured tracks when an audience
// becomes a broadcaster. The caller serializes lifecycle changes with Release.
func (conn *RtcConnection) preparePublishTracks() int {
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.publishConfig == nil {
		return -1
	}
	if conn.audioTrack == nil && conn.publishConfig.AudioPublishType != AudioPublishTypeNoPublish {
		conn.publishConfig.IsPublishAudio = true
		conn.createAudioPublishTrack()
		if conn.audioTrack == nil {
			if conn.audioSender != nil {
				conn.audioSender.Release()
				conn.audioSender = nil
			}
			if conn.encodedAudioSender != nil {
				conn.encodedAudioSender.Release()
				conn.encodedAudioSender = nil
			}
			return -1
		}
	}
	if conn.videoTrack == nil && conn.publishConfig.VideoPublishType != VideoPublishTypeNoPublish {
		conn.publishConfig.IsPublishVideo = true
		conn.createVideoPublishTrack()
		failure := -1
		if conn.videoTrack != nil && conn.videoEncoderConfiguration != nil {
			if code := conn.SetVideoEncoderConfiguration(conn.videoEncoderConfiguration); code != 0 {
				failure = code
				conn.videoTrack.Release()
				conn.videoTrack = nil
			}
		}
		if conn.videoTrack == nil {
			if conn.videoSender != nil {
				conn.videoSender.Release()
				conn.videoSender = nil
			}
			if conn.encodedVideoSender != nil {
				conn.encodedVideoSender.Release()
				conn.encodedVideoSender = nil
			}
			return failure
		}
	}
	return 0
}

// SetUserRole requests a native client role, creating configured publication
// tracks on promotion. Serialize it with other connection operations and Release.
// The native setter has no server completion acknowledgement.
func (conn *RtcConnection) SetUserRole(role int) int {
	if conn == nil || conn.localUser == nil {
		return -1
	}
	return conn.localUser.SetUserRole(role)
}

func (conn *RtcConnection) prepareAudience() int {
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.localUser == nil {
		return -1
	}
	if conn.audioTrack != nil && conn.localUser.publishFlag {
		if code := conn.localUser.unpublishAudio(conn.audioTrack); code != 0 {
			return code
		}
	}
	conn.audioPublicationRecovery = false
	if conn.pcmConsumeStats != nil {
		conn.pcmConsumeStats.reset()
	}
	if conn.videoTrack != nil && conn.localUser.videoPublishFlag {
		if code := conn.localUser.unpublishVideo(conn.videoTrack); code != 0 {
			return code
		}
	}
	return 0
}

// GetUserRole returns the most recently requested native role, without implying
// server acknowledgement. Serialize access with Release.
func (conn *RtcConnection) GetUserRole() int {
	if conn == nil {
		return 0
	}
	return conn.localUser.GetUserRole()
}

// GetPublicationState returns local successful publish/unpublish call state.
// Partial failures are reflected independently for audio and video; this is not
// server acknowledgement. Serialize lifecycle access with Release.
func (conn *RtcConnection) GetPublicationState() (audio, video bool) {
	if conn == nil {
		return false, false
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.localUser == nil {
		return false, false
	}
	return conn.localUser.publishFlag, conn.localUser.videoPublishFlag
}

func configureExtraSendFrameSpeed(params *SendExternalAudioParameters, setParameters func(string) int) int {
	if params == nil || !params.Enabled || params.SendMs <= 0 || params.SendSpeed <= 1 {
		return -2001
	}
	speed := min(params.SendSpeed, 5)
	return setParameters(fmt.Sprintf(`{"che.audio.extra_send_frames_per_interval_for_fake_adm":%d}`, speed))
}
