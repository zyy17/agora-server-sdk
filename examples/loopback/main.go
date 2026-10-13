package main

import (
	"context"
	"flag"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

func main() {
	example.Main(func() error {
		user := flag.String("user", "", "optional remote user ID filter")
		flag.Parse()
		return example.Run(false, nil, func(ctx context.Context, s *sdk.Session) error {
			for {
				frame, err := s.Audio().Next(ctx)
				if err != nil {
					return err
				}
				if *user != "" && frame.UserID != *user {
					continue
				}
				if err := s.SendPCM(frame.Data); err != nil {
					return err
				}
			}
		})
	})
}
