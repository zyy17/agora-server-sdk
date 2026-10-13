//go:build avcodec

package agoraservice

import (
	"bytes"
	"encoding/base64"
	"sync"
	"testing"
	"time"
)

// Offline fixtures generated with FFmpeg 7.1.5. The videos are one red 64x48
// yuv420p access unit (libx264 -tune zerolatency, libx265 -tune zerolatency, encoder SEI removed).
// Audio is a 20 ms libopus packet of a mono 1 kHz/48 kHz sine wave, extracted
// from the Ogg container. Tests do not need ffmpeg, network access, or RTC keys.
const testH264 = "AAAAAWdCwAraEewEQAAAAwBAAAAMo8SJqAAAAAFozg/IAAAAAWWIhDoRigACGPHAAED2OAAIeUnJyddddddddeA="
const testH265 = "AAAAAUABDAH//wFgAAADAJAAAAMAAAMAHpKAkAAAAAFCAQEBYAAAAwCQAAADAAADAB6gIIMWWSrkyvAWgIAAAAMAgAAADIQAAAABRAHBc9CJAAAAASgBr3j3BAP/26P/7SfS9sOUQIPAAHg="
const testOpus = "+J8QbWLei8NgB8hGSiwsmNu1/tV+gQ9PmGEDZ3QvALj6hXI7Krx+fRL9NUb/rGNLvUZ/ETPicE/ExhlV6j/W5jfKptf6PzdoozQ8Yn+mMUnFuMb/FuOrATQILi3h2y39BOcadlnE2USfacqbv1dNS2usbujmZCWcWVXtfg+emBZwIBC5U8/n6EJrWzR83S8p4raSuLwWcp3lmWEQgCnCkyjREKO7J5larKXxGrs8zUBl1sBJrOXjfpK0qqIt7JH0pxDUPYSYvh2q6LgiU6r54kcKzglX5Ia+Ha0NM8cZuClprV6h8ulsgDgElD9ywUNgK2+fgSxtElEYNTj9s3AhIEjwACkipG+vpgWZ4P1D6BSx"

func fixture(t *testing.T, value string) []byte {
	t.Helper()
	b, e := base64.StdEncoding.DecodeString(value)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestOptionalVideoDecodeAndCodecSwitch(t *testing.T) {
	d, err := NewVideoDecoder()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, tc := range []struct {
		codec VideoCodecType
		data  string
	}{{VideoCodecTypeH264, testH264}, {VideoCodecTypeH265, testH265}, {VideoCodecTypeH264, testH264}} {
		for _, size := range [][2]int{{0, 0}, {32, 24}, {128, 96}} {
			f, code := d.DecodeAndSave(fixture(t, tc.data), size[0], size[1], int(tc.codec))
			if code != 0 || f == nil {
				t.Fatalf("codec %d size %v: code %d", tc.codec, size, code)
			}
			w, h := size[0], size[1]
			if w == 0 {
				w, h = 64, 48
			}
			if f.Stride != w || f.Height != h || len(f.Buffer) != w*h*3/2 || f.Format != VideoPixelI420 {
				t.Fatalf("bad frame: %+v", f)
			}
			for plane, want := range []byte{81, 90, 240} {
				start, end := 0, w*h
				if plane == 1 {
					start, end = w*h, w*h+w*h/4
				}
				if plane == 2 {
					start, end = w*h+w*h/4, w*h*3/2
				}
				for _, pixel := range f.Buffer[start:end] {
					if pixel+1 < want || pixel > want+1 {
						t.Fatalf("codec %d plane %d pixel %d want %d", tc.codec, plane, pixel, want)
					}
				}
			}
		}
	}
}

func TestOptionalVideoInvalidAndClosed(t *testing.T) {
	var absent *VideoDecoder
	absent.Close()
	if f, c := absent.DecodeAndSave([]byte{1}, 0, 0, 2); f != nil || c >= 0 {
		t.Fatal("nil decoder accepted")
	}
	d, e := NewVideoDecoder()
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	for _, tc := range []struct {
		b       []byte
		w, h, c int
	}{{nil, 0, 0, 2}, {[]byte{1, 2, 3}, 0, 0, 2}, {fixture(t, testH264), -2, 48, 2}, {fixture(t, testH264), 63, 48, 2}, {fixture(t, testH264), 64, 48, 12}, {fixture(t, testH264), 1 << 30, 1 << 30, 2}} {
		if f, c := d.DecodeAndSave(tc.b, tc.w, tc.h, tc.c); f != nil || c >= 0 {
			t.Errorf("invalid input returned frame/code %d", c)
		}
	}
	d.Close()
	d.Close()
	if f, c := d.DecodeAndSave(fixture(t, testH264), 0, 0, 2); f != nil || c >= 0 {
		t.Fatal("closed decoder accepted input")
	}
}

func TestOptionalVideoConcurrentClose(t *testing.T) {
	d, e := NewVideoDecoder()
	if e != nil {
		t.Fatal(e)
	}
	packet := fixture(t, testH264)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 10; n++ {
				d.DecodeAndSave(packet, 0, 0, 2)
			}
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); d.Close(); d.Close() }()
	wg.Wait()
}

