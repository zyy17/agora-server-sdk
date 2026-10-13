package agorasdk

import (
	"errors"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestNormalizedValidatesNativeVideoOptions(t *testing.T) {
	for _, video := range []VideoConfig{
		{Encoder: &rtc.VideoEncoderConfiguration{Width: 640, Height: -1}},
		{Encoder: &rtc.VideoEncoderConfiguration{Framerate: -1}},
		{EncodedSender: &rtc.VideoEncodedImageSenderOptions{TargetBitrate: -1}},
	} {
		if _, err := normalized(ChannelConfig{Channel: "test", UserID: "test", Video: video}); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("accepted invalid video options: %v", err)
		}
	}
}
