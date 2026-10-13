package agoraservice

import "testing"

func TestCPUVideoRejectsPlatformHandles(t *testing.T) {
	for _, format := range []VideoPixelFormat{VideoPixelDefault, VideoTexture2D, VideoTextureOES, VideoCVPixelNV12, VideoCVPixelI420, VideoCVPixelBGRA, VideoCVPixelP010, VideoTextureID3D11Texture2D} {
		f := &ExternalVideoFrame{Type: VideoBufferRawData, Format: format, Stride: 2, Height: 2, Buffer: make([]byte, 16)}
		if f.Validate() == nil {
			t.Fatal("accepted unsupported format", format)
		}
	}
	f := &ExternalVideoFrame{Type: VideoBufferTexture, Format: VideoPixelRGBA, Stride: 2, Height: 2, Buffer: make([]byte, 16)}
	if f.Validate() == nil {
		t.Fatal("accepted texture type")
	}
	f.Type = VideoBufferRawData
	f.TextureID = 1
	if f.Validate() == nil {
		t.Fatal("silently dropped texture ID")
	}
	var absent *ExternalVideoFrame
	if absent.Validate() == nil {
		t.Fatal("accepted nil")
	}
}

func TestTrackBridgesAbsentResources(t *testing.T) {
	for _, conn := range []*RtcConnection{nil, {}} {
		for _, result := range []int{conn.SetAudioTrackEnabled(true), conn.SetVideoTrackEnabled(true), conn.AdjustAudioPublishVolume(100), conn.SetAudioSendDelayMs(10), conn.SetAudioBufferCapacity(30), conn.ClearAudioBuffer()} {
			if result != -2000 {
				t.Fatal("invalid handle passed", result)
			}
		}
	}
}
