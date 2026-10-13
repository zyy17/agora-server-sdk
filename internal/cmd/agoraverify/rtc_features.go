package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"path/filepath"
	"sync"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"github.com/zyy17/agora-server-sdk/token"
)

const (
	testWidth  = 160
	testHeight = 120
	testRate   = 16000
)

var rtcFeatureOperations = []string{
	"prepare", "initialize", "join", "peer_join", "renew_token",
	"audio_send", "audio_receive", "video_send", "video_receive", "data_send", "data_receive",
	"leave", "peer_leave", "close",
}

func runRTCFeatures(ctx context.Context, c credentials, dir, identity string, note record) {
	channel, senderID, receiverID := identity, identity+"-send", identity+"-recv"
	senderToken, err := token.RTC(token.Config{
		AppID: c.appID, Certificate: c.certificate, Channel: channel,
		UserID: senderID, Role: token.Publisher, Lifetime: tokenLifetime,
	})
	if err != nil {
		note("prepare", err)
		return
	}
	receiverToken, err := rtcToken(c, channel, receiverID)
	note("prepare", err)
	if err != nil {
		return
	}
	service, err := sdk.NewService(sdk.Config{
		AppID: c.appID, IdentityMode: sdk.StringIdentity, Video: true,
		LogPath: filepath.Join(dir, "rtc.log"), ConfigDir: dir, DataDir: dir,
		ConfigureNative: func(c *rtc.AgoraServiceConfig) error {
			c.EnableAudioProcessor = true
			c.EnableAudioDevice = false
			c.AudioScenario = rtc.AudioScenarioDefault
			c.LogLevel = 0
			return nil
		},
	})
	note("initialize", err)
	if err != nil {
		return
	}
	defer func() { note("close", service.Close()) }()
	receiver, err := service.Join(ctx, sdk.ChannelConfig{
		Channel: channel, UserID: receiverID, Token: receiverToken, Role: sdk.Audience,
		Audio: sdk.AudioConfig{Mode: sdk.NoAudio, Scenario: sdk.ScenarioDefault, Receive: sdk.PCMFormat{SampleRate: testRate, Channels: 1}},
		Video: sdk.VideoConfig{Mode: sdk.NoVideo}, Receive: sdk.ReceiveConfig{Video: true},
		JoinTimeout: 10 * time.Second, QueueCapacity: 512,
	})
	note("join", err)
	if err != nil {
		return
	}
	defer func() { note("leave", receiver.Close()) }()
	encoder := rtc.NewVideoEncoderConfiguration()
	encoder.Width, encoder.Height, encoder.Framerate, encoder.Bitrate = testWidth, testHeight, 10, 150
	sender, err := service.Join(ctx, sdk.ChannelConfig{
		Channel: channel, UserID: senderID, Token: senderToken, Role: sdk.Broadcaster,
		Audio: sdk.AudioConfig{Mode: sdk.PCM, Scenario: sdk.ScenarioDefault, Send: sdk.PCMFormat{SampleRate: testRate, Channels: 1}},
		Video: sdk.VideoConfig{Mode: sdk.RawVideo, Encoder: encoder}, Receive: sdk.ReceiveConfig{Disabled: true},
		JoinTimeout: 10 * time.Second,
	})
	note("peer_join", err)
	if err != nil {
		return
	}
	defer func() { note("peer_leave", sender.Close()) }()
	// Submit a fresh receiver token; this checks local renewal acceptance. A
	// 120-second token is not allowed to expire during this bounded test.
	replacement, err := rtcToken(c, channel, receiverID)
	if err == nil {
		err = receiver.RenewToken(replacement)
	}
	note("renew_token", err)
	testRTCExchange(ctx, rtcMediaAdapter{sender}, rtcMediaAdapter{receiver}, channel, senderID, note)
}

type rtcMedia interface {
	SendPCM([]byte) error
	SendVideo(sdk.VideoFrame) error
	SendMessage([]byte) error
	readAudio(context.Context) (sdk.AudioFrame, error)
	readVideo(context.Context) (sdk.ReceivedVideoFrame, error)
	readMessage(context.Context) (sdk.Message, error)
}

type rtcMediaAdapter struct{ *sdk.Session }

func (s rtcMediaAdapter) readAudio(ctx context.Context) (sdk.AudioFrame, error) {
	return s.Audio().Next(ctx)
}

func (s rtcMediaAdapter) readVideo(ctx context.Context) (sdk.ReceivedVideoFrame, error) {
	return s.Video().Next(ctx)
}

func (s rtcMediaAdapter) readMessage(ctx context.Context) (sdk.Message, error) {
	return s.Messages().Next(ctx)
}

