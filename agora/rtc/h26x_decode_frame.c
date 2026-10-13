/* Optional FFmpeg decoder. Each input is a complete Annex-B access unit. */
#include <limits.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include "h26x_decode_frame.h"

#ifdef USE_AVCODEC
#include <libavcodec/avcodec.h>
#include <libavutil/error.h>

struct H264Decoder {
    AVCodecContext *codec_ctx;
    AVFrame *frame;
    AVPacket *pkt;
    int frame_count;
    int flush_sent;
};

H264Decoder *h26x_decoder_init(int codec_type, int *error_code) {
    int ret = AVERROR(EINVAL);
    if (error_code) *error_code = ret;
    enum AVCodecID id;
    if (codec_type == 2) id = AV_CODEC_ID_H264;
    else if (codec_type == 3) id = AV_CODEC_ID_HEVC;
    else return NULL;
    const AVCodec *codec = avcodec_find_decoder(id);
    if (!codec) {
        if (error_code) *error_code = AVERROR_DECODER_NOT_FOUND;
        return NULL;
    }
    H264Decoder *decoder = calloc(1, sizeof(*decoder));
    if (!decoder) {
        if (error_code) *error_code = AVERROR(ENOMEM);
        return NULL;
    }
    decoder->codec_ctx = avcodec_alloc_context3(codec);
    decoder->frame = av_frame_alloc();
    decoder->pkt = av_packet_alloc();
    if (!decoder->codec_ctx || !decoder->frame || !decoder->pkt) {
        ret = AVERROR(ENOMEM);
        goto fail;
    }
    decoder->codec_ctx->flags |= AV_CODEC_FLAG_LOW_DELAY;
    decoder->codec_ctx->thread_count = 1;
    ret = avcodec_open2(decoder->codec_ctx, codec, NULL);
    if (ret < 0) goto fail;
    if (error_code) *error_code = 0;
    return decoder;
fail:
    if (error_code) *error_code = ret;
    h264_decoder_free(decoder);
    return NULL;
}

H264Decoder *h264_decoder_init(int *error_code) {
    return h26x_decoder_init(2, error_code);
}

/* I420 is explicitly 8-bit, planar 4:2:0. Reject other formats rather than
 * reading 10-bit or 4:4:4 samples with incorrect byte/plane sizes. */
static int convert_frame_to_yuv(const AVFrame *frame, int width, int height,
                                YUVFrame **out) {
    if (frame->format != AV_PIX_FMT_YUV420P && frame->format != AV_PIX_FMT_YUVJ420P)
        return AVERROR(ENOSYS);
    if (width == 0) width = frame->width;
    if (height == 0) height = frame->height;
    if (width <= 0 || height <= 0 || (width & 1) || (height & 1) ||
        frame->width <= 0 || frame->height <= 0 ||
        (int64_t)width * height > INT_MAX / 3 * 2)
        return AVERROR(EINVAL);
    YUVFrame *yuv = calloc(1, sizeof(*yuv));
    if (!yuv) return AVERROR(ENOMEM);
    yuv->width = width;
    yuv->height = height;
    yuv->y_size = width * height;
    yuv->uv_size = (width / 2) * (height / 2);
    yuv->total_size = yuv->y_size + yuv->uv_size * 2;
    yuv->y_data = malloc(yuv->y_size);
    yuv->u_data = malloc(yuv->uv_size);
    yuv->v_data = malloc(yuv->uv_size);
    if (!yuv->y_data || !yuv->u_data || !yuv->v_data) {
        yuv_frame_free(yuv);
        return AVERROR(ENOMEM);
    }
    uint8_t *planes[] = {yuv->y_data, yuv->u_data, yuv->v_data};
    for (int plane = 0; plane < 3; ++plane) {
        const int w = plane ? width / 2 : width;
        const int h = plane ? height / 2 : height;
        const int src_w = plane ? (frame->width + 1) / 2 : frame->width;
        const int src_h = plane ? (frame->height + 1) / 2 : frame->height;
        if (!frame->data[plane]) {
            yuv_frame_free(yuv);
            return AVERROR_INVALIDDATA;
        }
        for (int y = 0; y < h; ++y) {
            const int src_y = (int64_t)y * src_h / h;
            const uint8_t *src = frame->data[plane] + (ptrdiff_t)src_y * frame->linesize[plane];
            uint8_t *dst = planes[plane] + (ptrdiff_t)y * w;
            if (w == src_w) memcpy(dst, src, w);
            else for (int x = 0; x < w; ++x) dst[x] = src[(int64_t)x * src_w / w];
        }
    }
    *out = yuv;
    return 0;
}