func TestOptionalOpusDecode(t *testing.T) {
	packet := fixture(t, testOpus)
	for _, tc := range []struct{ in, rate, out, bytes int }{{0, 0, 0, 1920}, {1, 48000, 1, 1920}, {1, 48000, 2, 3840}, {1, 16000, 1, 608}} {
		d := NewOpusDecoder(tc.in, tc.rate, tc.out)
		if d == nil {
			t.Fatalf("new decoder %+v", tc)
		}
		pcm, e := d.Decode(packet)
		if e != nil {
			t.Fatal(e)
		}
		if len(pcm) != tc.bytes || bytes.Equal(pcm, make([]byte, len(pcm))) {
			t.Fatalf("bad PCM for %+v: %d bytes", tc, len(pcm))
		}
		saved := append([]byte(nil), pcm...)
		if _, e = d.Decode(packet); e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(pcm, saved) {
			t.Fatal("decoder reused returned PCM memory")
		}
		d.Release()
		d.Release()
		if _, e = d.Decode(packet); e == nil {
			t.Fatal("released decoder accepted packet")
		}
	}
}

func TestOptionalOpusInvalidAndConcurrentRelease(t *testing.T) {
	for _, p := range [][3]int{{-1, 48000, 1}, {3, 48000, 1}, {1, -1, 1}, {1, 1000000, 1}, {1, 48000, 3}} {
		if d := NewOpusDecoder(p[0], p[1], p[2]); d != nil {
			d.Release()
			t.Fatalf("invalid params accepted: %v", p)
		}
	}
	var nilDecoder *OpusDecoder
	nilDecoder.Release()
	if _, e := nilDecoder.Decode([]byte{1}); e == nil {
		t.Fatal("nil decoder accepted packet")
	}
	d := NewOpusDecoder(1, 48000, 1)
	if d == nil {
		t.Fatal("no decoder")
	}
	if _, e := d.Decode(nil); e == nil {
		t.Fatal("empty packet accepted")
	}
	packet := fixture(t, testOpus)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 10; n++ {
				d.Decode(packet)
			}
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); d.Release(); d.Release() }()
	wg.Wait()
}

func waitOptional(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("worker shutdown blocked")
	}
}

func TestOptionalWorkersLifecycle(t *testing.T) {
	a := NewAudioTranscodingWorker(nil, 1)
	v := NewTranscodingWorker(nil, 15)
	if a == nil || v == nil {
		t.Fatal("worker creation failed")
	}
	a.Stop()
	v.Stop()
	if a.decoder != nil || v.decoder != nil {
		t.Fatal("Stop before Start leaked decoder")
	}
	if a.Start() == 0 || v.Start() == 0 {
		t.Fatal("stopped worker restarted")
	}
	a.Stop()
	v.Stop()
	a = NewAudioTranscodingWorker(nil, 1)
	v = NewTranscodingWorker(nil, 30)
	if a.Start() != 0 || v.Start() != 0 {
		t.Fatal("start failed")
	}
	if a.Start() == 0 || v.Start() == 0 {
		t.Fatal("double start accepted")
	}
	audio, video := fixture(t, testOpus), fixture(t, testH264)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				a.PushEncodedData(audio, 1, int(AudioCodecOpus))
				v.PushEncodedData(video, &EncodedVideoFrameInfo{CodecType: VideoCodecTypeH264})
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); a.Stop(); v.Stop() }()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	waitOptional(t, done)
	a.PushEncodedData(audio, 1, 1)
	v.PushEncodedData(video, nil)
}

