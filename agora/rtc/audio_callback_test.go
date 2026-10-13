//go:build agora_test

package agoraservice

import "testing"

func TestNativeCallbackWritablePCM(t *testing.T) {
	for _, position := range []int{1, 2, 4, 8, 16} {
		for _, mode := range []RawAudioFrameOpModeType{RawAudioFrameOpModeReadOnly, RawAudioFrameOpModeReadWrite} {
			f := newCallbackABIFixture()
			accept, malformed := true, false
			edit := func(frame *AudioFrame) bool {
				frame.Buffer[0] = 99
				if malformed {
					frame.Channels = 2
				}
				return accept
			}
			params := func(*LocalUser) AudioFrameObserverAudioParams { return AudioFrameObserverAudioParams{Mode: mode} }
			f.con.audioObserver = &AudioFrameObserver{
				OnRecordAudioFrame:               func(_ *LocalUser, _ string, frame *AudioFrame) bool { return edit(frame) },
				OnPlaybackAudioFrame:             func(_ *LocalUser, _ string, frame *AudioFrame) bool { return edit(frame) },
				OnMixedAudioFrame:                func(_ *LocalUser, _ string, frame *AudioFrame) bool { return edit(frame) },
				OnEarMonitoringAudioFrame:        func(_ *LocalUser, frame *AudioFrame) bool { return edit(frame) },
				OnPlaybackAudioFrameBeforeMixing: func(_ *LocalUser, _, _ string, frame *AudioFrame, _ VadState, _ *AudioFrame) bool { return edit(frame) },
				OnGetPlaybackAudioFrameParam:     params, OnGetRecordAudioFrameParam: params,
				OnGetMixedAudioFrameParam: params, OnGetEarMonitoringAudioFrameParam: params,
			}
			writable := mode == RawAudioFrameOpModeReadWrite && position != 8
			got, first := f.pcm(position)
			want := byte(1)
			if writable {
				want = 99
			}
			if !got || first != want {
				t.Fatalf("position=%d mode=%d accepted=%v first=%d", position, mode, got, first)
			}
			accept = false
			if got, first = f.pcm(position); got || first != 1 {
				t.Fatal("rejected frame changed native samples")
			}
			accept, malformed = true, true
			got, first = f.pcm(position)
			if first != 1 || writable && got {
				t.Fatal("malformed frame changed native samples")
			}
			f.close()
		}
	}
}
