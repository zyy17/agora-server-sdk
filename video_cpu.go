package agorasdk

import rtc "github.com/zyy17/agora-server-sdk/agora/rtc"

// CPUBufferType selects equivalent native CPU buffer representations.
type CPUBufferType int

const (
	// CPURawData is the default raw CPU buffer representation.
	CPURawData CPUBufferType = 0
	// CPUArray selects the native array representation, equivalent to raw data.
	CPUArray CPUBufferType = 1
)

// AlphaMode describes the native placement of an accompanying alpha image.
type AlphaMode int

const (
	// AlphaNormal uses a normal frame.
	AlphaNormal AlphaMode = iota
	// AlphaAbove places alpha above the color image.
	AlphaAbove
	// AlphaBelow places alpha below the color image.
	AlphaBelow
	// AlphaLeft places alpha left of the color image.
	AlphaLeft
	// AlphaRight places alpha right of the color image.
	AlphaRight
)

// ColorSpace carries native H.273 primaries/transfer/matrix IDs and the WebM
// colour range ID. Zero leaves the native default; values are forwarded rather
// than color-converted. Use values supported by the deployed encoder/decoder.
type ColorSpace struct {
	PrimariesID, TransferID, MatrixID, RangeID int
}

// CPUVideoFrame describes a complete contiguous CPU allocation, including padded
// rows. Stride is the allocation width in pixels (not bytes); the visible image
// is Stride-CropLeft-CropRight by Height-CropTop-CropBottom. There is no separate
// byte-stride or per-plane offset: I420/I010/I422 use half-stride chroma planes,
// NV12/NV21 use full-stride interleaved chroma. Cropping never reduces Data size.
// Texture/native hardware handles are deliberately excluded because the pinned
// Go sender does not marshal those paths. Keep slices unchanged during sends.
type CPUVideoFrame struct {
	// Data contains the entire allocation. I420/NV12/NV21 use 1.5 bytes per
	// pixel, I422 uses 2, I010 uses 3, RGBA/BGRA use 4.
	Data []byte
	// Stride and Height are 1–16384 pixels. 4:2:0 dimensions and crop must be
	// even; I422 requires even stride and horizontal crop.
	Stride, Height                           int
	Format                                   PixelFormat
	BufferType                               CPUBufferType
	CropLeft, CropTop, CropRight, CropBottom int
	// Rotation is clockwise 0, 90, 180, or 270; TimestampMillis is native time.
	Rotation        int
	TimestampMillis int64
	// Metadata is forwarded to the native C metadata field. The SDK documents
	// this field for texture paths, so CPU transport/delivery is not guaranteed.
	Metadata []byte
	// Alpha contains Stride*Height bytes including padding, or is empty.
	// FillAlphaBuffer extracts RGBA/BGRA alpha and cannot accompany explicit Alpha.
	Alpha           []byte
	FillAlphaBuffer bool
	AlphaMode       AlphaMode
	ColorSpace      ColorSpace
}

// VideoFrameType selects an encoded-frame classification. These high-level
// values are independent of rtc.VideoFrameType; do not cast between the enums.
// VideoFrameAuto is send-only convenience. Received frames use an explicit
// classification, preserving all types in the pinned native SDK on retransmit.
type VideoFrameType int

const (
	// VideoFrameAuto selects key when EncodedVideoFrame.KeyFrame is true, and
	// delta otherwise. It is the zero value and is never returned by reception.
	VideoFrameAuto VideoFrameType = iota
	// VideoFrameBlank explicitly selects the native blank classification.
	VideoFrameBlank
	// VideoFrameKey is independently decodable.
	VideoFrameKey
	// VideoFrameDelta depends on prior decoded frames.
	VideoFrameDelta
	// VideoFrameB is a bidirectionally predicted frame.
	VideoFrameB
	// VideoFrameDroppable is eligible for dropping by the native pipeline.
	VideoFrameDroppable
	// VideoFrameUnknown explicitly selects the native unknown classification.
	// Unrecognized classifications from a newer native SDK are also reported here.
	VideoFrameUnknown
)

