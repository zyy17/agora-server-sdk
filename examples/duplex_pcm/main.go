// duplex_pcm sends a short synthetic stereo tone while consuming mono audio.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

func main() { example.Main(run) }

func run() (err error) {
	duration := flag.Duration("duration", 2*time.Second, "tone duration: integral milliseconds, up to 1 minute")
	pts := flag.Int64("pts", 0, "opaque presentation value forwarded unchanged on every chunk")
	flag.Parse()
	if err := validateDuration(*duration); err != nil {
		return err
	}
	cfg, channel, err := example.Config()
	if err != nil {
		return err
	}
	cfg.IdentityMode = sdk.NumericIdentity
	channel.Audio.Send = sdk.PCMFormat{SampleRate: 16000, Channels: 2}
	channel.Audio.Receive = sdk.PCMFormat{SampleRate: 16000, Channels: 1}
	ctx, stop := example.Context()
	defer stop()
	service, err := sdk.NewService(cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, service.Close()) }()
	session, err := service.Join(ctx, channel)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, session.Close()) }()
	log.Printf("joined numeric UID=%s; send=16000 Hz stereo, receive=16000 Hz mono", session.Info().UserID)
	return duplex(ctx, session, *duration, *pts)
}

func duplex(parent context.Context, session *sdk.Session, duration time.Duration, pts int64) (err error) {
	ctx, cancel := context.WithCancel(parent)
	var workers sync.WaitGroup
	results := make(chan error, 2) // Each consumer reports at most one error.
	start := func(consume func() error) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if e := consume(); e != nil && !errors.Is(e, context.Canceled) {
				results <- e
				cancel()
			}
		}()
	}
	defer func() {
		cancel()
		workers.Wait()
		close(results)
		var workerErr error
		for e := range results {
			workerErr = errors.Join(workerErr, e)
		}
		if workerErr != nil && errors.Is(err, context.Canceled) {
			err = workerErr // Keep the cause visible rather than suppressing it as Ctrl+C.
		} else {
			err = errors.Join(err, workerErr)
		}
		log.Printf("dropped: audio=%d events=%d", session.Audio().Stats().Dropped, session.Events().Stats().Dropped)
	}()
	start(func() error { return receiveMonoAudio(ctx, session) })
	start(func() error { return receiveEvents(ctx, session) })
	if err := streamTone(ctx, duration, pts, session.SendPCMAt, waitContext); err != nil {
		return err
	}
	// This is an estimate of local send completion, not a remote-playback receipt.
	finish, stop := context.WithTimeout(ctx, 5*time.Second)
	defer stop()
	if err := session.WaitPCM(finish); err != nil {
		return err
	}
	return nil
}

func receiveMonoAudio(ctx context.Context, session *sdk.Session) error {
	frames, bytes := 0, 0
	defer func() { log.Printf("received frames=%d bytes=%d", frames, bytes) }()
	for {
		frame, e := session.Audio().Next(ctx)
		if e != nil {
			return e
		}
		if frame.SampleRate != 16000 || frame.Channels != 1 {
			return fmt.Errorf("unexpected received format: %d Hz, %d channels", frame.SampleRate, frame.Channels)
		}
		frames++
		bytes += len(frame.Data)
	}
}

func receiveEvents(ctx context.Context, session *sdk.Session) error {
	for {
		event, e := session.Events().Next(ctx)
		if e != nil {
			return e
		}
		switch event.Kind {
		case sdk.TokenExpiring:
			log.Print("RTC token expiring: obtain a new numeric-UID token from your server and call RenewToken")
		case sdk.TokenExpired, sdk.ConnectionFailed, sdk.ConnectionLost, sdk.Disconnected:
			return fmt.Errorf("RTC event %s: reason=%d", event.Kind, event.Reason)
		case sdk.SDKError:
			return fmt.Errorf("RTC error %d: %s", event.Code, event.Message)
		}
	}
}
