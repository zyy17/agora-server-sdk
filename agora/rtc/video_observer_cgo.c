#include "_cgo_export.h"
#include <stdint.h>
#include "video_observer_cgo.h"

// this function declaration must be strictly same with the function exported by go
void cgo_on_video_frame(AGORA_HANDLE agora_video_frame_observer2, const char* channelId, user_id_t uid, const video_frame* frame) {
  goOnVideoFrame(agora_video_frame_observer2, (char*)channelId, (char*)uid, (video_frame*)frame);
}

int cgo_on_encoded_video_frame(AGORA_HANDLE agora_video_encoded_frame_observer, const char* channelId, const char* uid, const uint8_t* image_buffer, size_t length,
                                const encoded_video_frame_info* video_encoded_frame_info) {
  return goOnEncodedVideoFrame(agora_video_encoded_frame_observer, (char*)channelId, (char*)uid, (uint8_t*)image_buffer, length, (encoded_video_frame_info*)video_encoded_frame_info);
}
