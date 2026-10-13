package agoraservice

import (
	"fmt"
	"math"
)

// Validate checks the complete CPU layout accepted by SendVideoFrame. Stride is
// in pixels; all planes use that allocation width. Chroma planes use half the
// stride for planar formats. Data must include every padded row. Cropping does
// not change allocation sizes. The binding supports raw/array CPU buffers only:
// EGL textures, CVPixelBuffer and platform handles cannot be represented safely
// by its byte-slice send path and are rejected rather than silently discarded.
// Keep all buffers unchanged until SendVideoFrame returns.
func (f *ExternalVideoFrame) Validate() error {
	if f == nil {
		return fmt.Errorf("frame is nil")
	}
	if f.Type != VideoBufferRawData && f.Type != VideoBufferArray {
		return fmt.Errorf("buffer type must be raw or array CPU data")
	}
	if f.EGLContext != nil || f.EGLType != 0 || f.TextureID != 0 || f.Matrix != [16]float32{} {
		return fmt.Errorf("texture handles and transformations are unsupported by the CPU sender")
	}
	if f.Stride < 1 || f.Height < 1 || f.Stride > 16384 || f.Height > 16384 {
		return fmt.Errorf("stride and height must be 1–16384 pixels")
	}
	for _, crop := range []int{f.CropLeft, f.CropTop, f.CropRight, f.CropBottom} {
		if crop < 0 || crop > 16384 {
			return fmt.Errorf("crop must be nonnegative and inside the allocation")
		}
	}
	if f.CropLeft+f.CropRight >= f.Stride || f.CropTop+f.CropBottom >= f.Height {
		return fmt.Errorf("crop must leave a nonempty image")
	}
	var size int
	switch f.Format {
	case VideoPixelI420, VideoPixelNV12, VideoPixelNV21, VideoPixelI010:
		if f.Stride%2 != 0 || f.Height%2 != 0 || f.CropLeft%2 != 0 || f.CropRight%2 != 0 || f.CropTop%2 != 0 || f.CropBottom%2 != 0 {
			return fmt.Errorf("4:2:0 allocation and crop dimensions must be even")
		}
		size = f.Stride * f.Height * 3 / 2
		if f.Format == VideoPixelI010 {
			size *= 2
		}
	case VideoPixelI422:
		if f.Stride%2 != 0 || f.CropLeft%2 != 0 || f.CropRight%2 != 0 {
			return fmt.Errorf("4:2:2 stride and horizontal crop must be even")
		}
		size = f.Stride * f.Height * 2
	case VideoPixelRGBA, VideoPixelBGRA:
		size = f.Stride * f.Height * 4
	default:
		return fmt.Errorf("unsupported CPU pixel format")
	}
	if len(f.Buffer) != size {
		return fmt.Errorf("buffer size must be %d bytes for the allocated pixel layout", size)
	}
	switch f.Rotation {
	case VideoOrientation0, VideoOrientation90, VideoOrientation180, VideoOrientation270:
	default:
		return fmt.Errorf("rotation must be 0, 90, 180, or 270 degrees")
	}
	if len(f.MetadataBuffer) > math.MaxInt32 {
		return fmt.Errorf("metadata exceeds the native signed 32-bit length")
	}
	if len(f.AlphaBuffer) != 0 && len(f.AlphaBuffer) != f.Stride*f.Height {
		return fmt.Errorf("alpha buffer must contain stride × height bytes")
	}
	if f.FillAlphaBuffer && (len(f.AlphaBuffer) != 0 || (f.Format != VideoPixelRGBA && f.Format != VideoPixelBGRA)) {
		return fmt.Errorf("alpha extraction requires RGBA/BGRA and no explicit alpha buffer")
	}
	if f.AlphaMode < 0 || f.AlphaMode > 4 {
		return fmt.Errorf("alpha mode must be 0–4")
	}
	for _, id := range []int{f.ColorSpace.PrimariesId, f.ColorSpace.TransferId, f.ColorSpace.MatrixId, f.ColorSpace.RangeId} {
		if id < 0 || int64(id) > math.MaxInt32 {
			return fmt.Errorf("color-space IDs must fit nonnegative native int32")
		}
	}
	return nil
}
