package agorasdk

import (
	"context"
	"io"
	"sync"
)

// Stream is a bounded, multi-consumer queue. Slow consumers drop the oldest item.
// Each item belongs to the consumer; native buffers are never exposed.
type Stream[T any] struct {
	mu      sync.Mutex
	items   chan T
	closed  bool
	dropped uint64
}

// StreamStats is a point-in-time queue snapshot.
type StreamStats struct {
	Capacity, Buffered int
	Dropped            uint64
	Closed             bool
}

func newStream[T any](capacity int) *Stream[T] { return &Stream[T]{items: make(chan T, capacity)} }

func (s *Stream[T]) push(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	select {
	case s.items <- v:
		return
	default:
	}
	select {
	case <-s.items:
		s.dropped++
	default:
	}
	select {
	case s.items <- v:
	default:
		s.dropped++
	}
}

func (s *Stream[T]) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.items)
	}
}

// Next waits for an item or cancellation. Buffered items are drained before io.EOF.
func (s *Stream[T]) Next(ctx context.Context) (T, error) {
	var zero T
	if ctx == nil {
		return zero, invalid("context", "must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case v, ok := <-s.items:
		if !ok {
			return zero, io.EOF
		}
		return v, nil
	}
}

func (s *Stream[T]) Stats() StreamStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return StreamStats{cap(s.items), len(s.items), s.dropped, s.closed}
}