func testRTCExchange(ctx context.Context, sender, receiver rtcMedia, channel, user string, note record) {
	// Send and observe concurrently so a local send return cannot be mistaken
	// for successful delivery. All receive checks validate source and content.
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	payload := append([]byte{0, 1, 255, 127}, []byte(channel)...)
	var wg sync.WaitGroup
	observe := func(operation string, fn func(context.Context) error) {
		wg.Add(1)
		go func() { defer wg.Done(); note(operation, fn(ctx)) }()
	}
	observe("audio_receive", func(ctx context.Context) error {
		for {
			f, err := receiver.readAudio(ctx)
			if err != nil {
				return err
			}
			if matchesTone(f, channel, user) {
				return nil
			}
		}
	})
	observe("video_receive", func(ctx context.Context) error {
		for {
			f, err := receiver.readVideo(ctx)
			if err != nil {
				return err
			}
			if matchesPattern(f, channel, user) {
				return nil
			}
		}
	})
	observe("data_receive", func(ctx context.Context) error {
		for {
			m, err := receiver.readMessage(ctx)
			if err != nil {
				return err
			}
			if m.Channel == channel && m.UserID == user && bytes.Equal(m.Data, payload) {
				return nil
			}
		}
	})
	// Four seconds of synthetic media, paced at 10ms PCM and 10fps video.
	// Repeated transient data messages are intentional (the RTC stream is
	// unreliable); no durable data is created by this test.
	var audioErr, videoErr, dataErr error
	var audioSent, videoSent, dataSent bool
	start := time.Now()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	frame := patternFrame()
	for i := 0; i < 400; i++ {
		select {
		case <-ctx.Done():
			if audioErr == nil {
				audioErr = ctx.Err()
			}
			if videoErr == nil {
				videoErr = ctx.Err()
			}
			if dataErr == nil {
				dataErr = ctx.Err()
			}
			i = 400
			continue
		case <-ticker.C:
		}
		if audioErr == nil {
			audioErr = sender.SendPCM(toneFrame(i * 160))
			audioSent = true
		}
		if i%10 == 0 && videoErr == nil {
			frame.TimestampMillis = start.UnixMilli() + int64(i*10)
			videoErr = sender.SendVideo(frame)
			videoSent = true
		}
		if i%25 == 0 && dataErr == nil {
			dataErr = sender.SendMessage(payload)
			dataSent = true
		}
	}
	if !audioSent && audioErr == nil {
		audioErr = errors.New("audio not sent")
	}
	if !videoSent && videoErr == nil {
		videoErr = errors.New("video not sent")
	}
	if !dataSent && dataErr == nil {
		dataErr = errors.New("data not sent")
	}
	note("audio_send", audioErr)
	note("video_send", videoErr)
	note("data_send", dataErr)
	wg.Wait()
}

func toneFrame(offset int) []byte {
	out := make([]byte, 320)
	for i := 0; i < 160; i++ {
		value := int16(6000 * math.Sin(2*math.Pi*440*float64(offset+i)/testRate))
		binary.LittleEndian.PutUint16(out[i*2:], uint16(value))
	}
	return out
}

func matchesTone(f sdk.AudioFrame, channel, user string) bool {
	if f.Channel != channel || f.UserID != user || f.SampleRate != testRate || f.Channels != 1 || f.BytesPerSample != 2 || len(f.Data) < 160 || len(f.Data)%2 != 0 {
		return false
	}
	var energy, real, imaginary float64
	n := len(f.Data) / 2
	for i := 0; i < n; i++ {
		value := float64(int16(binary.LittleEndian.Uint16(f.Data[i*2:])))
		angle := 2 * math.Pi * 440 * float64(i) / testRate
		energy += value * value
		real += value * math.Cos(angle)
		imaginary += value * math.Sin(angle)
	}
	// Lossy codecs and resampling change bytes/amplitude. Require meaningful
	// energy plus concentration at the known 440Hz test tone, not merely any
	// nonempty or non-silent callback.
	return energy/float64(n) > 2500 && (real*real+imaginary*imaginary)/(energy*float64(n)) > 0.15
}

func patternFrame() sdk.VideoFrame {
	data := make([]byte, testWidth*testHeight*3/2)
	levels := [4]byte{50, 100, 150, 200}
	for y := 0; y < testHeight; y++ {
		for x := 0; x < testWidth; x++ {
			data[y*testWidth+x] = levels[(y/(testHeight/2))*2+x/(testWidth/2)]
		}
	}
	for i := testWidth * testHeight; i < testWidth*testHeight*5/4; i++ {
		data[i] = 90
	}
	for i := testWidth * testHeight * 5 / 4; i < len(data); i++ {
		data[i] = 160
	}
	return sdk.VideoFrame{Data: data, Width: testWidth, Height: testHeight, Format: sdk.I420}
}

func matchesPattern(f sdk.ReceivedVideoFrame, channel, user string) bool {
	if f.Channel != channel || f.UserID != user || f.Width != testWidth || f.Height != testHeight || f.YStride < testWidth || len(f.Y) < f.YStride*testHeight {
		return false
	}
	for i, expected := range []int{50, 100, 150, 200} {
		x, y := (i%2)*testWidth/2+testWidth/4, (i/2)*testHeight/2+testHeight/4
		sum := 0
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				sum += int(f.Y[(y+dy)*f.YStride+x+dx])
			}
		}
		if math.Abs(float64(sum)/25-float64(expected)) > 25 {
			return false
		}
	}
	return true
}
