package agoraservice

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
	if conn.videoTrack != nil && conn.localUser.videoPublishFlag {
		if code := conn.localUser.unpublishVideo(conn.videoTrack); code != 0 {
			return code
		}
	}
	return 0
}
