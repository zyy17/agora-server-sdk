//go:build agora_test

package agoraservice

/*
#include <stdint.h>
#include "observer_bindings.h"

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
*/
import "C"

func (f *callbackABIFixture) pcm(position int) (accepted bool, first byte) {
	result := int(C.callback_test_pcm(f.local, C.int(position)))
	return result&255 != 0, byte(result >> 8)
}
