//go:build agora_test

package agoraservice

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNativeCallbackStatisticsABI(t *testing.T) {
	for _, uid := range []string{"speaker-alpha", "4294967295", "0"} {
		t.Run(uid, func(t *testing.T) {
			f := newCallbackABIFixture()
			defer f.close()
			// Identity bookkeeping cannot depend on application subscription handlers.
			f.subscribe(uid)
			var gotLocalAudio *LocalAudioTrackStats
			var gotAudio *RemoteAudioTrackStats
			var gotLocalVideo *LocalVideoTrackStats
			var gotVideo *RemoteVideoTrackStats
			var audioUID, videoUID string
			f.con.localUserObserver = &LocalUserObserver{
				OnLocalAudioTrackStatistics:  func(_ *LocalUser, s *LocalAudioTrackStats) { gotLocalAudio = s },
				OnRemoteAudioTrackStatistics: func(_ *LocalUser, u string, s *RemoteAudioTrackStats) { audioUID = u; gotAudio = s },
				OnLocalVideoTrackStatistics:  func(_ *LocalUser, s *LocalVideoTrackStats) { gotLocalVideo = s },
				OnRemoteVideoTrackStatistics: func(_ *LocalUser, u string, s *RemoteVideoTrackStats) { videoUID = u; gotVideo = s },
			}
			f.stats(false)
			if audioUID != uid || videoUID != uid {
				t.Fatalf("public identities = %q/%q, want %q", audioUID, videoUID, uid)
			}
			wantLocalAudio := &LocalAudioTrackStats{NumChannels: 2, SentSampleRate: 48000, SentBitrate: 96, InternalCodec: 7, VoicePitch: 123.5}
			wantAudio := &RemoteAudioTrackStats{Uid: 999, Quality: 1, NetworkTransportDelay: 12, JitterBufferDelay: 13, AudioLossRate: 14, NumChannels: 2, ReceivedSampleRate: 48000, ReceivedBitrate: 96, TotalFrozenTime: 18, FrozenRate: 19, ReceivedBytes: 0x123456789ab}
			wantLocalVideo := &LocalVideoTrackStats{NumberOfStreams: 2, BytesMajorStream: 0x123456789ab, BytesMinorStream: 31, FramesEncoded: 32, SSRCMajorStream: 33, SSRCMinorStream: 34, CaptureFrameRate: 35, RegulatedCaptureFrameRate: 36, InputFrameRate: 37, EncodeFrameRate: 38, RenderFrameRate: 39, TargetMediaBitrateBps: 40, MediaBitrateBps: 41, TotalBitrateBps: 42, CaptureWidth: 43, CaptureHeight: 44, RegulatedCaptureWidth: 45, RegulatedCaptureHeight: 46, Width: 47, Height: 48, EncoderType: 49, UplinkCostTimeMs: 50, QualityAdaptIndication: 51}
			wantVideo := &RemoteVideoTrackStats{Uid: 888, Delay: 61, Width: 62, Height: 63, ReceivedBitrate: 64, DecoderOutputFrameRate: 65, RendererOutputFrameRate: 66, FrameLossRate: 67, PacketLossRate: 68, RxStreamType: 69, TotalFrozenTime: 70, FrozenRate: 71, TotalDecodedFrames: 72, AvSyncTimeMs: -73, DownlinkProcessTimeMs: 74, FrameRenderDelayMs: 75, TotalActiveTime: 0x123456789ab, PublishDuration: 0x123456789ac}
			for _, pair := range []struct {
				name      string
				got, want any
			}{{"local audio", gotLocalAudio, wantLocalAudio}, {"remote audio", gotAudio, wantAudio}, {"local video", gotLocalVideo, wantLocalVideo}, {"remote video", gotVideo, wantVideo}} {
				if !reflect.DeepEqual(pair.got, pair.want) {
					t.Errorf("%s stats = %+v, want %+v", pair.name, pair.got, pair.want)
				}
			}
			// Nil payloads must not be dereferenced or delivered.
			gotLocalAudio = nil
			gotAudio = nil
			gotLocalVideo = nil
			gotVideo = nil
			f.stats(true)
			if gotLocalAudio != nil || gotAudio != nil || gotLocalVideo != nil || gotVideo != nil {
				t.Fatal("nil stats delivered")
			}
		})
	}
}

func TestNativeCallbackTrackIdentityLifetime(t *testing.T) {
	f := newCallbackABIFixture()
	defer f.close()
	var identities []string
	f.con.localUserObserver = &LocalUserObserver{
		OnRemoteAudioTrackStatistics: func(_ *LocalUser, uid string, _ *RemoteAudioTrackStats) { identities = append(identities, uid) },
		OnRemoteVideoTrackStatistics: func(_ *LocalUser, uid string, _ *RemoteVideoTrackStats) { identities = append(identities, uid) },
	}
	f.stats(false)
	if len(identities) != 0 {
		t.Fatal("unknown opaque tracks were delivered")
	}
	for _, remove := range []struct {
		name string
		run  func()
	}{{"stopped", func() { f.state("speaker-alpha", 0) }}, {"left", func() { f.left("speaker-alpha") }}, {"disconnected", f.disconnect}, {"unsubscribed", func() {
		f.con.remoteTracks.remove(remoteAudioTrack, "speaker-alpha")
		f.con.remoteTracks.remove(remoteVideoTrack, "speaker-alpha")
	}}} {
		t.Run(remove.name, func(t *testing.T) {
			f.subscribe("speaker-alpha")
			f.stats(false)
			before := len(identities)
			remove.run()
			f.stats(false)
			if len(identities) != before {
				t.Fatal("retired track stats delivered")
			}
		})
	}
	// A reused native address must resolve to its new public identity.
	f.subscribe("speaker-beta")
	f.stats(false)
	if identities[len(identities)-1] != "speaker-beta" || identities[len(identities)-2] != "speaker-beta" {
		t.Fatal(identities)
	}
}

