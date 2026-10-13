package token

import (
	"strings"
	"testing"
	"time"
)

func TestRTC(t *testing.T) {
	c := Config{AppID: strings.Repeat("a", 32), Certificate: strings.Repeat("b", 32), Channel: "room", UserID: "user"}
	v, err := RTC(c)
	if err != nil || !strings.HasPrefix(v, "007") {
		t.Fatal("generation failed", err)
	}
	c.Role = Subscriber
	if _, err := RTC(c); err != nil {
		t.Fatal(err)
	}
}

func TestRTCValidation(t *testing.T) {
	base := Config{AppID: strings.Repeat("a", 32), Certificate: strings.Repeat("b", 32), Channel: "room", UserID: "user"}
	for _, change := range []func(*Config){func(c *Config) { c.AppID = "bad" }, func(c *Config) { c.Certificate = strings.Repeat("g", 32) }, func(c *Config) { c.Channel = "" }, func(c *Config) { c.UserID = "" }, func(c *Config) { c.Lifetime = -time.Second }, func(c *Config) { c.Role = 99 }} {
		c := base
		change(&c)
		if _, err := RTC(c); err == nil {
			t.Fatal("invalid token config accepted")
		}
	}
}
