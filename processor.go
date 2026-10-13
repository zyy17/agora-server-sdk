package agorasdk

import (
	"math"
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
	// Supported output rates are the same as PCMFormat.
	SampleRate, Channels int
	// Input selects the default input format for SendPCM and SendPCMAt. Each zero
	// field inherits the output field above. Input rates may be 8000–48000 Hz
	// divisible by 100; native processing performs any required conversion.
	Input PCMFormat
	// QueueCapacity bounds the output queue in items. Zero uses 64; explicit
	// values must be 1–65536. Overflow discards the oldest output frame.
	QueueCapacity int
	// APM selects native processing. Nil uses rtc.NewAPMConfig.
	// Custom values must supply AiNsConfig, AiAecConfig, BghvsCConfig, and AgcConfig.
	// All settings are deeply copied. Do not mutate them during construction.
	APM *rtc.APMConfig
	// DisableAPM skips native APM filter configuration and permits VAD-only
	// processing. It requires VAD and cannot be combined with an explicit APM.
	DisableAPM bool
	// VAD optionally enables the SDK VAD v2 segmenter on processed frames.
	// Nil disables segmentation. The value is copied and validated; this is not
	// the standalone native rtc.AudioVad detector.
	VAD *rtc.AudioVadConfigV2
}

type nativeAudioProcessor interface {
	PushAudioPcmData([]byte, int, int, int64) int
	Release()
}

// ProcessorStats is native submission/callback-duration telemetry, in milliseconds.
// Submitted duration includes native failures. Processed duration counts 10 ms
// per callback; neither value acknowledges playback or guarantees completion.
type ProcessorStats struct {
	SubmittedMillis, ProcessedMillis int64
}

