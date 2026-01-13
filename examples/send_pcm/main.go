package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	agorasdk "github.com/zyy17/agora-server-sdk"
	agoraservice "github.com/zyy17/agora-server-sdk/agora/rtc"
)

const (
	exampleName                = "send_pcm"
	defaultChannelName         = "agora_sdk_example"
	defaultAgoraBaseDir        = "./agora_example"
	defaultUserID              = "0"
	defaultSampleRate          = 16000
	defaultAudioChannelType    = agorasdk.AudioChannelTypeMono
	defaultAudioFile           = "./examples/testdata/send_audio_16k_1ch.pcm"
	defaultSecondsPerPush      = 1
	defaultAudioBytesPerSample = 2
	defaultPushInterval        = 5 * time.Second
	defaultSleepInterval       = 100 * time.Millisecond
)

func main() {
	var (
		appID         = flag.String("app-id", "", "The required Agora App ID (default: uses AGORA_APP_ID env if empty)")
		appCert       = flag.String("app-cert", "", "The required Agora App Certificate (default: uses AGORA_APP_CERT env if empty)")
		channelName   = flag.String("channel-name", defaultChannelName, "RTC channel name to join")
		userID        = flag.String("user-id", defaultUserID, "User ID for the connection (string identifier)")
		logPath       = flag.String("log-path", defaultAgoraBaseDir+"/logs/agorasdk.log", "Path to the log file")
		configDir     = flag.String("config-dir", defaultAgoraBaseDir+"/config", "Directory for SDK configuration files")
		dataDir       = flag.String("data-dir", defaultAgoraBaseDir+"/data", "Directory for SDK data files")
		audioFilePath = flag.String("audio-file-path", defaultAudioFile, "Path to the audio file")
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
		logFatalf("failed to create service: %v", err)
	}
	defer svc.Release()

	conn, err := svc.NewRTCConnection(agorasdk.WithOnConnected(onConnected),
		agorasdk.WithSampleRate(defaultSampleRate),
		agorasdk.WithAudioChannelType(defaultAudioChannelType))
	if err != nil {
		logFatalf("failed to create RTC connection: %v", err)
	}
	defer conn.Release()

	// Publish audio to the RTC connection.
	if err := conn.PublishAudio(); err != nil {
		logFatalf("failed to publish audio: %v", err)
	}

	file, err := os.Open(*audioFilePath)
	if err != nil {
		logFatalf("Error opening file: %v\n", err)
	}
	defer file.Close()

	// Read the audio file and push the audio data to the RTC connection.
	go readFileToConnection(file, conn, defaultPushInterval, stopCh, int(defaultSampleRate))

	for {
		select {
		case <-stopCh:
			// Disconnect the RTC connection.
			if err := conn.Disconnect(); err != nil {
				logFatalf("failed to disconnect RTC connection: %v", err)
			}
			return
		default:
			time.Sleep(defaultSleepInterval)
		}
	}
}

func pushAudioFromFile(file *os.File, conn *agorasdk.RTCConnection, sampleRate int) {
	chunkSize := sampleRate * defaultSecondsPerPush * defaultAudioBytesPerSample
	buffer := make([]byte, chunkSize)
	for {
		readLen, err := file.Read(buffer)
		if err != nil || readLen < chunkSize {
			logf("Read up to EOF at %d bytes", readLen)
			// Reset the file pointer to the beginning of the file.
			file.Seek(0, 0)
			break
		}
		logf("Read %d bytes of audio data from file", readLen)

		// Push the audio data to the RTC connection.
		if err := conn.PushAudioPCMData(buffer, 0); err != nil {
			logFatalf("failed to push audio PCM data: %v", err)
		}
	}
}

func readFileToConnection(file *os.File, conn *agorasdk.RTCConnection, interval time.Duration, stopCh <-chan struct{}, sampleRate int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logf("Start to read file to connection with interval %s", interval)

	for {
		select {
		case <-stopCh:
			logf("Received stop signal to read file to connection, exiting...")
			return
		case <-ticker.C:
			if conn.IsPushToRtcCompleted() {
				// Push the audio data to the RTC connection.
				pushAudioFromFile(file, conn, sampleRate)
				logf("Pushed audio data to RTC connection with sample rate %dHz", sampleRate)
				continue
			}
			logf("The audio data is not pushed to RTC connection yet, waiting for next interval...")
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
