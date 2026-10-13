#include "audio_vad_bridge.h"

extern "C" int agora_go_vad_create(void **handle, const Vad_Config *config) {
    try {
        return Agora_UAP_VAD_Create(handle, config);
    } catch (...) {
        return AGORA_GO_VAD_EXCEPTION;
    }
}

extern "C" int agora_go_vad_process(void *handle, const Vad_AudioData *input,
                                    Vad_AudioData *output, enum VAD_STATE *state) {
    try {
        return Agora_UAP_VAD_Proc(handle, input, output, state);
    } catch (...) {
        output->audioData = nullptr;
        output->size = 0;
        *state = VAD_STATE_NONE_SPEAKING;
        return AGORA_GO_VAD_EXCEPTION;
    }
}

extern "C" int agora_go_vad_destroy(void **handle) {
    try {
        return Agora_UAP_VAD_Destroy(handle);
    } catch (...) {
        *handle = nullptr;
        return AGORA_GO_VAD_EXCEPTION;
    }
}
