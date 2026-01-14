#ifndef C_I_AGORA_RTM_STORAGE_H
#define C_I_AGORA_RTM_STORAGE_H

#include "agora_api.h"

#include "C_AgoraRtmBase.h"

#ifdef __cplusplus
extern "C"
{
#endif // __cplusplus

#pragma region agora

#pragma region agora::rtm

  /**
   * Metadata options.
   */
  struct C_MetadataOptions
  {
    /**
     * Indicates whether or not to notify server update the modify timestamp of metadata
     */
    bool recordTs;
    /**
     * Indicates whether or not to notify server update the modify user id of metadata
     */
    bool recordUserId;
  };
  struct C_MetadataOptions *C_MetadataOptions_New();
  void C_MetadataOptions_Delete(struct C_MetadataOptions *this_);

  struct C_MetadataItem
  {
    /**
     * The key of the metadata item.
     */
    const char *key;
    /**
     * The value of the metadata item.
     */
    const char *value;
    /**
     * The User ID of the user who makes the latest update to the metadata item.
     */
    const char *authorUserId;
    /**
     * The revision of the metadata item.
     */
    int64_t revision;
    /**
     * The Timestamp when the metadata item was last updated.
     */
    int64_t updateTs;
  };
  struct C_MetadataItem *C_MetadataItem_New();
  void C_MetadataItem_Delete(struct C_MetadataItem *this_);

  struct C_Metadata
  {
    /**
     * the major revision of metadata.
     */
    int64_t majorRevision;
    /**
     * The metadata item array.
     */
    struct C_MetadataItem *items;
    /**
     * The items count.
     */
    size_t itemCount;
  };
  struct C_Metadata *C_Metadata_New();
  void C_Metadata_Delete(struct C_Metadata *this_);

  typedef void C_IRtmStorage;
#pragma region C_IRtmStorage
  /**
   * Set the metadata of a specified channel.
   *
   * @param [in] channelName The name of the channel.
   * @param [in] channelType Which channel type, RTM_CHANNEL_TYPE_STREAM or RTM_CHANNEL_TYPE_MESSAGE.
   * @param [in] data Metadata data.
   * @param [in] options The options of operate metadata.
   * @param [in] lock lock for operate channel metadata.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_set_channel_metadata(C_IRtmStorage *this_,
                                       const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const struct C_Metadata *data, const struct C_MetadataOptions *options, const char *lockName, uint64_t *requestId);
  /**
   * Update the metadata of a specified channel.
   *
   * @param [in] channelName The channel Name of the specified channel.
   * @param [in] channelType Which channel type, RTM_CHANNEL_TYPE_STREAM or RTM_CHANNEL_TYPE_MESSAGE.
   * @param [in] data Metadata data.
   * @param [in] options The options of operate metadata.
   * @param [in] lock lock for operate channel metadata.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_update_channel_metadata(C_IRtmStorage *this_,
                                          const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const struct C_Metadata *data, const struct C_MetadataOptions *options, const char *lockName, uint64_t *requestId);
  /**
   * Remove the metadata of a specified channel.
   *
   * @param [in] channelName The channel Name of the specified channel.
   * @param [in] channelType Which channel type, RTM_CHANNEL_TYPE_STREAM or RTM_CHANNEL_TYPE_MESSAGE.
   * @param [in] data Metadata data.
   * @param [in] options The options of operate metadata.
   * @param [in] lock lock for operate channel metadata.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_remove_channel_metadata(C_IRtmStorage *this_,
                                          const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, const struct C_Metadata *data, const struct C_MetadataOptions *options, const char *lockName, uint64_t *requestId);
  /**
   * Get the metadata of a specified channel.
   *
   * @param [in] channelName The channel Name of the specified channel.
   * @param [in] channelType Which channel type, RTM_CHANNEL_TYPE_STREAM or RTM_CHANNEL_TYPE_MESSAGE.
   * @param requestId The unique ID of this request.
   */
  void agora_rtm_storage_get_channel_metadata(C_IRtmStorage *this_,
                                       const char *channelName, enum C_RTM_CHANNEL_TYPE channelType, uint64_t *requestId);

  /**
   * Set the metadata of a specified user.
   *
   * @param [in] userId The user ID of the specified user.
   * @param [in] data Metadata data.
   * @param [in] options The options of operate metadata.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_set_user_metadata(C_IRtmStorage *this_,
                                    const char *userId, const struct C_Metadata *data, const struct C_MetadataOptions *options, uint64_t *requestId);
  /**
   * Update the metadata of a specified user.
   *
   * @param [in] userId The user ID of the specified user.
   * @param [in] data Metadata data.
   * @param [in] options The options of operate metadata.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_update_user_metadata(C_IRtmStorage *this_,
                                       const char *userId, const struct C_Metadata *data, const struct C_MetadataOptions *options, uint64_t *requestId);
  /**
   * Remove the metadata of a specified user.
   *
   * @param [in] userId The user ID of the specified user.
   * @param [in] data Metadata data.
   * @param [in] options The options of operate metadata.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_remove_user_metadata(C_IRtmStorage *this_,
                                       const char *userId, const struct C_Metadata *data, const struct C_MetadataOptions *options, uint64_t *requestId);
  /**
   * Get the metadata of a specified user.
   *
   * @param [in] userId The user ID of the specified user.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_get_user_metadata(C_IRtmStorage *this_, const char *userId, uint64_t *requestId);

  /**
   * Subscribe the metadata update event of a specified user.
   *
   * @param [in] userId The user ID of the specified user.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_subscribe_user_metadata(C_IRtmStorage *this_, const char *userId, uint64_t *requestId);
  /**
   * unsubscribe the metadata update event of a specified user.
   *
   * @param [in] userId The user ID of the specified user.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_storage_unsubscribe_user_metadata(C_IRtmStorage *this_, const char *userId, uint64_t *requestId);
#pragma endregion C_IRtmStorage

#pragma endregion agora::rtm

#pragma endregion agora

#ifdef __cplusplus
}
#endif // __cplusplus

#endif // C_I_AGORA_RTM_STORAGE_H