func TestNativeCallbackFrameABI(t *testing.T) {
	f := newCallbackABIFixture()
	defer f.close()
	var videoCalls, audioCalls int
	f.con.videoObserver = &VideoFrameObserver{OnFrame: func(channel, uid string, frame *VideoFrame) {
		videoCalls++
		if channel != "test-room" || uid != "speaker-alpha" || frame.Width != 2 || frame.Height != 2 || frame.RenderTimeMs != 123456 || !reflect.DeepEqual(frame.YBuffer, []byte{11, 12, 13, 14}) || !reflect.DeepEqual(frame.UBuffer, []byte{15}) || !reflect.DeepEqual(frame.VBuffer, []byte{16}) {
			t.Fatalf("video callback: %q %q %+v", channel, uid, frame)
		}
	}}
	f.videoFrame()
	if videoCalls != 1 {
		t.Fatal("void video callback was not dispatched")
	}
	if got := f.encodedAudio(false); got != 0 {
		t.Fatalf("absent encoded audio handler returned %d", got)
	}
	f.con.encodedAudioFrameObserver = &AudioEncodedFrameObserver{OnEncodedAudioFrameReceived: func(uid string, data []byte, ts int64, codec int) {
		audioCalls++
		if uid != "speaker-alpha" || !reflect.DeepEqual(data, []byte{21, 22, 23}) || ts != 123456 || codec != 7 {
			t.Fatalf("encoded audio: %q %v %d %d", uid, data, ts, codec)
		}
	}}
	if got := f.encodedAudio(false); got != 1 {
		t.Fatalf("delivered encoded audio returned %d", got)
	}
	if got := f.encodedAudio(true); got != 0 || audioCalls != 1 {
		t.Fatalf("nil info: result %d calls %d", got, audioCalls)
	}
	f.con.callbacks.pause()
	f.videoFrame()
	if got := f.encodedAudio(false); got != 0 {
		t.Fatalf("paused encoded audio returned %d", got)
	}
	f.con.callbacks.resume()
	if videoCalls != 1 || audioCalls != 1 {
		t.Fatal("callback dispatched during teardown")
	}
}

func TestNativeCallbackConcurrentIdentityTeardown(t *testing.T) {
	f := newCallbackABIFixture()
	defer f.close()
	var calls atomic.Int64
	f.con.localUserObserver = &LocalUserObserver{OnRemoteAudioTrackStatistics: func(*LocalUser, string, *RemoteAudioTrackStats) { calls.Add(1) }}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				f.state("speaker-alpha", 1)
				f.stats(false)
				f.state("speaker-alpha", 0)
			}
		}()
	}
	for range 20 {
		f.con.callbacks.pause()
		f.con.remoteTracks.clear()
		f.con.callbacks.resume()
	}
	wg.Wait()
	f.con.callbacks.pause()
	f.con.remoteTracks.clear()
	before := calls.Load()
	f.stats(false)
	f.state("speaker-alpha", 1)
	if calls.Load() != before {
		t.Fatal("stats delivered after callback gate closed")
	}
	if _, ok := f.con.remoteTracks.lookup(remoteAudioTrack, f.audio); ok {
		t.Fatal("identity recreated during teardown")
	}
	f.con.callbacks.resume()
}

func TestNativeCallbackStringZeroVideoIdentity(t *testing.T) {
	f := newCallbackABIFixture()
	defer f.close()
	var got string
	f.con.videoObserver = &VideoFrameObserver{OnFrame: func(_, uid string, _ *VideoFrame) { got = uid }}
	f.videoFrameFor("0")
	if got != "0" {
		t.Fatalf("string identity callback = %q, want 0", got)
	}
}

func TestNativeCallbackLateArrivalAfterTeardown(t *testing.T) {
	f := newCallbackABIFixture()
	f.subscribe("speaker-alpha")
	var calls atomic.Int64
	f.con.localUserObserver = &LocalUserObserver{OnRemoteAudioTrackStatistics: func(*LocalUser, string, *RemoteAudioTrackStats) { calls.Add(1) }}
	f.con.videoObserver = &VideoFrameObserver{OnFrame: func(string, string, *VideoFrame) { calls.Add(1) }}
	f.con.encodedAudioFrameObserver = &AudioEncodedFrameObserver{OnEncodedAudioFrameReceived: func(string, []byte, int64, int) { calls.Add(1) }}
	f.close()
	// A queued native callback may still carry retired opaque addresses. They
	// must be rejected by lookup/gating, never dereferenced as C strings.
	f.stats(false)
	f.videoFrame()
	if result := f.encodedAudio(false); result != 0 {
		t.Fatalf("late encoded callback returned %d", result)
	}
	if calls.Load() != 0 {
		t.Fatal("late callback reached application")
	}
}
