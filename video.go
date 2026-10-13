package agorasdk

import (
	"math"
	"slices"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// PixelFormat identifies a CPU pixel layout accepted by the native sender.
// A named format is not a guarantee of encoder support in every native build.
type PixelFormat int

const (
	// I420 is planar 8-bit YUV 4:2:0 with Y, U, and V planes.
	I420 PixelFormat = PixelFormat(rtc.VideoPixelI420)
	// NV12 is 8-bit YUV 4:2:0 with a Y plane followed by interleaved UV.
	NV12 PixelFormat = PixelFormat(rtc.VideoPixelNV12)
	// NV21 is 8-bit YUV 4:2:0 with a Y plane followed by interleaved VU.
	NV21 PixelFormat = PixelFormat(rtc.VideoPixelNV21)
	// I422 is planar 8-bit YUV 4:2:2 with Y, U, and V planes.
	I422 PixelFormat = PixelFormat(rtc.VideoPixelI422)
	// I010 is planar 10-bit YUV 4:2:0 in little-endian 16-bit samples (native preview).
	I010 PixelFormat = PixelFormat(rtc.VideoPixelI010)
	// RGBA is packed 8-bit red, green, blue, and alpha in that byte order.
	RGBA PixelFormat = PixelFormat(rtc.VideoPixelRGBA)
	// BGRA is packed 8-bit blue, green, red, and alpha in that byte order.
	BGRA PixelFormat = PixelFormat(rtc.VideoPixelBGRA)
)

// VideoCodec identifies a native encoded-video payload. Support for transport,
// decoding, and encoding varies with the installed native SDK and peer.
type VideoCodec int

const (
	// VP8 identifies VP8 video.
	VP8 VideoCodec = VideoCodec(rtc.VideoCodecTypeVp8)
	// VP9 identifies VP9 video.
	VP9 VideoCodec = VideoCodec(rtc.VideoCodecTypeVp9)
	// AV1 identifies AV1 video.
	AV1 VideoCodec = VideoCodec(rtc.VideoCodecTypeAv1)
	// GenericVideo identifies an application-defined native payload.
	GenericVideo VideoCodec = VideoCodec(rtc.VideoCodecTypeGeneric)
	// GenericH264 identifies the native generic-H264 payload.
	GenericH264 VideoCodec = VideoCodec(rtc.VideoCodecTypeGenericH264)
	// JPEG identifies the native generic-JPEG payload.
	JPEG VideoCodec = VideoCodec(rtc.VideoCodecTypeGenericJpeg)
	// H264 identifies H.264/AVC video.
	H264 VideoCodec = VideoCodec(rtc.VideoCodecTypeH264)
	// H265 identifies H.265/HEVC video; native and peer support must be verified.
	H265 VideoCodec = VideoCodec(rtc.VideoCodecTypeH265)
)

// VideoFrame contains tightly packed CPU pixels for SendVideo.
// For padded rows, cropping, color space, and alpha extraction use CPUVideoFrame.
// Texture and hardware handles are not supported by the Go CPU sender.
// Keep all supplied slices unchanged until the send call returns.
type VideoFrame struct {
	// Data holds exactly Width*Height*3/2 bytes for I420/NV12/NV21 or
	// Width*Height*4 bytes for RGBA/BGRA; I422 uses 2 and I010 uses 3 bytes
	// per pixel. Row padding is not accepted.
	Data []byte
	// Width and Height are pixel dimensions in 1–16384. 4:2:0 requires even dimensions; I422 requires even width.
	Width, Height int
	// Format selects a supported CPU pixel layout.
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
	// Type, AVSyncType, and AlphaMode preserve native frame descriptors.
	Type       rtc.VideoBufferType
	AVSyncType int
	AlphaMode  AlphaMode
	// TextureID and Matrix are copied descriptors only, not owned or usable GPU
	// resources. The borrowed native shared context is intentionally not exposed.
	TextureID int
	Matrix    [16]float32
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

// EncodedVideoFrame describes one native encoded-video frame.
// Validation checks metadata bounds but does not parse the bitstream or prove
// codec compatibility. Keep Data and SEI unchanged during a synchronous send.
type EncodedVideoFrame struct {
	// Data contains nonempty encoded video in the framing expected by the native
	// sender and peers. The Go wrapper does not convert container formats.
	Data []byte
	// Codec selects a known native payload type; native/peer support varies.
	Codec VideoCodec
	// Width and Height are in pixels, each 1–16384. FPS is the declared frame
	// rate in 0–240; zero is forwarded to the native SDK without a wrapper default.
	Width, Height, FPS int
	// FrameType defaults to VideoFrameAuto: KeyFrame selects key when true,
	// delta otherwise. Explicit classifications preserve native frame types,
	// including VideoFrameBlank. KeyFrame=true conflicts with an explicit
	// non-key classification, including Blank; false does not override FrameType.
	FrameType VideoFrameType
	KeyFrame  bool
	// TrackID is the signed native track identifier. UID is the native numeric
	// media UID, even when the channel uses string identities.
	TrackID int
	UID     uint32
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
		s.video.push(ReceivedVideoFrame{Type: f.Type, AVSyncType: f.AVSyncType, AlphaMode: AlphaMode(f.AlphaMode), TextureID: f.TextureID, Matrix: f.Matrix, Channel: channel, UserID: uid, Width: f.Width, Height: f.Height, YStride: f.YStride, UStride: f.UStride, VStride: f.VStride, Rotation: int(f.Rotation),
			Y: slices.Clone(f.YBuffer), U: slices.Clone(f.UBuffer), V: slices.Clone(f.VBuffer), Metadata: slices.Clone(f.MetadataBuffer), Alpha: slices.Clone(f.AlphaBuffer), TimestampMillis: f.RenderTimeMs})
	}}
}

