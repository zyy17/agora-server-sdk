package agorasdk

import (
	"context"
	"errors"
	"io"
	"math"
	"slices"
	"time"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// AudioFrame is copied, interleaved signed PCM16 little-endian audio.
// Data and Speech are Go-owned and remain valid after callbacks and Close.
// Only AudioBeforeMixing frames identify an individual remote UserID.
// Telemetry fields are native values; availability and interpretation depend
// on the installed SDK and selected processing path.
type AudioFrame struct {
	// Position identifies the PCM observation point; processor output leaves it zero.
	Position AudioPosition
	// Channel and UserID identify the source when available. Mixed, local, and
	// processor frames may have empty values; do not infer an individual speaker.
	Channel, UserID string
	// Data contains the complete copied PCM frame, including silence.
	Data []byte
	// SampleRate is in Hz; Channels is the number of interleaved channels.
	SampleRate, Channels int
	// SamplesPerChannel counts samples in each channel, not across all channels.
	// BytesPerSample is the byte width of one sample; PCM16 uses 2.
	SamplesPerChannel, BytesPerSample int
	// AVSyncType and FarField preserve native synchronization and far-field indicators.
	AVSyncType, FarField int
	// TimestampMillis is the native render timestamp in milliseconds.
	// PresentationMillis preserves the native presentation value, which may be an
	// opaque application tag. Neither is guaranteed to be Unix time.
	TimestampMillis, PresentationMillis int64
	// VoiceProbability, MusicProbability, RMS, and Pitch preserve native audio
	// telemetry without normalization; do not assume a probability or pitch scale
	// that the selected native SDK does not document.
	VoiceProbability, MusicProbability, RMS, Pitch int
	// VAD is the SDK VAD v2 state when enabled at AudioBeforeMixing, or processor
	// VAD state when configured. It does not invoke the standalone rtc.AudioVad.
	VAD rtc.VadState
	// Speech is the optional copied VAD-selected PCM. It may contain pre-roll at
	// speech onset and subsequent speech frames; it is not solely a start event.
	// Forward either Data or Speech to a recognizer, not both as duplicate audio.
	Speech []byte
}

// AudioCodec identifies a native encoded-audio codec.
// Named codecs preserve the upstream enum; transport and peer support vary.
type AudioCodec int

const (
	// Opus identifies the native Opus codec.
	Opus AudioCodec = AudioCodec(rtc.AudioCodecOpus)
	// PCMA identifies G.711 A-law.
	PCMA AudioCodec = AudioCodec(rtc.AudioCodecPcma)
	// PCMU identifies G.711 mu-law.
	PCMU AudioCodec = AudioCodec(rtc.AudioCodecPcmu)
	// G722 identifies G.722.
	G722 AudioCodec = AudioCodec(rtc.AudioCodecG722)
	// HEAAC identifies high-efficiency AAC.
	HEAAC AudioCodec = AudioCodec(rtc.AudioCodecHeAac)
	// HEAACv2 identifies high-efficiency AAC v2.
	HEAACv2 AudioCodec = AudioCodec(rtc.AudioCodecHeAac2)
	// JC1 identifies the native JC1 codec.
	JC1 AudioCodec = AudioCodec(rtc.AudioCodecJc1)
	// LPCNet identifies the native LPCNet codec.
	LPCNet AudioCodec = AudioCodec(rtc.AudioCodecLpcnet)
	// OpusMultiChannel identifies native Opus with 3–8 channels.
	OpusMultiChannel AudioCodec = AudioCodec(rtc.AudioCodecOpusMC)
	// AAC identifies AAC-LC, not an ADTS file or other AAC profiles.
	AAC AudioCodec = AudioCodec(rtc.AudioCodecAacLc)
)

// EncodedAudioFrame contains one encoded packet and its format metadata.
// The wrapper checks metadata bounds but does not decode the packet or prove
// that its contents match the declared codec, rate, channels, or sample count.
type EncodedAudioFrame struct {
	// Data contains one encoded packet. Keep it unchanged during the send call.
	// Empty data is permitted only when SendEvenIfEmpty is explicitly true.
	Data []byte
	// Codec selects the native codec for Data; peer support must be verified.
	Codec AudioCodec
	// SampleRate is in Hz; accepted values are 7350, 8000, 11025, 12000, 16000,
	// 22050, 24000, 32000, 44100, 48000, 64000, 88200, and 96000.
	// Channels must be 1 or 2, or 3–8 for OpusMultiChannel. SamplesPerChannel
	// must fit nonnegative int32 and is passed
	// through to the native SDK, including zero.
	SampleRate, Channels, SamplesPerChannel int
	// TimestampMillis is the native capture timestamp in milliseconds, not necessarily Unix time.
	// The connection binding rewrites zero to -1 to select native synchronization.
	TimestampMillis int64
	// Speech sets the native encoded-frame speech indicator.
	Speech bool
	// SendEvenIfEmpty forwards the native empty-packet policy. A true value
	// permits an empty Data slice; native codec/transport support still applies.
	SendEvenIfEmpty bool
}

// ReceivedEncodedAudio is a copied native encoded-audio packet.
// Its callback does not supply a sample rate, channel count, or decoded duration;
// obtain that information from the codec or application protocol.
type ReceivedEncodedAudio struct {
	// Channel and UserID identify the remote sender.
	Channel, UserID string
	// Data owns the received encoded packet bytes.
	Data []byte
	// Codec is the native-reported codec; it may be a value without a named convenience constant.
	Codec AudioCodec
	// SentAtMillis is the native-reported send timestamp in milliseconds.
	// Do not assume it shares a wall-clock origin with local time.
	SentAtMillis int64
}

func (s *Session) audioObserver() *rtc.AudioFrameObserver {
	capture := func(position AudioPosition, channel string, f *rtc.AudioFrame) bool {
		if f != nil {
			out := copyAudioFrame(f)
			out.Position, out.Channel = position, channel
			return s.observeAudioFrame(out, f, s.audioFrameParameters(position).Mode == rtc.RawAudioFrameOpModeReadWrite)
		}
		return true
	}
	return &rtc.AudioFrameObserver{
		OnGetAudioFramePosition:      func(*rtc.LocalUser) int { return int(s.cfg.Receive.AudioPositions) },
		OnGetPlaybackAudioFrameParam: func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams { return s.audioFrameParameters(AudioPlayback) },
		OnGetRecordAudioFrameParam:   func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams { return s.audioFrameParameters(AudioRecord) },
		OnGetMixedAudioFrameParam:    func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams { return s.audioFrameParameters(AudioMixed) },
		OnGetEarMonitoringAudioFrameParam: func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams {
			return s.audioFrameParameters(AudioEarMonitoring)
		},
		OnRecordAudioFrame:        func(_ *rtc.LocalUser, ch string, f *rtc.AudioFrame) bool { return capture(AudioRecord, ch, f) },
		OnPlaybackAudioFrame:      func(_ *rtc.LocalUser, ch string, f *rtc.AudioFrame) bool { return capture(AudioPlayback, ch, f) },
		OnMixedAudioFrame:         func(_ *rtc.LocalUser, ch string, f *rtc.AudioFrame) bool { return capture(AudioMixed, ch, f) },
		OnEarMonitoringAudioFrame: func(_ *rtc.LocalUser, f *rtc.AudioFrame) bool { return capture(AudioEarMonitoring, s.cfg.Channel, f) },
		OnPlaybackAudioFrameBeforeMixing: func(_ *rtc.LocalUser, channel, uid string, f *rtc.AudioFrame, vad rtc.VadState, speech *rtc.AudioFrame) bool {
			if f == nil {
				return true
			}
			frame := copyAudioFrame(f)
			frame.Position, frame.Channel, frame.UserID, frame.VAD = AudioBeforeMixing, channel, uid, vad
			if speech != nil {
				frame.Speech = slices.Clone(speech.Buffer)
			}
			return s.observeAudioFrame(frame, f, false)
		},
	}
}

// observeAudioFrame isolates synchronous hook buffers from the asynchronous
// queue. Only accepted ReadWrite Data edits affect native and queued PCM.
func (s *Session) observeAudioFrame(frame AudioFrame, native *rtc.AudioFrame, writable bool) bool {
	if s.cfg.Audio.Callbacks == nil || s.cfg.Audio.Callbacks.OnFrame == nil {
		s.audio.push(frame)
		return true
	}
	hook := frame
	hook.Data, hook.Speech = slices.Clone(frame.Data), slices.Clone(frame.Speech)
	accepted := s.processAudioCallback(hook)
	if accepted && writable {
		copy(native.Buffer, hook.Data)
		copy(frame.Data, hook.Data)
	}
	s.audio.push(frame)
	return accepted
}

func copyAudioFrame(f *rtc.AudioFrame) AudioFrame {
	return AudioFrame{Data: slices.Clone(f.Buffer), SampleRate: f.SamplesPerSec, Channels: f.Channels,
		SamplesPerChannel: f.SamplesPerChannel, BytesPerSample: f.BytesPerSample, AVSyncType: f.AvsyncType, FarField: f.FarFieldFlag,
		TimestampMillis: f.RenderTimeMs, PresentationMillis: f.PresentTimeMs, VoiceProbability: f.VoiceProb, MusicProbability: f.MusicProb, RMS: f.Rms, Pitch: f.Pitch}
}

func (s *Session) encodedAudioObserver() *rtc.AudioEncodedFrameObserver {
	return &rtc.AudioEncodedFrameObserver{OnEncodedAudioFrameReceived: func(uid string, data []byte, ts int64, codec int) {
		s.encodedAudio.push(ReceivedEncodedAudio{s.cfg.Channel, uid, slices.Clone(data), AudioCodec(codec), ts})
	}}
}

func validatePCM(data []byte, rate, channels int) error {
	if err := validatePCMFormat(rate, channels); err != nil {
		return err
	}
	samples := len(data) / (channels * 2)
	if len(data) == 0 || len(data)%(channels*2) != 0 || uint64(samples) > math.MaxUint32 || samples%(rate/gcd(rate, 1000)) != 0 {
		return invalid("PCM data", "must contain complete PCM16 samples with an integral millisecond duration (10 ms at 44100 Hz)")
	}
	return nil
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func validateProcessorPCM(data []byte, rate, channels int) error {
	if err := validatePCMFormat(rate, channels); err != nil {
		return err
	}
	if len(data) == 0 || len(data)%(rate/100*channels*2) != 0 {
		return invalid("processor PCM data", "must contain complete nonempty 10 ms PCM16 frames")
	}
	return nil
}

// SendPCM publishes on first use and submits PCM with presentation value zero.
// Data must contain complete nonempty PCM16 samples in AudioConfig.Send format,
// with an integral-millisecond duration (multiples of 10 ms at 44100 Hz) and at
// most uint32 samples per channel. Calls are not paced by the wrapper.
//
// The caller must keep data unchanged until the call returns. Use PlayPCM for
// paced readers and a single application writer per track to preserve ordering.
// Success means local native acceptance, not remote delivery or playback.
// Wrong publication modes or Audience return ErrNotPublishing.
func (s *Session) SendPCM(data []byte) error { return s.SendPCMAt(data, 0) }

// SendPCMAt has the format, lifetime, and unpaced-send contract of SendPCM.
// It forwards presentationMillis unchanged, including zero or opaque application
// tags; it never increments or replaces the supplied value.
func (s *Session) SendPCMAt(data []byte, presentationMillis int64) error {
	if err := validatePCM(data, s.cfg.Audio.Send.SampleRate, s.cfg.Audio.Send.Channels); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.cfg.Audio.Mode != PCM {
		return ErrNotPublishing
	}
	if err := s.publishAudioLocked(); err != nil {
		return err
	}
	return check("send PCM", s.transport.PushAudioPcmData(data, s.cfg.Audio.Send.SampleRate, s.cfg.Audio.Send.Channels, presentationMillis))
}

// PlayPCM reads and paces PCM16 in 10 ms frames with presentation value zero.
// It uses one reusable frame buffer and publishes on first send. A final partial
// frame containing complete samples is zero-padded; an incomplete PCM sample
// returns an error wrapping ErrInvalidConfig. Reader errors are propagated.
//
// ctx and reader must not be nil. A blocked Read cannot be interrupted by ctx;
// use a reader that unblocks on application cancellation. Cancellation stops
// future sends but does not flush previously queued native or remote audio.
// The fixed pacing schedule can catch up after a stalled reader; applications
// streaming live sources may need their own underrun policy.
// The reader remains owned by the caller and is not closed by this method.
func (s *Session) PlayPCM(ctx context.Context, reader io.Reader) error {
	return s.PlayPCMAt(ctx, reader, 0)
}

// PlayPCMAt has the reader, pacing, ownership, and cancellation contract of PlayPCM.
// A nonzero presentationMillis is advanced by 10 for each frame; zero remains
// zero throughout playback. Do not use this method to preserve an opaque tag
// unchanged across every frame; use SendPCMAt for that behavior.
func (s *Session) PlayPCMAt(ctx context.Context, reader io.Reader, presentationMillis int64) error {
	return playPCM(ctx, reader, s.cfg.Audio.Send.SampleRate, s.cfg.Audio.Send.Channels, presentationMillis, s.SendPCMAt)
}

func playPCM(ctx context.Context, reader io.Reader, rate, channels int, presentationMillis int64, send func([]byte, int64) error) error {
	if ctx == nil || reader == nil {
		return invalid("context/reader", "must not be nil")
	}
	frameBytes := rate / 100 * channels * 2
	buf := make([]byte, frameBytes)
	next := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := io.ReadFull(reader, buf)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}
		if n == 0 {
			return nil
		}
		if n%(2*channels) != 0 {
			return invalid("PCM reader", "ended with an incomplete sample")
		}
		clear(buf[n:])
		if wait := time.Until(next); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := send(buf, presentationMillis); err != nil {
			return err
		}
		next = next.Add(10 * time.Millisecond)
		if presentationMillis != 0 {
			presentationMillis += 10
		}
		if err != nil {
			return nil
		}
	}
}

