package token

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	builder "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/rtctokenbuilder2"
)

// Role selects the RTC privileges embedded by [RTC]. It is a token-building role,
// separate from a client's configured RTC connection role. The zero value is
// [Publisher].
type Role uint8

const (
	// Publisher includes joining the channel and publishing audio, video, and
	// data streams. It is the default role.
	Publisher Role = iota
	// Subscriber includes only the join-channel privilege. Enforcing publishing
	// restrictions requires Agora's co-host authentication to be enabled for the
	// project; do not rely on this role alone as an authorization boundary.
	Subscriber
)

// Config describes an RTC AccessToken2 for one channel and string user account.
// Its zero value is invalid: AppID, Certificate, Channel, and UserID are required.
// [RTC] reads a copy of Config and does not change the caller's value.
type Config struct {
	// AppID is the Agora project ID, exactly 32 hexadecimal characters.
	AppID string
	// Certificate is the project's App Certificate, exactly 32 hexadecimal
	// characters. Keep it on a trusted server; never expose it to clients or logs.
	Certificate string
	// Channel is the exact RTC channel name to authorize. It must contain
	// 1 through 64 bytes and no NUL byte; limits are in bytes, not runes.
	Channel string
	// UserID is the exact string user account to authorize. It must contain
	// 1 through 255 bytes and no NUL byte. RTC always uses the string-account
	// builder, even when UserID consists entirely of decimal digits.
	UserID string
	// Role selects the token's RTC privileges. The zero value is Publisher.
	Role Role
	// Lifetime is the validity duration relative to token creation, not an
	// absolute expiry timestamp. Zero defaults to one hour. Other values must
	// be between one second and 24 hours, inclusive. Fractional seconds are
	// discarded, and the same whole-second duration is used for the token
	// and all included privileges.
	Lifetime time.Duration
}

// RTC generates an RTC AccessToken2 locally for the string account and channel
// in c. It does not contact Agora or require the native SDK. The App ID and
// certificate are checked for hexadecimal shape, but their validity for an
// actual Agora project is not verified.
//
// RTC validates the fields described by [Config] and accepts only [Publisher]
// or [Subscriber]. A zero lifetime becomes one hour; otherwise the allowed range
// is one second through 24 hours. The lifetime is truncated to whole seconds
// and supplied as both the token expiry and each included privilege's expiry.
//
// RTC returns an empty string and an error for invalid configuration, or the
// underlying builder's result and error. Tokens contain creation-time and random
// data, so repeated calls need not return the same string. Treat successful
// results as credentials: avoid logging them and deliver them only to the
// intended client over a secure channel.
//
// RTC always uses Agora's string-user-account builder. It does not generate RTM
// login tokens or expose numeric-UID token generation; use the appropriate Agora
// builder for those workflows.
func RTC(c Config) (string, error) {
	for name, value := range map[string]string{"app ID": c.AppID, "certificate": c.Certificate} {
		if len(value) != 32 {
			return "", fmt.Errorf("token: %s must be 32 hexadecimal characters", name)
		}
		if _, err := hex.DecodeString(value); err != nil {
			return "", fmt.Errorf("token: %s must be hexadecimal", name)
		}
	}
	if c.Channel == "" || len(c.Channel) > 64 || c.UserID == "" || len(c.UserID) > 255 || strings.ContainsRune(c.Channel, 0) || strings.ContainsRune(c.UserID, 0) {
		return "", fmt.Errorf("token: invalid channel or user ID")
	}
	if c.Lifetime == 0 {
		c.Lifetime = time.Hour
	}
	if c.Lifetime < time.Second || c.Lifetime > 24*time.Hour {
		return "", fmt.Errorf("token: lifetime must be 1 second to 24 hours")
	}
	if c.Role > Subscriber {
		return "", fmt.Errorf("token: invalid role")
	}
	role := builder.Role(builder.RolePublisher)
	if c.Role == Subscriber {
		role = builder.RoleSubscriber
	}
	seconds := uint32(c.Lifetime / time.Second)
	return builder.BuildTokenWithUserAccount(c.AppID, c.Certificate, c.Channel, c.UserID, role, seconds, seconds)
}