func (s *Session) encodedVideoObserver() *rtc.VideoEncodedFrameObserver {
	return &rtc.VideoEncodedFrameObserver{OnEncodedVideoFrame: func(channel, uid string, data []byte, i *rtc.EncodedVideoFrameInfo) bool {
		if i == nil {
			return true
		}
		frame := ReceivedEncodedVideo{channel, uid, EncodedVideoFrame{Data: slices.Clone(data), Codec: VideoCodec(i.CodecType), Width: i.Width, Height: i.Height, FPS: i.FramesPerSecond, FrameType: receivedVideoFrameType(i.FrameType), KeyFrame: i.FrameType == rtc.VideoFrameTypeKeyFrame, TrackID: i.TrackId, UID: i.Uid, TimestampMillis: i.CaptureTimeMs, DecodeTimestampMillis: i.DecodeTimeMs, PresentationMillis: i.PresentTimeMs, Rotation: int(i.Rotation), StreamID: i.StreamType, SEI: slices.Clone(i.SeiData)}}
		accepted := s.processEncodedVideoCallback(frame)
		s.encodedVideo.push(frame)
		return accepted
	}}
}

// processEncodedVideoCallback isolates application hooks from the queued packet.
// Encoded observer decisions never write modified packet data back to native.
func (s *Session) processEncodedVideoCallback(frame ReceivedEncodedVideo) (accepted bool) {
	accepted = true
	if callback := s.cfg.Video.OnEncodedFrame; callback != nil {
		frame.Frame.Data, frame.Frame.SEI = slices.Clone(frame.Frame.Data), slices.Clone(frame.Frame.SEI)
		defer func() {
			if recover() != nil {
				accepted = false
				s.event(Event{Kind: CallbackPanic, Message: "encoded video callback panicked"})
			}
		}()
		accepted = callback(frame)
	}
	return accepted
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

func (f VideoFrame) cpuFrame() CPUVideoFrame {
	return CPUVideoFrame{Data: f.Data, Stride: f.Width, Height: f.Height, Format: f.Format,
		Rotation: f.Rotation, TimestampMillis: f.TimestampMillis, Metadata: f.Metadata, Alpha: f.Alpha}
}

func (f VideoFrame) validate() error { return f.cpuFrame().validate() }

// SendVideo validates and submits compact CPU pixels through SendCPUVideo.
// It requires Broadcaster and RawVideo publication and publishes on first use.
// The call does not pace frames; keep all input buffers unchanged until it returns.
// A nil result means local native acceptance, not delivery or rendering by peers.
func (s *Session) SendVideo(frame VideoFrame) error { return s.SendCPUVideo(frame.cpuFrame()) }

func (f EncodedVideoFrame) validate() error {
	if len(f.SEI) > 1024 {
		return invalid("SEI", "must not exceed 1024 bytes")
	}
	if len(f.Data) == 0 || uint64(len(f.Data)) > math.MaxUint32 {
		return invalid("encoded video", "must contain 1–uint32 bytes")
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
	if !validVideoCodec(f.Codec) {
		return invalid("video codec", "is unknown")
	}
	if len(f.SEI) != 0 && f.Codec != H264 && f.Codec != H265 {
		return invalid("SEI", "insertion supports only H264 and H265")
	}
	if _, err := f.nativeFrameType(); err != nil {
		return err
	}
	if int64(f.TrackID) < math.MinInt32 || int64(f.TrackID) > math.MaxInt32 {
		return invalid("video track ID", "must fit native int32")
	}
	if int64(f.StreamID) < math.MinInt32 || int64(f.StreamID) > math.MaxInt32 {
		return invalid("video stream ID", "must fit native int32")
	}
	return nil
}

func (f EncodedVideoFrame) nativeInfo() *rtc.EncodedVideoFrameInfo {
	kind, _ := f.nativeFrameType() // Callers validate before conversion.
	return &rtc.EncodedVideoFrameInfo{CodecType: rtc.VideoCodecType(f.Codec), Width: f.Width, Height: f.Height, FramesPerSecond: f.FPS, FrameType: kind, TrackId: f.TrackID, Uid: f.UID, CaptureTimeMs: f.TimestampMillis, DecodeTimeMs: f.DecodeTimestampMillis, PresentTimeMs: f.PresentationMillis, Rotation: rtc.VideoOrientation(f.Rotation), StreamType: f.StreamID, SeiData: f.SEI}
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
// Width/Height may both be zero for native sizing or 1–16384 pixels.
// Framerate is 0–240, with zero forwarded to the SDK. Bitrates are Kbps;
// negative native bitrate selectors are preserved when conversion fits int32.
// A valid enum/configuration is not proof of encoder availability.
func (s *Session) SetVideoEncoder(cfg rtc.VideoEncoderConfiguration) error {
	if err := validateVideoEncoder(&cfg); err != nil {
		return err
	}
	return s.withNative("set video encoder", func(c *rtc.RtcConnection) int { return c.SetVideoEncoderConfiguration(&cfg) })
}

// SetSimulcast forwards native simulcast settings for the session's video track.
// When enabling, dimensions must be 1–16384 and framerate 1–240. Bitrate
// must fit nonnegative int32 in the native simulcast configuration units.
// The native SDK determines feature/peer support; this call does not change
// ReceiveConfig or subscriptions.
func (s *Session) SetSimulcast(enabled bool, cfg rtc.SimulcastStreamConfig) error {
	if err := validateSimulcast(enabled, cfg); err != nil {
		return err
	}
	return s.withNative("set simulcast", func(c *rtc.RtcConnection) int { return c.SetSimulcastStream(enabled, &cfg) })
}

func validVideoCodec(codec VideoCodec) bool {
	switch codec {
	case VP8, H264, H265, GenericVideo, GenericH264, AV1, VP9, JPEG:
		return true
	default:
		return false
	}
}

func validateVideoEncoder(cfg *rtc.VideoEncoderConfiguration) error {
	if cfg == nil {
		return nil
	}
	if cfg.Width != 0 || cfg.Height != 0 {
		if err := validateDimensions(cfg.Width, cfg.Height); err != nil {
			return err
		}
	}
	if cfg.Framerate < 0 || cfg.Framerate > 240 {
		return invalid("video framerate", "must be 0–240")
	}
	if cfg.CodecType != rtc.VideoCodecTypeNone && !validVideoCodec(VideoCodec(cfg.CodecType)) {
		return invalid("video encoder codec", "is unknown")
	}
	for _, rate := range []int{cfg.Bitrate, cfg.MinBitrate} {
		if int64(rate) < math.MinInt32/1000 || int64(rate) > math.MaxInt32/1000 {
			return invalid("video bitrate", "Kbps conversion must fit native int32 bits/sec")
		}
	}
	if cfg.OrientationMode < rtc.OrientationModeAdaptive || cfg.OrientationMode > rtc.OrientationModeFixedPortrait {
		return invalid("video orientation", "is unknown")
	}
	if cfg.DegradePreference < rtc.DegradeMaintainQuality || cfg.DegradePreference > rtc.DegradeMaintainBalanced {
		return invalid("video degradation preference", "is unknown")
	}
	if cfg.MirrorMode < rtc.VideoMirrorModeAuto || cfg.MirrorMode > rtc.VideoMirrorModeDisabled {
		return invalid("video mirror mode", "is unknown")
	}
	return nil
}

func validateEncodedVideoSender(cfg *rtc.VideoEncodedImageSenderOptions) error {
	if cfg == nil {
		return nil
	}
	if cfg.CodecType != rtc.VideoCodecTypeNone && !validVideoCodec(VideoCodec(cfg.CodecType)) {
		return invalid("encoded video sender codec", "is unknown")
	}
	if cfg.CcMode != rtc.VideoSendCcEnabled && cfg.CcMode != rtc.VideoSendCcDisabled {
		return invalid("video congestion control", "is unknown")
	}
	if cfg.TargetBitrate < 0 || int64(cfg.TargetBitrate) > math.MaxInt32 {
		return invalid("video target bitrate", "must fit nonnegative int32 Kbps")
	}
	return nil
}

func validateSimulcast(enabled bool, cfg rtc.SimulcastStreamConfig) error {
	if enabled || cfg.Width != 0 || cfg.Height != 0 {
		if err := validateDimensions(cfg.Width, cfg.Height); err != nil {
			return err
		}
	}
	if cfg.Framerate < 0 || cfg.Framerate > 240 || (enabled && cfg.Framerate == 0) {
		return invalid("simulcast framerate", "must be 1–240 when enabled, or 0–240 when disabled")
	}
	if cfg.Bitrate < 0 || int64(cfg.Bitrate) > math.MaxInt32 {
		return invalid("simulcast bitrate", "must fit nonnegative int32")
	}
	return nil
}
