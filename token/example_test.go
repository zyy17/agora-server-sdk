package token_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/zyy17/agora-server-sdk/token"
)

func ExampleRTC() {
	// These are synthetic test values, not credentials for a real Agora project.
	// Load real credentials securely on a trusted server.
	value, err := token.RTC(token.Config{
		AppID:       strings.Repeat("a", 32),
		Certificate: strings.Repeat("b", 32),
		Channel:     "room",
		UserID:      "listener",
		Role:        token.Subscriber,
		Lifetime:    15 * time.Minute,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	// Report success without printing the generated credential.
	fmt.Println("token created:", value != "")
	// Output: token created: true
}

func ExampleRTC_validation() {
	// Shape-valid, synthetic values let this example test the channel check.
	// An empty channel is invalid; no token is produced.
	_, err := token.RTC(token.Config{
		AppID:       strings.Repeat("a", 32),
		Certificate: strings.Repeat("b", 32),
		UserID:      "listener",
	})
	fmt.Println(err)
	// Output: token: invalid channel or user ID
}
