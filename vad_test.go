package agorasdk

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

func TestVADConfigCopiesAndRejectsInvalid(t *testing.T) {
	cfg := VADConfig{StartRecognizeCount: 2, StopRecognizeCount: 3, PreStartRecognizeCount: 1, RmsThr: -40, VoiceProb: 0.7, Aggressive: 2}
	copied, err := copiedVADConfig(&cfg)
	if err != nil || copied == &cfg || *copied != cfg {
		t.Fatal(copied, err)
	}
	cfg.StartRecognizeCount++
	if copied.StartRecognizeCount != 2 {
		t.Fatal("aliased configuration")
	}
	if copied, err := copiedVADConfig(nil); copied != nil || err != nil {
		t.Fatal(copied, err)
	}
	for _, cfg := range []VADConfig{
		{StartRecognizeCount: -1}, {StopRecognizeCount: 10001}, {PreStartRecognizeCount: -1},
		{ActivePercent: float32(math.NaN())}, {InactivePercent: float32(math.Inf(1))},
		{VoiceProb: -0.1}, {JointThr: 1.1}, {RmsThr: 1}, {RmsThr: -101},
		{RmsThr: float32(math.NaN())}, {Aggressive: -1}, {Aggressive: 4}, {Aggressive: float32(math.NaN())},
	} {
		if _, err := NewAudioVAD(&cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(cfg, err)
		}
		if _, err := NewStereoVAD(nil, &cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(cfg, err)
		}
	}
}

type fakeMonoVAD struct {
	process  func(*rtc.AudioFrame) (*rtc.AudioFrame, int)
	releases atomic.Int32
}

func (v *fakeMonoVAD) ProcessPcmFrame(frame *rtc.AudioFrame) (*rtc.AudioFrame, int) {
	return v.process(frame)
}

func (v *fakeMonoVAD) Release() { v.releases.Add(1) }

