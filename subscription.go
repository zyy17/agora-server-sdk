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

// SubscribeAudio subscribes one user. Manual subscriptions are required in large-scale channels.
func (s *Session) SubscribeAudio(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("subscribe audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeAudio(userID) })
}
func (s *Session) UnsubscribeAudio(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("unsubscribe audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeAudio(userID) })
}
func (s *Session) SubscribeAllAudio() error {
	if s.cfg.Mode == LargeScale {
		return ErrUnsupported
	}
	return s.withNative("subscribe all audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeAllAudio() })
}
func (s *Session) UnsubscribeAllAudio() error {
	return s.withNative("unsubscribe all audio", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeAllAudio() })
}

type VideoSubscription struct{ LowQuality, EncodedOnly bool }

func (v VideoSubscription) native() *rtc.VideoSubscriptionOptions {
	stream := rtc.VideoStreamType(0)
	if v.LowQuality {
		stream = 1
	}
	return &rtc.VideoSubscriptionOptions{StreamType: stream, EncodedFrameOnly: v.EncodedOnly}
}
func (s *Session) SubscribeVideo(userID string, options VideoSubscription) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("subscribe video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeVideo(userID, options.native()) })
}
func (s *Session) UnsubscribeVideo(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("unsubscribe video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeVideo(userID) })
}
func (s *Session) SubscribeAllVideo(options VideoSubscription) error {
	if s.cfg.Mode == LargeScale {
		return ErrUnsupported
	}
	return s.withNative("subscribe all video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().SubscribeAllVideo(options.native()) })
}
func (s *Session) UnsubscribeAllVideo() error {
	return s.withNative("unsubscribe all video", func(c *rtc.RtcConnection) int { return c.GetLocalUser().UnsubscribeAllVideo() })
}

// SendMessage sends an RTC data-stream message, not an RTM publish. Native rate
// limits still apply. Delivery is neither reliable nor ordered on this stream.
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
