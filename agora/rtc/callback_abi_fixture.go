//go:build agora_test

package agoraservice

/*
#include <stdlib.h>
#include <stdint.h>
#include "observer_bindings.h"

static void* callback_test_handle(void) {
  unsigned char *p = malloc(1);
  if (p) *p = 0xff; // Deliberately not a NUL-terminated user ID.
  return p;
}

static void callback_test_subscribe(void *local, void *audio, void *video, const char *uid) {
  local_user_observer observer = {0};
  bind_local_user_observer(&observer);
  video_track_info info = {0};
  observer.on_user_audio_track_subscribed(local, uid, audio);
  observer.on_user_video_track_subscribed(local, uid, &info, video);
}

static void callback_test_stats(void *local, void *audio, void *video, int nil_stats) {
  local_user_observer observer = {0};
  bind_local_user_observer(&observer);
  local_audio_stats local_audio = { .num_channels = 2, .sent_sample_rate = 48000,
    .sent_bitrate = 96, .internal_codec = 7, .voice_pitch = 123.5 };
  remote_audio_track_stats remote_audio = { .uid = 999, .quality = 1,
    .network_transport_delay = 12, .jitter_buffer_delay = 13, .audio_loss_rate = 14,
    .num_channels = 2, .received_sample_rate = 48000, .received_bitrate = 96,
    .total_frozen_time = 18, .frozen_rate = 19, .received_bytes = INT64_C(0x123456789ab) };
  local_video_track_stats local_video = { .number_of_streams = 2,
    .bytes_major_stream = UINT64_C(0x123456789ab), .bytes_minor_stream = 31,
    .frames_encoded = 32, .ssrc_major_stream = 33, .ssrc_minor_stream = 34,
    .capture_frame_rate = 35, .regulated_capture_frame_rate = 36, .input_frame_rate = 37,
    .encode_frame_rate = 38, .render_frame_rate = 39, .target_media_bitrate_bps = 40,
    .media_bitrate_bps = 41, .total_bitrate_bps = 42, .capture_width = 43,
    .capture_height = 44, .regulated_capture_width = 45, .regulated_capture_height = 46,
    .width = 47, .height = 48, .encoder_type = 49, .uplink_cost_time_ms = 50,
    .quality_adapt_indication = 51 };
  remote_video_track_stats remote_video = { .uid = 888, .delay = 61, .width = 62,
    .height = 63, .received_bitrate = 64, .decoder_output_frame_rate = 65,
    .renderer_output_frame_rate = 66, .frame_loss_rate = 67, .packet_loss_rate = 68,
    .rx_stream_type = 69, .total_frozen_time = 70, .frozen_rate = 71,
    .total_decoded_frames = 72, .av_sync_time_ms = -73, .downlink_process_time_ms = 74,
    .frame_render_delay_ms = 75, .totalActiveTime = UINT64_C(0x123456789ab),
    .publishDuration = UINT64_C(0x123456789ac) };
  observer.on_local_audio_track_statistics(local, nil_stats ? NULL : &local_audio);
  observer.on_remote_audio_track_statistics(local, audio, nil_stats ? NULL : &remote_audio);
  // An opaque track argument deliberately precedes the real payload.
  observer.on_local_video_track_statistics(local, video, nil_stats ? NULL : &local_video);
  observer.on_remote_video_track_statistics(local, video, nil_stats ? NULL : &remote_video);
}

static void callback_test_state(void *local, void *audio, void *video, const char *uid, int state) {
  local_user_observer observer = {0};
  bind_local_user_observer(&observer);
  observer.on_user_audio_track_state_changed(local, uid, audio, state, 0, 0);
  observer.on_user_video_track_state_changed(local, uid, video, state, 0, 0);
}

static void callback_test_left(void *connection, const char *uid) {
  rtc_conn_observer observer = {0};
  bind_rtc_conn_observer(&observer);
  observer.on_user_left(connection, uid, 0);
}

static void callback_test_disconnected(void *connection) {
  rtc_conn_observer observer = {0};
  bind_rtc_conn_observer(&observer);
  observer.on_disconnected(connection, NULL, 0);
}

static void callback_test_video(void *handle, const char *uid) {
  video_frame_observer2 observer = {0};
  bind_video_frame_observer2(&observer);
  uint8_t pixels[] = {11, 12, 13, 14, 15, 16};
  video_frame frame = { .width = 2, .height = 2, .y_stride = 2, .u_stride = 1,
    .v_stride = 1, .y_buffer = pixels, .u_buffer = pixels + 4, .v_buffer = pixels + 5,
    .render_time_ms = 123456 };
  observer.on_frame(handle, "test-room", uid, &frame);
}

static int callback_test_encoded_audio(void *handle, int nil_info) {
  audio_encoded_frame_rev_observer observer = {0};
  bind_audio_encoded_frame_rev_observer(&observer);
  uint8_t packet[] = {21, 22, 23};
  encoded_audio_frame_rev_info info = { .sendTs = 123456, .codec = 7 };
  return observer.on_encoded_audio_frame_received(handle, packet, sizeof(packet), nil_info ? NULL : &info);
}
*/
import "C"