func receivedVideoFrameType(kind rtc.VideoFrameType) VideoFrameType {
	switch kind {
	case rtc.VideoFrameTypeBlankFrame:
		return VideoFrameBlank
	case rtc.VideoFrameTypeKeyFrame:
		return VideoFrameKey
	case rtc.VideoFrameTypeDeltaFrame:
		return VideoFrameDelta
	case rtc.VideoFrameTypeBFrame:
		return VideoFrameB
	case rtc.VideoFrameTypeDroppableFrame:
		return VideoFrameDroppable
	default:
		return VideoFrameUnknown
	}
}

func (f EncodedVideoFrame) nativeFrameType() (rtc.VideoFrameType, error) {
	if f.KeyFrame && f.FrameType != VideoFrameAuto && f.FrameType != VideoFrameKey {
		return 0, invalid("video frame type", "conflicts with KeyFrame")
	}
	switch f.FrameType {
	case VideoFrameAuto:
		if f.KeyFrame {
			return rtc.VideoFrameTypeKeyFrame, nil
		}
		return rtc.VideoFrameTypeDeltaFrame, nil
	case VideoFrameBlank:
		return rtc.VideoFrameTypeBlankFrame, nil
	case VideoFrameKey:
		return rtc.VideoFrameTypeKeyFrame, nil
	case VideoFrameDelta:
		return rtc.VideoFrameTypeDeltaFrame, nil
	case VideoFrameB:
		return rtc.VideoFrameTypeBFrame, nil
	case VideoFrameDroppable:
		return rtc.VideoFrameTypeDroppableFrame, nil
	case VideoFrameUnknown:
		return rtc.VideoFrameTypeUnknown, nil
	default:
		return 0, invalid("video frame type", "is unknown")
	}
}

func (f CPUVideoFrame) nativeFrame() rtc.ExternalVideoFrame {
	kind := rtc.VideoBufferRawData
	if f.BufferType == CPUArray {
		kind = rtc.VideoBufferArray
	}
	return rtc.ExternalVideoFrame{Type: kind, Format: rtc.VideoPixelFormat(f.Format), Buffer: f.Data,
		Stride: f.Stride, Height: f.Height, CropLeft: f.CropLeft, CropTop: f.CropTop, CropRight: f.CropRight, CropBottom: f.CropBottom,
		Rotation: rtc.VideoOrientation(f.Rotation), Timestamp: f.TimestampMillis, MetadataBuffer: f.Metadata,
		AlphaBuffer: f.Alpha, FillAlphaBuffer: f.FillAlphaBuffer, AlphaMode: int(f.AlphaMode),
		ColorSpace: rtc.ColorSpaceType{PrimariesId: f.ColorSpace.PrimariesID, TransferId: f.ColorSpace.TransferID, MatrixId: f.ColorSpace.MatrixID, RangeId: f.ColorSpace.RangeID}}
}

func (f CPUVideoFrame) validate() error {
	if f.BufferType != CPURawData && f.BufferType != CPUArray {
		return invalid("CPU buffer type", "must be CPURawData or CPUArray")
	}
	native := f.nativeFrame()
	if err := native.Validate(); err != nil {
		return invalid("video frame", err.Error())
	}
	return nil
}

// SendCPUVideo validates the allocation and submits a raw CPU frame, publishing
// on first use. It requires Broadcaster and RawVideo. The synchronous call does
// not pace frames or acknowledge peer receipt. Buffer sizes, crops and C integer
// bounds are checked in the same authoritative validator used by the raw sender.
func (s *Session) SendCPUVideo(frame CPUVideoFrame) error {
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
	native := frame.nativeFrame()
	return check("send video", s.native.PushVideoFrame(&native))
}
