package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"sync"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

func main() { example.Main(run) }

func run() error {
	reply := flag.String("reply", "", "optional PCM16LE response file; interrupted on incoming speech")
	flag.Parse()
	return example.Run(false, func(c *sdk.ChannelConfig) {
		c.Audio.VAD = true
		c.Audio.Scenario = sdk.ScenarioAIServer
	}, func(ctx context.Context, session *sdk.Session) error {
		return serveAudio(ctx, session, *reply)
	})
}

func serveAudio(ctx context.Context, session *sdk.Session, reply string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	playContext, stopPlayback := context.WithCancel(ctx)
	defer stopPlayback()
	var workers sync.WaitGroup
	defer func() { cancel(); workers.Wait() }()
	results := make(chan error, 2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		results <- handleEvents(ctx, session)
	}()
	go func() {
		defer workers.Done()
		results <- handleSpeech(ctx, session, stopPlayback)
	}()
	if err := session.SetVolumeIndication(200*time.Millisecond, 3, true); err != nil {
		return err
	}
	if reply != "" {
		file, err := os.Open(reply)
		if err != nil {
			return err
		}
		err = session.PlayPCM(playContext, file)
		err = errors.Join(err, file.Close())
		if err != nil && !(errors.Is(err, context.Canceled) && ctx.Err() == nil) {
			return err
		}
	}
	return <-results
}

func handleEvents(ctx context.Context, session *sdk.Session) error {
	for {
		event, err := session.Events().Next(ctx)
		if err != nil {
			return err
		}
		switch event.Kind {
		case sdk.TokenExpiring, sdk.TokenExpired:
			log.Print("refresh token through your server and call RenewToken")
		case sdk.CapabilityFallback:
			if err := session.SetAudioScenario(sdk.ScenarioChorus); err != nil {
				return err
			}
		case sdk.AudioVolume:
			log.Printf("volume: %+v", event.Data)
		case sdk.SDKError:
			log.Printf("SDK error %d: %s", event.Code, event.Message)
		}
	}
}

func handleSpeech(ctx context.Context, session *sdk.Session, stopPlayback context.CancelFunc) error {
	for {
		frame, err := session.Audio().Next(ctx)
		if err != nil {
			return err
		}
		if len(frame.Speech) > 0 {
			stopPlayback()
			if err := session.InterruptAudio(); err != nil {
				return err
			}
			log.Printf("speech from %s: %d bytes, VAD=%d", frame.UserID, len(frame.Speech), frame.VAD)
		}
	}
}
