package agorasdk

import (
	"errors"
	"math"
	"sync"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// ErrDecoderNeedsInput means a video access unit was accepted but no frame is
// available yet. Submit another unit; this is not a corrupt-bitstream verdict.
var ErrDecoderNeedsInput = errors.New("agora: decoder needs more input")

// VideoDecodeOptions optionally resizes decoded video. Both dimensions zero
// preserve the source size; explicit dimensions must be even and 2–16384 pixels.
type VideoDecodeOptions struct{ Width, Height int }

// VideoDecoder owns an optional FFmpeg H.264/H.265 decoder, independent of Service
// and Session. Create it with NewVideoDecoder and Close it after use. Decode and
// Close serialize access. No RTC channel, publishing, pacing or file I/O occurs.
type VideoDecoder struct {
	mu     sync.Mutex
	native *rtc.VideoDecoder
}

// NewVideoDecoder allocates a decoder. It returns ErrUnsupported unless built
// with avcodec and the required FFmpeg libraries. Native setup failures are
// returned unchanged. Only 8-bit 4:2:0 Annex-B input is supported by this path.
func NewVideoDecoder() (*VideoDecoder, error) {
	if !avcodecAvailable {
		return nil, ErrUnsupported
	}
	native, err := rtc.NewVideoDecoder()
	if err != nil {
		return nil, err
	}
	return &VideoDecoder{native: native}, nil
}

// Decode accepts one nonempty H.264/H.265 Annex-B access unit and returns owned
// compact I420 pixels. It may return ErrDecoderNeedsInput. When a unit yields
// multiple pictures, the upstream decoder returns its latest picture. No flush
// or per-picture timestamp API exists upstream; timestamps remain zero.
// A codec switch recreates native decoder state and discards buffered pictures.
// Keep data unchanged until the call returns. Results survive Decode and Close.
func (d *VideoDecoder) Decode(data []byte, codec VideoCodec, options VideoDecodeOptions) (VideoFrame, error) {
	if len(data) == 0 || int64(len(data)) > math.MaxInt32 {
		return VideoFrame{}, invalid("video access unit", "must contain 1–int32 bytes")
	}
	if codec != H264 && codec != H265 {
		return VideoFrame{}, invalid("decoder codec", "requires H264 or H265")
	}
	if options.Width != 0 || options.Height != 0 {
		if err := validateDimensions(options.Width, options.Height); err != nil {
			return VideoFrame{}, err
		}
		if options.Width%2 != 0 || options.Height%2 != 0 {
			return VideoFrame{}, invalid("decoder dimensions", "must be even")
		}
	}
	if d == nil {
		return VideoFrame{}, ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.native == nil {
		return VideoFrame{}, ErrClosed
	}
	frame, code := d.native.DecodeAndSave(data, options.Width, options.Height, int(codec))
	if code == 1 {
		return VideoFrame{}, ErrDecoderNeedsInput
	}
	if err := check("decode video", code); err != nil {
		return VideoFrame{}, err
	}
	if frame == nil {
		return VideoFrame{}, ErrDecoderNeedsInput
	}
	out := VideoFrame{Data: frame.Buffer, Width: frame.Stride, Height: frame.Height, Format: PixelFormat(frame.Format), Rotation: int(frame.Rotation), TimestampMillis: frame.Timestamp}
	if err := out.validate(); err != nil {
		return VideoFrame{}, err
	}
	return out, nil
}

// Close releases decoder state, serializes against Decode, and is idempotent.
// Buffered pictures are discarded; previously returned pixels remain valid.
func (d *VideoDecoder) Close() error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.native != nil {
		d.native.Close()
		d.native = nil
	}
	return nil
}

// OpusDecoderConfig selects the encoded input channel count and decoded PCM16
// output. InputChannels zero selects mono. Output.SampleRate zero selects 48000;
// Output.Channels zero preserves InputChannels. Explicit output rates may be
// 1–192000 Hz, subject to native FFmpeg resampler support. Channels are 1 or 2.
type OpusDecoderConfig struct {
	InputChannels int
	Output        PCMFormat
}

// OpusDecoder owns a synchronous optional FFmpeg Opus decoder/resampler. It is
// independent of Service and Session and never publishes or paces audio.
// Create with NewOpusDecoder and Close when finished; Decode and Close serialize.
type OpusDecoder struct {
	mu     sync.Mutex
	native *rtc.OpusDecoder
	format PCMFormat
}

func normalizedOpusDecoderConfig(cfg OpusDecoderConfig) (OpusDecoderConfig, error) {
	if cfg.InputChannels == 0 {
		cfg.InputChannels = 1
	}
	if cfg.Output.SampleRate == 0 {
		cfg.Output.SampleRate = 48000
	}
	if cfg.Output.Channels == 0 {
		cfg.Output.Channels = cfg.InputChannels
	}
	if cfg.InputChannels < 1 || cfg.InputChannels > 2 || cfg.Output.Channels < 1 || cfg.Output.Channels > 2 || cfg.Output.SampleRate < 1 || cfg.Output.SampleRate > 192000 {
		return cfg, invalid("Opus decoder configuration", "requires 1–2 channels and output rate 1–192000 Hz")
	}
	return cfg, nil
}

// NewOpusDecoder allocates a decoder with copied configuration. Invalid options
// wrap ErrInvalidConfig; without avcodec it returns ErrUnsupported; allocation
// failure is reported as *Error. It does not create or require an RTC Service.
func NewOpusDecoder(cfg OpusDecoderConfig) (*OpusDecoder, error) {
	cfg, err := normalizedOpusDecoderConfig(cfg)
	if err != nil {
		return nil, err
	}
	if !avcodecAvailable {
		return nil, ErrUnsupported
	}
	native := rtc.NewOpusDecoder(cfg.InputChannels, cfg.Output.SampleRate, cfg.Output.Channels)
	if native == nil {
		return nil, &Error{Op: "create Opus decoder", Code: -1}
	}
	return &OpusDecoder{native: native, format: cfg.Output}, nil
}

// Decode consumes one nonempty raw Opus packet (no Ogg/container framing) and
// returns owned interleaved PCM16 little-endian audio in the configured format.
// Keep packet unchanged during the call. Output may be empty while the native
// pipeline buffers input; the upstream API has no resampler flush operation.
// Timestamps and observer telemetry remain zero. Results survive later calls
// and Close. Native decoding failures preserve the raw decoder's error text.
func (d *OpusDecoder) Decode(packet []byte) (AudioFrame, error) {
	if len(packet) == 0 || int64(len(packet)) > math.MaxInt32 {
		return AudioFrame{}, invalid("Opus packet", "must contain 1–int32 bytes")
	}
	if d == nil {
		return AudioFrame{}, ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.native == nil {
		return AudioFrame{}, ErrClosed
	}
	data, err := d.native.Decode(packet)
	if err != nil {
		return AudioFrame{}, err
	}
	if len(data)%(d.format.Channels*2) != 0 {
		return AudioFrame{}, invalid("decoded Opus", "has incomplete PCM16 samples")
	}
	return AudioFrame{Data: data, SampleRate: d.format.SampleRate, Channels: d.format.Channels, BytesPerSample: 2, SamplesPerChannel: len(data) / (d.format.Channels * 2)}, nil
}

// Close serializes against Decode and releases native decoder/resampler state.
// It is safe to repeat. Buffered output is discarded without flushing.
func (d *OpusDecoder) Close() error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.native != nil {
		d.native.Release()
		d.native = nil
	}
	return nil
}
