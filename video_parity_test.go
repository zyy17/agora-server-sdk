package agorasdk

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestCPUVideoLayoutsAndCrop(t *testing.T) {
	for _, tc := range []struct {
		format PixelFormat
		bytes  int
	}{{I420, 48}, {NV12, 48}, {NV21, 48}, {I422, 64}, {I010, 96}, {RGBA, 128}, {BGRA, 128}} {
		f := CPUVideoFrame{Data: make([]byte, tc.bytes), Stride: 8, Height: 4, Format: tc.format, CropRight: 2, Alpha: make([]byte, 32), BufferType: CPUArray,
			ColorSpace: ColorSpace{1, 2, 3, 4}, AlphaMode: AlphaRight, Rotation: 90, TimestampMillis: -123, Metadata: []byte{7}}
		if err := f.validate(); err != nil {
			t.Fatalf("format %d: %v", tc.format, err)
		}
		n := f.nativeFrame()
		if n.Type != rtc.VideoBufferArray || n.Stride != 8 || n.Height != 4 || n.CropRight != 2 || n.AlphaMode != 4 || n.ColorSpace.TransferId != 2 || n.Timestamp != -123 || n.Rotation != rtc.VideoOrientation90 {
			t.Fatalf("native metadata lost: %+v", n)
		}
		if &n.Buffer[0] != &f.Data[0] || &n.AlphaBuffer[0] != &f.Alpha[0] {
			t.Fatal("send unnecessarily copies data")
		}
	}
	f := CPUVideoFrame{Data: make([]byte, 16), Stride: 2, Height: 2, Format: RGBA, FillAlphaBuffer: true}
	if err := f.validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCPUVideoRejectsUnsafeLayouts(t *testing.T) {
	base := CPUVideoFrame{Data: make([]byte, 48), Stride: 8, Height: 4, Format: I420}
	for _, change := range []func(*CPUVideoFrame){
		func(f *CPUVideoFrame) { f.Data = f.Data[:47] },
		func(f *CPUVideoFrame) { f.Stride = math.MaxInt },
		func(f *CPUVideoFrame) { f.Height = 0 },
		func(f *CPUVideoFrame) { f.CropLeft = -1 },
		func(f *CPUVideoFrame) { f.CropRight = 8 },
		func(f *CPUVideoFrame) { f.CropTop = 1 },
		func(f *CPUVideoFrame) { f.Alpha = []byte{1} },
		func(f *CPUVideoFrame) { f.AlphaMode = 5 },
		func(f *CPUVideoFrame) { f.FillAlphaBuffer = true },
		func(f *CPUVideoFrame) { f.BufferType = 20 },
		func(f *CPUVideoFrame) { f.Rotation = 45 },
		func(f *CPUVideoFrame) { f.ColorSpace.MatrixID = -1 },
		func(f *CPUVideoFrame) { f.Format = PixelFormat(rtc.VideoTexture2D) },
	} {
		f := base
		change(&f)
		if err := f.validate(); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("accepted %+v: %v", f, err)
		}
	}
	if err := (&Session{closed: true}).SendCPUVideo(base); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func TestEncodedVideoParity(t *testing.T) {
	for _, codec := range []VideoCodec{VP8, H264, H265, GenericVideo, GenericH264, AV1, VP9, JPEG} {
		for _, kind := range []VideoFrameType{VideoFrameBlank, VideoFrameKey, VideoFrameDelta, VideoFrameB, VideoFrameDroppable, VideoFrameUnknown} {
			f := EncodedVideoFrame{Data: []byte{1}, Codec: codec, Width: 2, Height: 2, FrameType: kind, TrackID: -7, UID: math.MaxUint32, StreamID: -3}
			if err := f.validate(); err != nil {
				t.Fatal(err)
			}
			n := f.nativeInfo()
			if receivedVideoFrameType(n.FrameType) != kind || n.TrackId != -7 || n.Uid != math.MaxUint32 || n.CodecType != rtc.VideoCodecType(codec) || n.StreamType != -3 {
				t.Fatalf("lost metadata: %+v", n)
			}
		}
	}
	f := EncodedVideoFrame{Data: []byte{1}, Codec: H264, Width: 2, Height: 2, KeyFrame: true}
	if f.nativeInfo().FrameType != rtc.VideoFrameTypeKeyFrame {
		t.Fatal("key convenience ignored")
	}
	f.FrameType = VideoFrameB
	if err := f.validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal("accepted contradictory frame type", err)
	}
	f.FrameType, f.KeyFrame, f.Codec, f.SEI = VideoFrameBlank, false, VP8, []byte{1}
	if err := f.validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal("accepted unsupported SEI", err)
	}
	f.SEI = nil
	if err := validateTranscodeVideo(f); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal("transcoder accepted unsupported codec", err)
	}
}

