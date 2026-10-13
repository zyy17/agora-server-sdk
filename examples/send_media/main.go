// Decode a local media file with the FFmpeg command-line tool and send paced PCM
// plus raw video. This example needs ffmpeg at runtime, not FFmpeg cgo headers.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

type mediaSource struct {
	input, binary      string
	width, height, fps int
}

func main() { example.Main(run) }

func run() error {
	var source mediaSource
	flag.StringVar(&source.input, "input", "", "local video file with an audio stream, e.g. MP4")
	flag.IntVar(&source.width, "width", 640, "output width")
	flag.IntVar(&source.height, "height", 360, "output height")
	flag.IntVar(&source.fps, "fps", 15, "output FPS")
	flag.Parse()
	if source.input == "" || source.width < 2 || source.height < 2 || source.width > 8192 || source.height > 8192 || source.width%2 != 0 || source.height%2 != 0 || source.fps < 1 || source.fps > 120 {
		return fmt.Errorf("provide -input, even dimensions 2–8192, and FPS1–120")
	}
	if _, err := os.Stat(source.input); err != nil {
		return err
	}
	var err error
	source.binary, err = exec.LookPath("ffmpeg")
	if err != nil {
		return err
	}
	return example.Run(true, func(c *sdk.ChannelConfig) { c.Video.Mode = sdk.RawVideo }, source.stream)
}

func (s mediaSource) stream(ctx context.Context, session *sdk.Session) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan error, 2)
	base := time.Now().UnixMilli()
	go func() {
		output := []string{"-vn", "-f", "s16le", "-ar", "16000", "-ac", "1", "pipe:1"}
		results <- pipe(ctx, s.binary, s.input, output, func(reader io.Reader) error {
			return session.PlayPCMAt(ctx, reader, base)
		})
	}()
	go func() {
		output := []string{
			"-an", "-f", "rawvideo", "-pix_fmt", "yuv420p",
			"-s", fmt.Sprintf("%dx%d", s.width, s.height), "-r", fmt.Sprint(s.fps), "pipe:1",
		}
		results <- pipe(ctx, s.binary, s.input, output, func(reader io.Reader) error {
			return s.sendVideo(ctx, reader, base, session.SendVideo)
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
	return session.WaitPCM(ctx)
}

func (s mediaSource) sendVideo(ctx context.Context, reader io.Reader, base int64, send func(sdk.VideoFrame) error) error {
	buffer := make([]byte, s.width*s.height*3/2)
	tick := time.NewTicker(time.Second / time.Duration(s.fps))
	defer tick.Stop()
	for index := int64(0); ; index++ {
		if _, err := io.ReadFull(reader, buffer); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		frame := sdk.VideoFrame{
			Data: buffer, Width: s.width, Height: s.height,
			Format: sdk.I420, TimestampMillis: base + index*1000/int64(s.fps),
		}
		if err := send(frame); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
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
