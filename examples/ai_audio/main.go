package main

import (
	"context"
	"errors"
	"flag"
	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
	"log"
	"os"
	"sync"
	"time"
)

func main() { example.Main(run) }
func run() error {
	reply := flag.String("reply", "", "optional PCM16LE response file; interrupted on incoming speech")
	flag.Parse()
	return example.Run(false, func(c *sdk.ChannelConfig) { c.Audio.VAD = true; c.Audio.Scenario = sdk.ScenarioAIServer }, func(ctx context.Context, s *sdk.Session) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		playContext, stopPlayback := context.WithCancel(ctx)
		defer stopPlayback()
		var workers sync.WaitGroup
		defer func() { cancel(); workers.Wait() }()
		results := make(chan error, 2)
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				event, err := s.Events().Next(ctx)
				if err != nil {
					results <- err
					return
				}
				switch event.Kind {
				case sdk.TokenExpiring, sdk.TokenExpired:
					log.Print("refresh token through your server and call RenewToken")
				case sdk.CapabilityFallback:
					if err := s.SetAudioScenario(sdk.ScenarioChorus); err != nil {
						results <- err
						return
					}
				case sdk.AudioVolume:
					log.Printf("volume: %+v", event.Data)
				case sdk.SDKError:
					log.Printf("SDK error %d: %s", event.Code, event.Message)
				}
			}
		}()
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				frame, err := s.Audio().Next(ctx)
				if err != nil {
					results <- err
					return
				}
				if len(frame.Speech) > 0 {
					stopPlayback()
					if err := s.InterruptAudio(); err != nil {
						results <- err
						return
					}
					log.Printf("speech from %s: %d bytes, VAD=%d", frame.UserID, len(frame.Speech), frame.VAD)
				}
			}
		}()
		if err := s.SetVolumeIndication(200*time.Millisecond, 3, true); err != nil {
			return err
		}
		if *reply != "" {
			f, err := os.Open(*reply)
			if err != nil {
				return err
			}
			err = s.PlayPCM(playContext, f)
			err = errors.Join(err, f.Close())
			if err != nil && !(errors.Is(err, context.Canceled) && ctx.Err() == nil) {
				return err
			}
		}
		return <-results
	})
}
