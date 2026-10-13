package agorasdk

import (
	"slices"
	"sync"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// ProcessorConfig configures native external PCM processing independently of
// a channel. Its zero value selects 16 kHz mono output, a 64-item queue, and
// rtc.NewAPMConfig. The installed SDK must contain the requested extensions;
// missing extensions produce native initialization errors.
type ProcessorConfig struct {
	// SampleRate is in Hz and Channels is 1 or 2; zero selects 16000 Hz mono.
	// Supported rates are the same as PCMFormat. This format is used for both
	// input validation and requested processed output.
	SampleRate, Channels int
	// QueueCapacity bounds the output queue in items. Zero uses 64; explicit
	// values must be 1–65536. Overflow discards the oldest output frame.
	QueueCapacity int
	// APM selects native processing. Nil uses rtc.NewAPMConfig.
	// Custom values must supply AiNsConfig, AiAecConfig, BghvsCConfig, and AgcConfig.
	// Do not mutate this configuration or its components during construction.
	APM *rtc.APMConfig
	// VAD optionally enables the SDK VAD v2 segmenter on processed frames.
	// Nil disables segmentation. The value is copied and validated; this is not
	// the standalone native rtc.AudioVad detector.
	VAD *rtc.AudioVadConfigV2
}

// AudioProcessor owns a native external PCM processor and a copied output queue.
// Construct it with Service.NewAudioProcessor; the zero value is not usable.
// SendPCM and Close serialize access to the processor. It does not publish
// a channel track automatically, so applications must forward output explicitly.
type AudioProcessor struct {
	mu             sync.Mutex
	native         *rtc.ExternalAudioProcessor
	service        *Service
	output         *Stream[AudioFrame]
	rate, channels int
	closed         bool
	closeOnce      sync.Once
}

func normalizedProcessorConfig(cfg ProcessorConfig) (ProcessorConfig, error) {
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 16000
	}
	if cfg.Channels == 0 {
		cfg.Channels = 1
	}
	if cfg.QueueCapacity == 0 {
		cfg.QueueCapacity = 64
	}
	if err := validatePCMFormat(cfg.SampleRate, cfg.Channels); err != nil {
		return cfg, err
	}
	if cfg.QueueCapacity < 1 || cfg.QueueCapacity > 65536 {
		return cfg, invalid("processor queue capacity", "must be 1–65536")
	}
	if cfg.VAD != nil {
		if err := validateVAD(*cfg.VAD); err != nil {
			return cfg, err
		}
		v := *cfg.VAD
		cfg.VAD = &v
	}
	if cfg.APM == nil {
		cfg.APM = rtc.NewAPMConfig()
	}
	if cfg.APM.AiNsConfig == nil || cfg.APM.AiAecConfig == nil || cfg.APM.BghvsCConfig == nil || cfg.APM.AgcConfig == nil {
		return cfg, invalid("APM configuration", "requires all component configurations")
	}
	return cfg, nil
}

// NewAudioProcessor allocates an external processor owned by s.
// Config.AudioProcessing must have been enabled when creating the Service.
// Requested native APM extensions must be installed; the pinned Linux archive
// does not include every optional extension, so defaults can fail initialization.
// A successful caller should close the processor; Service.Close also closes it.
// The returned error may wrap ErrInvalidConfig, be ErrClosed, or be *Error.
func (s *Service) NewAudioProcessor(cfg ProcessorConfig) (*AudioProcessor, error) {
	cfg, err := normalizedProcessorConfig(cfg)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	if !s.audioProcessing {
		return nil, invalid("audio processor", "requires Config.AudioProcessing=true")
	}
	p := &AudioProcessor{service: s, output: newStream[AudioFrame](cfg.QueueCapacity), rate: cfg.SampleRate, channels: cfg.Channels}
	p.native = rtc.NewExternalAudioProcessor()
	if p.native == nil {
		return nil, &Error{Op: "create audio processor", Code: -1}
	}
	if err := check("initialize audio processor", p.native.Initialize(cfg.APM, cfg.SampleRate, cfg.Channels, cfg.VAD, p.observer())); err != nil {
		p.native.Release()
		p.output.close()
		return nil, err
	}
	s.processors[p] = struct{}{}
	return p, nil
}

func (p *AudioProcessor) observer() *rtc.ExternalAudioProcessorObserver {
	return &rtc.ExternalAudioProcessorObserver{OnProcessedAudioFrame: func(_ *rtc.ExternalAudioProcessor, f *rtc.AudioFrame, v rtc.VadState, speech *rtc.AudioFrame) {
		if f == nil {
			return
		}
		out := copyAudioFrame(f)
		out.VAD = v
		if speech != nil {
			out.Speech = slices.Clone(speech.Buffer)
		}
		p.output.push(out)
	}}
}

// Output returns the bounded queue of copied processed PCM and optional VAD output.
// Native callbacks do not wait for readers; inspect Stream.Stats for drops.
func (p *AudioProcessor) Output() *Stream[AudioFrame] { return p.output }

// SendPCM submits nonempty PCM16 little-endian input in ProcessorConfig's format.
// The buffer must contain an integral number of complete 10 ms frames.
// Unlike standalone rtc.AudioVad, multiple 10 ms frames may be submitted together.
// The method does not pace input or wait for processed output.
// Keep data unchanged until the call returns; consume Output separately.
// It returns ErrClosed after Close or *Error on native submission failure.
func (p *AudioProcessor) SendPCM(data []byte) error {
	if err := validateProcessorPCM(data, p.rate, p.channels); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	return check("process PCM", p.native.PushAudioPcmData(data, p.rate, p.channels, 0))
}

// Close releases native processing resources and closes Output.
// It is idempotent and safe for concurrent calls. Buffered output remains readable
// with a live context until io.EOF. Pending input is not guaranteed to finish
// processing. Close currently returns nil; it is not a completion acknowledgement.
func (p *AudioProcessor) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.native.Release()
		p.native = nil
		p.mu.Unlock()
		p.output.close()
		p.service.mu.Lock()
		delete(p.service.processors, p)
		p.service.mu.Unlock()
	})
	return nil
}