func TestReceiveVideoPreservesDescriptorsAndOwnership(t *testing.T) {
	s := &Session{video: newStream[ReceivedVideoFrame](1), encodedVideo: newStream[ReceivedEncodedVideo](1)}
	defer s.video.close()
	defer s.encodedVideo.close()
	f := &rtc.VideoFrame{Type: rtc.VideoBufferArray, AVSyncType: 3, AlphaMode: 4, TextureID: 12, Matrix: [16]float32{1, 2}, YBuffer: []byte{1}, MetadataBuffer: []byte{2}, AlphaBuffer: []byte{3}}
	s.videoObserver().OnFrame("room", "alice", f)
	f.YBuffer[0], f.MetadataBuffer[0], f.AlphaBuffer[0] = 8, 8, 8
	got, err := s.video.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != f.Type || got.AVSyncType != 3 || got.AlphaMode != AlphaRight || got.TextureID != 12 || got.Matrix != f.Matrix || got.Y[0] != 1 || got.Metadata[0] != 2 || got.Alpha[0] != 3 {
		t.Fatalf("lost descriptors: %+v", got)
	}
	data := []byte{1}
	sei := []byte{2}
	info := &rtc.EncodedVideoFrameInfo{CodecType: rtc.VideoCodecTypeAv1, FrameType: rtc.VideoFrameTypeDroppableFrame, TrackId: -8, Uid: 123, SeiData: sei}
	s.encodedVideoObserver().OnEncodedVideoFrame("room", "alice", data, info)
	data[0], sei[0] = 9, 9
	encoded, err := s.encodedVideo.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if encoded.Frame.FrameType != VideoFrameDroppable || encoded.Frame.TrackID != -8 || encoded.Frame.UID != 123 || !reflect.DeepEqual(encoded.Frame.Data, []byte{1}) || !reflect.DeepEqual(encoded.Frame.SEI, []byte{2}) {
		t.Fatalf("lost encoded metadata: %+v", encoded)
	}
}

