package agorasdk

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestProcessorConfigDefaultsAndOwnership(t *testing.T) {
	vad := &rtc.AudioVadConfigV2{StartRecognizeCount: 7}
	cfg, err := normalizedProcessorConfig(ProcessorConfig{VAD: vad})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SampleRate != 16000 || cfg.Channels != 1 || cfg.QueueCapacity != 64 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.APM == nil || cfg.APM.AiNsConfig == nil || cfg.APM.AiAecConfig == nil || cfg.APM.BghvsCConfig == nil || cfg.APM.AgcConfig == nil {
		t.Fatal("missing default APM components")
	}
	vad.StartRecognizeCount = 99
	if cfg.VAD.StartRecognizeCount != 7 {
		t.Fatal("retained caller VAD configuration")
	}

	apm := rtc.NewAPMConfig()
	cfg, err = normalizedProcessorConfig(ProcessorConfig{SampleRate: 48000, Channels: 2, QueueCapacity: 8, APM: apm})
	if err != nil || cfg.SampleRate != 48000 || cfg.Channels != 2 || cfg.QueueCapacity != 8 || cfg.APM != apm {
		t.Fatalf("explicit configuration changed: %+v, %v", cfg, err)
	}
}

func TestProcessorConfigValidationPrecedesServiceState(t *testing.T) {
	service := &Service{closed: true}
	cases := []struct {
		name string
		cfg  ProcessorConfig
		want string
	}{
		{"format before queue", ProcessorConfig{SampleRate: -1, QueueCapacity: -1}, "sample rate"},
		{"queue before VAD", ProcessorConfig{QueueCapacity: -1, VAD: &rtc.AudioVadConfigV2{StartRecognizeCount: -1}}, "processor queue capacity"},
		{"VAD before APM", ProcessorConfig{VAD: &rtc.AudioVadConfigV2{StartRecognizeCount: -1}, APM: &rtc.APMConfig{}}, "VAD counts"},
		{"APM before closed", ProcessorConfig{APM: &rtc.APMConfig{}}, "APM configuration"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.NewAudioProcessor(tt.cfg)
			if !errors.Is(err, ErrInvalidConfig) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want invalid %s", err, tt.want)
			}
		})
	}
	if _, err := service.NewAudioProcessor(ProcessorConfig{}); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v, want closed", err)
	}
}

func TestProcessorObserverCopiesFrames(t *testing.T) {
	processor := &AudioProcessor{output: newStream[AudioFrame](1)}
	defer processor.output.close()
	observer := processor.observer()
	observer.OnProcessedAudioFrame(nil, nil, 0, nil)
	frame := &rtc.AudioFrame{Buffer: []byte{1, 2}, SamplesPerSec: 16000, Channels: 1, RenderTimeMs: 17}
	speech := &rtc.AudioFrame{Buffer: []byte{3, 4}}
	observer.OnProcessedAudioFrame(nil, frame, rtc.VadState(1), speech)
	frame.Buffer[0] = 9
	speech.Buffer[0] = 8
	got, err := processor.Output().Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Data, []byte{1, 2}) || !reflect.DeepEqual(got.Speech, []byte{3, 4}) || got.VAD != rtc.VadState(1) || got.TimestampMillis != 17 {
		t.Fatalf("unexpected copied frame: %+v", got)
	}
	observer.OnProcessedAudioFrame(nil, frame, 0, nil)
	got, err = processor.Output().Next(context.Background())
	if err != nil || got.Speech != nil {
		t.Fatalf("unexpected speech without a VAD frame: %+v, %v", got, err)
	}
}
