package main

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestStreamToneChunksAndOpaquePTS(t *testing.T) {
	var chunks []int
	var joined []byte
	var waited time.Duration
	const pts int64 = -123
	err := streamTone(context.Background(), 41*time.Millisecond, pts, func(data []byte, got int64) error {
		if got != pts {
			t.Fatalf("PTS changed: %d", got)
		}
		chunks = append(chunks, len(data))
		joined = append(joined, data...)
		return nil
	}, func(_ context.Context, d time.Duration) error { waited += d; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(chunks, []int{448, 832, 1280, 64}) || waited != 41*time.Millisecond {
		t.Fatalf("chunks=%v waited=%v", chunks, waited)
	}
	if !bytes.Equal(joined, stereoTone(0, 41*16)) {
		t.Fatal("sample continuity changed at a chunk boundary")
	}
}

func TestStreamToneCancellationAndSendFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sends := 0
	err := streamTone(ctx, time.Second, 0, func([]byte, int64) error { sends++; return nil }, func(ctx context.Context, _ time.Duration) error {
		cancel()
		return waitContext(ctx, time.Hour)
	})
	if !errors.Is(err, context.Canceled) || sends != 1 {
		t.Fatalf("err=%v sends=%d", err, sends)
	}
	want := errors.New("send failed")
	err = streamTone(context.Background(), time.Second, 0, func([]byte, int64) error { return want }, func(context.Context, time.Duration) error {
		t.Fatal("wait after failed send")
		return nil
	})
	if !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}

func TestDurationValidation(t *testing.T) {
	for _, value := range []time.Duration{0, -time.Millisecond, time.Microsecond, time.Minute + time.Millisecond} {
		if validateDuration(value) == nil {
			t.Fatalf("accepted %v", value)
		}
	}
}
