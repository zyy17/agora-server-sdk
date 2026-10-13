package agoraservice

import (
	"sync"
	"testing"
	"time"
)

func TestCallbackGateDrainsAndReopens(t *testing.T) {
	var gate callbackGate
	if !gate.begin() {
		t.Fatal("new gate rejected callback")
	}
	paused := make(chan struct{})
	go func() { gate.pause(); close(paused) }()
	deadline := time.After(time.Second)
	for {
		gate.mu.Lock()
		stopping := gate.paused > 0
		gate.mu.Unlock()
		if stopping {
			break
		}
		select {
		case <-deadline:
			t.Fatal("pause did not start")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if gate.begin() {
		gate.end()
		t.Fatal("paused gate admitted callback")
	}
	select {
	case <-paused:
		t.Fatal("pause returned before active callback finished")
	default:
	}
	gate.end()
	select {
	case <-paused:
	case <-time.After(time.Second):
		t.Fatal("pause did not drain")
	}
	gate.resume()
	if !gate.begin() {
		t.Fatal("resumed gate rejected callback")
	}
	gate.end()
}

func TestCallbackGateConcurrentCallbacks(t *testing.T) {
	var gate callbackGate
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				if gate.begin() {
					gate.end()
				}
			}
		}()
	}
	gate.pause()
	wg.Wait()
	gate.pause()
	gate.resume()
	if gate.begin() {
		gate.end()
		t.Fatal("nested pause reopened too soon")
	}
	gate.resume()
}

func TestInvalidInputsDoNotEnterNativeSDK(t *testing.T) {
	if Initialize(nil) == 0 {
		t.Fatal("nil service config accepted")
	}
	if NewRtcConnection(nil, nil) != nil {
		t.Fatal("nil connection config accepted")
	}
	if got := (*VideoFrameSender)(nil).SendVideoFrame(nil); got == 0 {
		t.Fatal("nil video frame accepted")
	}
	if got := (*VideoEncodedImageSender)(nil).SendEncodedVideoImage(nil, nil); got == 0 {
		t.Fatal("nil encoded video accepted")
	}
	if got := (*AudioPcmDataSender)(nil).SendAudioPcmData(nil); got == 0 {
		t.Fatal("nil PCM accepted")
	}
	if got := (*AudioEncodedFrameSender)(nil).SendEncodedAudioFrame(nil, nil); got == 0 {
		t.Fatal("nil encoded audio accepted")
	}
	ptr, pin := unsafeCBytes(nil)
	defer pin.Unpin()
	if ptr != nil {
		t.Fatal("empty buffer had a pointer")
	}
}

func TestPCMConsumptionAt44100Hz(t *testing.T) {
	var stats PcmConsumeStats
	stats.addPcmData(441*2*2, 44100, 2)
	if stats.duration != 10 {
		t.Fatalf("duration = %d, want 10ms", stats.duration)
	}
}

func TestVadManagerSeparatesChannelUserPairs(t *testing.T) {
	manager := NewAudioVadManager(nil)
	defer manager.Release()
	if manager.makeKey("ab", "c") == manager.makeKey("a", "bc") {
		t.Fatal("channel/user keys collide")
	}
	manager.Release()
	if frame, state := manager.Process("channel", "user", &AudioFrame{}); frame != nil || state != VadStateInvalid {
		t.Fatal("released VAD processed a frame")
	}
}

func TestPrecisionTimerStopDrainsTask(t *testing.T) {
	timer := NewPrecisionTimer(1)
	running := make(chan struct{})
	finish := make(chan struct{})
	var started sync.Once
	timer.Start(func() { started.Do(func() { close(running) }); <-finish })
	<-running
	stopped := make(chan struct{})
	go func() { timer.Stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("stop returned before task finished")
	case <-time.After(time.Millisecond):
	}
	close(finish)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop did not finish")
	}
	timer.Stop()
}
