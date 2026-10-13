package agorasdk_test

import (
	"fmt"

	agora "github.com/zyy17/agora-server-sdk"
)

func ExampleSession_SendCPUVideo() {
	// session was joined as Broadcaster with Video.Mode = RawVideo.
	var session *agora.Session
	if session == nil {
		return
	}
	// Four padded pixels at the right are excluded from the visible 640×480 image.
	frame := agora.CPUVideoFrame{Data: make([]byte, 644*480*4), Stride: 644, Height: 480,
		Format: agora.RGBA, CropRight: 4, FillAlphaBuffer: true,
		ColorSpace: agora.ColorSpace{PrimariesID: 1, TransferID: 1, MatrixID: 1, RangeID: 2}}
	if err := session.SendCPUVideo(frame); err != nil {
		fmt.Println(err)
	}
}

func ExampleEncodedVideoFrame() {
	frame := agora.EncodedVideoFrame{Data: []byte{1, 2, 3}, Codec: agora.AV1,
		Width: 640, Height: 480, FPS: 30, FrameType: agora.VideoFrameDroppable, TrackID: 7, UID: 42}
	// Payload bytes above illustrate metadata only; provide a real encoded
	// access unit and verify native/peer AV1 support before sending.
	fmt.Println(frame.TrackID, frame.UID, frame.FrameType == agora.VideoFrameDroppable)
	// Output: 7 42 true
}

func ExampleSession_SendPCMFrame() {
	// session was joined with Audio.Mode = PCM and 16 kHz mono send format.
	var session *agora.Session
	if session == nil {
		return
	}
	frame := agora.AudioFrame{Data: make([]byte, 320), SampleRate: 16000, Channels: 1,
		SamplesPerChannel: 160, BytesPerSample: 2, TimestampMillis: 1234, PresentationMillis: -17}
	if err := session.SendPCMFrame(frame); err != nil {
		fmt.Println(err)
	}
}

func ExampleNewVideoDecoder() {
	decoder, err := agora.NewVideoDecoder() // Requires -tags avcodec and FFmpeg.
	if err != nil {
		return
	}
	defer decoder.Close()
	// pixels, err := decoder.Decode(accessUnit, agora.H264, agora.VideoDecodeOptions{})
	// ErrDecoderNeedsInput means submit another unit. Successful pixels are Go-owned.
}
