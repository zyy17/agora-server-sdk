package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gordonklaus/portaudio"
	agorasdk "github.com/zyy17/agora-server-sdk"
	agoraservice "github.com/zyy17/agora-server-sdk/agora/rtc"
)

const (
	exampleName             = "recv_pcm"
	defaultChannelName      = "agora_sdk_example"
	defaultAgoraBaseDir     = "./agora_example"
	defaultUserID           = "0"
	defaultSampleRate       = 16000
	defaultAudioChannelType = agorasdk.AudioChannelTypeMono
	defaultAudioMode        = agorasdk.AudioModeChannel
	defaultPcmQueueSize     = 10
	defaultSleepInterval    = 40 * time.Millisecond

	// Max buffer size: 2 seconds at 48kHz.
	maxAudioBufferSize = 48000 * 2
)

func main() {
	var (
		appID       = flag.String("app-id", "", "The required Agora App ID (default: uses AGORA_APP_ID env if empty)")
		appCert     = flag.String("app-cert", "", "The required Agora App Certificate (default: uses AGORA_APP_CERT env if empty)")
		channelName = flag.String("channel-name", defaultChannelName, "RTC channel name to join")
		userID      = flag.String("user-id", defaultUserID, "User ID for the connection (string identifier)")
		logPath     = flag.String("log-path", defaultAgoraBaseDir+"/logs/agorasdk.log", "Path to the log file")
		configDir   = flag.String("config-dir", defaultAgoraBaseDir+"/config", "Directory for SDK configuration files")
		dataDir     = flag.String("data-dir", defaultAgoraBaseDir+"/data", "Directory for SDK data files")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Create a unbuffered channel to notify the global stop signal to all goroutines.
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		// Notify all goroutines to stop.
		close(stopCh)
		logf("Received termination signal, exiting application...")
	}()

	svc, err := agorasdk.NewService(
		agorasdk.WithAppID(*appID),
		agorasdk.WithAppCert(*appCert),
		agorasdk.WithChannelName(*channelName),
		agorasdk.WithUserID(*userID),
		agorasdk.WithLogPath(*logPath),
		agorasdk.WithConfigDir(*configDir),
		agorasdk.WithDataDir(*dataDir),
	)
	if err != nil {
		logFatalf("Failed to create service: %v", err)
	}
	defer svc.Release()

	conn, err := svc.NewRTCConnection(
		agorasdk.WithOnConnected(onConnected),
		agorasdk.WithOnDisconnected(onDisconnected),
		agorasdk.WithSampleRate(defaultSampleRate),
		agorasdk.WithAudioChannelType(defaultAudioChannelType),
		agorasdk.WithEnableReceiveAudioFrame(true),
		agorasdk.WithAudioMode(defaultAudioMode),
		agorasdk.WithPCMQueueSize(defaultPcmQueueSize),
	)
	if err != nil {
		logFatalf("Failed to create RTC connection: %v", err)
	}
	defer conn.Release()

	audioPlayer, err := newAudioPlayer(maxAudioBufferSize)
	if err != nil {
		logFatalf("Failed to create audio player: %v", err)
	}
	defer audioPlayer.terminate()

	go audioPlayer.play(stopCh)
	go audioPlayer.recvAudioFrames(conn, stopCh)

	// Wait for stop signal.
	for {
		select {
		case <-stopCh:
			if err := conn.Disconnect(); err != nil {
				logFatalf("Failed to disconnect RTC connection: %v", err)
			}
			return
		default:
			time.Sleep(defaultSleepInterval)
		}
	}
}

type audioPlayer struct {
	audioBuffer []int16
	mu          sync.Mutex
}

func newAudioPlayer(maxAudioBufferSize int) (*audioPlayer, error) {
	// Initialize portaudio.
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	return &audioPlayer{
		audioBuffer: make([]int16, 0, maxAudioBufferSize),
	}, nil
}

func (a *audioPlayer) terminate() {
	portaudio.Terminate()
}

