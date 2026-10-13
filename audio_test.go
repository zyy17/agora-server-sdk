package agorasdk

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
)

func TestPlayPCMPaddingAndTimestamp(t *testing.T) {
	var frames [][]byte
	var timestamps []int64
	input := make([]byte, 322)
	input[320] = 7
	err := playPCM(context.Background(), bytes.NewReader(input), 16000, 1, 1000, func(p []byte, pts int64) error {
		frames = append(frames, slices.Clone(p))
		timestamps = append(timestamps, pts)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || len(frames[1]) != 320 || frames[1][0] != 7 || frames[1][2] != 0 || timestamps[0] != 1000 || timestamps[1] != 1010 {
		t.Fatal("incorrect paced frames", timestamps)
	}
}
func TestPlayPCMCancellationStopsFutureFrames(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sent := 0
	err := playPCM(ctx, bytes.NewReader(make([]byte, 960)), 16000, 1, 0, func([]byte, int64) error { sent++; cancel(); return nil })
	if !errors.Is(err, context.Canceled) || sent != 1 {
		t.Fatal(sent, err)
	}
}
func TestPlayPCMRejectsIncompleteSample(t *testing.T) {
	err := playPCM(context.Background(), bytes.NewReader([]byte{1}), 16000, 1, 0, func([]byte, int64) error { t.Fatal("sent partial sample"); return nil })
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}
