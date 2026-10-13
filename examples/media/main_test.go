package main

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	sdk "github.com/zyy17/agora-server-sdk"
)

func TestChannelSetup(t *testing.T) {
	for _, test := range []struct {
		mode  string
		video bool
		want  sdk.ChannelConfig
	}{
		{"send-yuv", true, sdk.ChannelConfig{Video: sdk.VideoConfig{Mode: sdk.RawVideo}}},
		{"recv-yuv", true, sdk.ChannelConfig{Role: sdk.Audience, Receive: sdk.ReceiveConfig{Video: true}}},
		{"send-h264", true, sdk.ChannelConfig{Video: sdk.VideoConfig{Mode: sdk.EncodedVideo}}},
		{"recv-h264", true, sdk.ChannelConfig{Role: sdk.Audience, Receive: sdk.ReceiveConfig{EncodedVideo: true}}},
		{"send-aac", false, sdk.ChannelConfig{Audio: sdk.AudioConfig{Mode: sdk.EncodedAudio}}},
		{"send-opus", false, sdk.ChannelConfig{Audio: sdk.AudioConfig{Mode: sdk.EncodedAudio}}},
		{"recv-encoded-audio", false, sdk.ChannelConfig{Role: sdk.Audience, Receive: sdk.ReceiveConfig{EncodedAudio: true}}},
		{"transcode-h264", true, sdk.ChannelConfig{Video: sdk.VideoConfig{Mode: sdk.RawVideo}}},
		{"transcode-opus", false, sdk.ChannelConfig{Audio: sdk.AudioConfig{Mode: sdk.PCM}}},
	} {
		t.Run(test.mode, func(t *testing.T) {
			options := mediaOptions{mode: test.mode, width: 640, height: 360, fps: 15}
			video, configure, err := options.channelSetup()
			if err != nil {
				t.Fatal(err)
			}
			got := sdk.ChannelConfig{Channel: "channel", UserID: "user", Token: "token"}
			configure(&got)
			test.want.Channel, test.want.UserID, test.want.Token = got.Channel, got.UserID, got.Token
			if video != test.video || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("video=%v config=%+v, want video=%v config=%+v", video, got, test.video, test.want)
			}
			if got.Channel != "channel" || got.UserID != "user" || got.Token != "token" {
				t.Fatal("channel identity changed")
			}
		})
	}
}

func TestChannelSetupRejectsInvalidOptions(t *testing.T) {
	for _, options := range []mediaOptions{
		{mode: "unknown", width: 640, height: 360, fps: 15},
		{mode: "send-yuv", width: 3, height: 360, fps: 15},
		{mode: "send-yuv", width: 640, height: 1, fps: 15},
		{mode: "send-yuv", width: 8194, height: 360, fps: 15},
		{mode: "send-yuv", width: 640, height: 360, fps: 0},
		{mode: "send-yuv", width: 640, height: 360, fps: 121},
	} {
		if _, _, err := options.channelSetup(); err == nil {
			t.Fatalf("accepted %+v", options)
		}
	}
}

func TestOpenFileRequiresInputOrReceiveUser(t *testing.T) {
	for _, mode := range []string{"send-yuv", "send-opus", "recv-yuv", "recv-h264", "recv-encoded-audio"} {
		options := mediaOptions{mode: mode}
		if _, err := options.openFile(); err == nil {
			t.Fatalf("accepted incomplete options for %s", mode)
		}
	}
}

func TestWriteYUVRemovesPaddingAndRoundsChromaDimensions(t *testing.T) {
	frame := sdk.ReceivedVideoFrame{
		Width: 3, Height: 3, YStride: 4, UStride: 3, VStride: 3,
		Y: []byte{1, 2, 3, 99, 4, 5, 6, 99, 7, 8, 9, 99},
		U: []byte{10, 11, 99, 12, 13, 99},
		V: []byte{14, 15, 99, 16, 17, 99},
	}
	var output bytes.Buffer
	if err := writeYUV(&output, frame); err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}
	if !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("got %v, want %v", output.Bytes(), want)
	}
	frame.Y = frame.Y[:2]
	if err := writeYUV(&output, frame); err == nil {
		t.Fatal("accepted a truncated plane")
	}
}

func TestWaitUntilCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, at := range []time.Time{time.Now().Add(-time.Hour), time.Now().Add(time.Hour)} {
		if err := waitUntil(ctx, at); !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	}
	if err := waitUntil(context.Background(), time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func TestIsH264Keyframe(t *testing.T) {
	for _, test := range []struct {
		data []byte
		want bool
	}{
		{nil, false},
		{[]byte{0, 0, 1}, false},
		{[]byte{0, 0, 1, 0x65}, true},
		{[]byte{0, 0, 0, 1, 0x65}, true},
		{[]byte{0, 0, 1, 0x61}, false},
	} {
		if got := isH264Keyframe(test.data); got != test.want {
			t.Fatalf("isH264Keyframe(%x)=%v", test.data, got)
		}
	}
}
