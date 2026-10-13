//go:build agora_test

package agoraservice

import (
	"reflect"
	"testing"
)

func TestNativeMediaFrameABI(t *testing.T) {
	f := &ExternalVideoFrame{Type: VideoBufferArray, Format: VideoPixelRGBA, Buffer: make([]byte, 128), Stride: 8, Height: 4, CropLeft: 1, CropTop: 1, CropRight: 2, CropBottom: 1, Rotation: VideoOrientation270, Timestamp: -0x123456789ab, MetadataBuffer: []byte{1, 2}, AlphaBuffer: make([]byte, 32), AlphaMode: 4, ColorSpace: ColorSpaceType{1, 2, 3, 4}}
	f.Buffer[0], f.AlphaBuffer[0] = 7, 9
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := mediaVideoFrameRoundTrip(f); !reflect.DeepEqual(got, f) {
		t.Fatalf("round trip mismatch: %+v\nwant %+v", got, f)
	}
	f.AlphaBuffer = nil
	f.FillAlphaBuffer = true
	if got := mediaVideoFrameRoundTrip(f); !got.FillAlphaBuffer {
		t.Fatal("alpha extraction flag lost")
	}
	if got := mediaReceivedAlphaMode(); got != 4 {
		t.Fatal("received alpha mode lost", got)
	}
}

func TestNativeMediaEncodedABI(t *testing.T) {
	v := &EncodedVideoFrameInfo{CodecType: VideoCodecTypeAv1, Width: 123, Height: 234, FramesPerSecond: 60, FrameType: VideoFrameTypeBFrame, Rotation: VideoOrientation90, TrackId: -8, CaptureTimeMs: -0x123456789ab, DecodeTimeMs: 0x23456789abc, Uid: 0xffffffff, StreamType: -2, PresentTimeMs: -0x3456789abcd}
	if got := mediaEncodedVideoRoundTrip(v); !reflect.DeepEqual(got, v) {
		t.Fatalf("video metadata lost: %+v", got)
	}
	a := &EncodedAudioFrameInfo{Speech: true, Codec: AudioCodecOpusMC, SampleRateHz: 48000, SamplesPerChannel: 960, SendEvenIfEmpty: true, NumberOfChannels: 8, CaptureTimeMs: -0x123456789ab}
	if got := mediaEncodedAudioRoundTrip(a); !reflect.DeepEqual(got, a) {
		t.Fatalf("audio metadata lost: %+v", got)
	}
}
