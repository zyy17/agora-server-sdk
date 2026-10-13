package main

import (
	"encoding/hex"
	"errors"
	"time"

	rtm "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/rtmtokenbuilder2"

	"github.com/zyy17/agora-server-sdk/token"
)

const tokenLifetime = 2 * time.Minute

type credentials struct {
	appID, certificate string
}

func (c credentials) validate() error {
	for _, v := range []struct{ name, value string }{
		{"AGORA_APP_ID", c.appID}, {"AGORA_APP_CERTIFICATE", c.certificate},
	} {
		if v.value == "" {
			return errors.New(v.name + " is missing; configure the repository Actions secret")
		}
		if len(v.value) != 32 {
			return errors.New(v.name + " must contain exactly 32 hexadecimal characters")
		}
		if _, err := hex.DecodeString(v.value); err != nil {
			return errors.New(v.name + " must contain exactly 32 hexadecimal characters")
		}
	}
	return nil
}

func rtcToken(c credentials, channel, user string) (string, error) {
	return token.RTC(token.Config{
		AppID: c.appID, Certificate: c.certificate, Channel: channel,
		UserID: user, Role: token.Subscriber, Lifetime: tokenLifetime,
	})
}

func rtmToken(c credentials, user string) (string, error) {
	// The pinned official AccessToken2 API takes a relative lifetime in seconds,
	// NOT a Unix expiry timestamp. Keep the matching claim test below.
	return rtm.BuildToken(c.appID, c.certificate, user, uint32(tokenLifetime/time.Second))
}