// SendEncodedAudio validates packet metadata and publishes the configured encoded
// audio track on first use. It requires Broadcaster and AudioConfig.Mode
// EncodedAudio, otherwise ErrNotPublishing is returned. It does not pace,
// transcode, or parse the payload. Keep frame.Data unchanged during the call.
// A nil result is local native acceptance, not a remote decode acknowledgement.
func (s *Session) SendEncodedAudio(frame EncodedAudioFrame) error {
	if err := frame.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.cfg.Audio.Mode != EncodedAudio {
		return ErrNotPublishing
	}
	if err := s.publishAudioLocked(); err != nil {
		return err
	}
	info := rtc.EncodedAudioFrameInfo{Codec: rtc.AudioCodecType(frame.Codec), SampleRateHz: frame.SampleRate, NumberOfChannels: frame.Channels, SamplesPerChannel: frame.SamplesPerChannel, CaptureTimeMs: frame.TimestampMillis, Speech: frame.Speech, SendEvenIfEmpty: frame.SendEvenIfEmpty}
	return check("send encoded audio", s.native.PushAudioEncodedData(frame.Data, &info))
}

// InterruptAudio asks the native connection to interrupt queued audio.
// The AI-server path unpublishes and republishes the track; other scenarios
// clear its sender buffer. This does not cancel application producers or remove
// audio already delivered to peers. Stop producers and discard application
// queues before calling it to prevent stale audio from being enqueued afterward.
// The binding does not propagate each internal unpublish, publish, or clear
// failure individually, so a nil result is not a remote-silence guarantee.
func (s *Session) InterruptAudio() error {
	return s.withNative("interrupt audio", func(c *rtc.RtcConnection) int { return c.InterruptAudio() })
}

