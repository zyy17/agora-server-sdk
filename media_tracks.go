package agorasdk

import (
	"math"
	"time"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// SetAudioEnabled enables or disables the local audio track under the session's
// operation lock. It does not publish/unpublish the track. An audience session
// with configured audio stores the setting until promotion. Explicit audio-track
// settings survive scenario changes and automatic fallback. Native SetEnabled is
// void, so success is local application/storage, not remote state or playback.
func (s *Session) SetAudioEnabled(enabled bool) error {
	return s.withNative("set audio enabled", func(c *rtc.RtcConnection) int { return c.SetAudioTrackEnabled(enabled) })
}

// SetVideoEnabled is the video counterpart of SetAudioEnabled. It does not alter
// publication state and success is not a peer-rendering acknowledgement.
func (s *Session) SetVideoEnabled(enabled bool) error {
	return s.withNative("set video enabled", func(c *rtc.RtcConnection) int { return c.SetVideoTrackEnabled(enabled) })
}

// SetAudioPublishVolume adjusts the local published audio volume. volume must fit
// a nonnegative int32; the native SDK applies its supported range and gain policy.
// It does not affect received audio, queue duration, or publication state.
// Audience sessions defer native application until promotion; native validation
// can therefore fail at promotion. Successful settings survive track replacement.
func (s *Session) SetAudioPublishVolume(volume int) error {
	if volume < 0 || int64(volume) > math.MaxInt32 {
		return invalid("audio publish volume", "must fit nonnegative int32")
	}
	return s.withNative("set audio publish volume", func(c *rtc.RtcConnection) int { return c.AdjustAudioPublishVolume(volume) })
}

// SetAudioSendDelay forwards the local sender delay, truncated to milliseconds.
// delay must be nonnegative and fit signed int32 milliseconds. The native setter
// has no result code. The setting survives replacement and is deferred until
// promotion when an audience has no track yet.
func (s *Session) SetAudioSendDelay(delay time.Duration) error {
	if delay < 0 || delay/time.Millisecond > math.MaxInt32 {
		return invalid("audio send delay", "must fit nonnegative int32 milliseconds")
	}
	return s.withNative("set audio send delay", func(c *rtc.RtcConnection) int { return c.SetAudioSendDelayMs(int(delay / time.Millisecond)) })
}

// SetAudioBufferCapacity sets the native sender capacity in 10 ms frames. frames
// must be 1–int32. This temporary upstream feature has no native result code and
// does not change application Stream capacities or queue already delivered audio.
// It survives track replacement and is deferred until audience promotion.
func (s *Session) SetAudioBufferCapacity(frames int) error {
	if frames < 1 || int64(frames) > math.MaxInt32 {
		return invalid("audio buffer capacity", "must be 1–int32 frames")
	}
	return s.withNative("set audio buffer capacity", func(c *rtc.RtcConnection) int { return c.SetAudioBufferCapacity(frames) })
}

// ClearAudioBuffer clears only the local native sender queue. Unlike
// InterruptAudio it does not unpublish/republish in the AI-server scenario.
// Stop producers first to avoid immediately re-enqueuing stale audio. Audio
// already delivered to peers cannot be recalled by this method.
func (s *Session) ClearAudioBuffer() error {
	return s.withNative("clear audio buffer", func(c *rtc.RtcConnection) int { return c.ClearAudioBuffer() })
}
