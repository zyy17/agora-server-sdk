package main

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func await[T any](t *testing.T, ctx context.Context, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-ctx.Done():
		t.Fatal("operation did not finish:", ctx.Err())
		var zero T
		return zero
	}
}

func TestIdentityMode(t *testing.T) {
	for _, tc := range []struct {
		mode, bot, peer string
		want            sdk.IdentityMode
		valid           bool
	}{
		{"", "100", "200", sdk.NumericIdentity, true},
		{"numeric", "100", "200", sdk.NumericIdentity, true},
		{"string", "bot", "human", sdk.StringIdentity, true},
		{"string", "001", "1", sdk.StringIdentity, true},
		{"", "0", "200", 0, false},
		{"", "100", "100", 0, false},
		{"", "100", "0200", 0, false},
		{"", "100", "4294967296", 0, false},
		{"", "100", "", 0, false},
		{"other", "100", "200", 0, false},
		{"string", "bot", "\x00", 0, false},
	} {
		got, err := identityMode(tc.mode, tc.bot, tc.peer)
		if (err == nil) != tc.valid || (tc.valid && got != tc.want) {
			t.Fatalf("%+v: got %v, %v", tc, got, err)
		}
	}
}

func TestMicrophonePaddingDoesNotRepeatAudio(t *testing.T) {
	ctx, cancel := context.WithCancel(testContext(t))
	defer cancel()
	ready := make(chan struct{})
	close(ready)
	input := make(chan []byte)
	output := make(chan []byte, 1)
	ticks := make(chan time.Time)
	done := make(chan error, 1)
	go func() { done <- feedMicrophone(ctx, ready, input, output, ticks) }()
	pcm := bytes.Repeat([]byte{1, 2}, 400)
	input <- pcm
	ticks <- time.Time{}
	first := await(t, ctx, output)
	if !bytes.Equal(first[:len(pcm)], pcm) || !bytes.Equal(first[len(pcm):], make([]byte, inputBytes-len(pcm))) {
		t.Fatal("input was not consumed exactly once with zero padding")
	}
	ticks <- time.Time{}
	if next := await(t, ctx, output); !bytes.Equal(next, make([]byte, inputBytes)) {
		t.Fatal("silence repeated old input")
	}
	cancel()
	if err := await(t, testContext(t), done); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestMicrophoneBounds(t *testing.T) {
	queue := make(chan []byte, 1)
	if err := enqueueMicrophone(queue, []byte{1, 2}); err != nil {
		t.Fatal(err)
	}
	if err := enqueueMicrophone(queue, []byte{3, 4}); err == nil {
		t.Fatal("overflow silently accepted")
	}
	if got := <-queue; !bytes.Equal(got, []byte{1, 2}) {
		t.Fatal("overflow replaced earlier audio")
	}
	if err := enqueueMicrophone(queue, make([]byte, inputBytes*10+2)); err == nil {
		t.Fatal("oversized frame accepted")
	}
	for _, backlog := range []bool{false, true} {
		ctx := testContext(t)
		ready := make(chan struct{})
		close(ready)
		input := make(chan []byte)
		output := make(chan []byte, 1)
		ticks := make(chan time.Time)
		done := make(chan error, 1)
		go func() { done <- feedMicrophone(ctx, ready, input, output, ticks) }()
		if backlog {
			input <- make([]byte, inputBytes*10+2)
		} else {
			ticks <- time.Time{}
			ticks <- time.Time{}
		}
		if err := await(t, ctx, done); err == nil {
			t.Fatal("unbounded microphone backlog")
		}
	}
}

func TestPhraseSplitter(t *testing.T) {
	var phrases []string
	s := phraseSplitter{emit: func(text string) error { phrases = append(phrases, text); return nil }}
	for _, delta := range []string{"你好", "，世界。Hello", "!", strings.Repeat("界", 83), " tail"} {
		if err := s.add(delta); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.finish(); err != nil {
		t.Fatal(err)
	}
	want := []string{"你好，", "世界。", "Hello!", strings.Repeat("界", 80), "界界界 tail"}
	if !reflect.DeepEqual(phrases, want) {
		t.Fatal(phrases)
	}
	failure := errors.New("stop")
	s.emit = func(string) error { return failure }
	if err := s.add("done."); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestPCMFramer(t *testing.T) {
	var frames [][]byte
	f := pcmFramer{emit: func(data []byte) error { frames = append(frames, data); return nil }}
	pcm := bytes.Repeat([]byte{1, 2, 3, 4}, 401)
	for start := 0; start < len(pcm); start += 17 {
		if err := f.add(pcm[start:min(start+17, len(pcm))]); err != nil {
			t.Fatal(err)
		}
	}
	if len(frames) != 2 {
		t.Fatal("audio was not emitted before EOF")
	}
	if err := f.finish(); err != nil {
		t.Fatal(err)
	}
	got := bytes.Join(frames, nil)
	if !bytes.Equal(got[:len(pcm)], pcm) || !bytes.Equal(got[len(pcm):], make([]byte, len(got)-len(pcm))) {
		t.Fatal("frame ownership or final padding corrupted PCM")
	}
	_ = f.add([]byte{1})
	if err := f.finish(); err == nil {
		t.Fatal("odd final byte accepted")
	}
}

func TestAudioGateSerializesInterruptAndRejectsStale(t *testing.T) {
	ctx := testContext(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var order []string
	gate := &audioGate{
		send:  func([]byte) error { close(entered); <-release; order = append(order, "send"); return nil },
		flush: func() error { order = append(order, "flush"); return nil },
	}
	generation, _ := gate.replace()
	writeDone, replaceDone := make(chan error, 1), make(chan error, 1)
	go func() { writeDone <- gate.write(audioPacket{generation, []byte{1, 2}}) }()
	await(t, ctx, entered)
	go func() { _, err := gate.replace(); replaceDone <- err }()
	close(release)
	if err := await(t, ctx, writeDone); err != nil {
		t.Fatal(err)
	}
	if err := await(t, ctx, replaceDone); err != nil {
		t.Fatal(err)
	}
	if err := gate.write(audioPacket{generation, []byte{3, 4}}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"send", "flush"}) {
		t.Fatal(order)
	}
}

func TestCoordinatePartialsAndRepeatedUtterances(t *testing.T) {
	ctx, cancel := context.WithCancel(testContext(t))
	defer cancel()
	input, ready := make(chan transcript), make(chan struct{})
	started, stopped := make(chan string, 4), make(chan string, 4)
	gate := &audioGate{send: func([]byte) error { return nil }, flush: func() error { return nil }}
	done := make(chan error, 1)
	go func() {
		done <- coordinate(ctx, input, ready, gate, func(ctx context.Context, _ uint64, text string) error {
			started <- text
			<-ctx.Done()
			stopped <- text
			return ctx.Err()
		})
	}()
	input <- transcript{ready: true}
	await(t, ctx, ready)
	input <- transcript{text: ""}
	input <- transcript{text: "hello"}
	input <- transcript{text: "hello"} // Keepalive: no turn and no second cancellation.
	input <- transcript{text: "hello there"}
	input <- transcript{text: "hello there", final: true}
	if text := await(t, ctx, started); text != "hello there" {
		t.Fatal(text)
	}
	input <- transcript{text: ""}
	input <- transcript{text: "hello"}
	await(t, ctx, stopped)
	input <- transcript{text: "hello there", final: true} // Legitimate repeated utterance.
	if text := await(t, ctx, started); text != "hello there" {
		t.Fatal(text)
	}
	cancel()
	if err := await(t, testContext(t), done); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	await(t, testContext(t), stopped)
	if len(started) != 0 || len(stopped) != 0 {
		t.Fatal("partials created duplicate turns")
	}
}

func TestCoordinatePreservesFlushError(t *testing.T) {
	ctx, cancel := context.WithCancel(testContext(t))
	gate := &audioGate{sent: true, flush: func() error { return errors.New("flush failed") }}
	cancel()
	err := coordinate(ctx, make(chan transcript), make(chan struct{}), gate, nil)
	if err == nil || !strings.Contains(err.Error(), "flush failed") {
		t.Fatal(err)
	}
}

func TestPlaybackPacingResetsAfterUnderrun(t *testing.T) {
	ctx, cancel := context.WithCancel(testContext(t))
	defer cancel()
	var waits []time.Duration
	base := time.Unix(0, 0)
	nowValues := []time.Time{base, base, base.Add(time.Second), base.Add(time.Second), base.Add(time.Second), base.Add(time.Second + 20*time.Millisecond)}
	calls, sent := 0, 0
	gate := &audioGate{generation: 1, send: func([]byte) error {
		sent++
		if sent == 3 {
			cancel()
		}
		return nil
	}}
	input := make(chan audioPacket, 3)
	for range 3 {
		input <- audioPacket{1, make([]byte, frameBytes)}
	}
	err := playAudio(ctx, input, gate, func() time.Time { value := nowValues[calls]; calls++; return value }, func(_ context.Context, delay time.Duration) error { waits = append(waits, delay); return nil })
	if !errors.Is(err, context.Canceled) || sent != 3 || !reflect.DeepEqual(waits, []time.Duration{20 * time.Millisecond}) {
		t.Fatalf("sent=%d waits=%v err=%v", sent, waits, err)
	}
}

func TestPlaybackRechecksGenerationAfterWait(t *testing.T) {
	ctx, cancel := context.WithCancel(testContext(t))
	defer cancel()
	input := make(chan audioPacket, 2)
	input <- audioPacket{1, make([]byte, frameBytes)}
	input <- audioPacket{1, make([]byte, frameBytes)}
	sent := 0
	gate := &audioGate{generation: 1, send: func([]byte) error { sent++; return nil }, flush: func() error { return nil }}
	err := playAudio(ctx, input, gate, func() time.Time { return time.Unix(0, 0) }, func(_ context.Context, _ time.Duration) error {
		_, err := gate.replace()
		cancel()
		return err
	})
	if !errors.Is(err, context.Canceled) || sent != 1 {
		t.Fatalf("sent=%d err=%v", sent, err)
	}
}

func TestReplyStreamsBeforeLLMCompletesAndPreservesPhraseOrder(t *testing.T) {
	ctx := testContext(t)
	firstSpeech := make(chan struct{})
	var phrases []string
	output := make(chan audioPacket, 3)
	complete := func(ctx context.Context, _ string, emit func(string) error) error {
		if err := emit("One."); err != nil {
			return err
		}
		select {
		case <-firstSpeech:
		case <-ctx.Done():
			return ctx.Err()
		}
		return emit("Two.")
	}
	speak := func(_ context.Context, phrase string, emit func([]byte) error) error {
		phrases = append(phrases, phrase)
		if len(phrases) == 1 {
			close(firstSpeech)
		}
		return emit(bytes.Repeat([]byte{byte(len(phrases)), 0}, frameBytes/2))
	}
	if err := reply(ctx, complete, speak, "question", 7, output); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(phrases, []string{"One.", "Two."}) {
		t.Fatal(phrases)
	}
	for i := byte(1); i <= 2; i++ {
		packet := await(t, ctx, output)
		if packet.generation != 7 || packet.data[0] != i {
			t.Fatal(packet)
		}
	}
}

func TestReplyCancellationUnblocksFullPlaybackAndJoinsLLM(t *testing.T) {
	ctx, cancel := context.WithCancel(testContext(t))
	defer cancel()
	entered, llmExited := make(chan struct{}), make(chan struct{})
	complete := func(ctx context.Context, _ string, emit func(string) error) error {
		defer close(llmExited)
		for {
			if err := emit("More."); err != nil {
				return err
			}
		}
	}
	var once sync.Once
	speak := func(_ context.Context, _ string, emit func([]byte) error) error {
		once.Do(func() { close(entered) })
		return emit(make([]byte, frameBytes))
	}
	done := make(chan error, 1)
	go func() { done <- reply(ctx, complete, speak, "q", 1, make(chan audioPacket)) }()
	await(t, ctx, entered)
	cancel()
	if err := await(t, testContext(t), done); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	await(t, testContext(t), llmExited)
}

func TestReplyLLMFailureCancelsBlockedSpeech(t *testing.T) {
	ctx := testContext(t)
	speaking := make(chan struct{})
	failure := errors.New("LLM truncated")
	complete := func(ctx context.Context, _ string, emit func(string) error) error {
		if err := emit("First."); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-speaking:
			return failure
		}
	}
	speak := func(ctx context.Context, _ string, _ func([]byte) error) error {
		close(speaking)
		<-ctx.Done()
		return ctx.Err()
	}
	if err := reply(ctx, complete, speak, "q", 1, make(chan audioPacket)); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestInitialSubscriptionCanWaitForPeer(t *testing.T) {
	for _, err := range []error{nil, &sdk.Error{Op: "subscribe audio", Code: -2}} {
		if got := subscribeIfPresent(func(peer string) error {
			if peer != "1001" {
				t.Fatal("wrong selected peer", peer)
			}
			return err
		}, "1001"); got != nil {
			t.Fatal(got)
		}
	}
	failure := &sdk.Error{Op: "subscribe audio", Code: -7}
	if err := subscribeIfPresent(func(string) error { return failure }, "1001"); !errors.Is(err, failure) {
		t.Fatal("subscription failure suppressed", err)
	}
}

func TestCoordinatePreservesConcurrentFailureCause(t *testing.T) {
	for range 300 {
		ctx, fail := context.WithCancelCause(testContext(t))
		input := make(chan transcript)
		started := make(chan struct{})
		gate := &audioGate{flush: func() error { return nil }}
		done := make(chan error, 1)
		go func() {
			done <- coordinate(ctx, input, make(chan struct{}), gate, func(ctx context.Context, _ uint64, _ string) error {
				close(started)
				<-ctx.Done()
				return ctx.Err()
			})
		}()
		input <- transcript{text: "question", final: true}
		await(t, ctx, started)
		cause := errors.New("ASR died")
		fail(cause)
		if err := await(t, testContext(t), done); !errors.Is(err, cause) {
			t.Fatal("concurrent cancellation hid the actual failure:", err)
		}
	}
}
