package main

import (
	"context"
	"flag"
	"os"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

func main() { example.Main(run) }
func run() error {
	path := flag.String("input", "examples/testdata/send_audio_16k_1ch.pcm", "PCM16LE file")
	rate := flag.Int("rate", 16000, "sample rate in Hz")
	channels := flag.Int("channels", 1, "audio channels")
	flag.Parse()
	file, err := os.Open(*path)
	if err != nil {
		return err
	}
	defer file.Close()
	return example.Run(false, func(c *sdk.ChannelConfig) { c.Audio.Send.SampleRate = *rate; c.Audio.Send.Channels = *channels }, func(ctx context.Context, s *sdk.Session) error {
		if err := s.PlayPCM(ctx, file); err != nil {
			return err
		}
		return s.WaitPCM(ctx)
	})
}
