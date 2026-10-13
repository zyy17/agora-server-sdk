package main

import (
	"errors"
	"flag"
	"fmt"
	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
	"os"
)

func main() {
	example.Main(func() (err error) {
		count := flag.Int("count", 2, "independent receive-only sessions; each token must permit its user ID")
		flag.Parse()
		if *count < 1 || *count > 64 {
			return fmt.Errorf("count must be 1–64")
		}
		cfg, ch, err := example.Config()
		if err != nil {
			return err
		}
		ctx, stop := example.Context()
		defer stop()
		svc, err := sdk.NewService(cfg)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, svc.Close()) }()
		for i := 0; i < *count; i++ {
			join := ch
			join.UserID = fmt.Sprintf("%s-%d", ch.UserID, i)
			join.Token = os.Getenv(fmt.Sprintf("AGORA_TOKEN_%d", i))
			join.Role = sdk.Audience
			if _, err := svc.Join(ctx, join); err != nil {
				return err
			}
		}
		<-ctx.Done()
		return ctx.Err()
	})
}
