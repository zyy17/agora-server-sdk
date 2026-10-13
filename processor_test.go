package agorasdk

import (
	"context"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	if err != nil || cfg.SampleRate != 48000 || cfg.Channels != 2 || cfg.QueueCapacity != 8 || cfg.APM == apm || !reflect.DeepEqual(cfg.APM, apm) {
		t.Fatalf("explicit configuration changed: %+v, %v", cfg, err)
	}
	apm.AiNsConfig.AiNSModelPref = 999
	apm.AiAecConfig.SplitSrateFor48k = 999
	apm.BghvsCConfig.VadThr = 0.1
	apm.AgcConfig.Enabled = true
	apm.EnableDump = true
	if cfg.APM.AiNsConfig.AiNSModelPref == 999 || cfg.APM.AiAecConfig.SplitSrateFor48k == 999 || cfg.APM.BghvsCConfig.VadThr == 0.1 || cfg.APM.AgcConfig.Enabled || cfg.APM.EnableDump {
		t.Fatal("retained caller APM components")
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

func TestProcessorVADOnlyAndInputFormat(t *testing.T) {
	cfg, err := normalizedProcessorConfig(ProcessorConfig{SampleRate: 48000, Channels: 2, Input: PCMFormat{SampleRate: 24000}, DisableAPM: true, VAD: &rtc.AudioVadConfigV2{}})
	if err != nil || cfg.APM != nil || cfg.Input != (PCMFormat{SampleRate: 24000, Channels: 2}) {
		t.Fatal(cfg, err)
	}
	for _, cfg := range []ProcessorConfig{
		{DisableAPM: true},
		{DisableAPM: true, APM: rtc.NewAPMConfig(), VAD: &rtc.AudioVadConfigV2{}},
		{Input: PCMFormat{SampleRate: 22050}},
		{Input: PCMFormat{Channels: 3}},
		{VAD: &rtc.AudioVadConfigV2{ActivePercent: float32(math.NaN())}},
	} {
		if _, err := normalizedProcessorConfig(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(cfg, err)
		}
	}
}

type fakeAudioProcessor struct {
	push    func([]byte, int, int, int64) int
	release func()
}

func (f *fakeAudioProcessor) PushAudioPcmData(data []byte, rate, channels int, pts int64) int {
	return f.push(data, rate, channels, pts)
}

func (f *fakeAudioProcessor) Release() {
	if f.release != nil {
		f.release()
	}
}

func TestProcessorForwardsInputAndSignedPresentation(t *testing.T) {
	var calls int
	p := &AudioProcessor{rate: 24000, channels: 2}
	defer p.Close()
	p.native = &fakeAudioProcessor{push: func(data []byte, rate, channels int, pts int64) int {
		calls++
		if rate != 24000 || channels != 2 || len(data) != 960 || pts != -19 {
			t.Fatal(rate, channels, len(data), pts)
		}
		return -7
	}}
	frame := AudioFrame{Data: make([]byte, 960), SampleRate: 24000, Channels: 2, SamplesPerChannel: 240, BytesPerSample: 2, PresentationMillis: -19, TimestampMillis: 123}
	for _, submit := range []func() error{
		func() error { return p.SendPCMAt(frame.Data, -19) },
		func() error { return p.SendPCMFormat(frame.Data, PCMFormat{24000, 2}, -19) },
		func() error { return p.SendPCMFrame(frame) },
	} {
		var native *Error
		if err := submit(); !errors.As(err, &native) || native.Code != -7 || native.Op != "process PCM" {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Fatal(calls)
	}
	frame.SamplesPerChannel--
	if err := p.SendPCMFrame(frame); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	for _, format := range []PCMFormat{{7999, 1}, {48001, 1}, {22050, 1}, {16000, -1}, {16000, 3}} {
		if err := p.SendPCMFormat(frame.Data, format, 0); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(format, err)
		}
	}
	if calls != 3 {
		t.Fatal("forwarded invalid input")
	}
}

func TestProcessorSendDefaultsAndClose(t *testing.T) {
	var calls, releases atomic.Int32
	p := &AudioProcessor{rate: 16000, channels: 1, output: newStream[AudioFrame](2), stats: func() ProcessorStats { return ProcessorStats{10, 10} }}
	p.native = &fakeAudioProcessor{push: func(data []byte, rate, channels int, pts int64) int {
		calls.Add(1)
		if len(data) != 320 || rate != 16000 || channels != 1 || pts != 0 {
			t.Error(rate, channels, pts)
		}
		return 0
	}, release: func() { releases.Add(1) }}
	if err := p.SendPCM(make([]byte, 320)); err != nil {
		t.Fatal(err)
	}
	if err := p.SendPCMFormat(make([]byte, 320), PCMFormat{}, 0); err != nil {
		t.Fatal(err)
	}
	observer := p.observer()
	observer.OnProcessedAudioFrame(nil, standaloneVADFrame(make([]byte, 320), 1), rtc.VadStateInvalid, nil)
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() { _ = p.Close() })
	}
	wg.Wait()
	if err := p.SendPCM(make([]byte, 320)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if calls.Load() != 2 || releases.Load() != 1 {
		t.Fatal(calls.Load(), releases.Load())
	}
	if p.Stats() != (ProcessorStats{10, 10}) {
		t.Fatal(p.Stats())
	}
	if _, err := p.Output().Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	observer.OnProcessedAudioFrame(nil, standaloneVADFrame(make([]byte, 320), 1), rtc.VadStateInvalid, nil)
	if _, err := p.Output().Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func TestProcessorCloseWaitsForSend(t *testing.T) {
	entered, finish, released := make(chan struct{}), make(chan struct{}), make(chan struct{})
	p := &AudioProcessor{rate: 16000, channels: 1, native: &fakeAudioProcessor{push: func([]byte, int, int, int64) int {
		close(entered)
		<-finish
		return 0
	}, release: func() { close(released) }}}
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := p.SendPCM(make([]byte, 320)); err != nil {
			t.Error(err)
		}
	})
	<-entered
	wg.Go(func() { _ = p.Close() })
	select {
	case <-released:
		t.Fatal("release raced an active sender")
	case <-time.After(10 * time.Millisecond):
	}
	close(finish)
	wg.Wait()
	<-released
}

func TestNativeProcessorVADOnlyLifecycle(t *testing.T) {
	t.Chdir(t.TempDir())
	service, err := NewService(Config{AppID: strings.Repeat("0", 32), AudioProcessing: true})
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	processor, err := service.NewAudioProcessor(ProcessorConfig{DisableAPM: true, VAD: &rtc.AudioVadConfigV2{StartRecognizeCount: 1, StopRecognizeCount: 1}})
	if err != nil {
		t.Fatal(err)
	}
	defer processor.Close()
	// A native sink works without loading optional APM extensions or joining a channel.
	if err := processor.SendPCMAt(make([]byte, 320), -42); err != nil {
		t.Fatal(err)
	}
	if err := processor.SendPCMFormat(make([]byte, 19200), PCMFormat{SampleRate: 48000, Channels: 2}, -43); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	frame, err := processor.Output().Next(ctx)
	if err != nil {
		t.Fatal("native processor produced no output", err)
	}
	if frame.SampleRate != 16000 || frame.Channels != 1 || frame.BytesPerSample != 2 || len(frame.Data) == 0 {
		t.Fatal("unexpected processed format", frame)
	}
	if got := processor.Stats(); got.SubmittedMillis != 110 || got.ProcessedMillis < 10 {
		t.Fatal(got)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if err := processor.SendPCM(make([]byte, 320)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
