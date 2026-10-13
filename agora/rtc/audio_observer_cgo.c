#include "_cgo_export.h"
#include "audio_observer_cgo.h"

int cgo_on_record_audio_frame(AGORA_HANDLE agora_local_user,const char* channelId, const audio_frame* frame) {
  return goOnRecordAudioFrame(agora_local_user, (char*)channelId, (audio_frame*)frame);
}

int cgo_on_playback_audio_frame(AGORA_HANDLE agora_local_user, const char* channelId, const audio_frame* frame) {
  return goOnPlaybackAudioFrame(agora_local_user, (char*)channelId, (audio_frame*)frame);
}

int cgo_on_mixed_audio_frame(AGORA_HANDLE agora_local_user, const char* channelId, const audio_frame* frame) {
  return goOnMixedAudioFrame(agora_local_user, (char*)channelId, (audio_frame*)frame);
}

int cgo_on_ear_monitoring_audio_frame(AGORA_HANDLE agora_local_user, const audio_frame* frame) {
  return goOnEarMonitoringAudioFrame(agora_local_user, (audio_frame*)frame);
}

// this function declaration must be strictly same with the function exported by go
int cgo_on_playback_audio_frame_before_mixing(AGORA_HANDLE agora_local_user, const char* channelId, user_id_t uid, const audio_frame* frame) {
  return goOnPlaybackAudioFrameBeforeMixing(agora_local_user, (char*)channelId, (char*)uid, (audio_frame*)frame);
}

int cgo_on_get_audio_frame_position(AGORA_HANDLE agora_local_user) {
  return goOnGetAudioFramePosition(agora_local_user);
}

audio_params cgo_on_get_playback_audio_frame_param(AGORA_HANDLE agora_local_user) {
  return goOnGetPlaybackAudioFrameParam(agora_local_user);
}

audio_params cgo_on_get_record_audio_frame_param(AGORA_HANDLE agora_local_user) {
  return goOnGetRecordAudioFrameParam(agora_local_user);
}

audio_params cgo_on_get_mixed_audio_frame_param(AGORA_HANDLE agora_local_user) {
  return goOnGetMixedAudioFrameParam(agora_local_user);
}

audio_params cgo_on_get_ear_monitoring_audio_frame_param(AGORA_HANDLE agora_local_user) {
  return goOnGetEarMonitoringAudioFrameParam(agora_local_user);
}

int cgo_on_encoded_audio_frame_received(AGORA_HANDLE agora_audio_encoded_frame_observer, const uint8_t* packet, size_t length,
    const encoded_audio_frame_rev_info* encoded_audio_frame_info) {
  return goOnEncodedAudioFrameReceived(agora_audio_encoded_frame_observer, (uint8_t*)packet, length, (encoded_audio_frame_rev_info*)encoded_audio_frame_info);
}
