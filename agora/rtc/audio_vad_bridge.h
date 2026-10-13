#ifndef AGORA_GO_AUDIO_VAD_BRIDGE_H
#define AGORA_GO_AUDIO_VAD_BRIDGE_H

#include "vad.h"

/* No native C++ exception may cross a cgo boundary. */
#define AGORA_GO_VAD_EXCEPTION (-2002)
#ifdef __cplusplus
extern "C" {
#endif
int agora_go_vad_create(void **handle, const Vad_Config *config);
int agora_go_vad_process(void *handle, const Vad_AudioData *input,
                         Vad_AudioData *output, enum VAD_STATE *state);
int agora_go_vad_destroy(void **handle);
#ifdef __cplusplus
}
#endif
#endif
