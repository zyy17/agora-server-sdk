package agoraservice

import (
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"sync"
	"testing"
)

func TestStandaloneVADRejectsInvalidInputs(t *testing.T) {
	var mono *AudioVad
	var stereo *SteroAudioVad
	mono.Release()
	stereo.Release()
	if _, code := mono.ProcessPcmFrame(nil); code >= 0 {
		t.Fatal(code)
	}
	if _, l, _, r := stereo.ProcessAudioFrame(nil); l >= 0 || r >= 0 {
		t.Fatal(l, r)
	}
	for _, c := range []AudioVadConfig{{StartRecognizeCount: -1}, {VoiceProb: float32(math.NaN())}, {RmsThr: 1}, {Aggressive: 4}} {
		if NewAudioVad(&c) != nil {
			t.Fatal("accepted invalid settings")
		}
	}
	frame := &AudioFrame{Type: AudioFrameTypePCM16, SamplesPerChannel: 160, BytesPerSample: 2, Channels: 1, SamplesPerSec: 16000, Buffer: make([]byte, 320)}
	if !validVADFrame(frame, 1) {
		t.Fatal("valid frame rejected")
	}
	frame.SamplesPerChannel = 159
	if validVADFrame(frame, 1) {
		t.Fatal("inconsistent count accepted")
	}
	frame.SamplesPerChannel = 160
	frame.Buffer = make([]byte, 640)
	frame.SamplesPerChannel = 320
	if validVADFrame(frame, 1) {
		t.Fatal("multiple frames accepted")
	}
	frame.SamplesPerChannel = 160
	frame.Buffer = frame.Buffer[:319]
	if validVADFrame(frame, 1) {
		t.Fatal("partial sample accepted")
	}
}

func TestStereoVADSplitPreservesMetadata(t *testing.T) {
	frame := &AudioFrame{Type: AudioFrameTypePCM16, SamplesPerChannel: 2, BytesPerSample: 2, Channels: 2, SamplesPerSec: 16000, Buffer: []byte{1, 2, 3, 4, 5, 6, 7, 8}, RenderTimeMs: 12, PresentTimeMs: 34}
	left, right := splitStereoVADFrame(frame)
	if string(left.Buffer) != string([]byte{1, 2, 5, 6}) || string(right.Buffer) != string([]byte{3, 4, 7, 8}) || left.SamplesPerChannel != 2 || right.PresentTimeMs != 34 || left.RenderTimeMs != 12 || left.Channels != 1 || right.Channels != 1 {
		t.Fatal(left, right)
	}
	frame.Buffer[0] = 9
	if left.Buffer[0] != 1 {
		t.Fatal("aliased input")
	}
}

func TestNativeStandaloneVADLifecycle(t *testing.T) {
	vad := NewSteroVad(nil, nil)
	if vad == nil {
		t.Fatal("native VAD unavailable")
	}
	frame := &AudioFrame{Type: AudioFrameTypePCM16, SamplesPerChannel: 160, BytesPerSample: 2, Channels: 2, SamplesPerSec: 16000, Buffer: make([]byte, 640)}
	_, left, _, right := vad.ProcessAudioFrame(frame)
	if left < 0 || right < 0 {
		t.Fatal(left, right)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				vad.ProcessAudioFrame(frame)
			}
		}()
	}
	vad.Release()
	wg.Wait()
	vad.Release()
	if _, l, _, r := vad.ProcessAudioFrame(frame); l >= 0 || r >= 0 {
		t.Fatal("released detector accepted input")
	}
}

// Run in a subprocess so a native regression cannot terminate the test runner.
func TestNativeVADExceptionBoundary(t *testing.T) {
	const child = "AGORA_VAD_EXCEPTION_TEST"
	if os.Getenv(child) != "1" {
		command := exec.Command(os.Args[0], "-test.run=^TestNativeVADExceptionBoundary$")
		command.Env = append(os.Environ(), child+"=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("native boundary subprocess failed: %v\n%s", err, output)
		}
		return
	}
	config := AudioVadConfig{StartRecognizeCount: 1, StopRecognizeCount: 10000, PreStartRecognizeCount: 0, ActivePercent: 0, InactivePercent: 0, VoiceProb: 0, RmsThr: -100, JointThr: 0, Aggressive: 2}
	vad := NewAudioVad(&config)
	if vad == nil {
		t.Fatal("constructor failed")
	}
	defer vad.Release()
	data := make([]byte, 320)
	for i := 0; i < 160; i++ {
		sample := int16(20000 * math.Sin(2*math.Pi*440*float64(i)/16000))
		binary.LittleEndian.PutUint16(data[i*2:], uint16(sample))
	}
	frame := &AudioFrame{Type: AudioFrameTypePCM16, SamplesPerChannel: 160, BytesPerSample: 2, Channels: 1, SamplesPerSec: 16000, Buffer: data}
	for i := 0; i < 64; i++ {
		_, state := vad.ProcessPcmFrame(frame)
		if state < 0 {
			if state != -2002 {
				t.Fatal("unexpected native error", state)
			}
			if _, next := vad.ProcessPcmFrame(frame); next != -1 {
				t.Fatal("failed detector reused", next)
			}
			return
		}
	}
	// A native version may repair this input; it must still never abort the process.
}