func (a *audioPlayer) play(stopCh <-chan struct{}) {
	logf("Start audio play routine")

	outputBuffer := make([]int16, 1024)

	stream, err := portaudio.OpenDefaultStream(0, 1, float64(defaultSampleRate), len(outputBuffer), outputBuffer)
	if err != nil {
		logf("Failed to open audio stream: %v", err)
		return
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		logf("Failed to start audio stream: %v", err)
		return
	}
	defer stream.Stop()

	logf("Audio stream started successfully")

	// More frequent updates for smoother playback.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			logf("Received stop signal to audio playback routine, exiting...")
			return
		case <-ticker.C:
			// Get audio data from buffer.
			a.mu.Lock()
			available := len(a.audioBuffer)
			if available > 0 {
				// Copy up to outputBuffer size.
				copySize := min(available, len(outputBuffer))
				copy(outputBuffer, a.audioBuffer[:copySize])
				// Remove copied data from buffer.
				a.audioBuffer = a.audioBuffer[copySize:]
				a.mu.Unlock()

				// Write to audio stream.
				if err := stream.Write(); err != nil {
					logf("Failed to write to audio stream: %v", err)
				}
			} else {
				a.mu.Unlock()
				// Fill with silence when no data available.
				for i := range outputBuffer {
					outputBuffer[i] = 0
				}
				if err := stream.Write(); err != nil {
					logf("Failed to write silence to audio stream: %v", err)
				}
			}
		}
	}
}

func (a *audioPlayer) recvAudioFrames(conn *agorasdk.RTCConnection, stopCh <-chan struct{}) {
	logf("Start audio routine to process audio frames from queue")

	for {
		select {
		case <-stopCh:
			logf("Received stop signal to audio routine, exiting...")
			return
		default:
			frame, err := conn.FetchAudioFrame()
			if err == agorasdk.ErrEmptyPCMQueue {
				continue
			}
			if err != nil {
				logf("Failed to fetch audio frame: %v", err)
				continue
			}

			// Convert audio frame buffer to int16 slice and add to audio buffer for playback.
			if len(frame.Buffer) > 0 && frame.BytesPerSample == 2 {
				// Convert []byte to []int16 (little-endian).
				totalSamples := len(frame.Buffer) / 2
				rawSamples := make([]int16, totalSamples)
				for i := range rawSamples {
					rawSamples[i] = int16(binary.LittleEndian.Uint16(frame.Buffer[i*2 : (i+1)*2]))
				}
				// Add samples to audio buffer for playback.
				a.mu.Lock()
				a.audioBuffer = append(a.audioBuffer, rawSamples...)
				bufferSize := len(a.audioBuffer)
				// Limit buffer size to avoid memory issues.
				if len(a.audioBuffer) > maxAudioBufferSize {
					// Keep the most recent data.
					a.audioBuffer = a.audioBuffer[len(a.audioBuffer)-maxAudioBufferSize:]
				}
				a.mu.Unlock()
				// Log sample rate mismatch if any.
				if frame.SamplesPerSec != defaultSampleRate {
					logf("Warning: Audio frame sample rate (%d) differs from playback rate (%d)", frame.SamplesPerSec, defaultSampleRate)
				}
				// Reduce log frequency to avoid spam.
				if bufferSize%10000 < len(rawSamples) {
					logf("Added %d samples (from %d channels, %dHz) to audio buffer, total: %d", len(rawSamples), frame.Channels, frame.SamplesPerSec, bufferSize)
				}
			} else {
				logf("Skipping audio frame: Buffer len=%d, BytesPerSample=%d", len(frame.Buffer), frame.BytesPerSample)
			}

			if err := conn.PushAudioPCMData(frame.Buffer, 0); err != nil {
				logFatalf("Failed to push audio PCM data: %v", err)
			}
		}
	}
}

func logf(format string, args ...any) {
	log.Printf("[%s] %s", exampleName, fmt.Sprintf(format, args...))
}

func logFatalf(format string, args ...any) {
	log.Fatalf("[%s] %s", exampleName, fmt.Sprintf(format, args...))
}

func onConnected(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, reason int) {
	logf("OnConnected for reason %d", reason)
}

func onDisconnected(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, reason int) {
	logf("OnDisconnected for reason %d", reason)
}