/* Drain every available frame, retaining the newest for realtime rendering. */
static int receive_frames(H264Decoder *decoder, int width, int height, YUVFrame **out) {
    int ret;
    while ((ret = avcodec_receive_frame(decoder->codec_ctx, decoder->frame)) >= 0) {
        YUVFrame *next = NULL;
        ret = convert_frame_to_yuv(decoder->frame, width, height, &next);
        av_frame_unref(decoder->frame);
        if (ret < 0) return ret;
        yuv_frame_free(*out);
        *out = next;
        ++decoder->frame_count;
    }
    if (ret != AVERROR(EAGAIN) && ret != AVERROR_EOF) return ret;
    return *out ? 0 : (ret == AVERROR_EOF ? ret : 1);
}

int h264_decode_frame(H264Decoder *decoder, const uint8_t *data, int size,
                      int width, int height, YUVFrame **out) {
    if (!decoder || !out || !data || size <= 0 || width < 0 || height < 0)
        return AVERROR(EINVAL);
    *out = NULL;
    if (decoder->flush_sent) return AVERROR_EOF;
    av_packet_unref(decoder->pkt);
    int ret = av_new_packet(decoder->pkt, size);
    if (ret < 0) return ret;
    /* av_new_packet provides AV_INPUT_BUFFER_PADDING_SIZE zero bytes. Raw
     * Go slices do not provide that padding and must never be wrapped. */
    memcpy(decoder->pkt->data, data, size);
    ret = avcodec_send_packet(decoder->codec_ctx, decoder->pkt);
    if (ret == AVERROR(EAGAIN)) {
        ret = receive_frames(decoder, width, height, out);
        if (ret >= 0) ret = avcodec_send_packet(decoder->codec_ctx, decoder->pkt);
    }
    av_packet_unref(decoder->pkt);
    if (ret >= 0) ret = receive_frames(decoder, width, height, out);
    if (ret < 0) {
        yuv_frame_free(*out);
        *out = NULL;
    }
    return ret;
}

int h264_decode_stream(H264Decoder *decoder, const uint8_t *data, int size,
                       int width, int height, YUVFrame **out) {
    return h264_decode_frame(decoder, data, size, width, height, out);
}

int h264_decode_frame_flush(H264Decoder *decoder, int width, int height, YUVFrame **out) {
    if (!decoder || !out) return AVERROR(EINVAL);
    *out = NULL;
    if (!decoder->flush_sent) {
        int ret = avcodec_send_packet(decoder->codec_ctx, NULL);
        if (ret < 0 && ret != AVERROR_EOF) return ret;
        decoder->flush_sent = 1;
    }
    int ret = receive_frames(decoder, width, height, out);
    if (ret < 0) { yuv_frame_free(*out); *out = NULL; }
    return ret;
}

void yuv_frame_free(YUVFrame *frame) {
    if (!frame) return;
    free(frame->y_data);
    free(frame->u_data);
    free(frame->v_data);
    free(frame);
}
int h264_decode_frame_get_count(H264Decoder *decoder) {
    return decoder ? decoder->frame_count : 0;
}
void h264_decoder_free(H264Decoder *decoder) {
    if (!decoder) return;
    avcodec_free_context(&decoder->codec_ctx);
    av_frame_free(&decoder->frame);
    av_packet_free(&decoder->pkt);
    free(decoder);
}
#else
struct H264Decoder { int unused; };
H264Decoder *h26x_decoder_init(int codec_type, int *error_code) {
    if (error_code) *error_code = -1;
    return NULL;
}
H264Decoder *h264_decoder_init(int *error_code) { return h26x_decoder_init(2, error_code); }
int h264_decode_frame(H264Decoder *d, const uint8_t *p, int n, int w, int h, YUVFrame **f) { return -1; }
int h264_decode_stream(H264Decoder *d, const uint8_t *p, int n, int w, int h, YUVFrame **f) { return -1; }
int h264_decode_frame_flush(H264Decoder *d, int w, int h, YUVFrame **f) { return -1; }
void yuv_frame_free(YUVFrame *frame) { (void)frame; }
int h264_decode_frame_get_count(H264Decoder *decoder) { return 0; }
void h264_decoder_free(H264Decoder *decoder) { (void)decoder; }
#endif
