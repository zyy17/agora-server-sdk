#ifndef C_I_AGORA_STREAM_CHANNEL_H
#define C_I_AGORA_STREAM_CHANNEL_H

#include "agora_api.h"

#include "C_AgoraRtmBase.h"

#ifdef __cplusplus
extern "C"
{
#endif // __cplusplus

#pragma region agora

#pragma region agora::rtm

  /**
   * The qos of rtm message.
   */
  enum C_RTM_MESSAGE_QOS
  {
    /**
     * Will not ensure that messages arrive in order.
     */
    RTM_MESSAGE_QOS_UNORDERED = 0,
    /**
     * Will ensure that messages arrive in order.
     */
    RTM_MESSAGE_QOS_ORDERED = 1,
  };

  /**
   * The priority of rtm message.
   */
  enum C_RTM_MESSAGE_PRIORITY
  {
    /**
     * The highest priority
     */
    RTM_MESSAGE_PRIORITY_HIGHEST = 0,
    /**
     * The high priority
     */
    RTM_MESSAGE_PRIORITY_HIGH = 1,
    /**
     * The normal priority (Default)
     */
    RTM_MESSAGE_PRIORITY_NORMAL = 4,
    /**
     * The low priority
     */
    RTM_MESSAGE_PRIORITY_LOW = 8,
  };

  /**
   * Join channel options.
   */
  struct C_JoinChannelOptions
  {
    /**
     * Token used to join channel.
     */
    const char *token;
    /**
     * Whether to subscribe channel metadata information
     */
    bool withMetadata;
    /**
     * Whether to subscribe channel with user presence
     */
    bool withPresence;
    /**
     * Whether to subscribe channel with lock
     */
    bool withLock;
    /**
     * Whether to join channel in quiet mode
     */
    bool beQuiet;
  };
  struct C_JoinChannelOptions *C_JoinChannelOptions_New();
  void C_JoinChannelOptions_Delete(struct C_JoinChannelOptions *this_);

  /**
   * Join topic options.
   */
  struct C_JoinTopicOptions
  {
    /**
     * The qos of rtm message.
     */
    enum C_RTM_MESSAGE_QOS qos;

    /**
     * The priority of rtm message.
     */
    enum C_RTM_MESSAGE_PRIORITY priority;

    /**
     * The metaData of topic.
     */
    const char *meta;

    /**
     * The rtm data will sync with media
     */
    bool syncWithMedia;
  };
  struct C_JoinTopicOptions *C_JoinTopicOptions_New();
  void C_JoinTopicOptions_Delete(struct C_JoinTopicOptions *this_);

  /**
   * Topic options.
   */
  struct C_TopicOptions
  {
    /**
     * The list of users to subscribe.
     */
    const char **users;
    /**
     * The number of users.
     */
    size_t userCount;
  };
  struct C_TopicOptions *C_TopicOptions_New();
  void C_TopicOptions_Delete(struct C_TopicOptions *this_);

  /**
   * The IStreamChannel class.
   *
   * This class provides the stream channel methods that can be invoked by your app.
   */
  typedef void C_IStreamChannel;
#pragma region C_IStreamChannel
  /**
   * Join the channel.
   *
   * @param [in] options join channel options.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_join(C_IStreamChannel *this_, const struct C_JoinChannelOptions *options, uint64_t *requestId);

  /**
   * Renews the token. Once a token is enabled and used, it expires after a certain period of time.
   * You should generate a new token on your server, call this method to renew it.
   *
   * @param [in] token Token used renew.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_renew_token(C_IStreamChannel *this_, const char *token, uint64_t *requestId);

  /**
   * Leave the channel.
   *
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_leave(C_IStreamChannel *this_, uint64_t *requestId);

  /**
   * Return the channel name of this stream channel.
   *
   * @return The channel name.
   */
  const char *agora_rtm_stream_channel_get_channel_name(C_IStreamChannel *this_);

  /**
   * Join a topic.
   *
   * @param [in] topic The name of the topic.
   * @param [in] options The options of the topic.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_join_topic(C_IStreamChannel *this_, const char *topic, const struct C_JoinTopicOptions *options, uint64_t *requestId);

  /**
   * Publish a message in the topic.
   *
   * @param [in] topic The name of the topic.
   * @param [in] message The content of the message.
   * @param [in] length The length of the message.
   * @param [in] option The option of the message.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_publish_topic_message(C_IStreamChannel *this_, const char *topic, const char *message, size_t length, const struct C_TopicMessageOptions *option, uint64_t *requestId);

  /**
   * Leave the topic.
   *
   * @param [in] topic The name of the topic.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_leave_topic(C_IStreamChannel *this_, const char *topic, uint64_t *requestId);

  /**
   * Subscribe a topic.
   *
   * @param [in] topic The name of the topic.
   * @param [in] options The options of subscribe the topic.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_subscribe_topic(C_IStreamChannel *this_, const char *topic, const struct C_TopicOptions *options, uint64_t *requestId);

  /**
   * Unsubscribe a topic.
   *
   * @param [in] topic The name of the topic.
   * @param [in] options The options of unsubscribe the topic.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_unsubscribe_topic(C_IStreamChannel *this_, const char *topic, const struct C_TopicOptions *options, uint64_t *requestId);

  /**
   * Get subscribed user list
   *
   * @param [in] topic The name of the topic.
   * @param [out] requestId The unique ID of this request.
   */
  void agora_rtm_stream_channel_get_subscribed_user_list(C_IStreamChannel *this_, const char *topic, uint64_t *requestId);

  /**
   * Release the stream channel instance.
   *
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int agora_rtm_stream_channel_release(C_IStreamChannel *this_);

#pragma endregion C_IStreamChannel

#pragma endregion agora::rtm

#pragma endregion agora

#ifdef __cplusplus
}
#endif // __cplusplus

#endif // C_I_AGORA_STREAM_CHANNEL_H
