package agorasdk

import (
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"slices"
)

type PixelFormat int

const (
	I420 PixelFormat = PixelFormat(rtc.VideoPixelI420)
	NV12 PixelFormat = PixelFormat(rtc.VideoPixelNV12)
	RGBA PixelFormat = PixelFormat(rtc.VideoPixelRGBA)
	BGRA PixelFormat = PixelFormat(rtc.VideoPixelBGRA)
)

type VideoCodec int

const (
	H264 VideoCodec = VideoCodec(rtc.VideoCodecTypeH264)
	H265 VideoCodec = VideoCodec(rtc.VideoCodecTypeH265)
)

// VideoFrame is tightly packed CPU pixel data. Textures and native handles belong
// to the advanced API. TimestampMillis uses the same timebase as audio.
type VideoFrame struct {
	Data            []byte
	Width, Height   int
	Format          PixelFormat
	Rotation        int
	TimestampMillis int64
	Metadata, Alpha []byte
}
type ReceivedVideoFrame struct {
	Channel, UserID                                    string
	Width, Height, YStride, UStride, VStride, Rotation int
	Y, U, V, Metadata, Alpha                           []byte
	TimestampMillis                                    int64
}
type EncodedVideoFrame struct {
	Data                                                       []byte
	Codec                                                      VideoCodec
	Width, Height, FPS                                         int
	KeyFrame                                                   bool
	TimestampMillis, DecodeTimestampMillis, PresentationMillis int64
	Rotation, StreamID                                         int
	SEI                                                        []byte
}
type ReceivedEncodedVideo struct {
	Channel, UserID string
	Frame           EncodedVideoFrame
}

