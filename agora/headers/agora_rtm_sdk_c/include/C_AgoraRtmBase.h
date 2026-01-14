#ifndef C_AGORA_RTM_BASE_H
#define C_AGORA_RTM_BASE_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C"
{
#endif // __cplusplus

#pragma region agora

#pragma region agora::rtm

#define C_DEFAULT_LOG_SIZE_IN_KB 1024

/**
 * Rtm link state.
 */
enum C_RTM_LINK_STATE {
  /**
   * The initial state.
   */
  RTM_LINK_STATE_IDLE = 0,
  /**
   * The SDK is connecting to the server.
   */
  RTM_LINK_STATE_CONNECTING = 1,
  /**
   * The SDK has connected to the server.
   */
  RTM_LINK_STATE_CONNECTED = 2,
  /**
   * The SDK is disconnected from the server.
   */
  RTM_LINK_STATE_DISCONNECTED = 3,
  /**
   * The SDK link is suspended.
   */
  RTM_LINK_STATE_SUSPENDED = 4,
  /**
   * The SDK is failed to connect to the server.
   */
  RTM_LINK_STATE_FAILED = 5,
};

/**
 * Rtm link operation.
 */
enum C_RTM_LINK_OPERATION {
  /**
   * Login.
   */
  RTM_LINK_OPERATION_LOGIN = 0,
  /**
   * Logout.
   */
  RTM_LINK_OPERATION_LOGOUT = 1,
  /**
   * Join
   */
  RTM_LINK_OPERATION_JOIN = 2,
  /**
   * Leave.
   */
  RTM_LINK_OPERATION_LEAVE = 3,
  /**
   * Server reject
   */
  RTM_LINK_OPERATION_SERVER_REJECT = 4,
  /**
   * Auto reconnect
   */
  RTM_LINK_OPERATION_AUTO_RECONNECT = 5,
  /**
   * Reconnected
   */
  RTM_LINK_OPERATION_RECONNECTED = 6,
  /**
   * Heartbeat lost
   */
  RTM_LINK_OPERATION_HEARTBEAT_LOST = 7,
  /**
   * Server timeout
   */
  RTM_LINK_OPERATION_SERVER_TIMEOUT = 8,
  /**
   * Network change
   */
  RTM_LINK_OPERATION_NETWORK_CHANGE = 9,
};

/**
 * Rtm link state change reason.
 */
enum C_RTM_LINK_STATE_CHANGE_REASON {
  /**
   * Unknown reason.
   */
  RTM_LINK_STATE_CHANGE_REASON_UNKNOWN = 0,
  /**
   * Login.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOGIN = 1,
  /**
   * Login success.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOGIN_SUCCESS = 2,
  /**
   * Login timeout.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOGIN_TIMEOUT = 3,
  /**
   * Login not authorized.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOGIN_NOT_AUTHORIZED = 4,
  /**
   * Login rejected.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOGIN_REJECTED = 5,
  /**
   * Re-login.
   */
  RTM_LINK_STATE_CHANGE_REASON_RELOGIN = 6,
  /**
   * Logout.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOGOUT = 7,
  /**
   * Auto reconnect.
   */
  RTM_LINK_STATE_CHANGE_REASON_AUTO_RECONNECT = 8,
  /**
   * Reconnect timeout.
   */
  RTM_LINK_STATE_CHANGE_REASON_RECONNECT_TIMEOUT = 9,
  /**
   * Reconnect success.
   */
  RTM_LINK_STATE_CHANGE_REASON_RECONNECT_SUCCESS = 10,
  /**
   * Join.
   */
  RTM_LINK_STATE_CHANGE_REASON_JOIN = 11,
  /**
   * Join success.
   */
  RTM_LINK_STATE_CHANGE_REASON_JOIN_SUCCESS = 12,
  /**
   * Join failed.
   */
  RTM_LINK_STATE_CHANGE_REASON_JOIN_FAILED = 13,
  /**
   * Rejoin.
   */
  RTM_LINK_STATE_CHANGE_REASON_REJOIN = 14,
  /**
   * Leave.
   */
  RTM_LINK_STATE_CHANGE_REASON_LEAVE = 15,
  /**
   * Invalid token.
   */
  RTM_LINK_STATE_CHANGE_REASON_INVALID_TOKEN = 16,
  /**
   * Token expired.
   */
  RTM_LINK_STATE_CHANGE_REASON_TOKEN_EXPIRED = 17,
  /**
   * Inconsistent app ID.
   */
  RTM_LINK_STATE_CHANGE_REASON_INCONSISTENT_APP_ID = 18,
  /**
   * Invalid channel name.
   */
  RTM_LINK_STATE_CHANGE_REASON_INVALID_CHANNEL_NAME = 19,
  /**
   * Invalid user ID.
   */
  RTM_LINK_STATE_CHANGE_REASON_INVALID_USER_ID = 20,
  /**
   * Not initialized.
   */
  RTM_LINK_STATE_CHANGE_REASON_NOT_INITIALIZED = 21,
  /**
   * Rtm service not connected.
   */
  RTM_LINK_STATE_CHANGE_REASON_RTM_SERVICE_NOT_CONNECTED = 22,
  /**
   * Channel instance exceed limitation.
   */
  RTM_LINK_STATE_CHANGE_REASON_CHANNEL_INSTANCE_EXCEED_LIMITATION = 23,
  /**
   * Operation rate exceed limitation.
   */
  RTM_LINK_STATE_CHANGE_REASON_OPERATION_RATE_EXCEED_LIMITATION = 24,
  /**
   * Channel in error state.
   */
  RTM_LINK_STATE_CHANGE_REASON_CHANNEL_IN_ERROR_STATE = 25,
  /**
   * Presence not connected.
   */
  RTM_LINK_STATE_CHANGE_REASON_PRESENCE_NOT_CONNECTED = 26,
  /**
   * Same UID login.
   */
  RTM_LINK_STATE_CHANGE_REASON_SAME_UID_LOGIN = 27,
  /**
   * Kicked out by server.
   */
  RTM_LINK_STATE_CHANGE_REASON_KICKED_OUT_BY_SERVER = 28,
  /**
   * Keep alive timeout.
   */
  RTM_LINK_STATE_CHANGE_REASON_KEEP_ALIVE_TIMEOUT = 29,
  /**
   * Connection error.
   */
  RTM_LINK_STATE_CHANGE_REASON_CONNECTION_ERROR = 30,
  /**
   * Presence not ready.
   */
  RTM_LINK_STATE_CHANGE_REASON_PRESENCE_NOT_READY = 31,
  /**
   * Network change.
   */
  RTM_LINK_STATE_CHANGE_REASON_NETWORK_CHANGE = 32,
  /**
   * Service not supported.
   */
  RTM_LINK_STATE_CHANGE_REASON_SERVICE_NOT_SUPPORTED = 33,
  /**
   * Stream channel not available.
   */
  RTM_LINK_STATE_CHANGE_REASON_STREAM_CHANNEL_NOT_AVAILABLE = 34,
  /**
   * storage not available.
   */
  RTM_LINK_STATE_CHANGE_REASON_STORAGE_NOT_AVAILABLE = 35,
  /**
   * Lock not available.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOCK_NOT_AVAILABLE = 36,
  /**
   * Login too frequent.
   */
  RTM_LINK_STATE_CHANGE_REASON_LOGIN_TOO_FREQUENT = 37,
};

/**
 * Rtm protocol type for underlying connection.
 */
enum C_RTM_PROTOCOL_TYPE {
    /**
     * TCP and UDP (default).
     */
    RTM_PROTOCOL_TYPE_TCP_UDP = 0,
    /**
     * Use TCP only.
     */
    RTM_PROTOCOL_TYPE_TCP_ONLY = 1,
};

/**
 * Rtm service type.
 */
enum C_RTM_SERVICE_TYPE {
    /**
     * The type of rtm service not specified.
     */
    RTM_SERVICE_TYPE_NONE = 0x00000000,
    /**
     * The basic functionality of rtm service.
     */
    RTM_SERVICE_TYPE_MESSAGE = 0x00000001,
    /**
     * The advanced functionality of rtm service.
     */
    RTM_SERVICE_TYPE_STREAM = 0x00000002,
};

/**
 * IP areas.
 */
enum C_RTM_AREA_CODE
{
  /**
   * Mainland China.
   */
  RTM_AREA_CODE_CN = 0x00000001,
  /**
   * North America.
   */
  RTM_AREA_CODE_NA = 0x00000002,
  /**
   * Europe.
   */
  RTM_AREA_CODE_EU = 0x00000004,
  /**
   * Asia, excluding Mainland China.
   */
  RTM_AREA_CODE_AS = 0x00000008,
  /**
   * Japan.
   */
  RTM_AREA_CODE_JP = 0x00000010,
  /**
   * India.
   */
  RTM_AREA_CODE_IN = 0x00000020,
  /**
   * (Default) Global.
   */
  RTM_AREA_CODE_GLOB = (0xFFFFFFFF)
};

/**
 * The log level for rtm sdk.
 */
enum C_RTM_LOG_LEVEL
{
  /**
   * 0x0000: No logging.
   */
  RTM_LOG_LEVEL_NONE = 0x0000,
  /**
   * 0x0001: Informational messages.
   */
  RTM_LOG_LEVEL_INFO = 0x0001,
  /**
   * 0x0002: Warnings.
   */
  RTM_LOG_LEVEL_WARN = 0x0002,
  /**
   * 0x0004: Errors.
   */
  RTM_LOG_LEVEL_ERROR = 0x0004,
  /**
   * 0x0008: Critical errors that may lead to program termination.
   */
  RTM_LOG_LEVEL_FATAL = 0x0008,
};

/**
 * The encryption mode.
 */
enum C_RTM_ENCRYPTION_MODE
{
  /**
   * Disable message encryption.
   */
  RTM_ENCRYPTION_MODE_NONE = 0,
  /**
   * 128-bit AES encryption, GCM mode.
   */
  RTM_ENCRYPTION_MODE_AES_128_GCM = 1,
  /**
   * 256-bit AES encryption, GCM mode.
   */
  RTM_ENCRYPTION_MODE_AES_256_GCM = 2,
};

/**
 * The error codes of rtm client.
 */
enum C_RTM_ERROR_CODE
{
  /**
   * 0: No error occurs.
   */
  RTM_ERROR_OK = 0,

  /**
   * -10001 ~ -11000 : reserved for generic error.
   * -10001: The SDK is not initialized.
   */
  RTM_ERROR_NOT_INITIALIZED = -10001,
  /**
   * -10002: The user didn't login the RTM system.
   */
  RTM_ERROR_NOT_LOGIN = -10002,
  /**
   * -10003: The app ID is invalid.
   */
  RTM_ERROR_INVALID_APP_ID = -10003,
  /**
   * -10004: The event handler is invalid.
   */
  RTM_ERROR_INVALID_EVENT_HANDLER = -10004,
  /**
   * -10005: The token is invalid.
   */
  RTM_ERROR_INVALID_TOKEN = -10005,
  /**
   * -10006: The user ID is invalid.
   */
  RTM_ERROR_INVALID_USER_ID = -10006,
  /**
   * -10007: The service is not initialized.
   */
  RTM_ERROR_INIT_SERVICE_FAILED = -10007,
  /**
   * -10008: The channel name is invalid.
   */
  RTM_ERROR_INVALID_CHANNEL_NAME = -10008,
  /**
   * -10009: The token has expired.
   */
  RTM_ERROR_TOKEN_EXPIRED = -10009,
  /**
   * -10010: There is no server resources now.
   */
  RTM_ERROR_LOGIN_NO_SERVER_RESOURCES = -10010,
  /**
   * -10011: The login timeout.
   */
  RTM_ERROR_LOGIN_TIMEOUT = -10011,
  /**
   * -10012: The login is rejected by server.
   */
  RTM_ERROR_LOGIN_REJECTED = -10012,
  /**
   * -10013: The login is aborted due to unrecoverable error.
   */
  RTM_ERROR_LOGIN_ABORTED = -10013,
  /**
   * -10014: The parameter is invalid.
   */
  RTM_ERROR_INVALID_PARAMETER = -10014,
  /**
   * -10015: The login is not authorized. Happens user login the RTM system without granted from console.
   */
  RTM_ERROR_LOGIN_NOT_AUTHORIZED = -10015,
  /**
   * -10016: Try to login or join with inconsistent app ID.
   */
  RTM_ERROR_INCONSISTENT_APPID = -10016,
  /**
   * -10017: Already call same request.
   */
  RTM_ERROR_DUPLICATE_OPERATION = -10017,
  /**
   * -10018: Already call destroy or release, this instance is forbidden to call any api, please create new instance.
   */
  RTM_ERROR_INSTANCE_ALREADY_RELEASED = -10018,
  /**
   * -10019: The channel type is invalid.
   */
  RTM_ERROR_INVALID_CHANNEL_TYPE = -10019,
  /**
   * -10020: The encryption parameter is invalid.
   */
  RTM_ERROR_INVALID_ENCRYPTION_PARAMETER = -10020,
  /**
   * -10021: The operation is too frequent.
   */
  RTM_ERROR_OPERATION_RATE_EXCEED_LIMITATION = -10021,

  /**
   * -11001 ~ -12000 : reserved for channel error.
   * -11001: The user has not joined the channel.
   */
  RTM_ERROR_CHANNEL_NOT_JOINED = -11001,
  /**
   * -11002: The user has not subscribed the channel.
   */
  RTM_ERROR_CHANNEL_NOT_SUBSCRIBED = -11002,
  /**
   * -11003: The topic member count exceeds the limit.
   */
  RTM_ERROR_CHANNEL_EXCEED_TOPIC_USER_LIMITATION = -11003,
  /**
   * -11004: The channel is reused in RTC.
   */
  RTM_ERROR_CHANNEL_IN_REUSE = -11004,
  /**
   * -11005: The channel instance count exceeds the limit.
   */
  RTM_ERROR_CHANNEL_INSTANCE_EXCEED_LIMITATION = -11005,
  /**
   * -11006: The channel is in error state.
   */
  RTM_ERROR_CHANNEL_IN_ERROR_STATE = -11006,
  /**
   * -11007: The channel join failed.
   */
  RTM_ERROR_CHANNEL_JOIN_FAILED = -11007,
  /**
   * -11008: The topic name is invalid.
   */
  RTM_ERROR_CHANNEL_INVALID_TOPIC_NAME = -11008,
  /**
   * -11009: The message is invalid.
   */
  RTM_ERROR_CHANNEL_INVALID_MESSAGE = -11009,
  /**
   * -11010: The message length exceeds the limit.
   */
  RTM_ERROR_CHANNEL_MESSAGE_LENGTH_EXCEED_LIMITATION = -11010,
  /**
   * -11011: The user list is invalid.
   */
  RTM_ERROR_CHANNEL_INVALID_USER_LIST = -11011,
  /**
   * -11012: The stream channel is not available.
   */
  RTM_ERROR_CHANNEL_NOT_AVAILABLE = -11012,
  /**
   * -11013: The topic is not subscribed.
   */
  RTM_ERROR_CHANNEL_TOPIC_NOT_SUBSCRIBED = -11013,
  /**
   * -11014: The topic count exceeds the limit.
   */
  RTM_ERROR_CHANNEL_EXCEED_TOPIC_LIMITATION = -11014,
  /**
   * -11015: Join topic failed.
   */
  RTM_ERROR_CHANNEL_JOIN_TOPIC_FAILED = -11015,
  /**
   * -11016: The topic is not joined.
   */
  RTM_ERROR_CHANNEL_TOPIC_NOT_JOINED = -11016,
  /**
   * -11017: The topic does not exist.
   */
  RTM_ERROR_CHANNEL_TOPIC_NOT_EXIST = -11017,
  /**
   * -11018: The topic meta is invalid.
   */
  RTM_ERROR_CHANNEL_INVALID_TOPIC_META = -11018,
  /**
   * -11019: Subscribe channel timeout.
   */
  RTM_ERROR_CHANNEL_SUBSCRIBE_TIMEOUT = -11019,
  /**
   * -11020: Subscribe channel too frequent.
   */
  RTM_ERROR_CHANNEL_SUBSCRIBE_TOO_FREQUENT = -11020,
  /**
   * -11021: Subscribe channel failed.
   */
  RTM_ERROR_CHANNEL_SUBSCRIBE_FAILED = -11021,
  /**
   * -11022: Unsubscribe channel failed.
   */
  RTM_ERROR_CHANNEL_UNSUBSCRIBE_FAILED = -11022,
  /**
   * -11023: Encrypt message failed.
   */
  RTM_ERROR_CHANNEL_ENCRYPT_MESSAGE_FAILED = -11023,
  /**
   * -11024: Publish message failed.
   */
  RTM_ERROR_CHANNEL_PUBLISH_MESSAGE_FAILED = -11024,
  /**
   * -11025: Publish message too frequent.
   */
  RTM_ERROR_CHANNEL_PUBLISH_MESSAGE_TOO_FREQUENT = -11025,
  /**
   * -11026: Publish message timeout.
   */
  RTM_ERROR_CHANNEL_PUBLISH_MESSAGE_TIMEOUT = -11026,
  /**
   * -11027: The connection state is invalid.
   */
  RTM_ERROR_CHANNEL_NOT_CONNECTED = -11027,
  /**
   * -11028: Leave channel failed.
   */
  RTM_ERROR_CHANNEL_LEAVE_FAILED = -11028,
  /**
   * -11029: The custom type length exceeds the limit.
   */
  RTM_ERROR_CHANNEL_CUSTOM_TYPE_LENGTH_OVERFLOW = -11029,
  /**
   * -11030: The custom type is invalid.
   */
  RTM_ERROR_CHANNEL_INVALID_CUSTOM_TYPE = -11030,
  /**
   * -11031: unsupported message type (in MacOS/iOS platform，message only support NSString and NSData)
   */
  RTM_ERROR_CHANNEL_UNSUPPORTED_MESSAGE_TYPE = -11031,
  /**
   * -11032: The channel presence is not ready.
   */
  RTM_ERROR_CHANNEL_PRESENCE_NOT_READY = -11032,
  /**
   * -11033: The destination user of publish message is offline.
   */
  RTM_ERROR_CHANNEL_RECEIVER_OFFLINE = -11033,

  /**
   * -12001 ~ -13000 : reserved for storage error.
   * -12001: The storage operation failed.
   */
  RTM_ERROR_STORAGE_OPERATION_FAILED = -12001,
  /**
   * -12002: The metadata item count exceeds the limit.
   */
  RTM_ERROR_STORAGE_METADATA_ITEM_EXCEED_LIMITATION = -12002,
  /**
   * -12003: The metadata item is invalid.
   */
  RTM_ERROR_STORAGE_INVALID_METADATA_ITEM = -12003,
  /**
   * -12004: The argument in storage operation is invalid.
   */
  RTM_ERROR_STORAGE_INVALID_ARGUMENT = -12004,
  /**
   * -12005: The revision in storage operation is invalid.
   */
  RTM_ERROR_STORAGE_INVALID_REVISION = -12005,
  /**
   * -12006: The metadata length exceeds the limit.
   */
  RTM_ERROR_STORAGE_METADATA_LENGTH_OVERFLOW = -12006,
  /**
   * -12007: The lock name in storage operation is invalid.
   */
  RTM_ERROR_STORAGE_INVALID_LOCK_NAME = -12007,
  /**
   * -12008: The lock in storage operation is not acquired.
   */
  RTM_ERROR_STORAGE_LOCK_NOT_ACQUIRED = -12008,
  /**
   * -12009: The metadata key is invalid.
   */
  RTM_ERROR_STORAGE_INVALID_KEY = -12009,
  /**
   * -12010: The metadata value is invalid.
   */
  RTM_ERROR_STORAGE_INVALID_VALUE = -12010,
  /**
   * -12011: The metadata key length exceeds the limit.
   */
  RTM_ERROR_STORAGE_KEY_LENGTH_OVERFLOW = -12011,
  /**
   * -12012: The metadata value length exceeds the limit.
   */
  RTM_ERROR_STORAGE_VALUE_LENGTH_OVERFLOW = -12012,
  /**
   * -12013: The metadata key already exists.
   */
  RTM_ERROR_STORAGE_DUPLICATE_KEY = -12013,
  /**
   * -12014: The revision in storage operation is outdated.
   */
  RTM_ERROR_STORAGE_OUTDATED_REVISION = -12014,
  /**
   * -12015: The storage operation performed without subscribing.
   */
  RTM_ERROR_STORAGE_NOT_SUBSCRIBE = -12015,
  /**
   * -12016: The metadata item is invalid.
   */
  RTM_ERROR_STORAGE_INVALID_METADATA_INSTANCE = -12016,
  /**
   * -12017: The user count exceeds the limit when try to subscribe.
   */
  RTM_ERROR_STORAGE_SUBSCRIBE_USER_EXCEED_LIMITATION = -12017,
  /**
   * -12018: The storage operation timeout.
   */
  RTM_ERROR_STORAGE_OPERATION_TIMEOUT = -12018,
  /**
   * -12019: The storage service not available.
   */
  RTM_ERROR_STORAGE_NOT_AVAILABLE = -12019,

  /**
   * -13001 ~ -14000 : reserved for presence error.
   * -13001: The user is not connected.
   */
  RTM_ERROR_PRESENCE_NOT_CONNECTED = -13001,
  /**
   * -13002: The presence is not writable.
   */
  RTM_ERROR_PRESENCE_NOT_WRITABLE = -13002,
  /**
   * -13003: The argument in presence operation is invalid.
   */
  RTM_ERROR_PRESENCE_INVALID_ARGUMENT = -13003,
  /**
   * -13004: The cached presence state count exceeds the limit.
   */
  RTM_ERROR_PRESENCE_CACHED_TOO_MANY_STATES = -13004,
  /**
   * -13005: The state count exceeds the limit.
   */
  RTM_ERROR_PRESENCE_STATE_COUNT_OVERFLOW = -13005,
  /**
   * -13006: The state key is invalid.
   */
  RTM_ERROR_PRESENCE_INVALID_STATE_KEY = -13006,
  /**
   * -13007: The state value is invalid.
   */
  RTM_ERROR_PRESENCE_INVALID_STATE_VALUE = -13007,
  /**
   * -13008: The state key length exceeds the limit.
   */
  RTM_ERROR_PRESENCE_STATE_KEY_SIZE_OVERFLOW = -13008,
  /**
   * -13009: The state value length exceeds the limit.
   */
  RTM_ERROR_PRESENCE_STATE_VALUE_SIZE_OVERFLOW = -13009,
  /**
   * -13010: The state key already exists.
   */
  RTM_ERROR_PRESENCE_STATE_DUPLICATE_KEY = -13010,
  /**
   * -13011: The user is not exist.
   */
  RTM_ERROR_PRESENCE_USER_NOT_EXIST = -13011,
  /**
   * -13012: The presence operation timeout.
   */
  RTM_ERROR_PRESENCE_OPERATION_TIMEOUT = -13012,
  /**
   * -13013: The presence operation failed.
   */
  RTM_ERROR_PRESENCE_OPERATION_FAILED = -13013,

  /**
   * -14001 ~ -15000 : reserved for lock error.
   * -14001: The lock operation failed.
   */
  RTM_ERROR_LOCK_OPERATION_FAILED = -14001,
  /**
   * -14002: The lock operation timeout.
   */
  RTM_ERROR_LOCK_OPERATION_TIMEOUT = -14002,
  /**
   * -14003: The lock operation is performing.
   */
  RTM_ERROR_LOCK_OPERATION_PERFORMING = -14003,
  /**
   * -14004: The lock already exists.
   */
  RTM_ERROR_LOCK_ALREADY_EXIST = -14004,
  /**
   * -14005: The lock name is invalid.
   */
  RTM_ERROR_LOCK_INVALID_NAME = -14005,
  /**
   * -14006: The lock is not acquired.
   */
  RTM_ERROR_LOCK_NOT_ACQUIRED = -14006,
  /**
   * -14007: Acquire lock failed.
   */
  RTM_ERROR_LOCK_ACQUIRE_FAILED = -14007,
  /**
   * -14008: The lock is not exist.
   */
  RTM_ERROR_LOCK_NOT_EXIST = -14008,
  /**
   * -14009: The lock service is not available.
   */
  RTM_ERROR_LOCK_NOT_AVAILABLE = -14009,
};

/**
 * Connection states between rtm sdk and agora server.
 */
enum C_RTM_CONNECTION_STATE
{
  /**
   * 1: The SDK is disconnected with server.
   */
  RTM_CONNECTION_STATE_DISCONNECTED = 1,
  /**
   * 2: The SDK is connecting to the server.
   */
  RTM_CONNECTION_STATE_CONNECTING = 2,
  /**
   * 3: The SDK is connected to the server and has joined a channel. You can now publish or subscribe to
   * a track in the channel.
   */
  RTM_CONNECTION_STATE_CONNECTED = 3,
  /**
   * 4: The SDK keeps rejoining the channel after being disconnected from the channel, probably because of
   * network issues.
   */
  RTM_CONNECTION_STATE_RECONNECTING = 4,
  /**
   * 5: The SDK fails to connect to the server or join the channel.
   */
  RTM_CONNECTION_STATE_FAILED = 5,
};

/**
 * Reasons for connection state change.
 */
enum C_RTM_CONNECTION_CHANGE_REASON
{
  /**
   * 0: The SDK is connecting to the server.
   */
  RTM_CONNECTION_CHANGED_CONNECTING = 0,
  /**
   * 1: The SDK has joined the channel successfully.
   */
  RTM_CONNECTION_CHANGED_JOIN_SUCCESS = 1,
  /**
   * 2: The connection between the SDK and the server is interrupted.
   */
  RTM_CONNECTION_CHANGED_INTERRUPTED = 2,
  /**
   * 3: The connection between the SDK and the server is banned by the server.
   */
  RTM_CONNECTION_CHANGED_BANNED_BY_SERVER = 3,
  /**
   * 4: The SDK fails to join the channel for more than 20 minutes and stops reconnecting to the channel.
   */
  RTM_CONNECTION_CHANGED_JOIN_FAILED = 4,
  /**
   * 5: The SDK has left the channel.
   */
  RTM_CONNECTION_CHANGED_LEAVE_CHANNEL = 5,
  /**
   * 6: The connection fails because the App ID is not valid.
   */
  RTM_CONNECTION_CHANGED_INVALID_APP_ID = 6,
  /**
   * 7: The connection fails because the channel name is not valid.
   */
  RTM_CONNECTION_CHANGED_INVALID_CHANNEL_NAME = 7,
  /**
   * 8: The connection fails because the token is not valid.
   */
  RTM_CONNECTION_CHANGED_INVALID_TOKEN = 8,
  /**
   * 9: The connection fails because the token has expired.
   */
  RTM_CONNECTION_CHANGED_TOKEN_EXPIRED = 9,
  /**
   * 10: The connection is rejected by the server.
   */
  RTM_CONNECTION_CHANGED_REJECTED_BY_SERVER = 10,
  /**
   * 11: The connection changes to reconnecting because the SDK has set a proxy server.
   */
  RTM_CONNECTION_CHANGED_SETTING_PROXY_SERVER = 11,
  /**
   * 12: When the connection state changes because the app has renewed the token.
   */
  RTM_CONNECTION_CHANGED_RENEW_TOKEN = 12,
  /**
   * 13: The IP Address of the app has changed. A change in the network type or IP/Port changes the IP
   * address of the app.
   */
  RTM_CONNECTION_CHANGED_CLIENT_IP_ADDRESS_CHANGED = 13,
  /**
   * 14: A timeout occurs for the keep-alive of the connection between the SDK and the server.
   */
  RTM_CONNECTION_CHANGED_KEEP_ALIVE_TIMEOUT = 14,
  /**
   * 15: The SDK has rejoined the channel successfully.
   */
  RTM_CONNECTION_CHANGED_REJOIN_SUCCESS = 15,
  /**
   * 16: The connection between the SDK and the server is lost.
   */
  RTM_CONNECTION_CHANGED_LOST = 16,
  /**
   * 17: The change of connection state is caused by echo test.
   */
  RTM_CONNECTION_CHANGED_ECHO_TEST = 17,
  /**
   * 18: The local IP Address is changed by user.
   */
  RTM_CONNECTION_CHANGED_CLIENT_IP_ADDRESS_CHANGED_BY_USER = 18,
  /**
   * 19: The connection is failed due to join the same channel on another device with the same uid.
   */
  RTM_CONNECTION_CHANGED_SAME_UID_LOGIN = 19,
  /**
   * 20: The connection is failed due to too many broadcasters in the channel.
   */
  RTM_CONNECTION_CHANGED_TOO_MANY_BROADCASTERS = 20,
  /**
   * 21: The connection is failed due to license validation failure.
   */
  RTM_CONNECTION_CHANGED_LICENSE_VALIDATION_FAILURE = 21,
  /**
   * 22: The connection is failed due to user vid not support stream channel.
   */
  RTM_CONNECTION_CHANGED_STREAM_CHANNEL_NOT_AVAILABLE = 22,
  /**
   * 23: The connection is failed due to token and appid inconsistent.
   */
  RTM_CONNECTION_CHANGED_INCONSISTENT_APPID = 23,
  /**
   * 10001: The connection of rtm edge service has been successfully established.
   */
  RTM_CONNECTION_CHANGED_LOGIN_SUCCESS = 10001,
  /**
   * 10002: User log out Agora RTM system.
   */
  RTM_CONNECTION_CHANGED_LOGOUT = 10002,
  /**
   * 10003: User log out Agora RTM system.
   */
  RTM_CONNECTION_CHANGED_PRESENCE_NOT_READY = 10003,
};

/**
 * RTM channel type.
 */
enum C_RTM_CHANNEL_TYPE
{
  /**
   * 0: Unknown channel type.
   */
  RTM_CHANNEL_TYPE_NONE = 0,
  /**
   * 1: Message channel.
   */
  RTM_CHANNEL_TYPE_MESSAGE = 1,
  /**
   * 2: Stream channel.
   */
  RTM_CHANNEL_TYPE_STREAM = 2,
  /**
   * 3: User.
   */
  RTM_CHANNEL_TYPE_USER = 3,
};

/**
  @brief Message type when user publish message to channel or topic
  */
enum C_RTM_MESSAGE_TYPE
{
  /**
    0: The binary message.
    */
  RTM_MESSAGE_TYPE_BINARY = 0,
  /**
    1: The ascii message.
    */
  RTM_MESSAGE_TYPE_STRING = 1,
};

/**
  @brief Storage type indicate the storage event was triggered by user or channel
  */
enum C_RTM_STORAGE_TYPE
{
  /**
    0: Unknown type.
    */
  RTM_STORAGE_TYPE_NONE = 0,
  /**
    1: The user storage event.
    */
  RTM_STORAGE_TYPE_USER = 1,
  /**
    2: The channel storage event.
    */
  RTM_STORAGE_TYPE_CHANNEL = 2,
};

/**
 * The storage event type, indicate storage operation
 */
enum C_RTM_STORAGE_EVENT_TYPE
{
  /**
    0: Unknown event type.
    */
  RTM_STORAGE_EVENT_TYPE_NONE = 0,
  /**
    1: Triggered when user subscribe user metadata state or join channel with options.withMetadata = true
    */
  RTM_STORAGE_EVENT_TYPE_SNAPSHOT = 1,
  /**
    2: Triggered when a remote user set metadata
    */
  RTM_STORAGE_EVENT_TYPE_SET = 2,
  /**
    3: Triggered when a remote user update metadata
    */
  RTM_STORAGE_EVENT_TYPE_UPDATE = 3,
  /**
    4: Triggered when a remote user remove metadata
    */
  RTM_STORAGE_EVENT_TYPE_REMOVE = 4,
};

/**
 * The lock event type, indicate lock operation
 */
enum C_RTM_LOCK_EVENT_TYPE
{
  /**
   * 0: Unknown event type
   */
  RTM_LOCK_EVENT_TYPE_NONE = 0,
  /**
   * 1: Triggered when user subscribe lock state
   */
  RTM_LOCK_EVENT_TYPE_SNAPSHOT = 1,
  /**
   * 2: Triggered when a remote user set lock
   */
  RTM_LOCK_EVENT_TYPE_LOCK_SET = 2,
  /**
   * 3: Triggered when a remote user remove lock
   */
  RTM_LOCK_EVENT_TYPE_LOCK_REMOVED = 3,
  /**
   * 4: Triggered when a remote user acquired lock
   */
  RTM_LOCK_EVENT_TYPE_LOCK_ACQUIRED = 4,
  /**
   * 5: Triggered when a remote user released lock
   */
  RTM_LOCK_EVENT_TYPE_LOCK_RELEASED = 5,
  /**
   * 6: Triggered when user reconnect to rtm service,
   * detect the lock has been acquired and released by others.
   */
  RTM_LOCK_EVENT_TYPE_LOCK_EXPIRED = 6,
};

/**
 * The proxy type
 */
enum C_RTM_PROXY_TYPE
{
  /**
   * 0: Link without proxy
   */
  RTM_PROXY_TYPE_NONE = 0,
  /**
   * 1: Link with http proxy
   */
  RTM_PROXY_TYPE_HTTP = 1,
  /**
   * 2: Link with tcp cloud proxy
   */
  RTM_PROXY_TYPE_CLOUD_TCP = 2,
};

/**
  @brief Topic event type
  */
enum C_RTM_TOPIC_EVENT_TYPE
{
  /**
   * 0: Unknown event type
   */
  RTM_TOPIC_EVENT_TYPE_NONE = 0,
  /**
   * 1: The topic snapshot of this channel
   */
  RTM_TOPIC_EVENT_TYPE_SNAPSHOT = 1,
  /**
   * 2: Triggered when remote user join a topic
   */
  RTM_TOPIC_EVENT_TYPE_REMOTE_JOIN_TOPIC = 2,
  /**
   * 3: Triggered when remote user leave a topic
   */
  RTM_TOPIC_EVENT_TYPE_REMOTE_LEAVE_TOPIC = 3,
};

/**
  @brief Presence event type
  */
enum C_RTM_PRESENCE_EVENT_TYPE
{
  /**
   * 0: Unknown event type
   */
  RTM_PRESENCE_EVENT_TYPE_NONE = 0,
  /**
   * 1: The presence snapshot of this channel
   */
  RTM_PRESENCE_EVENT_TYPE_SNAPSHOT = 1,
  /**
   * 2: The presence event triggered in interval mode
   */
  RTM_PRESENCE_EVENT_TYPE_INTERVAL = 2,
  /**
   * 3: Triggered when remote user join channel
   */
  RTM_PRESENCE_EVENT_TYPE_REMOTE_JOIN_CHANNEL = 3,
  /**
   * 4: Triggered when remote user leave channel
   */
  RTM_PRESENCE_EVENT_TYPE_REMOTE_LEAVE_CHANNEL = 4,
  /**
   * 5: Triggered when remote user's connection timeout
   */
  RTM_PRESENCE_EVENT_TYPE_REMOTE_TIMEOUT = 5,
  /**
   * 6: Triggered when user changed state
   */
  RTM_PRESENCE_EVENT_TYPE_REMOTE_STATE_CHANGED = 6,
  /**
   * 7: Triggered when user joined channel without presence service
   */
  RTM_PRESENCE_EVENT_TYPE_ERROR_OUT_OF_SERVICE = 7,
};

/**
 * Definition of LogConfiguration
 */
struct C_RtmLogConfig
{
  /**
   * The log file path, default is NULL for default log path
   */
  const char *filePath;
  /**
   * The log file size, KB , set 1024KB to use default log size
   */
  uint32_t fileSizeInKB;
  /**
   *  The log level, set LOG_LEVEL_INFO to use default log level
   */
  enum C_RTM_LOG_LEVEL level;
};
struct C_RtmLogConfig *C_RtmLogConfig_New();
void C_RtmLogConfig_Delete(struct C_RtmLogConfig *this_);

/**
 * User list.
 */
struct C_UserList
{
  /**
   * The list of users.
   */
  const char **users;
  /**
   * The number of users.
   */
  size_t userCount;
};
struct C_UserList *C_UserList_New();
void C_UserList_Delete(struct C_UserList *this_);

/**
  @brief Topic publisher information
  */
struct C_PublisherInfo
{
  /**
   * The publisher user ID
   */
  const char *publisherUserId;
  /**
   * The metadata of the publisher
   */
  const char *publisherMeta;
};
struct C_PublisherInfo *C_PublisherInfo_New();
void C_PublisherInfo_Delete(struct C_PublisherInfo *this_);

/**
  @brief Topic information
  */
struct C_TopicInfo
{
  /**
   * The name of the topic
   */
  const char *topic;
  /**
   * The publisher array
   */
  struct C_PublisherInfo *publishers;
  /**
   * The count of publisher in current topic
   */
  size_t publisherCount;
};
struct C_TopicInfo *C_TopicInfo_New();
void C_TopicInfo_Delete(struct C_TopicInfo *this_);

/**
  @brief User state property
  */
struct C_StateItem
{
  /**
   * The key of the state item.
   */
  const char *key;
  /**
   * The value of the state item.
   */
  const char *value;
};
struct C_StateItem *C_StateItem_New();
void C_StateItem_Delete(struct C_StateItem *this_);

/**
 *  The information of a Lock.
 */
struct C_LockDetail
{
  /**
   * The name of the lock.
   */
  const char *lockName;
  /**
   * The owner of the lock. Only valid when user getLocks or receive LockEvent with RTM_LOCK_EVENT_TYPE_SNAPSHOT
   */
  const char *owner;
  /**
   * The ttl of the lock.
   */
  uint32_t ttl;
};
struct C_LockDetail *C_LockDetail_New();
void C_LockDetail_Delete(struct C_LockDetail *this_);

/**
 *  The states of user.
 */
struct C_UserState
{
  /**
   * The user id.
   */
  const char *userId;
  /**
   * The user states.
   */
  const struct C_StateItem *states;
  /**
   * The count of user states.
   */
  size_t statesCount;
};
struct C_UserState *C_UserState_New();
void C_UserState_Delete(struct C_UserState *this_);

/**
 *  The subscribe option.
 */
struct C_SubscribeOptions
{
  /**
   * Whether to subscribe channel with message
   */
  bool withMessage;
  /**
   * Whether to subscribe channel with metadata
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
   * Whether to subscribe channel in quiet mode
   * Quiet mode means remote user will not receive any notification when we subscribe or
   * unsubscribe or change our presence state
   */
  bool beQuiet;
};
struct C_SubscribeOptions *C_SubscribeOptions_New();
void C_SubscribeOptions_Delete(struct C_SubscribeOptions *this_);

/**
 *  The channel information.
 */
struct C_ChannelInfo
{
  /**
   * The channel which the message was published
   */
  const char *channelName;
  /**
   * Which channel type, RTM_CHANNEL_TYPE_STREAM or RTM_CHANNEL_TYPE_MESSAGE
   */
  enum C_RTM_CHANNEL_TYPE channelType;
};
struct C_ChannelInfo *C_ChannelInfo_New();
void C_ChannelInfo_Delete(struct C_ChannelInfo *this_);

/**
 *  The option to query user presence.
 */
struct C_PresenceOptions
{
  /**
   * Whether to display user id in query result
   */
  bool includeUserId;
  /**
   * Whether to display user state in query result
   */
  bool includeState;
  /**
   * The paging object used for pagination.
   */
  const char *page;
};
struct C_PresenceOptions *C_PresenceOptions_New();
void C_PresenceOptions_Delete(struct C_PresenceOptions *this_);

/**
 *  The option to query user presence.
 */
struct C_GetOnlineUsersOptions
{
  /**
   * Whether to display user id in query result
   */
  bool includeUserId;
  /**
   * Whether to display user state in query result
   */
  bool includeState;
  /**
   * The paging object used for pagination.
   */
  const char *page;
};
struct C_GetOnlineUsersOptions *C_GetOnlineUsersOptions_New();
void C_GetOnlineUsersOptions_Delete(struct C_GetOnlineUsersOptions *this_);

/**
  @brief Publish message option
  */
struct C_PublishOptions
{
 /*
  The channel type.
 */
 enum C_RTM_CHANNEL_TYPE channelType;
 /**
   The message type.
   */
 enum C_RTM_MESSAGE_TYPE messageType;
 /**
   The custom type of the message, up to 32 bytes for customize
   */
 const char *customType;
 /**
   Whether to store in history, true to enable
   */
  bool storeInHistory;
};
struct C_PublishOptions *C_PublishOptions_New();
void C_PublishOptions_Delete(struct C_PublishOptions *this_);

/**
  @brief topic message option
  */
struct C_TopicMessageOptions
{
  /**
    The message type.
    */
  enum C_RTM_MESSAGE_TYPE messageType;
  /**
    The time to calibrate data with media,
    only valid when user join topic with syncWithMedia in stream channel
    */
  uint64_t sendTs;
  /**
    The custom type of the message, up to 32 bytes for customize
    */
  const char *customType;
};
struct C_TopicMessageOptions *C_TopicMessageOptions_New();
void C_TopicMessageOptions_Delete(struct C_TopicMessageOptions *this_);

/**
  @brief Proxy configuration
  */
struct C_RtmProxyConfig
{
  /**
    The Proxy type.
    */
  enum C_RTM_PROXY_TYPE proxyType;
  /**
    The Proxy server address.
    */
  const char *server;
  /**
    The Proxy server port.
    */
  uint16_t port;
  /**
    The Proxy user account.
    */
  const char *account;
  /**
    The Proxy password.
    */
  const char *password;
};
struct C_RtmProxyConfig *C_RtmProxyConfig_New();
void C_RtmProxyConfig_Delete(struct C_RtmProxyConfig *this_);

/**
  @brief encryption configuration
  */
struct C_RtmEncryptionConfig
{
  /**
   * The encryption mode.
   */
  enum C_RTM_ENCRYPTION_MODE encryptionMode;

  /**
   * The encryption key in the string format.
   */
  const char *encryptionKey;

  /**
   * The encryption salt.
   */
  uint8_t encryptionSalt[32];
};
struct C_RtmEncryptionConfig *C_RtmEncryptionConfig_New();
void C_RtmEncryptionConfig_Delete(struct C_RtmEncryptionConfig *this_);

/**
 * @brief Private configuration
 */
struct C_RtmPrivateConfig {
    /**
     * Rtm service type.
     */
    enum C_RTM_SERVICE_TYPE serviceType;
    /**
     * Local access point hosts list.
     */
    const char** accessPointHosts;
    /**
     * The count of access point hosts list.
     */
    size_t accessPointHostsCount;
};

struct C_HistoryMessage {
  /**
   * Message type
   */
  enum C_RTM_MESSAGE_TYPE messageType;
  /**
   * The publisher
   */
  const char* publisher;
  /**
   * The payload
   */
  const char* message;
  /**
   * The payload length
   */
  size_t messageLength;
  /**
   * The custom type of the message
   */
  const char* customType;
  /**
   * Timestamp of the message received by rtm server
   */
  uint64_t timestamp;
};

#pragma endregion agora::rtm

#pragma endregion agora

#ifdef __cplusplus
}
#endif // __cplusplus

#endif // C_AGORA_RTM_BASE_H
