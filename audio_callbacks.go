package agorasdk

import (
	"math"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// AudioFrameParameters configures one native PCM observation point.
// Format zero fields inherit AudioConfig.Receive. Native/device availability
// determines whether that observation point actually produces frames.
type AudioFrameParameters struct {
	// Format selects the sample rate and channel count for this observation point.
	Format PCMFormat
	// SamplesPerCall counts interleaved samples across all channels. Zero selects
	// 10 ms. Explicit values must be positive int32 multiples of Channels.
	SamplesPerCall int
	// ReadWrite allows OnFrame to modify PCM bytes before native processing
	// continues. Shape/format changes are unsupported. False is read-only.
	ReadWrite bool
}

// AudioReceiveParameters independently configures mixed/local observation
// points. Before-mixing audio uses AudioConfig.Receive and is read-only.
// A nil entry inherits the shared receive format with 10 ms read-only frames.
type AudioReceiveParameters struct {
	// Playback configures PCM after remote speakers have been mixed.
	Playback *AudioFrameParameters
	// Record configures recorded PCM when a native recording device exists.
	Record *AudioFrameParameters
	// Mixed configures mixed playback and recording PCM.
	Mixed *AudioFrameParameters
	// EarMonitoring configures the native ear-monitoring path.
	EarMonitoring *AudioFrameParameters
}

// AudioCallbacks provides synchronous native PCM accept/reject control.
// These optional functions run on native callback threads and must return
// promptly. They must not call Session or Service methods that take lifecycle
// locks (including Close, WithNative, Send, and SetRole), wait for events, or do
// blocking I/O. Info, Role, Done, and SetAudioFallback are safe snapshot methods.
// Callbacks can be concurrent, so the application must synchronize its state.
type AudioCallbacks struct {
	// OnFrame returns whether native processing should accept the frame. The
	// frame is still queued for observation, including rejected frames. Data edits
	// affect native PCM only at a ReadWrite observation point, on true return.
	// Do not retain or concurrently access the callback buffers. Only in-place
	// Data edits are supported; format, metadata, and slice-header edits are ignored.
	// Panics are contained, emit CallbackPanic, and reject that frame.
	OnFrame func(AudioFrame) bool
}

func (s *Session) processAudioCallback(frame AudioFrame) (accepted bool) {
	accepted = true
	if callbacks := s.cfg.Audio.Callbacks; callbacks != nil && callbacks.OnFrame != nil {
		defer func() {
			if recover() != nil {
				accepted = false
				s.event(Event{Kind: CallbackPanic, Message: "audio frame callback panicked"})
			}
		}()
		accepted = callbacks.OnFrame(frame)
	}
	return accepted
}

func (s *Session) audioFrameParameters(position AudioPosition) rtc.AudioFrameObserverAudioParams {
	base := s.cfg.Audio.Receive
	value := AudioFrameParameters{Format: base}
	if all := s.cfg.Audio.ReceiveParameters; all != nil {
		var selected *AudioFrameParameters
		switch position {
		case AudioPlayback:
			selected = all.Playback
		case AudioRecord:
			selected = all.Record
		case AudioMixed:
			selected = all.Mixed
		case AudioEarMonitoring:
			selected = all.EarMonitoring
		}
		if selected != nil {
			value = *selected
		}
	}
	value = value.withDefaults(base)
	mode := rtc.RawAudioFrameOpModeReadOnly
	if value.ReadWrite {
		mode = rtc.RawAudioFrameOpModeReadWrite
	}
	return rtc.AudioFrameObserverAudioParams{SampleRate: value.Format.SampleRate, Channels: value.Format.Channels, Mode: mode, SamplesPerCall: value.SamplesPerCall}
}

func (p AudioFrameParameters) withDefaults(base PCMFormat) AudioFrameParameters {
	if p.Format.SampleRate == 0 {
		p.Format.SampleRate = base.SampleRate
	}
	if p.Format.Channels == 0 {
		p.Format.Channels = base.Channels
	}
	if p.SamplesPerCall == 0 {
		p.SamplesPerCall = p.Format.SampleRate / 100 * p.Format.Channels
	}
	return p
}

func (c ChannelConfig) validateAudioCallbacks() error {
	if policy := c.Audio.Fallback; policy != nil && policy.Scenario > ScenarioAIClient {
		return invalid("fallback scenario", "is unknown")
	}
	if params := c.Audio.ReceiveParameters; params != nil {
		for _, value := range []*AudioFrameParameters{params.Playback, params.Record, params.Mixed, params.EarMonitoring} {
			if value == nil {
				continue
			}
			p := value.withDefaults(c.Audio.Receive)
			if err := validatePCMFormat(p.Format.SampleRate, p.Format.Channels); err != nil {
				return err
			}
			if p.SamplesPerCall <= 0 || p.SamplesPerCall > math.MaxInt32 || p.SamplesPerCall%p.Format.Channels != 0 {
				return invalid("samples per call", "must be a positive int32 multiple of Channels")
			}
		}
	}
	return nil
}

func copyAudioOptions(config AudioConfig) AudioConfig {
	if config.Fallback != nil {
		copy := *config.Fallback
		config.Fallback = &copy
	}
	if config.Callbacks != nil {
		copy := *config.Callbacks
		config.Callbacks = &copy
	}
	if config.ReceiveParameters != nil {
		copy := *config.ReceiveParameters
		for _, pointer := range []**AudioFrameParameters{&copy.Playback, &copy.Record, &copy.Mixed, &copy.EarMonitoring} {
			if *pointer != nil {
				value := **pointer
				*pointer = &value
			}
		}
		config.ReceiveParameters = &copy
	}
	return config
}
