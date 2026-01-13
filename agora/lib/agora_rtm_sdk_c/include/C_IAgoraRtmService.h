#ifndef C_I_AGORA_RTM_SERVICE_H
#define C_I_AGORA_RTM_SERVICE_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C"
{
#endif // __cplusplus

#pragma region agora

#pragma region agora::rtm

  enum C_PEER_MESSAGE_STATE
  {
    PEER_MESSAGE_INIT = 0,
    PEER_MESSAGE_FAILURE = 1,
    PEER_MESSAGE_PEER_UNREACHABLE = 2,
    PEER_MESSAGE_RECEIVED_BY_PEER = 3,
    PEER_MESSAGE_SENT_TIMEOUT = 4,
  };

  /**
   * The login error code.
   */
  enum C_LOGIN_ERR_CODE
  {
    /**
     * 0: Login succeeds. No error occurs.
     */
    LOGIN_ERR_OK = 0,
    /**
     * 1: Login fails for reasons unknown.
     */
    LOGIN_ERR_UNKNOWN = 1,
    /**
     * 2: The server rejects the login, either because the user has already logged in, or because
     * the RTM service is not initialized.
     */
    LOGIN_ERR_REJECTED = 2, // ALREADY LOGIN OR NOT INITIALIZED, SERVER REJECT
    /**
     * 3: Invalid login arguments.
     */
    LOGIN_ERR_INVALID_ARGUMENT = 3,
    /**
     * 4: The App ID is invalid.
     */
    LOGIN_ERR_INVALID_APP_ID = 4,
    /**
     * 5: The token is invalid.
     */
    LOGIN_ERR_INVALID_TOKEN = 5,
    /**
     * 6: The login is rejected because the token has expired.
     */
    LOGIN_ERR_TOKEN_EXPIRED = 6,
    /**
     * 7: Authentication of the RTMP token fails.
     */
    LOGIN_ERR_NOT_AUTHORIZED = 7,
    /**
     * 8: The login times out. The current timeout is set as six seconds.
     */
    LOGIN_ERR_TIMEOUT = 8,
  };
  /**
   * The logout error code.
   */
  enum C_LOGOUT_ERR_CODE
  {
    /**
     * 0: Logout succeeds. No error occurs.
     */
    LOGOUT_ERR_OK = 0,
    /**
     * 1: Logout fails.
     */
    LOGOUT_ERR_REJECTED = 1,
  };
  /**
   * The connection state.
   */
  enum C_CONNECTION_STATE
  {
    /**
     * 1: The SDK has logged in the RTM service.
     */
    CONNECTION_STATE_CONNECTED = 1,
    /**
     * 2: The initial state. The SDK is disconnected from the RTM service.
     */
    CONNECTION_STATE_DISCONNECTED = 2,
    /**
     * 3: The SDK gives up logging in the RTM service, mainly because another instance has logged in the RTM
     * service with the same user ID.
     *
     * Call the logout() method before calling login to log in the RTM service again.
     */
    CONNECTION_STATE_ABORTED = 3,
  };

  /**
   * The state of the channel message.
   */
  enum C_CHANNEL_MESSAGE_STATE
  {
    /**
     * 1: The channel message is received by the server.
     */
    CHANNEL_MESSAGE_RECEIVED_BY_SERVER = 1,
    /**
     * 3: The SDK has not received a response from the server in five seconds. The current timeout is set as
     * five seconds.
     */
    CHANNEL_MESSAGE_SENT_TIMEOUT = 3,
  };
  /**
   * The join channel error.
   */
  enum C_JOIN_CHANNEL_ERR
  {
    /**
     0: The method call succeeds, or the user joins the channel successfully.
    */
    JOIN_CHANNEL_ERR_OK = 0,

    /**
     1: Common failure. The user fails to join the channel.
    */
    JOIN_CHANNEL_ERR_FAILURE = 1,

    /**
     2: **RESERVED FOR FUTURE USE**
    */
    JOIN_CHANNEL_ERR_REJECTED = 2, // Usually occurs when the user is already in the channel

    /**
     3: The user fails to join the channel because the argument is invalid.
    */
    JOIN_CHANNEL_ERR_INVALID_ARGUMENT = 3,

    /**
     4: A timeout occurs when joining the channel. The current timeout is set as five seconds. Possible reasons: The user is in the \ref agora::rtm::CONNECTION_STATE_ABORTED "CONNECTION_STATE_ABORTED" state.
    */
    JOIN_CHANNEL_TIMEOUT = 4,

    /**
     5: The number of the RTM channels you are in exceeds the limit of 20.
    */
    JOIN_CHANNEL_ERR_EXCEED_LIMIT = 5,

    /**
     6: The user is joining or has joined the channel.
    */
    JOIN_CHANNEL_ERR_ALREADY_JOINED = 6,

    /**
     7: The method call frequency exceeds 50 queries every three seconds.
    */
    JOIN_CHANNEL_ERR_TOO_OFTEN = 7,

    /**
     8: The frequency of joining the same channel exceeds two times every five seconds.
    */
    JOIN_CHANNEL_ERR_JOIN_SAME_CHANNEL_TOO_OFTEN = 8,

    /**
     101: \ref agora::rtm::IRtmService "IRtmService" is not initialized.
    */
    JOIN_CHANNEL_ERR_NOT_INITIALIZED = 101,

    /**
     102: The user does not call the \ref agora::rtm::IRtmService::login "login" method, or the method call of \ref agora::rtm::IRtmService::login "login" does not succeed before joining the channel.
    */
    JOIN_CHANNEL_ERR_USER_NOT_LOGGED_IN = 102,
  };
  /**
   @brief Error codes related to leaving a channel.
    */
  enum C_LEAVE_CHANNEL_ERR
  {

    /**
     0: The method call succeeds, or the user leaves the channel successfully.
      */
    LEAVE_CHANNEL_ERR_OK = 0,

    /**
     1: Common failure. The user fails to leave the channel.
      */
    LEAVE_CHANNEL_ERR_FAILURE = 1,

    /**
     2: **RESERVED FOR FUTURE USE**
      */
    LEAVE_CHANNEL_ERR_REJECTED = 2,

    /**
     3: The user is not in the channel.
      */
    LEAVE_CHANNEL_ERR_NOT_IN_CHANNEL = 3,

    /**
     101: \ref agora::rtm::IRtmService "IRtmService" is not initialized.
      */
    LEAVE_CHANNEL_ERR_NOT_INITIALIZED = 101,

    /**
     102: The user does not call the \ref agora::rtm::IRtmService::login "login" method, or the method call of \ref agora::rtm::IRtmService::login "login" does not succeed before calling the \ref agora::rtm::IChannel::leave "leave" method.
      */
    LEAVE_CHANNEL_ERR_USER_NOT_LOGGED_IN = 102,
  };

  /**
   * The reason for a user to leave the channel.
   */
  enum C_LEAVE_CHANNEL_REASON
  {
    /**
     * 1: The user quits the channel.
     */
    LEAVE_CHANNEL_REASON_QUIT = 1,
    /**
     * 2: The user is kicked off the channel.
     */
    LEAVE_CHANNEL_REASON_KICKED = 2,
  };
  /**
   @brief Error codes related to sending a channel message.
    */
  enum C_CHANNEL_MESSAGE_ERR_CODE
  {

    /**
     0: The method call succeeds, or the server receives the channel message.
      */
    CHANNEL_MESSAGE_ERR_OK = 0,

    /**
     1: Common failure. The user fails to send the channel message.
      */
    CHANNEL_MESSAGE_ERR_FAILURE = 1,

    /**
     2: The SDK does not receive a response from the server in 10 seconds. The current timeout is set as 10 seconds. Possible reasons: The user is in the \ref agora::rtm::CONNECTION_STATE_ABORTED "CONNECTION_STATE_ABORTED" state.
      */
    CHANNEL_MESSAGE_ERR_SENT_TIMEOUT = 2,

    /**
     3: The method call frequency exceeds the limit of (RTM SDK for Windows C++) 180 calls every three seconds or (RTM SDK for Linux C++) 1500 calls every three seconds, with channel and peer messages taken together..
      */
    CHANNEL_MESSAGE_ERR_TOO_OFTEN = 3,

    /**
     4: The message is null or exceeds 32 KB in length.
      */
    CHANNEL_MESSAGE_ERR_INVALID_MESSAGE = 4,

    /**
     101: \ref agora::rtm::IRtmService "IRtmService" is not initialized.
      */
    CHANNEL_MESSAGE_ERR_NOT_INITIALIZED = 101,

    /**
     102: The user does not call the \ref agora::rtm::IRtmService::login "login" method, or the method call of \ref agora::rtm::IRtmService::login "login" does not succeed before sending out a channel message.
      */
    CHANNEL_MESSAGE_ERR_USER_NOT_LOGGED_IN = 102,
  };

  /**
   * The response code.
   */
  enum C_RESPONSE_CODE
  {
    /**
     * 1: The response
     */
    RESPONSE_CODE_SUCCESS = 1,
    RESPONSE_CODE_FAILURE = 2,
  };
  /**
   * The message type.
   */
  enum C_MESSAGE_TYPE
  {
    /**
     * 0: The message type is undefined.
     */
    MESSAGE_TYPE_UNDEFINED = 0,
    /**
     * 1: A text message.
     */
    MESSAGE_TYPE_TEXT = 1,
    /**
     * 2: A raw message in binary, for example, audio data, or video data.
     */
    MESSAGE_TYPE_BINARY = 2,
    /**
     * 4: A converge message.
     */
    MESSAGE_TYPE_CONVERGE = 4,
  };

  /**
   * The IMessage class.
   */
  typedef void C_IMessage;
#pragma region C_IMessage
  // note: undefined reference to `agora::rtm::IMessage::createMessage()'
  // may be deprecated?
  // C_IMessage *C_IMessage_createMessage();
  void C_IMessage_Delete(C_IMessage *this_);

  /**
   * Gets the ID of the message.
   * @return The ID of the current IMessage instance.
   */
  int64_t C_IMessage_getMessageId(C_IMessage *this_);
  /**
   * Gets the message type.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IMessage_getMessageType(C_IMessage *this_);
  /**
   * Sets the content of the text message, or the text description of the raw message.
   * @param str The content of the text message, or the text description of the raw message. The maximum length
   * is 32 KB.
   */
  void C_IMessage_setText(C_IMessage *this_, const char *str);
  /**
   * Gets the content of the text messsage, or the text description of the raw message.
   * @return
   * - The content of the text message or the text description of the raw message.
   */
  const char *C_IMessage_getText(C_IMessage *this_);
  /**
   * Get pointer of custom raw message
   * @return
   * - The content of binary raw message
   */
  const unsigned char *C_IMessage_getRawMessageData(C_IMessage *this_);
  /**
   * Get length of custom raw message
   * @return
   * - The length of binary raw message in bytes
   */
  int C_IMessage_getRawMessageLength(C_IMessage *this_);
  /**
   * Set message type
   */
  void C_IMessage_setMessageType(C_IMessage *this_, int32_t type);
  /**
   * Set raw binary message
   */
  void C_IMessage_setRawMessage(C_IMessage *this_, const uint8_t *data, int length);
  /**
   * Releases the IMessage instance.
   */
  void C_IMessage_release(C_IMessage *this_);
#pragma endregion C_IMessage

  /**
   * The IChannelMember class.
   */
  typedef void C_IChannelMember;
#pragma region C_IChannelMember
  void C_IChannelMember_Delete(C_IChannelMember *this_);
  /**
   * Gets the ID of the channel member.
   * @return The ID of the channel member.
   */
  const char *C_IChannelMember_getMemberId(C_IChannelMember *this_);
  /**
   * Gets the ID of the channel.
   * @return The ID of the channel.
   */
  const char *C_IChannelMember_getChannelId(C_IChannelMember *this_);
  /**
   * Releases the IChannelMember class.
   */
  void C_IChannelMember_release(C_IChannelMember *this_);
#pragma endregion C_IChannelMember

  /**
   * The IChannelAttributes class.
   */
  typedef void C_IChannelAttributes;
#pragma region C_IChannelAttributes
  /**
   * Creates an IChannelAttributes instance.
   * @return
   * - An IChannelAttributes instance, if the method call succeeds.
   */
  // note: undefined reference to `agora::rtm::IChannelAttributes::createChannelAttributes()'
  // may be deprecated?
  // C_IChannelAttributes *C_IChannelAttributes_createChannelAttributes();
  /**
   * Adds an attribute to a specified channel.
   *
   * @param key The pointer to the attribute key.
   * @param value The pointer to the attribute value.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannelAttributes_addAttribute(C_IChannelAttributes *this_, const char *key, const char *value);
  /**
   * Removes an attribute from the channel.
   * @param key The pointer to the attribute key.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannelAttributes_removeAttribute(C_IChannelAttributes *this_, const char *key);
  /**
   * Gets the size of the attributes.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannelAttributes_getAttributesSize(C_IChannelAttributes *this_);
  /**
   * Gets the channel attributes.
   * @param size The size of the channel attributes.
   * @param key The pointer to the key of each channel attribute.
   * @param value The pointer to the value of each channel attribute.
   */
  void C_IChannelAttributes_getAttributes(C_IChannelAttributes *this_, int size, char **key,
                                          char **value); // todo: discussion, how to traveral
  /**
   * Gets the value of a channel attribute using the attribute key.
   * @param key The pointer to the key of the channel attribute that you want to get.
   */
  const char *C_IChannelAttributes_getAttributeValue(C_IChannelAttributes *this_, const char *key);
  /**
   * Releases the IChannelAttributes instance.
   * @param
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannelAttributes_release(C_IChannelAttributes *this_);
#pragma endregion C_IChannelAttributes

  /**
   * The IChannelEventHandler class.
   */
  typedef void C_IChannelEventHandler;
#pragma region C_IChannelEventHandler
  void C_IChannelEventHandler_Delete(C_IChannelEventHandler *this_);
  /**
   * Occurs when the local user successfully joins a channel.
   */
  void C_IChannelEventHandler_onJoinSuccess(C_IChannelEventHandler *this_);
  /**
   * Occurs when the local user fails to join a channel.
   * @param errorCode The error code: #JOIN_CHANNEL_ERR.
   */
  void C_IChannelEventHandler_onJoinFailure(C_IChannelEventHandler *this_, enum C_JOIN_CHANNEL_ERR errorCode);
  /**
   * Occurs when the local user leaves a channel.
   * @param errorCode The error code. See #LEAVE_CHANNEL_ERR.
   */
  void C_IChannelEventHandler_onLeave(C_IChannelEventHandler *this_, enum C_LEAVE_CHANNEL_ERR errorCode);
  /**
   * Occurs when the local user receives a channel message.
   * @param message The pointer to the messsage: IMessage.
   */
  void C_IChannelEventHandler_onMessageReceived(C_IChannelEventHandler *this_, const char *userId, const C_IMessage *message);
  /**
   * Reports the state of the message sent by the local user.
   * @param messageId ID of the message.
   * @param state The state of the message: #CHANNEL_MESSAGE_STATE.
   */
  void C_IChannelEventHandler_onSendMessageState(C_IChannelEventHandler *this_, int64_t messageId, enum C_CHANNEL_MESSAGE_STATE state);
  /**
   Returns the result of the \ref agora::rtm::IChannel::sendMessage "sendMessage" method call.

    @param messageId The ID of the sent channel message.
    @param state The error codes. See #CHANNEL_MESSAGE_ERR_CODE.
    */
  void C_IChannelEventHandler_onSendMessageResult(C_IChannelEventHandler *this_, long long messageId, enum C_CHANNEL_MESSAGE_ERR_CODE state);
  /**
   * Occurs when another member joins the channel.
   * @param member The pointer to the member who joins the channel: IChannelMember.
   */
  void C_IChannelEventHandler_onMemberJoined(C_IChannelEventHandler *this_, C_IChannelMember *member);
  /**
   * Occurs when the other member leaves the channel.
   * @param member The pointer to the member who leaves the channel: IChannelMember.
   */
  void C_IChannelEventHandler_onMemberLeft(C_IChannelEventHandler *this_, C_IChannelMember *member);
  /**
   * Reports all the members in the channel.
   * @param members The pointer to each member in the channel: IChannelMember.
   * @param userCount The number of users in the channel.
   */
  void C_IChannelEventHandler_onMembersGotten(C_IChannelEventHandler *this_, C_IChannelMember **members, int userCount);
  /**
   * Occurs when the channel attributes are updated.
   * @param attributes The pointer to the current channel attributes: IChannelAttributes.
   */
  void C_IChannelEventHandler_onAttributesUpdated(C_IChannelEventHandler *this_, const C_IChannelAttributes *attributes);
  /**
   * Occurs when the local user calls updateAttributes().
   * @param requestId ID of the current attribute update request.
   * @param resCode The response code: #RESPONSE_CODE.
   */
  void C_IChannelEventHandler_onUpdateAttributesResponse(C_IChannelEventHandler *this_, int64_t requestId, enum C_RESPONSE_CODE resCode);
  /**
   * Occurs when the channel attributes are deleted.
   * @param attributes The pointer to the channel attributes that you want to remove: IChannelAttributes.
   */
  void C_IChannelEventHandler_onAttributesDeleted(C_IChannelEventHandler *this_, const C_IChannelAttributes *attributes);
  /**
   * Occurs when the local user calls deleteAttributes().
   * @param requestId ID of the current attribute delete request.
   * @param resCode The response code: #RESPONSE_CODE.
   */
  void C_IChannelEventHandler_onDeleteAttributesResponse(C_IChannelEventHandler *this_, int64_t requestId, enum C_RESPONSE_CODE resCode);
#pragma endregion C_IChannelEventHandler

  /**
   * The IChannel class.
   */
  typedef void C_IChannel;
#pragma region C_IChannel
  /**
   * Sets an event handler for IChannel.
   * @param eventHandler The pointer to the event handler of IChannel: IChannelEventHandler.
   */
  void C_IChannel_setEventHandler(C_IChannel *this_, C_IChannelEventHandler *eventHandler);
  /**
   * Joins the current channel.
   *
   * A successful method call triggers either onJoinSuccess() or onJoinFailure() on the local client, to report
   * the state of joining the channel.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannel_join(C_IChannel *this_);
  /**
   * Leaves the current channel.
   *
   * After the local user successfully leaves the channel, the SDK triggers the onLeave() on the local client.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannel_leave(C_IChannel *this_);
  /**
   * Sends a channel message.
   *
   * After you successfully send a channel message, all members in the channel receive the message in the
   * onMessageReceived() callback.
   * @param message The pointer to the channel message that you want to send: IMessage.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannel_sendMessage(C_IChannel *this_, const C_IMessage *message);
  /**
   * Updates the channel attributes.
   *
   * A successful method call triggers the onUpdateAttributesResponse() callback on the local client.
   * @param attributes The pointer to the channel attributes that you want to update: IChannelAttributes.
   * @param requestId ID of the current update request.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannel_updateAttributes(C_IChannel *this_, C_IChannelAttributes *attributes, int64_t *requestId);
  /**
   * Removes the channel attributes.
   *
   * A successful method call triggers the onDeleteAttributesResponse() callback on the local client.
   * @param attributes The pointer to the channel attributes that you want to remove: IChannelAttributes.
   * @param requestId ID of the current delete request.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannel_deleteAttributes(C_IChannel *this_, C_IChannelAttributes *attributes, int64_t *requestId);
  /**
   * Gets the current request ID.
   * @return
   * - The pointer to the request ID, if the method call succeeds.
   * - An empty pointer NULL, if the method call fails.
   */
  const char *C_IChannel_getId(C_IChannel *this_);

  // sync_call
  /**
   * Releases the IChannel instance.
   *
   * This is a synchronous method call, which means that the SDK reports the result of this method call
   * after the IChannel instance is successfully released.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IChannel_release(C_IChannel *this_);
#pragma endregion C_IChannel

  /**
   * The IRtmServiceEventHandler class.
   */
  typedef void C_IRtmServiceEventHandler;
#pragma region C_IRtmServiceEventHandler
  void C_IRtmServiceEventHandler_Delete(C_IRtmServiceEventHandler *this_);
  /**
   * Occurs when the user successfully logs in the RTM service.
   */
  void C_IRtmServiceEventHandler_onLoginSuccess(C_IRtmServiceEventHandler *this_);
  /**
   * Occurs when the user fails to log in the RTM service.
   * @param errorCode The reason for the login failure: #LOGIN_ERR_CODE.
   */
  void C_IRtmServiceEventHandler_onLoginFailure(C_IRtmServiceEventHandler *this_, enum C_LOGIN_ERR_CODE errorCode);
  /**
   * Occurs when the user successfully logs out of the RTM service.
   */
  void C_IRtmServiceEventHandler_onLogout(C_IRtmServiceEventHandler *this_);
  /**
   * Occurs when the connection state of the local user has changed.
   * @param state The current connection state: #CONNECTION_STATE.
   */
  void C_IRtmServiceEventHandler_onConnectionStateChanged(C_IRtmServiceEventHandler *this_, enum C_CONNECTION_STATE state);
  /**
   * Reports the state of sending a message.
   * @param messageId ID of the message.
   * @param state The current state of the message: #PEER_MESSAGE_STATE.
   */
  void C_IRtmServiceEventHandler_onSendMessageState(C_IRtmServiceEventHandler *this_, int64_t messageId, enum C_PEER_MESSAGE_STATE state);
  /**
   * Occurs when the local user receives a message from a remote user.
   * @param peerId ID of the remote user that sends the message.
   * @param message The pointer to the message: IMessage.
   */
  void C_IRtmServiceEventHandler_onMessageReceivedFromPeer(C_IRtmServiceEventHandler *this_, const char *peerId, const C_IMessage *message);
#pragma endregion C_IRtmServiceEventHandler

  /**
   * The IRtmService class.
   */
  typedef void C_IRtmService;
#pragma region C_IRtmService
  void C_IRtmService_Delete(C_IRtmService *this_);
  /**
   * Creates and gets an IRtmService instance.
   * @param appId The pointer to the app ID.
   * @param eventHandler The pointer to the IRtmServiceEventHandler object.
   * @param eventSpace The connection specific ID, used during report to argus.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IRtmService_initialize(C_IRtmService *this_, const char *appId, C_IRtmServiceEventHandler *eventHandler);
  /**
   * Releases the IRtmServiceEventHandler object.
   * @param eventHandler The pointer to the IRtmServiceEventHandler object.
   */
  void C_IRtmService_unregisterObserver(C_IRtmService *this_, C_IRtmServiceEventHandler *eventHandler);
  /**
   * Releases the IRtmService instance.
   * @param sync Determines whether to report the result of this method call synchronously.
   * - true: Report the result of this method call after the IRtmService instance is released.
   * - false: (Default) Report the result of this method call immediately, even when the IRtmService is not
   * released.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IRtmService_release(C_IRtmService *this_, bool sync);
  /**
   * Logs in the RTM service.
   *
   * @note
   * - If you login with the same user ID from a different instance, your previous login will be kicked.
   * - The call frequency limit of this method is 2 queries per second.
   * @param token The token used to log in the RTM service.
   * @param userId ID of the user logging in the RTM service.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IRtmService_login(C_IRtmService *this_, const char *token, const char *userId);
  /**
   * Logs out of the RTM service.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IRtmService_logout(C_IRtmService *this_);
  /**
   * Sends a peer message to a specified remote user.
   *
   * @param peerId The pointer to the ID of the remote user.
   * @param message The pointer to message: IMessage.
   * @return
   * - 0: Success.
   * - < 0: Failure.
   */
  int C_IRtmService_sendMessageToPeer(C_IRtmService *this_, const char *peerId, const C_IMessage *message);
  /**
   * Creates an RTM channel.
   *
   * @param channelId The unique channel name for an RTM session. Supported character scopes are:
   * - All lowercase English letters: a to z.
   * - All uppercase English letters: A to Z.
   * - All numeric characters: 0 to 9.
   * - The space character.
   * - Punctuation characters and other symbols, including: "!", "#", "$", "%", "&", "(", ")", "+", "-", ":", ";", "<", "=",
   * ".", ">", "?", "@", "[", "]", "^", "_", " {", "}", "|", "~", ","
   *
   * @param eventHandler The pointer to IChannelEventHandler.
   * @return
   * - The pointer to an IChannel instance, if the method call succeeds.
   * - An empty pointer NULL, if the method call fails.
   */
  C_IChannel *C_IRtmService_createChannel(C_IRtmService *this_, const char *channelId, C_IChannelEventHandler *eventHandler);
#pragma endregion C_IRtmService

#pragma endregion agora::rtm

#pragma endregion agora

#ifdef __cplusplus
}
#endif // __cplusplus

#endif // C_I_AGORA_RTM_SERVICE_H
