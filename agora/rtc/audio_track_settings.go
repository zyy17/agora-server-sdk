package agoraservice

// Explicit connection-owned settings survive native PCM track replacement.
// Every access after construction is protected by RtcConnection.audioMu.
type audioTrackSettings struct {
	enabledSet, enabled bool
	volumeSet           bool
	volume              int
	delaySet            bool
	delay               int
	capacitySet         bool
	capacity            int
}

type audioTrackControls interface {
	SetEnabled(bool)
	AdjustPublishVolume(int) int
	SetSendDelayMs(int) int
	SetMaxBufferedAudioFrameNumber(int)
}

func (settings audioTrackSettings) apply(track audioTrackControls) int {
	// Restore mute before any chance of publishing the replacement track.
	if settings.enabledSet {
		track.SetEnabled(settings.enabled)
	}
	if settings.volumeSet {
		if code := track.AdjustPublishVolume(settings.volume); code != 0 {
			return code
		}
	}
	if settings.delaySet {
		if code := track.SetSendDelayMs(settings.delay); code != 0 {
			return code
		}
	}
	if settings.capacitySet {
		track.SetMaxBufferedAudioFrameNumber(settings.capacity)
	}
	return 0
}

func (conn *RtcConnection) canConfigureAudioTrack() bool {
	if conn.cConnection == nil {
		return false
	}
	if conn.audioTrack != nil {
		return conn.audioTrack.cTrack != nil
	}
	return conn.publishConfig != nil && conn.publishConfig.AudioPublishType != AudioPublishTypeNoPublish
}

// recoverAudioPublication retries only a publication lost by a failed scenario
// rollback. It never starts publication for a deliberately unpublished track.
// Call with audioMu held, so automatic fallback cannot race the retry and send.
func (conn *RtcConnection) recoverAudioPublication(publish func() int) int {
	if !conn.audioPublicationRecovery {
		return 0
	}
	if code := publish(); code != 0 {
		return code
	}
	conn.audioPublicationRecovery = false
	return 0
}

// rollbackAudioScenario records incomplete restoration without claiming the old
// native state was restored. A later explicit scenario request retries dirty
// state, and a later send retries a lost previously active publication.
func (conn *RtcConnection) rollbackAudioScenario(scenario AudioScenario, wasPublished bool, setScenario func(AudioScenario) int, publish func() int) {
	conn.audioScenarioDirty = setScenario(scenario) != 0
	if wasPublished {
		conn.audioPublicationRecovery = publish() != 0
		if conn.audioPublicationRecovery && conn.pcmConsumeStats != nil {
			conn.pcmConsumeStats.reset()
		}
	}
}
