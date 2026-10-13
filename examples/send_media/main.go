// Decode a local media file with the FFmpeg command-line tool and send paced PCM
// plus raw video. This example needs ffmpeg at runtime, not FFmpeg cgo headers.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
	"io"
	"os"
	"os/exec"
	"time"
)

func main() { example.Main(run) }
func run() error {
	input := flag.String("input", "", "local video file with an audio stream, e.g. MP4")
	width := flag.Int("width", 640, "output width")
	height := flag.Int("height", 360, "output height")
	fps := flag.Int("fps", 15, "output FPS")
	flag.Parse()
	if *input == "" || *width < 2 || *height < 2 || *width > 8192 || *height > 8192 || *width%2 != 0 || *height%2 != 0 || *fps < 1 || *fps > 120 {
		return fmt.Errorf("provide -input, even dimensions 2–8192, and FPS1–120")
	}
	if _, err := os.Stat(*input); err != nil {
		return err
	}
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		return err
	}
	return example.Run(true, func(c *sdk.ChannelConfig) { c.Video.Mode = sdk.RawVideo }, func(ctx context.Context, s *sdk.Session) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		results := make(chan error, 2)
		base := time.Now().UnixMilli()
		go func() {
			results <- pipe(ctx, binary, *input, []string{"-vn", "-f", "s16le", "-ar", "16000", "-ac", "1", "pipe:1"}, func(r io.Reader) error { return s.PlayPCMAt(ctx, r, base) })
		}()
		go func() {
			results <- pipe(ctx, binary, *input, []string{"-an", "-f", "rawvideo", "-pix_fmt", "yuv420p", "-s", fmt.Sprintf("%dx%d", *width, *height), "-r", fmt.Sprint(*fps), "pipe:1"}, func(r io.Reader) error {
				buf := make([]byte, *width**height*3/2)
				tick := time.NewTicker(time.Second / time.Duration(*fps))
				defer tick.Stop()
				var index int64
				for {
					if _, err := io.ReadFull(r, buf); err != nil {
						if errors.Is(err, io.EOF) {
							return nil
						}
						return err
					}
					if err := s.SendVideo(sdk.VideoFrame{Data: buf, Width: *width, Height: *height, Format: sdk.I420, TimestampMillis: base + index*1000/int64(*fps)}); err != nil {
						return err
					}
					index++
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-tick.C:
					}
				}
			})
		}()
		var combined error
		for i := 0; i < 2; i++ {
			if err := <-results; err != nil {
				combined = errors.Join(combined, err)
				cancel()
			}
		}
		if combined != nil {
			return combined
		}
		return s.WaitPCM(ctx)
	})
}
func pipe(ctx context.Context, binary, input string, output []string, consume func(io.Reader) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	args := append([]string{"-nostdin", "-hide_banner", "-loglevel", "error", "-i", input}, output...)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	err = consume(stdout)
	if err != nil {
		cancel()
	}
	return errors.Join(err, cmd.Wait())
}
