package agorasdk

import (
	"encoding/json"
	"strings"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func validateUserID(uid string) error {
	if strings.TrimSpace(uid) == "" || strings.ContainsRune(uid, 0) || len(uid) > 255 {
		return invalid("user ID", "must contain 1–255 bytes without NUL")
	}
	return nil
}

func validateParameters(value string) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &obj); err != nil || obj == nil {
		return invalid("parameters", "must be a JSON object")
	}
	return nil
}

// SubscribeAudio requests one remote user's audio in the Service's identity
// namespace. Numeric IDs must be nonzero canonical uint32 text.
// LargeScale channels require per-user manual subscriptions. Native acceptance
// is not evidence that the peer is publishing or that frames have arrived.
func (s *Session) SubscribeAudio(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("subscribe audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeAudio(userID) })
}

// UnsubscribeAudio requests removal of one remote user's audio subscription.
// userID must match the Service's identity namespace. Frames already queued
// for the application remain available.
func (s *Session) UnsubscribeAudio(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("unsubscribe audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeAudio(userID) })
}

// SubscribeAllAudio requests audio from all remote users.
// LargeScale channels return ErrUnsupported; use SubscribeAudio for each user.
// Observer enablement and received PCM format remain unchanged.
func (s *Session) SubscribeAllAudio() error {
	if s.cfg.Mode == LargeScale {
		return ErrUnsupported
	}
	return s.withNative("subscribe all audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeAllAudio() })
}

// UnsubscribeAllAudio requests removal of all remote audio subscriptions.
// Already buffered frames are not discarded.
func (s *Session) UnsubscribeAllAudio() error {
	return s.withNative("unsubscribe all audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeAllAudio() })
}

// VideoSubscription controls which native remote video stream is requested.
// The zero value requests the high-quality stream with decoded frames.
// Availability still depends on publication, observers, and native/peer support.
type VideoSubscription struct {
	// LowQuality requests the native low-quality stream when the publisher offers it.
	LowQuality bool
	// EncodedOnly requests encoded frames without native decoding.
	EncodedOnly bool
}

func (v VideoSubscription) native() *rtc.VideoSubscriptionOptions {
	stream := rtc.VideoStreamType(0)
	if v.LowQuality {
		stream = 1
	}
	return &rtc.VideoSubscriptionOptions{StreamType: stream, EncodedFrameOnly: v.EncodedOnly}
}

// SubscribeVideo requests one remote user's video with options.
// userID must match the Service's identity namespace; numeric IDs must be nonzero.
// Enable the corresponding ReceiveConfig observer to consume returned frames.
// LargeScale channels require these explicit per-user requests.
func (s *Session) SubscribeVideo(userID string, options VideoSubscription) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("subscribe video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeVideo(userID, options.native()) })
}

// UnsubscribeVideo requests removal of one remote user's video subscription.
// Frames already buffered in receive queues are not discarded.
func (s *Session) UnsubscribeVideo(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("unsubscribe video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeVideo(userID) })
}

// SubscribeAllVideo requests video from all remote users using options.
// LargeScale channels return ErrUnsupported. This does not enable an observer
// that was disabled when the session was created.
func (s *Session) SubscribeAllVideo(options VideoSubscription) error {
	if s.cfg.Mode == LargeScale {
		return ErrUnsupported
	}
	return s.withNative("subscribe all video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeAllVideo(options.native()) })
}

// UnsubscribeAllVideo requests removal of all remote video subscriptions.
// Already buffered frames remain available.
func (s *Session) UnsubscribeAllVideo() error {
	return s.withNative("unsubscribe all video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeAllVideo() })
}

// SendMessage submits 1–1024 bytes to the session's RTC data stream, not RTM.
// The stream is unordered and unreliable; success is local native acceptance.
// The wrapper does not pace, rate-limit, acknowledge, or retry messages.
// Keep data unchanged until the call returns.
//
// Native live-broadcasting data sends require Broadcaster. The pinned native
// contract limits a channel to 30 packets/second, a client to 6 KB/second,
// messages to 1 KB, and a user to 5 streams per channel. The wrapper uses its
// internally created stream; native limits remain authoritative.
func (s *Session) SendMessage(data []byte) error {
	if len(data) == 0 || len(data) > 1024 {
		return invalid("message", "must contain 1–1024 bytes")
	}
	return s.withNative("send message", func(c *rtc.RtcConnection) int { return c.SendStreamMessage(data) })
}

func (s *Session) validateRemoteUserID(uid string) error {
	mode := StringIdentity
	if s.service != nil {
		mode = s.service.identityMode
	}
	return mode.validateUserID(uid, false)
}
