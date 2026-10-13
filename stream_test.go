package agorasdk

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func TestStreamBoundedDropOldest(t *testing.T) {
	s := newStream[int](2)
	s.push(1)
	s.push(2)
	s.push(3)
	if got := s.Stats(); got.Buffered != 2 || got.Dropped != 1 {
		t.Fatalf("stats=%+v", got)
	}
	for _, want := range []int{2, 3} {
		got, err := s.Next(context.Background())
		if err != nil || got != want {
			t.Fatalf("got %d,%v want %d", got, err, want)
		}
	}
	s.close()
	s.close()
	s.push(4)
	if _, err := s.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("got %v", err)
	}
}
func TestStreamDrainsAfterClose(t *testing.T) {
	s := newStream[int](1)
	s.push(7)
	s.close()
	v, err := s.Next(context.Background())
	if v != 7 || err != nil {
		t.Fatal(v, err)
	}
}
func TestStreamCancellation(t *testing.T) {
	s := newStream[int](1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.push(7)
	if _, err := s.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	//lint:ignore SA1012 This negative test verifies that nil contexts are rejected.
	if _, err := s.Next(nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	s2 := newStream[int](1)
	if _, err := s2.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}
func TestStreamConcurrentClose(t *testing.T) {
	s := newStream[int](8)
	var wg sync.WaitGroup
	for p := 0; p < 8; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				s.push(i)
			}
		}()
	}
	for c := 0; c < 3; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if _, err := s.Next(context.Background()); err != nil {
					return
				}
			}
		}()
	}
	s.close()
	wg.Wait()
	if got := s.Stats(); got.Buffered > 8 || !got.Closed {
		t.Fatalf("stats=%+v", got)
	}
}
