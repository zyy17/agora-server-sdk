package main

import (
	"context"
	"errors"
	"io"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/mediaio"
)

func (o mediaOptions) sendYUV(ctx context.Context, session *sdk.Session, input io.Reader) error {
	start := time.Now()
	for index := int64(0); ; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		data := make([]byte, o.width*o.height*3/2)
		if _, err := io.ReadFull(input, data); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		at := time.Duration(index) * time.Second / time.Duration(o.fps)
		if err := waitUntil(ctx, start.Add(at)); err != nil {
			return err
		}
		frame := sdk.VideoFrame{
			Data: data, Width: o.width, Height: o.height,
			Format: sdk.I420, TimestampMillis: at.Milliseconds(),
		}
		if err := session.SendVideo(frame); err != nil {
			return err
		}
	}
}

func (o mediaOptions) sendH264(ctx context.Context, session *sdk.Session, input io.Reader) error {
	start := time.Now()
	for index := int64(0); ; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		data, err := mediaio.ReadPacket(input)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		at := time.Duration(index) * time.Second / time.Duration(o.fps)
		if err := waitUntil(ctx, start.Add(at)); err != nil {
			return err
		}
		frame := sdk.EncodedVideoFrame{
			Data: data, Codec: sdk.H264, Width: o.width, Height: o.height, FPS: o.fps,
			KeyFrame: isH264Keyframe(data), TimestampMillis: at.Milliseconds(),
			PresentationMillis: at.Milliseconds(), SEI: []byte(o.sei),
		}
		if o.mode == "transcode-h264" {
			err = session.TranscodeVideo(frame)
		} else {
			err = session.SendEncodedVideo(frame)
		}
		if err != nil {
			return err
		}
	}
}

func sendAAC(ctx context.Context, session *sdk.Session, input io.Reader) error {
	start := time.Now()
	var samples int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		packet, err := mediaio.ReadADTS(input)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		at := time.Duration(samples) * time.Second / time.Duration(packet.SampleRate)
		if err := waitUntil(ctx, start.Add(at)); err != nil {
			return err
		}
		frame := sdk.EncodedAudioFrame{
			Data: packet.Data, Codec: sdk.AAC, SampleRate: packet.SampleRate,
			Channels: packet.Channels, SamplesPerChannel: packet.SamplesPerChannel,
			TimestampMillis: at.Milliseconds(),
		}
		if err := session.SendEncodedAudio(frame); err != nil {
			return err
		}
		samples += int64(packet.SamplesPerChannel)
	}
}

func (o mediaOptions) sendOpus(ctx context.Context, session *sdk.Session, input io.Reader) error {
	start := time.Now()
	for index := int64(0); ; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		data, err := mediaio.ReadPacket(input)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		at := time.Duration(index) * 20 * time.Millisecond
		if err := waitUntil(ctx, start.Add(at)); err != nil {
			return err
		}
		if o.mode == "transcode-opus" {
			err = session.TranscodeOpus(data, o.channels)
		} else {
			err = session.SendEncodedAudio(sdk.EncodedAudioFrame{
				Data: data, Codec: sdk.Opus, SampleRate: o.rate, Channels: o.channels,
				SamplesPerChannel: o.rate / 50, TimestampMillis: at.Milliseconds(), Speech: true,
			})
		}
		if err != nil {
			return err
		}
	}
}

func waitUntil(ctx context.Context, at time.Time) error {
	delay := time.Until(at)
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

func isH264Keyframe(data []byte) bool {
	for i := 0; i+3 < len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 && data[i+3]&31 == 5 {
			return true
		}
	}
	return false
}
