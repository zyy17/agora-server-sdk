package main

import (
	"context"
	"errors"
	"flag"
	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
	"io"
	"os"
	"time"
)

func main() { example.Main(run) }
func run() (err error) {
	input := flag.String("input", "examples/testdata/send_audio_16k_1ch.pcm", "PCM16LE 16 kHz mono")
	output := flag.String("output", "processed.pcm", "output file")
	flag.Parse()
	cfg, _, err := example.Config()
	if err != nil {
		return err
	}
	cfg.AudioProcessing = true
	svc, err := sdk.NewService(cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, svc.Close()) }()
	processor, err := svc.NewAudioProcessor(sdk.ProcessorConfig{})
	if err != nil {
		return err
	}
	defer processor.Close()
	in, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(*output)
	if err != nil {
		return err
	}
	defer out.Close()
	ctx, stop := example.Context()
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error, 1)
	done := make(chan struct{})
	defer func() { cancel(); <-done }()
	go func() {
		defer close(done)
		for {
			frame, err := processor.Output().Next(ctx)
			if err != nil {
				result <- err
				return
			}
			if _, err := out.Write(frame.Data); err != nil {
				result <- err
				return
			}
		}
	}()
	buf := make([]byte, 320)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		n, err := io.ReadFull(in, buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if n > 0 {
			if err := processor.SendPCM(buf); err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-result:
			return err
		case <-tick.C:
		}
	}
	timer := time.NewTimer(300 * time.Millisecond)
	defer timer.Stop()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	cancel()
	return <-result
}
