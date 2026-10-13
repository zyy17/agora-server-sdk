#include "observer_bindings.h"
#include "rtc_callbacks_cgo.h"
#include "audio_observer_cgo.h"
#include "video_observer_cgo.h"

// Make signature drift a build error, including on compilers that only warn by default.
#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic error "-Wincompatible-pointer-types"
#endif

void bind_rtc_conn_observer(rtc_conn_observer *observer) {
  observer->on_connected = cgo_on_connected;
  observer->on_disconnected = cgo_on_disconnected;
  observer->on_connecting = cgo_on_connecting;
  observer->on_reconnecting = cgo_on_reconnecting;
  observer->on_reconnected = cgo_on_reconnected;
  observer->on_connection_lost = cgo_on_connection_lost;
  observer->on_connection_failure = cgo_on_connection_failure;
  observer->on_token_privilege_will_expire = cgo_on_token_privilege_will_expire;
  observer->on_token_privilege_did_expire = cgo_on_token_privilege_did_expire;
  observer->on_user_joined = cgo_on_user_joined;
  observer->on_user_left = cgo_on_user_left;
  observer->on_error = cgo_on_error;
  observer->on_stream_message_error = cgo_on_stream_message_error;
  observer->on_encryption_error = cgo_on_encryption_error;
}

void bind_local_user_observer(local_user_observer *observer) {
  observer->on_stream_message = cgo_on_stream_message;
  observer->on_user_info_updated = cgo_on_user_info_updated;
  observer->on_user_audio_track_subscribed = cgo_on_user_audio_track_subscribed;
  observer->on_user_video_track_subscribed = cgo_on_user_video_track_subscribed;
  observer->on_user_audio_track_state_changed = cgo_on_user_audio_track_state_changed;
  observer->on_user_video_track_state_changed = cgo_on_user_video_track_state_changed;
  observer->on_audio_publish_state_changed = cgo_on_audio_publish_state_changed;
  observer->on_audio_volume_indication = cgo_on_audio_volume_indication;
  observer->on_audio_meta_data_received = cgo_on_audio_meta_data_received;
  observer->on_local_audio_track_statistics = cgo_on_local_audio_track_statistics;
  observer->on_remote_audio_track_statistics = cgo_on_remote_audio_track_statistics;
  observer->on_local_video_track_statistics = cgo_on_local_video_track_statistics;
  observer->on_remote_video_track_statistics = cgo_on_remote_video_track_statistics;
  observer->on_audio_track_publish_success = cgo_on_audio_track_publish_success;
  observer->on_audio_track_unpublished = cgo_on_audio_track_unpublished;
  observer->on_video_track_publish_success = cgo_on_video_track_publish_success;
  observer->on_video_track_unpublished = cgo_on_video_track_unpublished;
  observer->on_intra_request_received = cgo_on_intra_request_received;
}

void bind_audio_frame_observer(audio_frame_observer *observer) {
  observer->on_record_audio_frame = cgo_on_record_audio_frame;
  observer->on_playback_audio_frame = cgo_on_playback_audio_frame;
  observer->on_mixed_audio_frame = cgo_on_mixed_audio_frame;
  observer->on_ear_monitoring_audio_frame = cgo_on_ear_monitoring_audio_frame;
  observer->on_playback_audio_frame_before_mixing = cgo_on_playback_audio_frame_before_mixing;
  observer->on_get_audio_frame_position = cgo_on_get_audio_frame_position;
  observer->on_get_playback_audio_frame_param = cgo_on_get_playback_audio_frame_param;
  observer->on_get_record_audio_frame_param = cgo_on_get_record_audio_frame_param;
  observer->on_get_mixed_audio_frame_param = cgo_on_get_mixed_audio_frame_param;
  observer->on_get_ear_monitoring_audio_frame_param = cgo_on_get_ear_monitoring_audio_frame_param;
}

void bind_video_frame_observer2(video_frame_observer2 *observer) {
  observer->on_frame = cgo_on_video_frame;
}

void bind_video_encoded_frame_observer(video_encoded_frame_observer *observer) {
  observer->on_encoded_video_frame = cgo_on_encoded_video_frame;
}

void bind_capabilites_observer(capabilites_observer *observer) {
  observer->on_capabilities_changed = cgo_on_capabilities_changed;
}

void bind_audio_encoded_frame_rev_observer(audio_encoded_frame_rev_observer *observer) {
  observer->on_encoded_audio_frame_received = cgo_on_encoded_audio_frame_received;
}
