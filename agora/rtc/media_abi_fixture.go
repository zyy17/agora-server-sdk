//go:build agora_test

package agoraservice

/*
#include "agora_media_node_factory.h"
static int media_frame_probe(external_video_frame *f, int length, int alpha_length) {
  volatile unsigned char value = ((unsigned char *)f->buffer)[length - 1];
  if (f->metadata_size) value ^= f->metadata_buffer[f->metadata_size - 1];
  if (alpha_length) value ^= f->alpha_buffer[alpha_length - 1];
  return value >= 0;
}
*/
import "C"
import "unsafe"

func mediaVideoFrameRoundTrip(f *ExternalVideoFrame) *ExternalVideoFrame {
	var out *ExternalVideoFrame
	withCExternalVideoFrame(f, func(c *C.struct__external_video_frame) int {
		if C.media_frame_probe(c, C.int(len(f.Buffer)), C.int(len(f.AlphaBuffer))) != 1 {
			panic("native media probe failed")
		}
		out = &ExternalVideoFrame{Type: VideoBufferType(c._type), Format: VideoPixelFormat(c.format), Stride: int(c.stride), Height: int(c.height),
			CropLeft: int(c.crop_left), CropTop: int(c.crop_top), CropRight: int(c.crop_right), CropBottom: int(c.crop_bottom), Rotation: VideoOrientation(c.rotation), Timestamp: int64(c.timestamp),
			Buffer: C.GoBytes(c.buffer, C.int(len(f.Buffer))), MetadataBuffer: C.GoBytes(unsafe.Pointer(c.metadata_buffer), c.metadata_size),
			AlphaBuffer: C.GoBytes(unsafe.Pointer(c.alpha_buffer), C.int(len(f.AlphaBuffer))), FillAlphaBuffer: c.fill_alpha_buffer != 0, AlphaMode: int(c.alpha_mode),
			ColorSpace: ColorSpaceType{PrimariesId: int(c.color_space.primaries_id), TransferId: int(c.color_space.transfer_id), MatrixId: int(c.color_space.matrix_id), RangeId: int(c.color_space.range_id)}}
		return 0
	})
	return out
}

func mediaEncodedVideoRoundTrip(f *EncodedVideoFrameInfo) *EncodedVideoFrameInfo {
	c := cEncodedVideoFrameInfo(f)
	return &EncodedVideoFrameInfo{CodecType: VideoCodecType(c.codec_type), Width: int(c.width), Height: int(c.height), FramesPerSecond: int(c.frames_per_second), FrameType: VideoFrameType(c.frame_type), Rotation: VideoOrientation(c.rotation), TrackId: int(c.track_id), CaptureTimeMs: int64(c.capture_time_ms), DecodeTimeMs: int64(c.decode_time_ms), Uid: uint32(c.uid), StreamType: int(c.stream_type), PresentTimeMs: int64(c.presentation_ms)}
}

func mediaEncodedAudioRoundTrip(f *EncodedAudioFrameInfo) *EncodedAudioFrameInfo {
	c := cEncodedAudioFrameInfo(f)
	return &EncodedAudioFrameInfo{Speech: c.speech != 0, Codec: AudioCodecType(c.codec), SampleRateHz: int(c.sample_rate_hz), SamplesPerChannel: int(c.samples_per_channel), SendEvenIfEmpty: c.send_even_if_empty != 0, NumberOfChannels: int(c.number_of_channels), CaptureTimeMs: int64(c.capture_time_ms)}
}

func mediaReceivedAlphaMode() int {
	f := C.struct__video_frame{alpha_mode: 4}
	return GoVideoFrame(&f).AlphaMode
}
