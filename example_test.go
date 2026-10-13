package agorasdk_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	agorasdk "github.com/zyy17/agora-server-sdk"
)

// This example is entirely local and does not initialize the native SDK.
func ExamplePCMFormat() {
	format := agorasdk.PCMFormat{SampleRate: 16000, Channels: 1}
	tenMilliseconds := make([]byte, format.SampleRate/100*format.Channels*2)
	fmt.Println(len(tenMilliseconds))
	// Output: 320
}

func ExampleError() {
	var err error = &agorasdk.Error{Op: "example operation", Code: -1}
	var native *agorasdk.Error
	if errors.As(err, &native) {
		fmt.Println(native.Op, native.Code)
	}
	// Output: example operation -1
}

// This compile-only example requires an installed native SDK and a real App ID.
// It creates no channel connection.
func ExampleNewService() {
	service, err := agorasdk.NewService(agorasdk.Config{
		AppID: os.Getenv("AGORA_APP_ID"),
	})
	if err != nil {
		return
	}
	if err := service.Close(); err != nil {
		fmt.Println(err)
	}
}

// This compile-only example shows ownership for an already configured Service.
// The application supplies valid channel credentials and consumes Events in
// addition to audio; no connection is opened by the test suite.
func ExampleService_Join() {
	receive := func(ctx context.Context, service *agorasdk.Service, cfg agorasdk.ChannelConfig) (err error) {
		cfg.Role = agorasdk.Audience
		session, err := service.Join(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, session.Close()) }()
		for {
			frame, nextErr := session.Audio().Next(ctx)
			if errors.Is(nextErr, io.EOF) {
				return nil
			}
			if nextErr != nil {
				return nextErr
			}
			if frame.Position == agorasdk.AudioBeforeMixing {
				fmt.Println(frame.UserID, len(frame.Data))
			}
		}
	}
	_ = receive
}

// This compile-only example receives an existing Broadcaster session configured
// for 16 kHz mono PCM. The application owns the session and closes it separately.
func ExampleSession_PlayPCM() {
	play := func(ctx context.Context, session *agorasdk.Session) error {
		silence := make([]byte, 16000*2/10) // 100 ms of PCM16 mono.
		if err := session.PlayPCM(ctx, bytes.NewReader(silence)); err != nil {
			return err
		}
		return session.WaitPCM(ctx) // Estimate only, not a remote acknowledgement.
	}
	_ = play
}
