package agorasdk_test

import (
	"fmt"

	agorasdk "github.com/zyy17/agora-server-sdk"
)

// This compile-only example uses the installed native VAD extension, with no
// Service, App ID, or channel connection.
func ExampleNewAudioVAD() {
	detector, err := agorasdk.NewAudioVAD(nil)
	if err != nil {
		return
	}
	defer detector.Close()
	// Replace silence with a single 10 ms, 16 kHz mono PCM16 frame.
	result, err := detector.ProcessPCM(make([]byte, 320))
	if err != nil {
		return
	}
	if result.Speech != nil {
		fmt.Println(result.State, len(result.Speech.Data))
	}
}

// Left and right may have independent thresholds and independently selected speech.
func ExampleNewStereoVAD() {
	detector, err := agorasdk.NewStereoVAD(nil, nil)
	if err != nil {
		return
	}
	defer detector.Close()
	result, err := detector.ProcessPCM(make([]byte, 640))
	if err != nil {
		return
	}
	fmt.Println(result.Left.State, result.Right.State)
}

// Native telemetry is required; PCM bytes alone do not drive this segmenter.
func ExampleNewTelemetryVAD() {
	detector, err := agorasdk.NewTelemetryVAD(nil)
	if err != nil {
		return
	}
	defer detector.Close()
	consume := func(frame agorasdk.AudioFrame) error {
		result, err := detector.Process(frame)
		if err != nil {
			return err
		}
		if result.Speech != nil {
			fmt.Println(result.State, len(result.Speech.Data))
		}
		return nil
	}
	_ = consume // Call for each telemetry-bearing Session.Audio or processor frame.
}

// The application owns the service, which must enable Config.AudioProcessing.
// This compile-only example creates no service or channel connection.
func ExampleAudioProcessor_SendPCMFormat() {
	process := func(service *agorasdk.Service) error {
		processor, err := service.NewAudioProcessor(agorasdk.ProcessorConfig{
			SampleRate: 16000, Channels: 1,
			Input:      agorasdk.PCMFormat{SampleRate: 48000, Channels: 2},
			DisableAPM: true, VAD: &agorasdk.TelemetryVADConfig{},
		})
		if err != nil {
			return err
		}
		defer processor.Close()
		// Consume processor.Output concurrently in the application. Presentation
		// values are signed tags; this overrides the default input format per call.
		return processor.SendPCMFormat(make([]byte, 480), agorasdk.PCMFormat{SampleRate: 24000, Channels: 1}, -10)
	}
	_ = process
}
