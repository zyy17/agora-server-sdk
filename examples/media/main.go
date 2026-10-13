package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
	"github.com/zyy17/agora-server-sdk/mediaio"
	"io"
	"os"
	"time"
)

func main() { example.Main(run) }
func run() error {
	mode := flag.String("mode", "send-yuv", "send-yuv, recv-yuv, send-h264, recv-h264, send-aac, send-opus, recv-encoded-audio, transcode-h264, transcode-opus")
	input := flag.String("input", "", "input YUV420p, AAC-LC ADTS, or length-prefixed encoded packets")
	output := flag.String("output", "received.bin", "output YUV420p or length-prefixed encoded packets")
	width := flag.Int("width", 640, "video width")
	height := flag.Int("height", 360, "video height")
	fps := flag.Int("fps", 15, "video framerate")
	rate := flag.Int("rate", 48000, "encoded Opus sample rate")
	channels := flag.Int("channels", 1, "encoded Opus channels")
	user := flag.String("user", "", "required remote user for receive modes; prevents mixing independent bitstreams")
	sei := flag.String("sei", "", "optional SEI payload on outgoing H264 packets")
	flag.Parse()
	if *width < 2 || *height < 2 || *width > 8192 || *height > 8192 || *width%2 != 0 || *height%2 != 0 || *fps < 1 || *fps > 120 {
		return fmt.Errorf("video requires even dimensions 2–8192 and FPS 1–120")
	}
	video := false
	var configure func(*sdk.ChannelConfig)
	switch *mode {
	case "send-yuv":
		video = true
		configure = func(c *sdk.ChannelConfig) { c.Video.Mode = sdk.RawVideo }
	case "recv-yuv":
		video = true
		configure = func(c *sdk.ChannelConfig) { c.Role = sdk.Audience; c.Receive.Video = true }
	case "send-h264":
		video = true
		configure = func(c *sdk.ChannelConfig) { c.Video.Mode = sdk.EncodedVideo }
	case "recv-h264":
		video = true
		configure = func(c *sdk.ChannelConfig) { c.Role = sdk.Audience; c.Receive.EncodedVideo = true }
	case "send-aac", "send-opus":
		configure = func(c *sdk.ChannelConfig) { c.Audio.Mode = sdk.EncodedAudio }
	case "recv-encoded-audio":
		configure = func(c *sdk.ChannelConfig) { c.Role = sdk.Audience; c.Receive.EncodedAudio = true }
	case "transcode-h264":
		video = true
		configure = func(c *sdk.ChannelConfig) { c.Video.Mode = sdk.RawVideo }
	case "transcode-opus":
		configure = func(c *sdk.ChannelConfig) { c.Audio.Mode = sdk.PCM }
	default:
		return fmt.Errorf("unknown mode %q", *mode)
	}
	receiving := *mode == "recv-yuv" || *mode == "recv-h264" || *mode == "recv-encoded-audio"
	var file *os.File
	var err error
	if receiving {
		if *user == "" {
			return fmt.Errorf("receive modes require -user to keep streams separate")
		}
		file, err = os.Create(*output)
	} else {
		if *input == "" {
			return fmt.Errorf("send modes require -input")
		}
		file, err = os.Open(*input)
	}
	if err != nil {
		return err
	}
	defer file.Close()
	return example.Run(video, configure, func(ctx context.Context, s *sdk.Session) error {
		var frameIndex int64
		start := time.Now()
		wait := func(at time.Duration) error {
			delay := time.Until(start.Add(at))
			if delay <= 0 {
				return ctx.Err()
			}
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			switch *mode {
			case "recv-yuv":
				f, err := s.Video().Next(ctx)
				if err != nil {
					return err
				}
				if f.UserID != *user {
					continue
				}
				for _, plane := range []struct {
					data                  []byte
					stride, width, height int
				}{{f.Y, f.YStride, f.Width, f.Height}, {f.U, f.UStride, (f.Width + 1) / 2, (f.Height + 1) / 2}, {f.V, f.VStride, (f.Width + 1) / 2, (f.Height + 1) / 2}} {
					for y := 0; y < plane.height; y++ {
						offset := y * plane.stride
						if offset+plane.width > len(plane.data) {
							return fmt.Errorf("invalid received video stride")
						}
						if _, err := file.Write(plane.data[offset : offset+plane.width]); err != nil {
							return err
						}
					}
				}
			case "recv-h264":
				f, err := s.EncodedVideo().Next(ctx)
				if err != nil {
					return err
				}
				if f.UserID != *user {
					continue
				}
				if err := mediaio.WritePacket(file, f.Frame.Data); err != nil {
					return err
				}
			case "recv-encoded-audio":
				f, err := s.EncodedAudio().Next(ctx)
				if err != nil {
					return err
				}
				if f.UserID != *user {
					continue
				}
				if err := mediaio.WritePacket(file, f.Data); err != nil {
					return err
				}
			case "send-yuv":
				data := make([]byte, *width**height*3/2)
				if _, err := io.ReadFull(file, data); err != nil {
					if errors.Is(err, io.EOF) {
						return nil
					}
					return err
				}
				at := time.Duration(frameIndex) * time.Second / time.Duration(*fps)
				if err := wait(at); err != nil {
					return err
				}
				if err := s.SendVideo(sdk.VideoFrame{Data: data, Width: *width, Height: *height, Format: sdk.I420, TimestampMillis: at.Milliseconds()}); err != nil {
					return err
				}
				frameIndex++
			case "send-h264", "transcode-h264":
				data, err := mediaio.ReadPacket(file)
				if errors.Is(err, io.EOF) {
					return nil
				}
				if err != nil {
					return err
				}
				at := time.Duration(frameIndex) * time.Second / time.Duration(*fps)
				if err := wait(at); err != nil {
					return err
				}
				frame := sdk.EncodedVideoFrame{Data: data, Codec: sdk.H264, Width: *width, Height: *height, FPS: *fps, KeyFrame: isH264Keyframe(data), TimestampMillis: at.Milliseconds(), PresentationMillis: at.Milliseconds(), SEI: []byte(*sei)}
				if *mode == "transcode-h264" {
					err = s.TranscodeVideo(frame)
				} else {
					err = s.SendEncodedVideo(frame)
				}
				if err != nil {
					return err
				}
				frameIndex++
			case "send-aac":
				f, err := mediaio.ReadADTS(file)
				if errors.Is(err, io.EOF) {
					return nil
				}
				if err != nil {
					return err
				}
				at := time.Duration(frameIndex) * time.Second / time.Duration(f.SampleRate)
				if err := wait(at); err != nil {
					return err
				}
				if err := s.SendEncodedAudio(sdk.EncodedAudioFrame{Data: f.Data, Codec: sdk.AAC, SampleRate: f.SampleRate, Channels: f.Channels, SamplesPerChannel: f.SamplesPerChannel, TimestampMillis: at.Milliseconds()}); err != nil {
					return err
				}
				frameIndex += int64(f.SamplesPerChannel)
			case "send-opus", "transcode-opus":
				data, err := mediaio.ReadPacket(file)
				if errors.Is(err, io.EOF) {
					return nil
				}
				if err != nil {
					return err
				}
				at := time.Duration(frameIndex) * 20 * time.Millisecond
				if err := wait(at); err != nil {
					return err
				}
				if *mode == "transcode-opus" {
					err = s.TranscodeOpus(data, *channels)
				} else {
					err = s.SendEncodedAudio(sdk.EncodedAudioFrame{Data: data, Codec: sdk.Opus, SampleRate: *rate, Channels: *channels, SamplesPerChannel: *rate / 50, TimestampMillis: at.Milliseconds(), Speech: true})
				}
				if err != nil {
					return err
				}
				frameIndex++
			}
		}
	})
}
func isH264Keyframe(data []byte) bool {
	for i := 0; i+3 < len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 && data[i+3]&31 == 5 {
			return true
		}
	}
	return false
}
