package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

type mediaOptions struct {
	mode, input, output, user, sei string
	width, height, fps             int
	rate, channels                 int
}

func main() { example.Main(run) }

func run() error {
	var options mediaOptions
	flag.StringVar(&options.mode, "mode", "send-yuv", "send-yuv, recv-yuv, send-h264, recv-h264, send-aac, send-opus, recv-encoded-audio, transcode-h264, transcode-opus")
	flag.StringVar(&options.input, "input", "", "input YUV420p, AAC-LC ADTS, or length-prefixed encoded packets")
	flag.StringVar(&options.output, "output", "received.bin", "output YUV420p or length-prefixed encoded packets")
	flag.IntVar(&options.width, "width", 640, "video width")
	flag.IntVar(&options.height, "height", 360, "video height")
	flag.IntVar(&options.fps, "fps", 15, "video framerate")
	flag.IntVar(&options.rate, "rate", 48000, "encoded Opus sample rate")
	flag.IntVar(&options.channels, "channels", 1, "encoded Opus channels")
	flag.StringVar(&options.user, "user", "", "required remote user for receive modes; prevents mixing independent bitstreams")
	flag.StringVar(&options.sei, "sei", "", "optional SEI payload on outgoing H264 packets")
	flag.Parse()
	video, configure, err := options.channelSetup()
	if err != nil {
		return err
	}
	file, err := options.openFile()
	if err != nil {
		return err
	}
	defer file.Close()
	return example.Run(video, configure, func(ctx context.Context, session *sdk.Session) error {
		return options.stream(ctx, session, file)
	})
}

func (o mediaOptions) channelSetup() (bool, func(*sdk.ChannelConfig), error) {
	if o.width < 2 || o.height < 2 || o.width > 8192 || o.height > 8192 || o.width%2 != 0 || o.height%2 != 0 || o.fps < 1 || o.fps > 120 {
		return false, nil, fmt.Errorf("video requires even dimensions 2–8192 and FPS 1–120")
	}
	switch o.mode {
	case "send-yuv", "transcode-h264":
		return true, func(c *sdk.ChannelConfig) { c.Video.Mode = sdk.RawVideo }, nil
	case "recv-yuv":
		return true, func(c *sdk.ChannelConfig) {
			c.Role = sdk.Audience
			c.Receive.Video = true
		}, nil
	case "send-h264":
		return true, func(c *sdk.ChannelConfig) { c.Video.Mode = sdk.EncodedVideo }, nil
	case "recv-h264":
		return true, func(c *sdk.ChannelConfig) {
			c.Role = sdk.Audience
			c.Receive.EncodedVideo = true
		}, nil
	case "send-aac", "send-opus":
		return false, func(c *sdk.ChannelConfig) { c.Audio.Mode = sdk.EncodedAudio }, nil
	case "recv-encoded-audio":
		return false, func(c *sdk.ChannelConfig) {
			c.Role = sdk.Audience
			c.Receive.EncodedAudio = true
		}, nil
	case "transcode-opus":
		return false, func(c *sdk.ChannelConfig) { c.Audio.Mode = sdk.PCM }, nil
	default:
		return false, nil, fmt.Errorf("unknown mode %q", o.mode)
	}
}

func (o mediaOptions) openFile() (*os.File, error) {
	switch o.mode {
	case "recv-yuv", "recv-h264", "recv-encoded-audio":
		if o.user == "" {
			return nil, fmt.Errorf("receive modes require -user to keep streams separate")
		}
		return os.Create(o.output)
	default:
		if o.input == "" {
			return nil, fmt.Errorf("send modes require -input")
		}
		return os.Open(o.input)
	}
}

func (o mediaOptions) stream(ctx context.Context, session *sdk.Session, file *os.File) error {
	switch o.mode {
	case "recv-yuv":
		return receiveYUV(ctx, session, file, o.user)
	case "recv-h264":
		return receiveH264(ctx, session, file, o.user)
	case "recv-encoded-audio":
		return receiveEncodedAudio(ctx, session, file, o.user)
	case "send-yuv":
		return o.sendYUV(ctx, session, file)
	case "send-h264", "transcode-h264":
		return o.sendH264(ctx, session, file)
	case "send-aac":
		return sendAAC(ctx, session, file)
	case "send-opus", "transcode-opus":
		return o.sendOpus(ctx, session, file)
	default:
		return fmt.Errorf("unknown mode %q", o.mode)
	}
}