import "unsafe"

// These synthetic native entry points are compiled only with the agora_test tag.
// They exercise the registered C function pointers without creating native tracks
// or connecting to an RTC service.
type callbackABIFixture struct {
	con                          *RtcConnection
	local, audio, video, encoded unsafe.Pointer
}

func newCallbackABIFixture() *callbackABIFixture {
	f := &callbackABIFixture{local: C.callback_test_handle(), audio: C.callback_test_handle(), video: C.callback_test_handle(), encoded: C.callback_test_handle()}
	f.con = &RtcConnection{localUser: &LocalUser{cLocalUser: f.local}}
	agoraService.setConFromHandle(f.local, f.con, ConTypeCLocalUser)
	agoraService.setConFromHandle(f.local, f.con, ConTypeCCon)
	agoraService.setConFromHandle(f.video, f.con, ConTypeCVideoObserver)
	agoraService.setAudioEncObserToMap(&EncAudioFrameObserverItem{Uid: "speaker-alpha", Con: f.con, CObserverHandle: f.encoded})
	return f
}

func (f *callbackABIFixture) close() {
	f.con.callbacks.pause()
	f.con.remoteTracks.clear()
	agoraService.deleteConFromHandle(f.local, ConTypeCLocalUser)
	agoraService.deleteConFromHandle(f.local, ConTypeCCon)
	agoraService.deleteConFromHandle(f.video, ConTypeCVideoObserver)
	agoraService.encAudioFrameObserverItemsMap.Delete(f.encoded)
	C.free(f.local)
	C.free(f.audio)
	C.free(f.video)
	C.free(f.encoded)
}

func (f *callbackABIFixture) subscribe(uid string) {
	cuid := C.CString(uid)
	defer C.free(unsafe.Pointer(cuid))
	C.callback_test_subscribe(f.local, f.audio, f.video, cuid)
}
func (f *callbackABIFixture) stats(nilStats bool) {
	C.callback_test_stats(f.local, f.audio, f.video, CIntFromBool(nilStats))
}
func (f *callbackABIFixture) state(uid string, state int) {
	cuid := C.CString(uid)
	defer C.free(unsafe.Pointer(cuid))
	C.callback_test_state(f.local, f.audio, f.video, cuid, C.int(state))
}
func (f *callbackABIFixture) left(uid string) {
	cuid := C.CString(uid)
	defer C.free(unsafe.Pointer(cuid))
	C.callback_test_left(f.local, cuid)
}
func (f *callbackABIFixture) disconnect() { C.callback_test_disconnected(f.local) }
func (f *callbackABIFixture) videoFrameFor(uid string) {
	cuid := C.CString(uid)
	defer C.free(unsafe.Pointer(cuid))
	C.callback_test_video(f.video, cuid)
}
func (f *callbackABIFixture) videoFrame() { f.videoFrameFor("speaker-alpha") }
func (f *callbackABIFixture) encodedAudio(nilInfo bool) int {
	return int(C.callback_test_encoded_audio(f.encoded, CIntFromBool(nilInfo)))
}
