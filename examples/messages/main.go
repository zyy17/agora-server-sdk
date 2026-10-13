package main

import (
	"context"
	"flag"
	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
	"log"
)

func main() {
	example.Main(func() error {
		text := flag.String("send", "hello from Go", "initial RTC data message")
		flag.Parse()
		return example.Run(false, nil, func(ctx context.Context, s *sdk.Session) error {
			if err := s.SendMessage([]byte(*text)); err != nil {
				return err
			}
			for {
				message, err := s.Messages().Next(ctx)
				if err != nil {
					return err
				}
				log.Printf("channel=%s user=%s sent_ms=%d data=%q", message.Channel, message.UserID, message.SentAtMillis, message.Data)
			}
		})
	})
}
