package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	sdk "github.com/zyy17/agora-server-sdk"
)

func TestSendVideoFramesAndTimestamps(t *testing.T) {
	source := mediaSource{width: 2, height: 2, fps: 120}
	input := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	frames := 0
	err := source.sendVideo(context.Background(), bytes.NewReader(input), 1000, func(frame sdk.VideoFrame) error {
		if frame.Width != 2 || frame.Height != 2 || frame.Format != sdk.I420 || frame.TimestampMillis != 1000+int64(frames)*1000/120 {
			t.Fatalf("unexpected frame: %+v", frame)
		}
		if !bytes.Equal(frame.Data, input[frames*6:(frames+1)*6]) {
			t.Fatalf("unexpected frame data: %v", frame.Data)
		}
		frames++
		return nil
	})
	if err != nil || frames != 2 {
		t.Fatalf("frames=%d err=%v", frames, err)
	}
}

func TestSendVideoRejectsPartialFrame(t *testing.T) {
	source := mediaSource{width: 2, height: 2, fps: 120}
	err := source.sendVideo(context.Background(), bytes.NewReader(make([]byte, 5)), 0, func(sdk.VideoFrame) error {
		t.Fatal("partial frame sent")
		return nil
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v", err)
	}
}

func TestSendVideoCancellationAndSendFailure(t *testing.T) {
	source := mediaSource{width: 2, height: 2, fps: 120}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	frames := 0
	err := source.sendVideo(ctx, bytes.NewReader(make([]byte, 12)), 0, func(sdk.VideoFrame) error {
		frames++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || frames != 1 {
		t.Fatalf("frames=%d err=%v", frames, err)
	}
	want := errors.New("send failed")
	err = source.sendVideo(context.Background(), bytes.NewReader(make([]byte, 6)), 0, func(sdk.VideoFrame) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
}
