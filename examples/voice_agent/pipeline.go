package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	inputBytes = 3200 // 100 ms, PCM16LE, 16 kHz mono.
	frameBytes = 640  // 20 ms, PCM16LE, 16 kHz mono.
)

func enqueueMicrophone(output chan<- []byte, pcm []byte) error {
	if len(pcm) > inputBytes*10 {
		return errors.New("microphone frame exceeds one second")
	}
	select {
	case output <- pcm:
		return nil
	default:
		return errors.New("microphone queue overflow; restart a fresh ASR session")
	}
}

func feedMicrophone(ctx context.Context, ready <-chan struct{}, input <-chan []byte, output chan<- []byte, ticks <-chan time.Time) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ready:
	}
	pending := make([]byte, 0, inputBytes*10)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pcm := <-input:
			if len(pending)+len(pcm) > inputBytes*10 {
				return errors.New("microphone backlog exceeds one second; restart ASR")
			}
			pending = append(pending, pcm...)
		case <-ticks:
			chunk := make([]byte, inputBytes)
			n := copy(chunk, pending)
			pending = append(pending[:0], pending[n:]...)
			// Zero padding accounts for RTC DTX/silence. Every real byte is
			// consumed once; previous speech is never reused as a keepalive.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case output <- chunk:
			default:
				return errors.New("ASR writer fell behind; refusing to drop microphone audio")
			}
		}
	}
}

type audioPacket struct {
	generation uint64
	data       []byte
}

// audioGate serializes generation changes, native sends and native interruption.
// Cancellation alone cannot stop a sender that already selected a queued packet.
type audioGate struct {
	mu         sync.Mutex
	generation uint64
	sent       bool
	send       func([]byte) error
	flush      func() error
}

func (g *audioGate) replace() (uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.generation++
	if g.sent {
		g.sent = false
		return g.generation, g.flush()
	}
	return g.generation, nil
}

func (g *audioGate) current(generation uint64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return generation == g.generation
}

func (g *audioGate) write(packet audioPacket) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if packet.generation != g.generation {
		return nil
	}
	if err := g.send(packet.data); err != nil {
		return err
	}
	g.sent = true
	return nil
}

func coordinate(ctx context.Context, input <-chan transcript, ready chan struct{}, gate *audioGate, respond func(context.Context, uint64, string) error) (err error) {
	var turnCancel context.CancelFunc
	var turnDone chan error
	stopTurn := func() error {
		if turnCancel != nil {
			turnCancel()
		}
		_, err := gate.replace()
		if turnDone != nil {
			<-turnDone
		}
		turnCancel, turnDone = nil, nil
		return err
	}
	defer func() {
		err = errors.Join(err, stopTurn())
		if cause := context.Cause(ctx); cause != nil && !errors.Is(err, cause) {
			err = errors.Join(err, cause)
		}
	}()
	started, segment := false, false
	partial := ""
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case err := <-turnDone:
			turnCancel()
			turnCancel, turnDone = nil, nil
			if err != nil {
				return err
			}
		case t := <-input:
			if t.ready {
				if !started {
					close(ready)
					started = true
					log.Print("ASR ready; the selected peer can speak now (transcripts appear in this terminal)")
				}
				continue
			}
			text := strings.TrimSpace(t.text)
			if !t.final {
				if text != "" && !segment {
					segment = true
					if err := stopTurn(); err != nil {
						return err
					}
					log.Print("speech detected; previous reply canceled")
				}
				if text != "" && text != partial {
					log.Printf("partial: %s", text)
					partial = text
				}
				continue // Partial text replaces prior text; it is not a delta.
			}
			segment, partial = false, ""
			if text == "" {
				continue
			}
			if err := stopTurn(); err != nil {
				return err
			}
			generation, err := gate.replace()
			if err != nil {
				return err
			}
			turnCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			turnCancel = cancel
			turnDone = make(chan error, 1)
			log.Printf("user: %s", text)
			go func(done chan<- error) { done <- respond(turnCtx, generation, text) }(turnDone)
		}
	}
}

func reply(ctx context.Context, complete func(context.Context, string, func(string) error) error, speak func(context.Context, string, func([]byte) error) error, text string, generation uint64, output chan<- audioPacket) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	phrases := make(chan string, 4)
	llmDone := make(chan error, 1)
	go func() {
		defer close(phrases)
		splitter := phraseSplitter{emit: func(phrase string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case phrases <- phrase:
				return nil
			}
		}}
		err := complete(ctx, text, splitter.add)
		if err == nil {
			err = splitter.finish()
		}
		if err != nil {
			cancel() // Stop speech immediately on truncated/error LLM output.
		}
		llmDone <- err
	}()
	framer := pcmFramer{emit: func(data []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case output <- audioPacket{generation, data}:
			return nil
		}
	}}
	var speechErr error
	for phrase := range phrases {
		if err := ctx.Err(); err != nil {
			speechErr = err
			break
		}
		if speechErr = speak(ctx, phrase, framer.add); speechErr != nil {
			break
		}
	}
	if speechErr != nil {
		cancel()
	}
	llmErr := <-llmDone // Always join the HTTP/SSE worker, including on barge-in.
	if speechErr != nil || llmErr != nil {
		return errors.Join(speechErr, llmErr)
	}
	return framer.finish()
}

type phraseSplitter struct {
	pending []rune
	emit    func(string) error
}

func (s *phraseSplitter) add(delta string) error {
	for _, r := range delta {
		s.pending = append(s.pending, r)
		if strings.ContainsRune("。！？.!?；;，,\n", r) || len(s.pending) >= 80 {
			if err := s.finish(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *phraseSplitter) finish() error {
	phrase := strings.TrimSpace(string(s.pending))
	s.pending = s.pending[:0]
	if phrase != "" {
		return s.emit(phrase)
	}
	return nil
}

type pcmFramer struct {
	pending []byte
	emit    func([]byte) error
}

func (f *pcmFramer) add(pcm []byte) error {
	for len(pcm) > 0 {
		n := min(frameBytes-len(f.pending), len(pcm))
		f.pending = append(f.pending, pcm[:n]...)
		pcm = pcm[n:]
		if len(f.pending) == frameBytes {
			if err := f.emit(f.pending); err != nil {
				return err
			}
			f.pending = nil // Ownership passed to the bounded playback queue.
		}
	}
	return nil
}

func (f *pcmFramer) finish() error {
	if len(f.pending)%2 != 0 {
		return errors.New("TTS ended with an incomplete PCM16 sample")
	}
	if len(f.pending) == 0 {
		return nil
	}
	frame := make([]byte, frameBytes)
	copy(frame, f.pending)
	f.pending = nil
	return f.emit(frame) // Pad the final short frame with less than 20 ms silence.
}

func playAudio(ctx context.Context, input <-chan audioPacket, gate *audioGate, now func() time.Time, wait func(context.Context, time.Duration) error) error {
	var next time.Time
	var generation uint64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case packet := <-input:
			if !gate.current(packet.generation) {
				continue
			}
			current := now()
			if generation != packet.generation || current.After(next) {
				next = current // Underruns never trigger a catch-up burst.
				generation = packet.generation
			}
			if delay := next.Sub(current); delay > 0 {
				if err := wait(ctx, delay); err != nil {
					return err
				}
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := gate.write(packet); err != nil {
				return err
			}
			next = now().Add(20 * time.Millisecond)
		}
	}
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
