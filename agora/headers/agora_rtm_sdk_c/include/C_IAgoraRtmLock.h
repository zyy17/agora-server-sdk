#ifndef C_I_AGORA_RTM_LOCK_H
#define C_I_AGORA_RTM_LOCK_H

#include "agora_api.h"

#include "C_AgoraRtmBase.h"

#ifdef __cplusplus
extern "C"
{
#endif // __cplusplus

#pragma region agora

#pragma region agora::rtm

  /**
   * The IRtmLock class.
   *
   * This class provides the rtm lock methods that can be invoked by your app.
   */
  typedef void C_IRtmLock;
#pragma region C_IRtmLock
  /**
   * sets a lock
   *
   * @param [in] channelName The name of the channel.
   * @param [in] channelType The type of the channel.
   * @param [in] lockName The name of the lock.
   * @param [in] ttl The lock ttl.
   * @param [out] requestId The related request id of this operation.
   */
  void agora_rtm_lock_set_lock(C_IRtmLock *this_, const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const char *lockName, uint32_t ttl, uint64_t *requestId);

  /**
   * gets locks in the channel
   *
   * @param [in] channelName The name of the channel.
   * @param [in] channelType The type of the channel.
   * @param [out] requestId The related request id of this operation.
   */
  void agora_rtm_lock_get_locks(C_IRtmLock *this_, const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, uint64_t *requestId);

  /**
   * removes a lock
   *
   * @param [in] channelName The name of the channel.
   * @param [in] channelType The type of the channel.
   * @param [in] lockName The name of the lock.
   * @param [out] requestId The related request id of this operation.
   */
  void agora_rtm_lock_remove_lock(C_IRtmLock *this_, const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const char *lockName, uint64_t *requestId);

  /**
   * acquires a lock
   *
   * @param [in] channelName The name of the channel.
   * @param [in] channelType The type of the channel.
   * @param [in] lockName The name of the lock.
   * @param [in] retry Whether to automatically retry when acquires lock failed
   * @param [out] requestId The related request id of this operation.
   */
  void agora_rtm_lock_acquire_lock(C_IRtmLock *this_, const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const char *lockName, bool retry, uint64_t *requestId);

  /**
   * releases a lock
   *
   * @param [in] channelName The name of the channel.
   * @param [in] channelType The type of the channel.
   * @param [in] lockName The name of the lock.
   * @param [out] requestId The related request id of this operation.
   */
  void agora_rtm_lock_release_lock(C_IRtmLock *this_, const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const char *lockName, uint64_t *requestId);

  /**
   * disables a lock
   *
   * @param [in] channelName The name of the channel.
   * @param [in] channelType The type of the channel.
   * @param [in] lockName The name of the lock.
   * @param [in] owner The lock owner.
   * @param [out] requestId The related request id of this operation.
   */
  void agora_rtm_lock_revoke_lock(C_IRtmLock *this_, const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const char *lockName, const char *owner, uint64_t *requestId);
#pragma endregion C_IRtmLock

#pragma endregion agora::rtm

#pragma endregion agora

#ifdef __cplusplus
}
#endif // __cplusplus

#endif // C_I_AGORA_RTM_LOCK_H