// WaitPCM polls the native duration-based PCM completion estimate every 10 ms.
// The estimate includes a fixed native margin (180 ms in the pinned binding);
// it is not a native queue or remote-playback acknowledgement.
// It requires PCM mode and a non-nil context, and returns ErrUnsupported for
// other modes, ErrClosed after Close, or ctx.Err on cancellation.
// It does not stop producers; sending more PCM can extend the wait.
func (s *Session) WaitPCM(ctx context.Context) error {
	if s.cfg.Audio.Mode != PCM {
		return ErrUnsupported
	}
	if ctx == nil {
		return invalid("context", "must not be nil")
	}
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return ErrClosed
		}
		done := s.native.IsPushToRtcCompleted()
		s.mu.Unlock()
		if done {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.done:
			return ErrClosed
		case <-tick.C:
		}
	}
}

// SendAudioMetadata submits metadata for the native audio path.
// The wrapper accepts 1–1024 bytes, but the pinned upstream release notes
// document a 64-byte native allowance. Keep payloads at most 64 bytes unless
// the deployed native version has been independently verified.
// Keep data unchanged during the call. Incoming values appear as AudioMetadata
// events with copied []byte Data; native publishing and peer requirements apply.
func (s *Session) SendAudioMetadata(data []byte) error {
	if len(data) == 0 || len(data) > 1024 {
		return invalid("audio metadata", "must contain 1–1024 bytes")
	}
	return s.withNative("send audio metadata", func(c *rtc.RtcConnection) int { return c.SendAudioMetaData(data) })
}

