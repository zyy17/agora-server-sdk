#ifndef H26X_DECODE_FRAME_H
#define H26X_DECODE_FRAME_H
#include <stdint.h>

typedef struct H264Decoder H264Decoder;
typedef struct {
    uint8_t *y_data, *u_data, *v_data;
    int width, height;
    int y_size, uv_size, total_size;
} YUVFrame;

/* Agora codec types: 2 = H.264, 3 = H.265/HEVC. */
H264Decoder *h26x_decoder_init(int codec_type, int *error_code);
H264Decoder *h264_decoder_init(int *error_code);
/* Decode one complete Annex-B access unit (not arbitrary byte chunks). The
 * output is the most recent available 8-bit I420 frame; caller must free it.
 * Zero dimensions preserve the source size. Nonzero dimensions scale using
 * nearest-neighbor sampling and must be positive and even.
 * Returns 0 for a frame, 1 if buffered, or a negative FFmpeg error. */
int h264_decode_frame(H264Decoder *, const uint8_t *, int, int, int, YUVFrame **);
int h264_decode_stream(H264Decoder *, const uint8_t *, int, int, int, YUVFrame **);
int h264_decode_frame_flush(H264Decoder *, int, int, YUVFrame **);
void yuv_frame_free(YUVFrame *);
int h264_decode_frame_get_count(H264Decoder *);
void h264_decoder_free(H264Decoder *);
#endif
