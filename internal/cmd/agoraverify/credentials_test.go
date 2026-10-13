package main

import (
	"strings"
	"testing"
	"time"

	access "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/accesstoken2"
)

func syntheticCredentials() credentials {
	return credentials{strings.Repeat("a", 32), strings.Repeat("b", 32)}
}

func TestCredentialValidationDoesNotExposeValues(t *testing.T) {
	for _, value := range []string{"", "secret-sentinel", strings.Repeat("z", 32), strings.Repeat("a", 31), strings.Repeat("a", 33), "\n" + strings.Repeat("a", 31)} {
		for _, field := range []string{"app ID", "certificate"} {
			c := syntheticCredentials()
			if field == "app ID" {
				c.appID = value
			} else {
				c.certificate = value
			}
			err := c.validate()
			if err == nil {
				t.Fatalf("accepted invalid %s", field)
			}
			if value != "" && strings.Contains(err.Error(), value) {
				t.Fatal("credential echoed by validation")
			}
		}
	}
	c := syntheticCredentials()
	c.appID = strings.Repeat("AB", 16)
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenClaimsAreShortLivedAndSeparate(t *testing.T) {
	c := syntheticCredentials()
	for _, component := range []string{"rtc", "rtm"} {
		var value string
		var err error
		if component == "rtc" {
			value, err = rtcToken(c, "test-room", "test-string-user")
		} else {
			value, err = rtmToken(c, "test-string-user")
		}
		if err != nil {
			t.Fatal("synthetic token creation failed")
		}
		parsed := access.NewAccessToken("", "", 0)
		ok, err := parsed.Parse(value)
		if err != nil || !ok {
			t.Fatal("synthetic token parsing failed")
		}
		if parsed.AppId != c.appID || parsed.Expire != 120 || len(parsed.Services) != 1 {
			t.Fatal("unexpected token app, relative lifetime, or service count")
		}
		if age := time.Now().Unix() - int64(parsed.IssueTs); age < -1 || age > 5 {
			t.Fatal("unexpected issue time")
		}
		if component == "rtc" {
			s, ok := parsed.Services[access.ServiceTypeRtc].(*access.ServiceRtc)
			if !ok || s.ChannelName != "test-room" || s.Uid != "test-string-user" || len(s.Privileges) != 1 || s.Privileges[access.PrivilegeJoinChannel] != 120 {
				t.Fatal("RTC token should authorize only this channel/string account and join privilege")
			}
		} else {
			s, ok := parsed.Services[access.ServiceTypeRtm].(*access.ServiceRtm)
			if !ok || s.UserId != "test-string-user" || len(s.Privileges) != 1 || s.Privileges[access.PrivilegeLogin] != 120 {
				t.Fatal("RTM token should authorize only this string account and login privilege")
			}
		}
	}
}
