package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestProcessFramesFormats(t *testing.T) {
	for _, channels := range []int{1, 2} {
		input := bytes.Repeat([]byte{1, 2, 3, 4}, 160*channels)
		frames := 0
		err := processFrames(context.Background(), bytes.NewReader(input), channels, func(frame *rtc.AudioFrame) error {
			if frame.Type != rtc.AudioFrameTypePCM16 || frame.SamplesPerSec != 16000 || frame.Channels != channels || frame.SamplesPerChannel != 160 || frame.BytesPerSample != 2 || frame.RenderTimeMs != int64(frames*10) {
				t.Fatalf("unexpected frame: %+v", frame)
			}
			if !bytes.Equal(frame.Buffer, input[frames*160*channels*2:(frames+1)*160*channels*2]) {
				t.Fatal("PCM bytes changed")
			}
			frames++
			return nil
		})
		if err != nil || frames != 2 {
			t.Fatalf("channels=%d frames=%d err=%v", channels, frames, err)
		}
	}
}

func TestProcessFramesRejectsPartialAndInvalidInput(t *testing.T) {
	for _, channels := range []int{1, 2} {
		err := processFrames(context.Background(), bytes.NewReader(make([]byte, 160*channels*2-1)), channels, func(*rtc.AudioFrame) error {
			t.Fatal("partial frame consumed")
			return nil
		})
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("err=%v", err)
		}
	}
	if processFrames(context.Background(), bytes.NewReader(nil), 3, nil) == nil {
		t.Fatal("invalid channel count accepted")
	}
}

func TestProcessFramesCancellationAndConsumerFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	frames := 0
	err := processFrames(ctx, bytes.NewReader(make([]byte, 640)), 1, func(*rtc.AudioFrame) error {
		frames++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || frames != 1 {
		t.Fatalf("frames=%d err=%v", frames, err)
	}
	want := errors.New("consumer failed")
	err = processFrames(context.Background(), bytes.NewReader(make([]byte, 320)), 1, func(*rtc.AudioFrame) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}
