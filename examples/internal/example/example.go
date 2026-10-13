// Package example contains shared setup for the executable examples.
package example

import (
	"context"
	"errors"
	"fmt"
	sdk "github.com/zyy17/agora-server-sdk"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Main(run func() error) {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		log.Print(err)
		os.Exit(1)
	}
}
func Config() (sdk.Config, sdk.ChannelConfig, error) {
	appID, uid := os.Getenv("AGORA_APP_ID"), os.Getenv("AGORA_USER_ID")
	if appID == "" || uid == "" {
		return sdk.Config{}, sdk.ChannelConfig{}, fmt.Errorf("set AGORA_APP_ID and a unique AGORA_USER_ID; set AGORA_TOKEN if the project requires authentication")
	}
	channel := os.Getenv("AGORA_CHANNEL")
	if channel == "" {
		channel = "go-sdk-example"
	}
	return sdk.Config{AppID: appID}, sdk.ChannelConfig{Channel: channel, UserID: uid, Token: os.Getenv("AGORA_TOKEN")}, nil
}
func Context() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
func Run(video bool, configure func(*sdk.ChannelConfig), run func(context.Context, *sdk.Session) error) (err error) {
	cfg, channel, err := Config()
	if err != nil {
		return err
	}
	cfg.Video = video
	if configure != nil {
		configure(&channel)
	}
	ctx, stop := Context()
	defer stop()
	service, err := sdk.NewService(cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, service.Close()) }()
	session, err := service.Join(ctx, channel)
	if err != nil {
		return err
	}
	return run(ctx, session)
}
