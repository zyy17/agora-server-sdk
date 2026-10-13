// voice_agent connects one RTC peer to streaming speech and language providers.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/examples/internal/example"
)

func main() { example.Main(run) }

func run() (err error) {
	cfg, channel, err := example.Config()
	if err != nil {
		return err
	}
	peer := os.Getenv("AGORA_REMOTE_USER_ID")
	cfg.IdentityMode, err = identityMode(os.Getenv("AGORA_IDENTITY_MODE"), channel.UserID, peer)
	if err != nil {
		return err
	}
	p := providers{
		httpClient: &http.Client{Timeout: 90 * time.Second},
		elevenKey:  os.Getenv("ELEVENLABS_API_KEY"), openAIKey: os.Getenv("OPENAI_API_KEY"),
		voiceID:  envDefault("ELEVENLABS_VOICE_ID", "JBFqnCBsd6RMkjVDRZzb"),
		language: os.Getenv("VOICE_LANGUAGE"), model: envDefault("OPENAI_MODEL", "gpt-4.1-mini"),
		systemPrompt: envDefault("VOICE_SYSTEM_PROMPT", "You are a concise, friendly voice assistant. Reply in the user's language in one to three short sentences. Use plain speech, no Markdown, and words that are easy to read aloud."),
	}
	if p.elevenKey == "" || p.openAIKey == "" {
		return errors.New("set ELEVENLABS_API_KEY and OPENAI_API_KEY in the server environment")
	}
	defer p.httpClient.CloseIdleConnections()
	channel.Audio = sdk.AudioConfig{Send: sdk.PCMFormat{SampleRate: 16000, Channels: 1}, Receive: sdk.PCMFormat{SampleRate: 16000, Channels: 1}}
	channel.Receive = sdk.ReceiveConfig{AudioPositions: sdk.AudioBeforeMixing, ManualSubscriptions: true}
	channel.QueueCapacity = 128
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
	log.Printf("joined as %s; selected peer=%s; waiting for ASR readiness", session.Info().UserID, peer)
	return serve(ctx, session, peer, p)
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func identityMode(mode, bot, peer string) (sdk.IdentityMode, error) {
	if peer == "" || bot == peer {
		return 0, errors.New("AGORA_REMOTE_USER_ID must select one peer distinct from AGORA_USER_ID")
	}
	switch mode {
	case "", "numeric":
		for _, id := range []string{bot, peer} {
			value, err := strconv.ParseUint(id, 10, 32)
			if err != nil || value == 0 || strconv.FormatUint(value, 10) != id {
				return 0, errors.New("numeric identities must be nonzero canonical uint32 values; use matching numeric-UID RTC tokens")
			}
		}
		return sdk.NumericIdentity, nil
	case "string":
		for _, id := range []string{bot, peer} {
			if strings.TrimSpace(id) == "" || len(id) > 255 || strings.ContainsRune(id, 0) {
				return 0, errors.New("string identities must contain 1–255 bytes without NUL")
			}
		}
		return sdk.StringIdentity, nil
	default:
		return 0, errors.New("AGORA_IDENTITY_MODE must be numeric or string")
	}
}

func serve(parent context.Context, session *sdk.Session, peer string, p providers) (err error) {
	ctx, fail := context.WithCancelCause(parent)
	var workers sync.WaitGroup
	start := func(work func() error) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if e := work(); e != nil {
				fail(e)
			}
		}()
	}
	gate := &audioGate{send: session.SendPCM, flush: session.InterruptAudio}
	defer func() {
		fail(err)
		workers.Wait()
		if cause := context.Cause(ctx); cause != nil && !errors.Is(err, cause) {
			err = errors.Join(err, cause)
		}
		_, flushErr := gate.replace()
		err = errors.Join(err, flushErr)
		log.Printf("stopped; dropped audio=%d events=%d", session.Audio().Stats().Dropped, session.Events().Stats().Dropped)
	}()
	ready := make(chan struct{})
	microphone := make(chan []byte, 50)
	asrAudio := make(chan []byte, 2)
	transcripts := make(chan transcript, 16)
	playback := make(chan audioPacket, 25) // At most 500 ms of generated PCM.
	start(func() error { return capture(ctx, session, peer, ready, microphone) })
	start(func() error { return rtcEvents(ctx, session, peer) })
	start(func() error {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		return feedMicrophone(ctx, ready, microphone, asrAudio, ticker.C)
	})
	start(func() error {
		return p.recognize(ctx, asrAudio, func(t transcript) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case transcripts <- t:
				return nil
			default:
				return errors.New("ASR transcript queue overflow")
			}
		})
	})
	start(func() error { return playAudio(ctx, playback, gate, time.Now, waitContext) })
	return coordinate(ctx, transcripts, ready, gate, func(turnCtx context.Context, generation uint64, text string) error {
		return reply(turnCtx, p.complete, p.speak, text, generation, playback)
	})
}

func capture(ctx context.Context, session *sdk.Session, peer string, ready <-chan struct{}, output chan<- []byte) error {
	for {
		frame, err := session.Audio().Next(ctx)
		if err != nil {
			return err
		}
		if session.Audio().Stats().Dropped != 0 {
			return errors.New("RTC microphone stream dropped frames; restart a fresh ASR session")
		}
		if frame.UserID != peer || frame.Position != sdk.AudioBeforeMixing {
			continue
		}
		select {
		case <-ready:
		default:
			continue // Discard pre-readiness audio, never replay it into ASR.
		}
		if frame.SampleRate != 16000 || frame.Channels != 1 || frame.BytesPerSample != 2 || len(frame.Data)%2 != 0 {
			return errors.New("received audio must be PCM16LE, 16000 Hz mono")
		}
		if err := enqueueMicrophone(output, frame.Data); err != nil {
			return err
		}
	}
}

func rtcEvents(ctx context.Context, session *sdk.Session, peer string) error {
	// Subscribe now as well as on join: the selected peer may already be present.
	if err := subscribeIfPresent(session.SubscribeAudio, peer); err != nil {
		return err
	}
	for {
		event, err := session.Events().Next(ctx)
		if err != nil {
			return err
		}
		if session.Events().Stats().Dropped != 0 || session.Audio().Stats().Dropped != 0 {
			return errors.New("RTC receive queue dropped data; restart the demo")
		}
		switch event.Kind {
		case sdk.UserJoined:
			if event.UserID == peer {
				if err := session.SubscribeAudio(peer); err != nil {
					return err
				}
				log.Print("selected peer joined")
			}
		case sdk.UserLeft:
			if event.UserID == peer {
				return errors.New("selected peer left; restart for a fresh conversation")
			}
		case sdk.TokenExpiring:
			log.Print("RTC token is expiring; restart with a fresh token from your server")
		case sdk.TokenExpired, sdk.ConnectionFailed, sdk.ConnectionLost, sdk.Disconnected, sdk.Reconnecting, sdk.EncryptionError, sdk.SDKError:
			return fmt.Errorf("RTC event %s: code=%d reason=%d", event.Kind, event.Code, event.Reason)
		}
	}
}

// subscribeIfPresent permits starting the bot before the selected human joins.
// Identity syntax is already validated; native -2 means this peer is absent.
// UserJoined will retry. Other failures must remain visible.
func subscribeIfPresent(subscribe func(string) error, peer string) error {
	err := subscribe(peer)
	var native *sdk.Error
	if errors.As(err, &native) && native.Code == -2 {
		return nil
	}
	return err
}
