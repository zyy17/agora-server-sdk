#ifndef C_I_AGORA_RTM_HISTORY_H
#define C_I_AGORA_RTM_HISTORY_H

#include "C_AgoraRtmBase.h"

#ifdef __cplusplus
extern "C"
{
#endif // __cplusplus

// Get History Messages Options
typedef struct {
    int64_t start;
    int64_t end;
    int count;
} C_GetHistoryMessagesOptions;

#pragma region agora

#pragma region agora::rtm

typedef void C_IRtmHistory;
#pragma region C_IRtmHistory

/**
 * Gets history messages in the channel.
 *
 * @param [in] channelName The name of the channel.
 * @param [in] channelType The type of the channel.
 * @param [in] options The query options.
 * @param [out] requestId The related request id of this operation.
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
int agora_rtm_history_get_messages(C_IRtmHistory *this_, const char* channelName, enum C_RTM_CHANNEL_TYPE channelType, const C_GetHistoryMessagesOptions* options, uint64_t* requestId);

#pragma endregion agora::rtm

#pragma endregion agora

#ifdef __cplusplus
}
#endif // __cplusplus

#endif // C_I_AGORA_RTM_HISTORY_H 