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

// AudioFrame is interleaved signed 16-bit little-endian PCM owned by the reader.
type AudioFrame struct {
	Position                                       AudioPosition
	Channel, UserID                                string
	Data                                           []byte
	SampleRate, Channels                           int
	SamplesPerChannel, BytesPerSample              int
	AVSyncType, FarField                           int
	TimestampMillis, PresentationMillis            int64
	VoiceProbability, MusicProbability, RMS, Pitch int
	VAD                                            rtc.VadState
	Speech                                         []byte
}
type AudioCodec int

const (
	Opus AudioCodec = AudioCodec(rtc.AudioCodecOpus)
	AAC  AudioCodec = AudioCodec(rtc.AudioCodecAacLc)
)

// EncodedAudioFrame contains one encoded packet and its timing/format.
type EncodedAudioFrame struct {
	Data                                    []byte
	Codec                                   AudioCodec
	SampleRate, Channels, SamplesPerChannel int
	TimestampMillis                         int64
	Speech                                  bool
}
type ReceivedEncodedAudio struct {
	Channel, UserID string
	Data            []byte
	Codec           AudioCodec
	SentAtMillis    int64
}

func (s *Session) audioObserver() *rtc.AudioFrameObserver {
	params := rtc.AudioFrameObserverAudioParams{SampleRate: s.cfg.Audio.Receive.SampleRate, Channels: s.cfg.Audio.Receive.Channels, Mode: rtc.RawAudioFrameOpModeReadOnly, SamplesPerCall: s.cfg.Audio.Receive.SampleRate / 100 * s.cfg.Audio.Receive.Channels}
	capture := func(position AudioPosition, channel string, f *rtc.AudioFrame) bool {
		if f != nil {
			out := copyAudioFrame(f)
			out.Position, out.Channel = position, channel
			s.audio.push(out)
		}
		return true
	}
	return &rtc.AudioFrameObserver{
		OnGetAudioFramePosition:           func(*rtc.LocalUser) int { return int(s.cfg.Receive.AudioPositions) },
		OnGetPlaybackAudioFrameParam:      func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams { return params },
		OnGetRecordAudioFrameParam:        func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams { return params },
		OnGetMixedAudioFrameParam:         func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams { return params },
		OnGetEarMonitoringAudioFrameParam: func(*rtc.LocalUser) rtc.AudioFrameObserverAudioParams { return params },
		OnRecordAudioFrame:                func(_ *rtc.LocalUser, ch string, f *rtc.AudioFrame) bool { return capture(AudioRecord, ch, f) },
		OnPlaybackAudioFrame:              func(_ *rtc.LocalUser, ch string, f *rtc.AudioFrame) bool { return capture(AudioPlayback, ch, f) },
		OnMixedAudioFrame:                 func(_ *rtc.LocalUser, ch string, f *rtc.AudioFrame) bool { return capture(AudioMixed, ch, f) },
		OnEarMonitoringAudioFrame:         func(_ *rtc.LocalUser, f *rtc.AudioFrame) bool { return capture(AudioEarMonitoring, s.cfg.Channel, f) },
		OnPlaybackAudioFrameBeforeMixing: func(_ *rtc.LocalUser, channel, uid string, f *rtc.AudioFrame, vad rtc.VadState, speech *rtc.AudioFrame) bool {
			if f == nil {
				return true
			}
			frame := copyAudioFrame(f)
			frame.Position, frame.Channel, frame.UserID, frame.VAD = AudioBeforeMixing, channel, uid, vad
			if speech != nil {
				frame.Speech = slices.Clone(speech.Buffer)
			}
			s.audio.push(frame)
			return true
		},
	}
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

// SendPCM publishes on first use. Data must contain complete PCM16 samples with
// an integral millisecond duration (10 ms at 44100 Hz) in Audio.Send format.
// Calls are not paced or buffered by the wrapper; use PlayPCM for paced files.
func (s *Session) SendPCM(data []byte) error { return s.SendPCMAt(data, 0) }

// SendPCMAt forwards presentationMillis unchanged, including opaque application
// tags. It never increments or replaces the supplied value.
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

// PlayPCM reads and paces PCM16 using a single 10 ms buffer. Reader must return
// promptly; a blocked Reader cannot be interrupted by context cancellation.
// A partial final frame is zero-padded. Cancellation stops future sends.
func (s *Session) PlayPCM(ctx context.Context, reader io.Reader) error {
	return s.PlayPCMAt(ctx, reader, 0)
}

// PlayPCMAt paces PCM with an explicit initial presentation timestamp in milliseconds.
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
func (s *Session) SendEncodedAudio(frame EncodedAudioFrame) error {
	if len(frame.Data) == 0 {
		return invalid("encoded audio", "must not be empty")
	}
	if err := validateEncodedAudioFormat(frame.SampleRate, frame.Channels); err != nil {
		return err
	}
	if !validAudioCodec(frame.Codec) {
		return invalid("audio codec", "is unsupported")
	}
	if frame.SamplesPerChannel < 0 {
		return invalid("samples per channel", "must not be negative")
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
	info := rtc.EncodedAudioFrameInfo{Codec: rtc.AudioCodecType(frame.Codec), SampleRateHz: frame.SampleRate, NumberOfChannels: frame.Channels, SamplesPerChannel: frame.SamplesPerChannel, CaptureTimeMs: frame.TimestampMillis, Speech: frame.Speech}
	return check("send encoded audio", s.native.PushAudioEncodedData(frame.Data, &info))
}
func (s *Session) InterruptAudio() error {
	return s.withNative("interrupt audio", func(c *rtc.RtcConnection) int { return c.InterruptAudio() })
}

// WaitPCM estimates when sent PCM has played out using the upstream duration
// clock plus its fixed margin. It is not a native queue or remote-playback acknowledgment.
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
func (s *Session) SendAudioMetadata(data []byte) error {
	if len(data) == 0 || len(data) > 1024 {
		return invalid("audio metadata", "must contain 1–1024 bytes")
	}
	return s.withNative("send audio metadata", func(c *rtc.RtcConnection) int { return c.SendAudioMetaData(data) })
}
func (s *Session) SetVolumeIndication(interval time.Duration, smoothing int, reportVAD bool) error {
	if interval < 0 || interval > time.Hour || smoothing < 0 || smoothing > 10 {
		return invalid("volume indication", "requires nonnegative interval and smoothing 0–10")
	}
	return s.withNative("set volume indication", func(c *rtc.RtcConnection) int {
		return c.GetLocalUser().SetAudioVolumeIndicationParameters(int(interval/time.Millisecond), smoothing, reportVAD)
	})
}

// SetAudioScenario changes processing outside the native callback thread. Handle
// CapabilityFallback events here if your application chooses to accept a fallback.
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
