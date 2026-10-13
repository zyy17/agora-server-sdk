package main

import (
	"context"
	"encoding/binary"
	"math"
	"sync"
	"testing"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
)

func TestAudioReceiptValidatesToneAndSource(t *testing.T) {
	f := sdk.AudioFrame{Channel: "test", UserID: "sender", SampleRate: testRate, Channels: 1, BytesPerSample: 2, Data: toneFrame(0)}
	if !matchesTone(f, "test", "sender") {
		t.Fatal("test tone rejected")
	}
	if matchesTone(f, "other", "sender") || matchesTone(f, "test", "other") {
		t.Fatal("wrong source accepted")
	}
	f.Data = make([]byte, 320)
	if matchesTone(f, "test", "sender") {
		t.Fatal("silence accepted as delivered tone")
	}
	for i := 0; i < 160; i++ {
		binary.LittleEndian.PutUint16(f.Data[2*i:], uint16(int16(6000*math.Sin(2*math.Pi*1000*float64(i)/testRate))))
	}
	if matchesTone(f, "test", "sender") {
		t.Fatal("unrelated audio accepted as delivered tone")
	}
}

func receivedPattern() sdk.ReceivedVideoFrame {
	f := patternFrame()
	return sdk.ReceivedVideoFrame{Channel: "test", UserID: "sender", Width: f.Width, Height: f.Height, YStride: f.Width, Y: f.Data[:f.Width*f.Height]}
}

func TestVideoReceiptValidatesPatternAndSource(t *testing.T) {
	f := receivedPattern()
	if !matchesPattern(f, "test", "sender") {
		t.Fatal("pattern rejected")
	}
	if matchesPattern(f, "other", "sender") || matchesPattern(f, "test", "other") {
		t.Fatal("wrong source accepted")
	}
	f.Y = make([]byte, testWidth*testHeight)
	if matchesPattern(f, "test", "sender") {
		t.Fatal("empty pixels counted as delivered pattern")
	}
	f.YStride = 0
	if matchesPattern(f, "test", "sender") {
		t.Fatal("invalid stride accepted")
	}
}

type fakeMedia struct {
	audio    chan sdk.AudioFrame
	video    chan sdk.ReceivedVideoFrame
	messages chan sdk.Message
}

func (*fakeMedia) SendPCM([]byte) error { return nil }

func (*fakeMedia) SendVideo(sdk.VideoFrame) error { return nil }

func (*fakeMedia) SendMessage([]byte) error { return nil }

func (f *fakeMedia) readAudio(ctx context.Context) (sdk.AudioFrame, error) {
	select {
	case v := <-f.audio:
		return v, nil
	case <-ctx.Done():
		return sdk.AudioFrame{}, ctx.Err()
	}
}

func (f *fakeMedia) readVideo(ctx context.Context) (sdk.ReceivedVideoFrame, error) {
	select {
	case v := <-f.video:
		return v, nil
	case <-ctx.Done():
		return sdk.ReceivedVideoFrame{}, ctx.Err()
	}
}

func (f *fakeMedia) readMessage(ctx context.Context) (sdk.Message, error) {
	select {
	case v := <-f.messages:
		return v, nil
	case <-ctx.Done():
		return sdk.Message{}, ctx.Err()
	}
}

func TestMediaSuccessRequiresReceivedContent(t *testing.T) {
	for _, valid := range []bool{true, false} {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Millisecond)
		peer := &fakeMedia{audio: make(chan sdk.AudioFrame, 1), video: make(chan sdk.ReceivedVideoFrame, 1), messages: make(chan sdk.Message, 1)}
		if valid {
			peer.audio <- sdk.AudioFrame{Channel: "test", UserID: "sender", SampleRate: testRate, Channels: 1, BytesPerSample: 2, Data: toneFrame(0)}
			peer.video <- receivedPattern()
			peer.messages <- sdk.Message{Channel: "test", UserID: "sender", Data: append([]byte{0, 1, 255, 127}, []byte("test")...)}
		} else {
			peer.messages <- sdk.Message{Channel: "test", UserID: "sender", Data: []byte("wrong payload")}
		}
		var mu sync.Mutex
		results := make(map[string]error)
		testRTCExchange(ctx, peer, peer, "test", "sender", func(operation string, err error) { mu.Lock(); defer mu.Unlock(); results[operation] = err })
		cancel()
		for _, operation := range []string{"audio_receive", "video_receive", "data_receive"} {
			err, ok := results[operation]
			if !ok || valid && err != nil || !valid && err == nil {
				t.Fatalf("%s did not reflect received content", operation)
			}
		}
	}
}

func TestFeatureReportsDistinguishAuthAndCoverage(t *testing.T) {
	r := report{}
	r.add("prepare", nil)
	r.add("initialize", nil)
	r.add("join", nil)
	r.add("peer_join", nil)
	r.fillMissing("rtc_features")
	if !validReport(r, "rtc_features") || !r.authenticated("rtc_features") || r.passed("rtc_features") {
		t.Fatal("authentication and feature completion were conflated")
	}
	for _, v := range r.Results {
		if v.Operation == "audio_receive" && v.Status != "not_run" {
			t.Fatal("missing operation not disclosed")
		}
	}
}
