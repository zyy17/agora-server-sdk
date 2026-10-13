//go:build agora_test

package agoraservice

/*
#include <stdint.h>
#include "observer_bindings.h"
#include "agora_audio_track.h"

static int callback_test_pcm(void *local, int position) {
  audio_frame_observer observer = {0};
  bind_audio_frame_observer(&observer);
  unsigned char samples[] = {1, 2, 3, 4};
  audio_frame frame = { .samples_per_channel = 2, .bytes_per_sample = 2,
    .channels = 1, .samples_per_sec = 16000, .buffer = samples };
  int accepted = 0;
  switch (position) {
    case 1: accepted = observer.on_playback_audio_frame(local, "room", &frame); break;
    case 2: accepted = observer.on_record_audio_frame(local, "room", &frame); break;
    case 4: accepted = observer.on_mixed_audio_frame(local, "room", &frame); break;
    case 8: accepted = observer.on_playback_audio_frame_before_mixing(local, "room", "peer", &frame); break;
    case 16: accepted = observer.on_ear_monitoring_audio_frame(local, &frame); break;
  }
  return accepted + ((int)samples[0] << 8);
}
static void callback_test_missing_ai_capability(void *handle) {
  capabilites_observer observer = {0};
  bind_capabilites_observer(&observer);
  capabilities caps = { .capability_type = 19 };
  observer.on_capabilities_changed(handle, &caps, 1);
}
*/
import "C"

func (f *callbackABIFixture) pcm(position int) (accepted bool, first byte) {
	result := int(C.callback_test_pcm(f.local, C.int(position)))
	return result&255 != 0, byte(result >> 8)
}

func inspectNativeAudioTrack(track *LocalAudioTrack) (enabled bool, volume, code int) {
	var nativeVolume C.int
	result := C.agora_local_audio_track_get_publish_volume(track.cTrack, &nativeVolume)
	return C.agora_local_audio_track_is_enabled(track.cTrack) != 0, int(nativeVolume), int(result)
}

func invokeMissingAIQoSCapability(conn *RtcConnection) {
	C.callback_test_missing_ai_capability(conn.cCapObserverHandle)
}
