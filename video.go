package agorasdk

import (
	"slices"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// PixelFormat identifies a tightly packed CPU pixel layout accepted by SendVideo.
type PixelFormat int

const (
	// I420 is planar 8-bit YUV 4:2:0 with Y, U, and V planes.
	I420 PixelFormat = PixelFormat(rtc.VideoPixelI420)
	// NV12 is 8-bit YUV 4:2:0 with a Y plane followed by interleaved UV.
	NV12 PixelFormat = PixelFormat(rtc.VideoPixelNV12)
	// RGBA is packed 8-bit red, green, blue, and alpha in that byte order.
	RGBA PixelFormat = PixelFormat(rtc.VideoPixelRGBA)
	// BGRA is packed 8-bit blue, green, red, and alpha in that byte order.
	BGRA PixelFormat = PixelFormat(rtc.VideoPixelBGRA)
)

// VideoCodec identifies a native encoded-video codec accepted by this wrapper.
type VideoCodec int

const (
	// H264 identifies H.264/AVC video.
	H264 VideoCodec = VideoCodec(rtc.VideoCodecTypeH264)
	// H265 identifies H.265/HEVC video; native and peer support must be verified.
	H265 VideoCodec = VideoCodec(rtc.VideoCodecTypeH265)
)

// VideoFrame contains tightly packed CPU pixels for SendVideo.
// Textures, hardware buffers, cropping, and native handles require the raw API.
// Keep all supplied slices unchanged until the send call returns.
type VideoFrame struct {
	// Data holds exactly Width*Height*3/2 bytes for I420/NV12 or
	// Width*Height*4 bytes for RGBA/BGRA. Row padding is not accepted.
	Data []byte
	// Width and Height are pixel dimensions in 1–16384. I420 and NV12 require even dimensions.
	Width, Height int
	// Format selects I420, NV12, RGBA, or BGRA.
	Format PixelFormat
	// Rotation is clockwise orientation in degrees: 0, 90, 180, or 270.
	Rotation int
	// TimestampMillis is the native capture timestamp in milliseconds.
	// Use a timebase consistent with related audio; it need not be Unix time.
	TimestampMillis int64
	// Metadata carries optional native frame metadata. Alpha is an optional
	// Width*Height-byte alpha plane. Availability depends on the native media path.
	Metadata, Alpha []byte
}

// ReceivedVideoFrame owns copied decoded planes and metadata.
// Respect the reported strides; receive planes are not guaranteed to be tightly
// packed like VideoFrame.Data. The value remains valid after the callback or Close.
type ReceivedVideoFrame struct {
	// Channel and UserID identify the remote source.
	Channel, UserID string
	// Width and Height are pixel dimensions. YStride, UStride, and VStride are
	// native plane row strides in bytes. Rotation is the native orientation in degrees.
	Width, Height, YStride, UStride, VStride, Rotation int
	// Y, U, and V own the copied native image planes. Metadata and Alpha own
	// optional native auxiliary buffers; empty slices mean no such data was supplied.
	Y, U, V, Metadata, Alpha []byte
	// TimestampMillis is the native render timestamp in milliseconds; its origin
	// is not guaranteed to be the local wall clock.
	TimestampMillis int64
}

// EncodedVideoFrame describes an H.264 or H.265 encoded frame.
// Validation checks metadata bounds but does not parse the bitstream or prove
// codec compatibility. Keep Data and SEI unchanged during a synchronous send.
type EncodedVideoFrame struct {
	// Data contains nonempty encoded video in the framing expected by the native
	// sender and peers. The Go wrapper does not convert container formats.
	Data []byte
	// Codec must be H264 or H265.
	Codec VideoCodec
	// Width and Height are in pixels, each 1–16384. FPS is the declared frame
	// rate in 0–240; zero is forwarded to the native SDK without a wrapper default.
	Width, Height, FPS int
	// KeyFrame declares an independently decodable key frame; false declares a delta frame.
	KeyFrame bool
	// TimestampMillis, DecodeTimestampMillis, and PresentationMillis are native
	// capture, decode, and presentation timestamps in milliseconds, respectively.
	// They are forwarded unchanged and are not necessarily Unix timestamps.
	TimestampMillis, DecodeTimestampMillis, PresentationMillis int64
	// Rotation is 0, 90, 180, or 270 degrees. StreamID is the native stream type,
	// passed through unchanged; it is not an RTC data-stream message identifier.
	Rotation, StreamID int
	// SEI contains optional supplemental enhancement information, at most 1024 bytes.
	// Codec and native-version restrictions still apply.
	SEI []byte
}

// ReceivedEncodedVideo owns a copied encoded frame and its source identity.
type ReceivedEncodedVideo struct {
	// Channel and UserID identify the remote sender.
	Channel, UserID string
	// Frame owns the received encoded payload and SEI; metadata reflects native values.
	Frame EncodedVideoFrame
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

// SendVideo validates and submits tightly packed raw CPU pixels.
// It requires Broadcaster and RawVideo publication and publishes on first use.
// The call does not pace frames; keep all input buffers unchanged until it returns.
// A nil result means local native acceptance, not delivery or rendering by peers.
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

// SendEncodedVideo validates metadata and submits an encoded frame without
// decoding or pacing it. It requires Broadcaster and EncodedVideo publication
// and publishes on first use. Keep Data and SEI unchanged until it returns.
// A nil result does not acknowledge remote decoding or playback.
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

// RequestKeyFrame requests an intra frame from a remote user in the Service's
// identity namespace. Numeric identities must be nonzero canonical uint32 text.
// The native result reports request submission, not arrival of a key frame.
func (s *Session) RequestKeyFrame(userID string) error {
	if err := s.validateRemoteUserID(userID); err != nil {
		return err
	}
	return s.withNative("request keyframe", func(c *rtc.RtcConnection) int { return c.SendIntraRequest(userID) })
}

// SetVideoEncoder updates the native video encoder configuration.
// Width and Height must be 1–16384 pixels and Framerate must be 1–240 frames
// per second. Remaining fields and applicability are validated by the native SDK.
func (s *Session) SetVideoEncoder(cfg rtc.VideoEncoderConfiguration) error {
	if err := validateDimensions(cfg.Width, cfg.Height); err != nil {
		return err
	}
	if cfg.Framerate < 1 || cfg.Framerate > 240 {
		return invalid("video framerate", "must be 1–240")
	}
	return s.withNative("set video encoder", func(c *rtc.RtcConnection) int { return c.SetVideoEncoderConfiguration(&cfg) })
}

// SetSimulcast forwards native simulcast settings for the session's video track.
// The native SDK validates cfg and determines feature/peer support; this call
// does not change ReceiveConfig or subscriptions.
func (s *Session) SetSimulcast(enabled bool, cfg rtc.SimulcastStreamConfig) error {
	return s.withNative("set simulcast", func(c *rtc.RtcConnection) int { return c.SetSimulcastStream(enabled, &cfg) })
}