// AudioProcessor owns a native external PCM processor and a copied output queue.
// Construct it with Service.NewAudioProcessor; the zero value is not usable.
// SendPCM and Close serialize access to the processor. It does not publish
// a channel track automatically, so applications must forward output explicitly.
type AudioProcessor struct {
	mu             sync.Mutex
	native         nativeAudioProcessor
	service        *Service
	output         *Stream[AudioFrame]
	rate, channels int
	stats          func() ProcessorStats
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
	if cfg.Input.SampleRate == 0 {
		cfg.Input.SampleRate = cfg.SampleRate
	}
	if cfg.Input.Channels == 0 {
		cfg.Input.Channels = cfg.Channels
	}
	if err := validateProcessorInputFormat(cfg.Input); err != nil {
		return cfg, err
	}
	if cfg.QueueCapacity < 1 || cfg.QueueCapacity > 65536 {
		return cfg, invalid("processor queue capacity", "must be 1–65536")
	}
	if cfg.VAD != nil {
		if err := validateTelemetryVADConfig(*cfg.VAD); err != nil {
			return cfg, err
		}
		v := *cfg.VAD
		cfg.VAD = &v
	}
	if cfg.DisableAPM {
		if cfg.APM != nil || cfg.VAD == nil {
			return cfg, invalid("APM configuration", "DisableAPM requires VAD and no explicit APM")
		}
		return cfg, nil
	}
	if cfg.APM == nil {
		cfg.APM = rtc.NewAPMConfig()
	}
	if cfg.APM.AiNsConfig == nil || cfg.APM.AiAecConfig == nil || cfg.APM.BghvsCConfig == nil || cfg.APM.AgcConfig == nil {
		return cfg, invalid("APM configuration", "requires all component configurations")
	}
	apm := *cfg.APM
	ns, aec, bghvs, agc := *apm.AiNsConfig, *apm.AiAecConfig, *apm.BghvsCConfig, *apm.AgcConfig
	apm.AiNsConfig, apm.AiAecConfig, apm.BghvsCConfig, apm.AgcConfig = &ns, &aec, &bghvs, &agc
	cfg.APM = &apm
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
	p := &AudioProcessor{service: s, output: newStream[AudioFrame](cfg.QueueCapacity), rate: cfg.Input.SampleRate, channels: cfg.Input.Channels}
	native := rtc.NewExternalAudioProcessor()
	if native == nil {
		p.output.close()
		return nil, &Error{Op: "create audio processor", Code: -1}
	}
	p.native = native
	p.stats = func() ProcessorStats {
		return ProcessorStats{native.InStreamInMs.Load(), native.ProcessedStreamInMs.Load()}
	}
	if err := check("initialize audio processor", native.Initialize(cfg.APM, cfg.SampleRate, cfg.Channels, cfg.VAD, p.observer())); err != nil {
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

// Stats returns a concurrency-safe native telemetry snapshot, including after Close.
func (p *AudioProcessor) Stats() ProcessorStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stats == nil {
		return ProcessorStats{}
	}
	return p.stats()
}

// SendPCM submits PCM16 little-endian input in ProcessorConfig.Input format with
// presentation value zero. Input must contain complete nonempty 10 ms frames.
// Calls are unpaced and do not wait for processed output. Keep data unchanged
// until return; consume Output separately. Errors wrap ErrInvalidConfig, are
// ErrClosed after Close, or preserve native submission failure in *Error.
func (p *AudioProcessor) SendPCM(data []byte) error { return p.SendPCMAt(data, 0) }

// SendPCMAt forwards the signed presentation value unchanged, including negative
// values and opaque tags. It otherwise has SendPCM's input and lifetime contract.
func (p *AudioProcessor) SendPCMAt(data []byte, presentationMillis int64) error {
	return p.SendPCMFormat(data, PCMFormat{SampleRate: p.rate, Channels: p.channels}, presentationMillis)
}

// SendPCMFormat submits input with a per-call format, independently of requested
// output format. Zero format fields use 16000 Hz and mono. Input rates may be
// 8000–48000 Hz divisible by 100, and channels must be 1 or 2. Native processing
// performs conversion. The PCM and presentation contracts are those of SendPCMAt.
func (p *AudioProcessor) SendPCMFormat(data []byte, format PCMFormat, presentationMillis int64) error {
	if format.SampleRate == 0 {
		format.SampleRate = 16000
	}
	if format.Channels == 0 {
		format.Channels = 1
	}
	if err := validateProcessorInput(data, format); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.native == nil {
		return ErrClosed
	}
	return check("process PCM", p.native.PushAudioPcmData(data, format.SampleRate, format.Channels, presentationMillis))
}

// SendPCMFrame forwards Data, format, and PresentationMillis from a complete
// PCM16 frame. BytesPerSample must be 2 and SamplesPerChannel must match Data;
// sample rate and channels must be explicit. TimestampMillis and telemetry are
// not submission options in the native external processor and are ignored.
func (p *AudioProcessor) SendPCMFrame(frame AudioFrame) error {
	format := PCMFormat{SampleRate: frame.SampleRate, Channels: frame.Channels}
	if err := validateProcessorInput(frame.Data, format); err != nil {
		return err
	}
	if frame.BytesPerSample != 2 || frame.SamplesPerChannel != len(frame.Data)/(2*frame.Channels) {
		return invalid("processor PCM frame", "requires PCM16 with matching sample count")
	}
	return p.SendPCMFormat(frame.Data, format, frame.PresentationMillis)
}

func validateProcessorInputFormat(format PCMFormat) error {
	if format.SampleRate < 8000 || format.SampleRate > 48000 || format.SampleRate%100 != 0 {
		return invalid("processor input sample rate", "must be 8000–48000 Hz divisible by 100")
	}
	if format.Channels != 1 && format.Channels != 2 {
		return invalid("processor input channels", "must be 1 or 2")
	}
	return nil
}

func validateProcessorInput(data []byte, format PCMFormat) error {
	if err := validateProcessorInputFormat(format); err != nil {
		return err
	}
	if len(data) == 0 || len(data)%(format.SampleRate/100*format.Channels*2) != 0 || uint64(len(data)/(2*format.Channels)) > math.MaxUint32 {
		return invalid("processor PCM data", "requires complete nonempty 10 ms PCM16 frames and at most uint32 samples per channel")
	}
	return nil
}

// Close releases native processing resources and closes Output.
// It is idempotent and safe for concurrent calls. Buffered output remains readable
// with a live context until io.EOF. Pending input is not guaranteed to finish
// processing. Close currently returns nil; it is not a completion acknowledgement.
func (p *AudioProcessor) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		if p.native != nil {
			p.native.Release()
			p.native = nil
		}
		p.mu.Unlock()
		if p.output != nil {
			p.output.close()
		}
		if p.service != nil {
			p.service.mu.Lock()
			delete(p.service.processors, p)
			p.service.mu.Unlock()
		}
	})
	return nil
}
