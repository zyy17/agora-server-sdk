package agorasdk

import (
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"slices"
	"sync"
)

// ProcessorConfig configures native external audio processing. The installed SDK
// must contain the requested APM extensions; missing extensions return native errors.
type ProcessorConfig struct {
	SampleRate, Channels int
	QueueCapacity        int
	APM                  *rtc.APMConfig
	VAD                  *rtc.AudioVadConfigV2
}

// AudioProcessor owns an external native PCM processor and a copied output stream.
type AudioProcessor struct {
	mu             sync.Mutex
	native         *rtc.ExternalAudioProcessor
	service        *Service
	output         *Stream[AudioFrame]
	rate, channels int
	closed         bool
	closeOnce      sync.Once
}

func (s *Service) NewAudioProcessor(cfg ProcessorConfig) (*AudioProcessor, error) {
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
		return nil, err
	}
	if cfg.QueueCapacity < 1 || cfg.QueueCapacity > 65536 {
		return nil, invalid("processor queue capacity", "must be 1–65536")
	}
	if cfg.VAD != nil {
		if err := validateVAD(*cfg.VAD); err != nil {
			return nil, err
		}
		v := *cfg.VAD
		cfg.VAD = &v
	}
	if cfg.APM == nil {
		cfg.APM = rtc.NewAPMConfig()
	}
	if cfg.APM.AiNsConfig == nil || cfg.APM.AiAecConfig == nil || cfg.APM.BghvsCConfig == nil || cfg.APM.AgcConfig == nil {
		return nil, invalid("APM configuration", "requires all component configurations")
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
	observer := &rtc.ExternalAudioProcessorObserver{OnProcessedAudioFrame: func(_ *rtc.ExternalAudioProcessor, f *rtc.AudioFrame, v rtc.VadState, speech *rtc.AudioFrame) {
		if f == nil {
			return
		}
		out := AudioFrame{Data: slices.Clone(f.Buffer), SampleRate: f.SamplesPerSec, Channels: f.Channels, TimestampMillis: f.RenderTimeMs, PresentationMillis: f.PresentTimeMs, VAD: v, RMS: f.Rms, VoiceProbability: f.VoiceProb}
		if speech != nil {
			out.Speech = slices.Clone(speech.Buffer)
		}
		p.output.push(out)
	}}
	if err := check("initialize audio processor", p.native.Initialize(cfg.APM, cfg.SampleRate, cfg.Channels, cfg.VAD, observer)); err != nil {
		p.native.Release()
		p.output.close()
		return nil, err
	}
	s.processors[p] = struct{}{}
	return p, nil
}
func (p *AudioProcessor) Output() *Stream[AudioFrame] { return p.output }
func (p *AudioProcessor) SendPCM(data []byte) error {
	if err := validatePCM(data, p.rate, p.channels); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	return check("process PCM", p.native.PushAudioPcmData(data, p.rate, p.channels, 0))
}
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