func TestMonoVADCopiesOutputAndPreservesErrors(t *testing.T) {
	out := standaloneVADFrame([]byte{1, 2}, 1)
	out.RenderTimeMs = 90
	code := int(VADSpeechStart)
	calls := 0
	native := &fakeMonoVAD{process: func(f *rtc.AudioFrame) (*rtc.AudioFrame, int) {
		calls++
		if f.Type != rtc.AudioFrameTypePCM16 || f.SamplesPerChannel != 160 || f.BytesPerSample != 2 || f.Channels != 1 || f.SamplesPerSec != 16000 || len(f.Buffer) != 320 {
			t.Fatal(f)
		}
		return out, code
	}}
	v := &AudioVAD{native: native}
	result, err := v.ProcessPCM(make([]byte, 320))
	if err != nil || result.State != VADSpeechStart || result.Speech.TimestampMillis != 90 || result.Speech.VAD != VADSpeechStart {
		t.Fatal(result, err)
	}
	out.Buffer[0] = 9
	if result.Speech.Data[0] != 1 {
		t.Fatal("aliased raw output")
	}
	code = -2002
	var nativeErr *Error
	if result, err := v.ProcessPCM(make([]byte, 320)); !errors.As(err, &nativeErr) || nativeErr.Code != -2002 || result.State != VADInvalid {
		t.Fatal(result, err)
	}
	for _, data := range [][]byte{nil, make([]byte, 319), make([]byte, 640)} {
		if _, err := v.ProcessPCM(data); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("forwarded invalid frame")
	}
	_ = v.Close()
	_ = v.Close()
	if native.releases.Load() != 1 {
		t.Fatal(native.releases.Load())
	}
	if _, err := v.ProcessPCM(make([]byte, 320)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

type fakeStereoVAD struct {
	process  func(*rtc.AudioFrame) (*rtc.AudioFrame, int, *rtc.AudioFrame, int)
	releases atomic.Int32
}

func (v *fakeStereoVAD) ProcessAudioFrame(f *rtc.AudioFrame) (*rtc.AudioFrame, int, *rtc.AudioFrame, int) {
	return v.process(f)
}

func (v *fakeStereoVAD) Release() { v.releases.Add(1) }

func TestStereoVADPreservesPartialResultsAndErrors(t *testing.T) {
	leftCode, rightCode := -7, int(VADSpeaking)
	native := &fakeStereoVAD{process: func(f *rtc.AudioFrame) (*rtc.AudioFrame, int, *rtc.AudioFrame, int) {
		if f.Channels != 2 || len(f.Buffer) != 640 {
			t.Fatal(f)
		}
		return nil, leftCode, standaloneVADFrame([]byte{3, 4}, 1), rightCode
	}}
	v := &StereoVAD{native: native}
	defer v.Close()
	result, err := v.ProcessPCM(make([]byte, 640))
	var nativeErr *Error
	if !errors.As(err, &nativeErr) || nativeErr.Code != -7 || nativeErr.Op != "process left VAD" || result.Left.State != VADInvalid || result.Right.State != VADSpeaking || !reflect.DeepEqual(result.Right.Speech.Data, []byte{3, 4}) {
		t.Fatal(result, err)
	}
	rightCode = -8
	result, err = v.ProcessPCM(make([]byte, 640))
	if err == nil || !strings.Contains(err.Error(), "left VAD") || !strings.Contains(err.Error(), "right VAD") || result.Right.State != VADInvalid {
		t.Fatal(result, err)
	}
	leftCode = int(VADSilent)
	if _, err := v.ProcessPCM(make([]byte, 320)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	_ = v.Close()
	if _, err := v.ProcessPCM(make([]byte, 640)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if native.releases.Load() != 1 {
		t.Fatal(native.releases.Load())
	}
}

func TestVADNilAndZeroValueClose(t *testing.T) {
	var mono *AudioVAD
	var stereo *StereoVAD
	var telemetry *TelemetryVAD
	for _, close := range []func() error{mono.Close, stereo.Close, telemetry.Close, new(AudioVAD).Close, new(StereoVAD).Close, new(TelemetryVAD).Close} {
		if err := close(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := mono.ProcessPCM(make([]byte, 320)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if _, err := stereo.ProcessPCM(make([]byte, 640)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if _, err := telemetry.Process(telemetryTestFrame()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func telemetryTestFrame() AudioFrame {
	return AudioFrame{Data: make([]byte, 320), SampleRate: 16000, Channels: 1, BytesPerSample: 2, SamplesPerChannel: 160, VoiceProbability: 1, RMS: 100, TimestampMillis: 17, PresentationMillis: -4, Position: AudioBeforeMixing, Channel: "channel", UserID: "speaker"}
}

func TestTelemetryVADOwnsRetainedFramesAndStates(t *testing.T) {
	cfg := TelemetryVADConfig{StartRecognizeCount: 2, StopRecognizeCount: 1, StartRms: -70}
	v, err := NewTelemetryVAD(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	cfg.StartRecognizeCount = 99
	frame := telemetryTestFrame()
	frame.Data[0] = 1
	if result, err := v.Process(frame); err != nil || result.State != VADSilent || result.Speech != nil {
		t.Fatal(result, err)
	}
	frame.Data[0] = 2
	result, err := v.Process(frame)
	if err != nil || result.State != VADSpeechStart || len(result.Speech.Data) != 640 || result.Speech.Data[0] != 1 || result.Speech.Data[320] != 2 || result.Speech.SamplesPerChannel != 320 || result.Speech.PresentationMillis != -4 || result.Speech.UserID != "speaker" {
		t.Fatal(result, err)
	}
	frame.Data[0] = 3
	if result.Speech.Data[320] != 2 {
		t.Fatal("retained caller memory")
	}
	result, err = v.Process(frame)
	if err != nil || result.State != VADSpeaking {
		t.Fatal(result, err)
	}
	result.Speech.Data[0] = 42
	if frame.Data[0] != 3 {
		t.Fatal("aliased during-speech frame")
	}
	frame.VoiceProbability, frame.RMS = 0, 0
	result, err = v.Process(frame)
	if err != nil || result.State != VADSpeechEnd {
		t.Fatal(result, err)
	}
}

func TestTelemetryVADValidationAndClose(t *testing.T) {
	for _, cfg := range []TelemetryVADConfig{{StartRecognizeCount: -1}, {ActivePercent: float32(math.NaN())}, {InactivePercent: float32(math.NaN())}, {AdaptiveRmsThresholdFactor: float32(math.NaN())}} {
		if _, err := NewTelemetryVAD(&cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(cfg, err)
		}
	}
	v, err := NewTelemetryVAD(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	frame := telemetryTestFrame()
	if _, err := v.Process(frame); err != nil {
		t.Fatal(err)
	}
	invalidFrames := []AudioFrame{frame, frame, frame, frame}
	invalidFrames[0].BytesPerSample = 1
	invalidFrames[1].SamplesPerChannel = 159
	invalidFrames[2].Data = nil
	invalidFrames[3].SampleRate = 32000
	for _, f := range invalidFrames {
		if _, err := v.Process(f); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(f, err)
		}
	}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			for range 10 {
				if _, err := v.Process(frame); err != nil && !errors.Is(err, ErrClosed) {
					t.Error(err)
				}
			}
			_ = v.Close()
		})
	}
	wg.Wait()
	if _, err := v.Process(frame); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func TestNativeOwnedVADLifecycle(t *testing.T) {
	mono, err := NewAudioVAD(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mono.Close()
	stereo, err := NewStereoVAD(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer stereo.Close()
	if result, err := mono.ProcessPCM(make([]byte, 320)); err != nil || result.State < 0 {
		t.Fatal(result, err)
	}
	if result, err := stereo.ProcessPCM(make([]byte, 640)); err != nil || result.Left.State < 0 || result.Right.State < 0 {
		t.Fatal(result, err)
	}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			for range 10 {
				if _, err := mono.ProcessPCM(make([]byte, 320)); err != nil && !errors.Is(err, ErrClosed) {
					t.Error(err)
				}
				if _, err := stereo.ProcessPCM(make([]byte, 640)); err != nil && !errors.Is(err, ErrClosed) {
					t.Error(err)
				}
			}
			_ = mono.Close()
			_ = stereo.Close()
		})
	}
	wg.Wait()
	if _, err := mono.ProcessPCM(make([]byte, 320)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if _, err := stereo.ProcessPCM(make([]byte, 640)); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func TestStandaloneVADFrameValidation(t *testing.T) {
	mono := &AudioVAD{}
	stereo := &StereoVAD{}
	frame := telemetryTestFrame()
	if _, err := mono.Process(frame); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	for _, mutate := range []func(*AudioFrame){
		func(f *AudioFrame) { f.SampleRate = 8000 },
		func(f *AudioFrame) { f.Channels = 2 },
		func(f *AudioFrame) { f.BytesPerSample = 1 },
		func(f *AudioFrame) { f.SamplesPerChannel = 80 },
		func(f *AudioFrame) { f.Data = f.Data[:319] },
	} {
		invalidFrame := frame
		mutate(&invalidFrame)
		if _, err := mono.Process(invalidFrame); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(invalidFrame, err)
		}
	}
	frame.Channels, frame.Data = 2, make([]byte, 640)
	if _, err := stereo.Process(frame); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	frame.SampleRate = 48000
	if _, err := stereo.Process(frame); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
}