func (s *Session) videoObserver() *rtc.VideoFrameObserver {
	return &rtc.VideoFrameObserver{OnFrame: func(channel, uid string, f *rtc.VideoFrame) {
		if f == nil {
			return
		}
		s.video.push(ReceivedVideoFrame{Channel: channel, UserID: uid, Width: f.Width, Height: f.Height, YStride: f.YStride, UStride: f.UStride, VStride: f.VStride, Rotation: int(f.Rotation),
			Y: slices.Clone(f.YBuffer), U: slices.Clone(f.UBuffer), V: slices.Clone(f.VBuffer), Metadata: slices.Clone(f.MetadataBuffer), Alpha: slices.Clone(f.AlphaBuffer), TimestampMillis: f.RenderTimeMs})
	}}
}
func (s *Session) encodedVideoObserver() *rtc.VideoEncodedFrameObserver {
	return &rtc.VideoEncodedFrameObserver{OnEncodedVideoFrame: func(channel, uid string, data []byte, i *rtc.EncodedVideoFrameInfo) bool {
		if i == nil {
			return true
		}
		s.encodedVideo.push(ReceivedEncodedVideo{channel, uid, EncodedVideoFrame{Data: slices.Clone(data), Codec: VideoCodec(i.CodecType), Width: i.Width, Height: i.Height, FPS: i.FramesPerSecond, KeyFrame: i.FrameType == rtc.VideoFrameTypeKeyFrame, TimestampMillis: i.CaptureTimeMs, DecodeTimestampMillis: i.DecodeTimeMs, PresentationMillis: i.PresentTimeMs, Rotation: int(i.Rotation), StreamID: i.StreamType, SEI: slices.Clone(i.SeiData)}})
		return true
	}}
}
func validateDimensions(width, height int) error {
	if width < 1 || height < 1 || width > 16384 || height > 16384 {
		return invalid("video dimensions", "must be between 1 and 16384")
	}
	return nil
}
func validateRotation(rotation int) error {
	switch rotation {
	case 0, 90, 180, 270:
		return nil
	default:
		return invalid("rotation", "must be 0, 90, 180, or 270")
	}
}
func (f VideoFrame) validate() error {
	if err := validateDimensions(f.Width, f.Height); err != nil {
		return err
	}
	if err := validateRotation(f.Rotation); err != nil {
		return err
	}
	var size int
	switch f.Format {
	case I420, NV12:
		if f.Width%2 != 0 || f.Height%2 != 0 {
			return invalid("YUV dimensions", "must be even")
		}
		size = f.Width * f.Height * 3 / 2
	case RGBA, BGRA:
		size = f.Width * f.Height * 4
	default:
		return invalid("pixel format", "must be I420, NV12, RGBA, or BGRA")
	}
	if len(f.Data) != size {
		return invalid("video data", "size does not match pixel format and dimensions")
	}
	if len(f.Alpha) > 0 && len(f.Alpha) != f.Width*f.Height {
		return invalid("alpha data", "size must equal width × height")
	}
	return nil
}
func (s *Session) SendVideo(frame VideoFrame) error {
	if err := frame.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.cfg.Video.Mode != RawVideo {
		return ErrNotPublishing
	}
	if err := s.publishVideoLocked(); err != nil {
		return err
	}
	native := rtc.ExternalVideoFrame{Type: rtc.VideoBufferRawData, Format: rtc.VideoPixelFormat(frame.Format), Buffer: frame.Data, Stride: frame.Width, Height: frame.Height, Rotation: rtc.VideoOrientation(frame.Rotation), Timestamp: frame.TimestampMillis, MetadataBuffer: frame.Metadata, AlphaBuffer: frame.Alpha}
	return check("send video", s.native.PushVideoFrame(&native))
}
func (f EncodedVideoFrame) validate() error {
	if len(f.SEI) > 1024 {
		return invalid("SEI", "must not exceed 1024 bytes")
	}
	if len(f.Data) == 0 {
		return invalid("encoded video", "must not be empty")
	}
	if err := validateDimensions(f.Width, f.Height); err != nil {
		return err
	}
	if err := validateRotation(f.Rotation); err != nil {
		return err
	}
	if f.FPS < 0 || f.FPS > 240 {
		return invalid("video FPS", "must be between 0 and 240")
	}
	if f.Codec != H264 && f.Codec != H265 {
		return invalid("video codec", "must be H264 or H265")
	}
	return nil
}
func (f EncodedVideoFrame) nativeInfo() *rtc.EncodedVideoFrameInfo {
	kind := rtc.VideoFrameTypeDeltaFrame
	if f.KeyFrame {
		kind = rtc.VideoFrameTypeKeyFrame
	}
	return &rtc.EncodedVideoFrameInfo{CodecType: rtc.VideoCodecType(f.Codec), Width: f.Width, Height: f.Height, FramesPerSecond: f.FPS, FrameType: kind, CaptureTimeMs: f.TimestampMillis, DecodeTimeMs: f.DecodeTimestampMillis, PresentTimeMs: f.PresentationMillis, Rotation: rtc.VideoOrientation(f.Rotation), StreamType: f.StreamID, SeiData: f.SEI}
}
func (s *Session) SendEncodedVideo(frame EncodedVideoFrame) error {
	if err := frame.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.cfg.Video.Mode != EncodedVideo {
		return ErrNotPublishing
	}
	if err := s.publishVideoLocked(); err != nil {
		return err
	}
	return check("send encoded video", s.native.PushVideoEncodedData(frame.Data, frame.nativeInfo()))
}
func (s *Session) RequestKeyFrame(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("request keyframe", func(c *rtc.RtcConnection) int { return c.SendIntraRequest(userID) })
}
func (s *Session) SetVideoEncoder(cfg rtc.VideoEncoderConfiguration) error {
	if err := validateDimensions(cfg.Width, cfg.Height); err != nil {
		return err
	}
	if cfg.Framerate < 1 || cfg.Framerate > 240 {
		return invalid("video framerate", "must be 1–240")
	}
	return s.withNative("set video encoder", func(c *rtc.RtcConnection) int { return c.SetVideoEncoderConfiguration(&cfg) })
}
func (s *Session) SetSimulcast(enabled bool, cfg rtc.SimulcastStreamConfig) error {
	return s.withNative("set simulcast", func(c *rtc.RtcConnection) int { return c.SetSimulcastStream(enabled, &cfg) })
}