func TestVideoEncoderAndSimulcastBounds(t *testing.T) {
	cfg := rtc.NewVideoEncoderConfiguration()
	if err := validateVideoEncoder(cfg); err != nil {
		t.Fatal("native defaults rejected", err)
	}
	for _, change := range []func(*rtc.VideoEncoderConfiguration){
		func(c *rtc.VideoEncoderConfiguration) { c.Width = 1 },
		func(c *rtc.VideoEncoderConfiguration) { c.Framerate = 241 },
		func(c *rtc.VideoEncoderConfiguration) { c.Bitrate = math.MaxInt32 },
		func(c *rtc.VideoEncoderConfiguration) { c.MinBitrate = math.MinInt32 },
		func(c *rtc.VideoEncoderConfiguration) { c.CodecType = 99 },
		func(c *rtc.VideoEncoderConfiguration) { c.OrientationMode = 99 },
		func(c *rtc.VideoEncoderConfiguration) { c.DegradePreference = 99 },
		func(c *rtc.VideoEncoderConfiguration) { c.MirrorMode = 99 },
	} {
		copy := *cfg
		change(&copy)
		if err := validateVideoEncoder(&copy); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(copy, err)
		}
	}
	if err := validateEncodedVideoSender(&rtc.VideoEncodedImageSenderOptions{CodecType: rtc.VideoCodecTypeAv1, TargetBitrate: 2000}); err != nil {
		t.Fatal(err)
	}
	if err := validateEncodedVideoSender(&rtc.VideoEncodedImageSenderOptions{CcMode: 99}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if err := validateSimulcast(false, rtc.SimulcastStreamConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := validateSimulcast(true, rtc.SimulcastStreamConfig{Width: 320, Height: 240, Framerate: 15, Bitrate: 500}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []rtc.SimulcastStreamConfig{{}, {Width: 320, Height: 240, Framerate: 241}, {Width: 320, Height: 240, Framerate: 15, Bitrate: math.MaxInt32 + 1}} {
		if err := validateSimulcast(true, c); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(c, err)
		}
	}
}

func TestEncodedVideoFrameTypeDefaultsAndExplicitMapping(t *testing.T) {
	for _, tt := range []struct {
		name    string
		kind    VideoFrameType
		key     bool
		want    rtc.VideoFrameType
		invalid bool
	}{
		{name: "zero delta", want: rtc.VideoFrameTypeDeltaFrame},
		{name: "zero key", key: true, want: rtc.VideoFrameTypeKeyFrame},
		{name: "explicit blank", kind: VideoFrameBlank, want: rtc.VideoFrameTypeBlankFrame},
		{name: "explicit key", kind: VideoFrameKey, want: rtc.VideoFrameTypeKeyFrame},
		{name: "explicit key convenience", kind: VideoFrameKey, key: true, want: rtc.VideoFrameTypeKeyFrame},
		{name: "explicit delta", kind: VideoFrameDelta, want: rtc.VideoFrameTypeDeltaFrame},
		{name: "explicit B", kind: VideoFrameB, want: rtc.VideoFrameTypeBFrame},
		{name: "explicit droppable", kind: VideoFrameDroppable, want: rtc.VideoFrameTypeDroppableFrame},
		{name: "explicit unknown", kind: VideoFrameUnknown, want: rtc.VideoFrameTypeUnknown},
		{name: "blank key conflict", kind: VideoFrameBlank, key: true, invalid: true},
		{name: "delta key conflict", kind: VideoFrameDelta, key: true, invalid: true},
		{name: "B key conflict", kind: VideoFrameB, key: true, invalid: true},
		{name: "droppable key conflict", kind: VideoFrameDroppable, key: true, invalid: true},
		{name: "unknown key conflict", kind: VideoFrameUnknown, key: true, invalid: true},
		{name: "invalid enum", kind: 100, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := EncodedVideoFrame{Data: []byte{1}, Codec: H264, Width: 2, Height: 2, FrameType: tt.kind, KeyFrame: tt.key}
			err := f.validate()
			if tt.invalid {
				if !errors.Is(err, ErrInvalidConfig) {
					t.Fatal(err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := f.nativeInfo().FrameType; got != tt.want {
				t.Fatalf("frame type = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEncodedVideoFrameTypeReceiveSendRoundTrip(t *testing.T) {
	for _, nativeKind := range []rtc.VideoFrameType{rtc.VideoFrameTypeBlankFrame, rtc.VideoFrameTypeKeyFrame, rtc.VideoFrameTypeDeltaFrame, rtc.VideoFrameTypeBFrame, rtc.VideoFrameTypeDroppableFrame, rtc.VideoFrameTypeUnknown} {
		s := &Session{encodedVideo: newStream[ReceivedEncodedVideo](1)}
		info := &rtc.EncodedVideoFrameInfo{CodecType: rtc.VideoCodecTypeH264, Width: 2, Height: 2, FrameType: nativeKind}
		s.encodedVideoObserver().OnEncodedVideoFrame("room", "alice", []byte{1}, info)
		frame, err := s.encodedVideo.Next(context.Background())
		s.encodedVideo.close()
		if err != nil {
			t.Fatal(err)
		}
		if frame.Frame.FrameType == VideoFrameAuto {
			t.Fatal("receive returned send-only automatic type")
		}
		if err := frame.Frame.validate(); err != nil {
			t.Fatal(err)
		}
		if got := frame.Frame.nativeInfo().FrameType; got != nativeKind {
			t.Fatalf("native type %d round-tripped as %d", nativeKind, got)
		}
	}
}