// SetVolumeIndication configures native volume events.
// interval must be between zero and one hour and is truncated to whole
// milliseconds; zero requests the native disabled setting. smoothing must be
// 0–10. reportVAD requests the native voice-activity indicator.
// Results arrive as AudioVolume events with []Volume Data and total volume in
// Event.State; the native SDK determines supported reporting behavior.
func (s *Session) SetVolumeIndication(interval time.Duration, smoothing int, reportVAD bool) error {
	if interval < 0 || interval > time.Hour || smoothing < 0 || smoothing > 10 {
		return invalid("volume indication", "requires nonnegative interval and smoothing 0–10")
	}
	return s.withNative("set volume indication", func(c *rtc.RtcConnection) int {
		return c.GetLocalUser().SetAudioVolumeIndicationParameters(int(interval/time.Millisecond), smoothing, reportVAD)
	})
}

// SetAudioScenario selects native processing outside the callback thread.
// Handle CapabilityFallback events and call this method if the application
// chooses a supported fallback; the wrapper does not accept one automatically.
// Changing the scenario does not change configured send/receive PCM formats.
// A native rejection is returned as *Error.
func (s *Session) SetAudioScenario(scenario AudioScenario) error {
	if scenario > ScenarioAIClient {
		return invalid("audio scenario", "is unknown")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if err := check("set audio scenario", s.native.UpdateAudioSenario(scenario.native())); err != nil {
		return err
	}
	return nil
}

func validAudioCodec(codec AudioCodec) bool {
	switch rtc.AudioCodecType(codec) {
	case rtc.AudioCodecOpus, rtc.AudioCodecPcma, rtc.AudioCodecPcmu, rtc.AudioCodecG722, rtc.AudioCodecAacLc, rtc.AudioCodecHeAac, rtc.AudioCodecJc1, rtc.AudioCodecHeAac2, rtc.AudioCodecLpcnet, rtc.AudioCodecOpusMC:
		return true
	default:
		return false
	}
}

func validateEncodedAudioFormat(rate, channels int) error {
	switch rate {
	case 7350, 8000, 11025, 12000, 16000, 22050, 24000, 32000, 44100, 48000, 64000, 88200, 96000:
	default:
		return invalid("encoded audio sample rate", "is unsupported")
	}
	if channels != 1 && channels != 2 {
		return invalid("encoded audio channels", "must be 1 or 2")
	}
	return nil
}

func (f EncodedAudioFrame) validate() error {
	if (len(f.Data) == 0 && !f.SendEvenIfEmpty) || uint64(len(f.Data)) > math.MaxUint32 {
		return invalid("encoded audio", "must fit uint32 bytes; empty requires SendEvenIfEmpty")
	}
	channels := f.Channels
	if f.Codec == OpusMultiChannel {
		if channels < 3 || channels > 8 {
			return invalid("encoded audio channels", "OpusMultiChannel requires 3–8")
		}
		channels = 1 // Validate rate separately from multichannel layout.
	}
	if err := validateEncodedAudioFormat(f.SampleRate, channels); err != nil {
		return err
	}
	if !validAudioCodec(f.Codec) {
		return invalid("audio codec", "is unsupported")
	}
	if f.SamplesPerChannel < 0 || int64(f.SamplesPerChannel) > math.MaxInt32 {
		return invalid("samples per channel", "must fit nonnegative int32")
	}
	return nil
}
