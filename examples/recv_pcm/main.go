package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

func main() { example.Main(run) }

func run() error {
	dir := flag.String("output", "recordings", "directory for per-user PCM16LE files")
	rate := flag.Int("rate", 16000, "sample rate")
	channels := flag.Int("channels", 1, "audio channels")
	flag.Parse()
	if err := os.MkdirAll(*dir, 0700); err != nil {
		return err
	}
	return example.Run(false, func(c *sdk.ChannelConfig) {
		c.Role = sdk.Audience
		c.Audio.Receive.SampleRate = *rate
		c.Audio.Receive.Channels = *channels
	}, func(ctx context.Context, s *sdk.Session) (err error) {
		files := map[string]*os.File{}
		defer func() {
			for _, file := range files {
				err = errors.Join(err, file.Close())
			}
		}()
		for {
			frame, err := s.Audio().Next(ctx)
			if err != nil {
				return err
			}
			file := files[frame.UserID]
			if file == nil {
				if len(files) >= 64 {
					return fmt.Errorf("example recording limit: 64 users")
				}
				name := hex.EncodeToString([]byte(frame.UserID)) + ".pcm"
				file, err = os.OpenFile(filepath.Join(*dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
				if err != nil {
					return err
				}
				files[frame.UserID] = file
			}
			if _, err = file.Write(frame.Data); err != nil {
				return err
			}
		}
	})
}
