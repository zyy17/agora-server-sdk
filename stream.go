package agorasdk

import (
	"context"
	"io"
	"sync"
)

// Stream is a bounded concurrent queue returned by a Session or AudioProcessor.
// Its zero value is not usable. Each item is consumed once; multiple readers
// compete rather than receive a broadcast. On overflow the oldest buffered item
// is discarded so native callbacks do not wait for readers.
//
// High-level receive payloads own their Go memory and remain valid after the
// callback and resource closure. Sharing or mutating a received value across
// goroutines still requires application synchronization.
type Stream[T any] struct {
	mu      sync.Mutex
	items   chan T
	closed  bool
	dropped uint64
}

// StreamStats is a point-in-time queue snapshot. Counts refer to items, not bytes
// or milliseconds; the queue may change immediately after the snapshot.
type StreamStats struct {
	// Capacity and Buffered are the queue's maximum and currently buffered item counts.
	Capacity, Buffered int
	// Dropped counts discarded items over the lifetime of the queue.
	Dropped uint64
	// Closed reports that no further items will be enqueued; buffered items may remain.
	Closed bool
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

// Next waits for one item or ctx cancellation. ctx must not be nil.
// With a live context it drains buffered items after closure, then returns io.EOF.
// An already canceled context returns ctx.Err without consuming an item.
// If cancellation races with an available item, either result may be selected.
// Cancellation affects this read only; it does not close the queue or its owner.
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

// Stats returns a synchronized snapshot of queue capacity, occupancy, drops,
// and closure. Use Dropped to detect loss; a bounded queue is not a lossless log.
func (s *Stream[T]) Stats() StreamStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return StreamStats{cap(s.items), len(s.items), s.dropped, s.closed}
}
