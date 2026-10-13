#pragma once
#include "agora_local_user.h"
#include "agora_rtc_conn.h"
void bind_rtc_conn_observer(rtc_conn_observer *observer);
void bind_local_user_observer(local_user_observer *observer);
void bind_audio_frame_observer(audio_frame_observer *observer);
void bind_video_frame_observer2(video_frame_observer2 *observer);
void bind_video_encoded_frame_observer(video_encoded_frame_observer *observer);
void bind_capabilites_observer(capabilites_observer *observer);
void bind_audio_encoded_frame_rev_observer(audio_encoded_frame_rev_observer *observer);
