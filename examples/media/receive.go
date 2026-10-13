package main

import (
	"context"
	"fmt"
	"io"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/mediaio"
)

func receiveYUV(ctx context.Context, session *sdk.Session, output io.Writer, user string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		frame, err := session.Video().Next(ctx)
		if err != nil {
			return err
		}
		if frame.UserID == user {
			if err := writeYUV(output, frame); err != nil {
				return err
			}
		}
	}
}

func writeYUV(output io.Writer, frame sdk.ReceivedVideoFrame) error {
	planes := []struct {
		data                  []byte
		stride, width, height int
	}{
		{frame.Y, frame.YStride, frame.Width, frame.Height},
		{frame.U, frame.UStride, (frame.Width + 1) / 2, (frame.Height + 1) / 2},
		{frame.V, frame.VStride, (frame.Width + 1) / 2, (frame.Height + 1) / 2},
	}
	for _, plane := range planes {
		for y := 0; y < plane.height; y++ {
			offset := y * plane.stride
			if offset+plane.width > len(plane.data) {
				return fmt.Errorf("invalid received video stride")
			}
			if _, err := output.Write(plane.data[offset : offset+plane.width]); err != nil {
				return err
			}
		}
	}
	return nil
}

func receiveH264(ctx context.Context, session *sdk.Session, output io.Writer, user string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		frame, err := session.EncodedVideo().Next(ctx)
		if err != nil {
			return err
		}
		if frame.UserID == user {
			if err := mediaio.WritePacket(output, frame.Frame.Data); err != nil {
				return err
			}
		}
	}
}

func receiveEncodedAudio(ctx context.Context, session *sdk.Session, output io.Writer, user string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		frame, err := session.EncodedAudio().Next(ctx)
		if err != nil {
			return err
		}
		if frame.UserID == user {
			if err := mediaio.WritePacket(output, frame.Data); err != nil {
				return err
			}
		}
	}
}
