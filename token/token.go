// Package token generates RTC AccessToken2 values locally. App certificates must
// remain on a trusted server; never embed them in client apps or commit them.
package token

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	builder "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/rtctokenbuilder2"
)

type Role uint8

const (
	Publisher Role = iota
	Subscriber
)

type Config struct {
	AppID, Certificate, Channel, UserID string
	Role                                Role
	Lifetime                            time.Duration
}

// RTC generates a string-user-ID token. Zero lifetime means one hour.
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
