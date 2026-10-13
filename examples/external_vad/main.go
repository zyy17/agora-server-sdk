// external_vad runs standalone mono or stereo VAD without an RTC service.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

func main() { example.Main(run) }

func run() (err error) {
	input := flag.String("input", "", "explicit PCM16LE 16 kHz file; empty uses 1 second of synthetic silence")
	channels := flag.Int("channels", 1, "input channels: 1 or 2")
	flag.Parse()
	if *channels != 1 && *channels != 2 {
		return errors.New("-channels must be 1 or 2")
	}
	var reader io.Reader = bytes.NewReader(make([]byte, 16000*2*(*channels)))
	if *input != "" {
		file, e := os.Open(*input)
		if e != nil {
			return e
		}
		defer func() { err = errors.Join(err, file.Close()) }()
		reader = file
	}
	ctx, stop := example.Context()
	defer stop()
	frames, leftBytes, rightBytes := 0, 0, 0
	if *channels == 1 {
		detector := rtc.NewAudioVad(nil)
		if detector == nil {
			return errors.New("cannot initialize standalone VAD; check installed native libraries")
		}
		defer detector.Release()
		lastState := -1
		err = processFrames(ctx, reader, 1, func(frame *rtc.AudioFrame) error {
			output, state := detector.ProcessPcmFrame(frame)
			if state < 0 {
				return fmt.Errorf("mono VAD failed: %d", state)
			}
			frames++
			leftBytes += outputBytes(output)
			if state != lastState {
				log.Printf("frame=%d state=%d", frames, state)
				lastState = state
			}
			return nil
		})
	} else {
		// NewSteroVad is the compatibility API spelling. Each channel has its
		// own detector and can produce different speech boundaries and lengths.
		detector := rtc.NewSteroVad(nil, nil)
		if detector == nil {
			return errors.New("cannot initialize stereo VAD; check installed native libraries")
		}
		defer detector.Release()
		lastLeft, lastRight := -1, -1
		err = processFrames(ctx, reader, 2, func(frame *rtc.AudioFrame) error {
			left, leftState, right, rightState := detector.ProcessAudioFrame(frame)
			if leftState < 0 || rightState < 0 {
				return fmt.Errorf("stereo VAD failed: left=%d right=%d", leftState, rightState)
			}
			frames++
			leftBytes += outputBytes(left)
			rightBytes += outputBytes(right)
			if leftState != lastLeft || rightState != lastRight {
				log.Printf("frame=%d left_state=%d right_state=%d", frames, leftState, rightState)
				lastLeft, lastRight = leftState, rightState
			}
			return nil
		})
	}
	log.Printf("input_frames=%d returned_mono_bytes=%d returned_right_bytes=%d", frames, leftBytes, rightBytes)
	return err
}

func outputBytes(frame *rtc.AudioFrame) int {
	if frame == nil {
		return 0
	}
	return len(frame.Buffer)
}

// processFrames rejects a partial tail instead of silently inventing samples.
// Calls are synchronous; no channel, callback queue, or pacing is needed offline.
func processFrames(ctx context.Context, reader io.Reader, channels int, consume func(*rtc.AudioFrame) error) error {
	if channels != 1 && channels != 2 {
		return errors.New("VAD requires one or two channels")
	}
	for index := int64(0); ; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		data := make([]byte, 160*channels*2)
		_, err := io.ReadFull(reader, data)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read complete 10 ms PCM frame: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		frame := &rtc.AudioFrame{
			Type: rtc.AudioFrameTypePCM16, Buffer: data,
			SamplesPerSec: 16000, Channels: channels,
			SamplesPerChannel: 160, BytesPerSample: 2,
			RenderTimeMs: index * 10,
		}
		if err := consume(frame); err != nil {
			return err
		}
	}
}