func TestOptionalWorkersCopyInputsAndUnblockOnStop(t *testing.T) {
	a := &AudioTranscodingWorker{isRunning: true, in_channels: 1, encodeDataQueue: make(chan *AudioTransCodeInData, 1), stopChan: make(chan bool)}
	v := &TranscodingWorker{isRunning: true, encodeDataQueue: make(chan *TransCodeInData, 1), stopChan: make(chan bool)}
	data := []byte{1, 2, 3}
	info := &EncodedVideoFrameInfo{Width: 64, Height: 48, CodecType: VideoCodecTypeH264}
	a.PushEncodedData(data, 1, 1)
	v.PushEncodedData(data, info)
	data[0] = 99
	info.Width = 99
	ai, vi := <-a.encodeDataQueue, <-v.encodeDataQueue
	if ai.data[0] != 1 || vi.data[0] != 1 || vi.frameInfo.Width != 64 {
		t.Fatal("queued caller-owned storage")
	}
	// A saturated queue must not prevent Stop or leave a producer blocked.
	a.encodeDataQueue <- ai
	v.encodeDataQueue <- vi
	ad, vd := make(chan struct{}), make(chan struct{})
	go func() { a.PushEncodedData(data, 1, 1); close(ad) }()
	go func() { v.PushEncodedData(data, info); close(vd) }()
	a.Stop()
	v.Stop()
	waitOptional(t, ad)
	waitOptional(t, vd)
	if len(a.encodeDataQueue) != 0 || len(v.encodeDataQueue) != 0 {
		t.Fatal("Stop retained queued media")
	}
}

const testShortOpus = "6J8SdJG3zW3Qrj+oWWvJG8RxM9qZmhbaBJp5aohfzYhb+dkohcBe+iGoFCQhDDcpwwDJc3jWEoXCPVM9bGOGqVwbNJE="

func TestOptionalWorkerShortOpusAndChannelSwitch(t *testing.T) {
	worker := NewAudioTranscodingWorker(nil, 1)
	if worker == nil {
		t.Fatal("worker init failed")
	}
	defer worker.Stop()
	chunks := make(chan []byte, 8)
	worker.pushPCM = func(pcm []byte, rate, channels int, pts int64) int {
		if rate != 48000 || channels != 1 {
			t.Errorf("bad PCM format: %d/%d", rate, channels)
		}
		chunks <- append([]byte(nil), pcm...)
		return 0
	}
	if worker.Start() != 0 {
		t.Fatal("start failed")
	}
	packet := fixture(t, testShortOpus)
	worker.PushEncodedData(packet, 1, int(AudioCodecOpus))
	worker.PushEncodedData(packet, 1, int(AudioCodecOpus))
	select {
	case pcm := <-chunks:
		if len(pcm) != 960 {
			t.Fatalf("short packets produced %d bytes", len(pcm))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("short Opus packets dropped")
	}
	worker.PushEncodedData(fixture(t, testOpus), 2, int(AudioCodecOpus))
	select {
	case pcm := <-chunks:
		if len(pcm) != 1920 {
			t.Fatalf("channel switch produced %d bytes", len(pcm))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("channel switch dropped packet")
	}
	worker.Stop()
}

// HEVC Main10 is rejected because the public output is 8-bit I420.
const testH265TenBit = "AAAAAUABDAH//wIgAAADAJAAAAMAAAMAHpKAkAAAAAFCAQECIAAAAwCQAAADAAADAB6gIIMTZZKuTK8BaAgAAAMACAAAAwDIQAAAAAFEAcFz0IkAAAABKAGvePcEA//bo//tJ9L2w5RAg8AAeA=="

func TestOptionalVideoUnsupportedPixelFormat(t *testing.T) {
	d, e := NewVideoDecoder()
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	if f, code := d.DecodeAndSave(fixture(t, testH265TenBit), 0, 0, int(VideoCodecTypeH265)); code >= 0 || f != nil {
		t.Fatalf("Main10 decoded as 8-bit I420: %d", code)
	}
	// A rejected frame does not invalidate subsequent codec switches.
	if f, code := d.DecodeAndSave(fixture(t, testH264), 0, 0, int(VideoCodecTypeH264)); code != 0 || f == nil {
		t.Fatalf("decode after rejected pixel format: %d", code)
	}
}
