// Code generated from the pinned C_AgoraRtmBase.h enum declarations; DO NOT EDIT.
package messaging

import native "github.com/zyy17/agora-server-sdk/agora/rtm"

// TokenEventType preserves the native RTM_TOKEN_EVENT_TYPE enumeration.
type TokenEventType = native.RtmTokenEventType

const (
	// TokenEventWillExpire is RTM_TOKEN_EVENT_TYPE_WILL_EXPIRE.
	TokenEventWillExpire TokenEventType = native.RtmTokenEventTypeWillExpire
	// TokenEventReadPermissionRevoked is RTM_TOKEN_EVENT_TYPE_READ_PERMISSION_REVOKED.
	TokenEventReadPermissionRevoked TokenEventType = native.RtmTokenEventTypeReadPermissionRevoked
)

// LinkState preserves the native RTM_LINK_STATE enumeration.
type LinkState = native.RtmLinkState

const (
	// LinkStateIdle is RTM_LINK_STATE_IDLE.
	LinkStateIdle LinkState = native.RtmLinkStateIdle
	// LinkStateConnecting is RTM_LINK_STATE_CONNECTING.
	LinkStateConnecting LinkState = native.RtmLinkStateConnecting
	// LinkStateConnected is RTM_LINK_STATE_CONNECTED.
	LinkStateConnected LinkState = native.RtmLinkStateConnected
	// LinkStateDisconnected is RTM_LINK_STATE_DISCONNECTED.
	LinkStateDisconnected LinkState = native.RtmLinkStateDisconnected
	// LinkStateSuspended is RTM_LINK_STATE_SUSPENDED.
	LinkStateSuspended LinkState = native.RtmLinkStateSuspended
	// LinkStateFailed is RTM_LINK_STATE_FAILED.
	LinkStateFailed LinkState = native.RtmLinkStateFailed
)

// LinkOperation preserves the native RTM_LINK_OPERATION enumeration.
type LinkOperation = native.RtmLinkOperation

const (
	// LinkOperationLogin is RTM_LINK_OPERATION_LOGIN.
	LinkOperationLogin LinkOperation = native.RtmLinkOperationLogin
	// LinkOperationLogout is RTM_LINK_OPERATION_LOGOUT.
	LinkOperationLogout LinkOperation = native.RtmLinkOperationLogout
	// LinkOperationJoin is RTM_LINK_OPERATION_JOIN.
	LinkOperationJoin LinkOperation = native.RtmLinkOperationJoin
	// LinkOperationLeave is RTM_LINK_OPERATION_LEAVE.
	LinkOperationLeave LinkOperation = native.RtmLinkOperationLeave
	// LinkOperationServerReject is RTM_LINK_OPERATION_SERVER_REJECT.
	LinkOperationServerReject LinkOperation = native.RtmLinkOperationServerReject
	// LinkOperationAutoReconnect is RTM_LINK_OPERATION_AUTO_RECONNECT.
	LinkOperationAutoReconnect LinkOperation = native.RtmLinkOperationAutoReconnect
	// LinkOperationReconnected is RTM_LINK_OPERATION_RECONNECTED.
	LinkOperationReconnected LinkOperation = native.RtmLinkOperationReconnected
	// LinkOperationHeartbeatLost is RTM_LINK_OPERATION_HEARTBEAT_LOST.
	LinkOperationHeartbeatLost LinkOperation = native.RtmLinkOperationHeartbeatLost
	// LinkOperationServerTimeout is RTM_LINK_OPERATION_SERVER_TIMEOUT.
	LinkOperationServerTimeout LinkOperation = native.RtmLinkOperationServerTimeout
	// LinkOperationNetworkChange is RTM_LINK_OPERATION_NETWORK_CHANGE.
	LinkOperationNetworkChange LinkOperation = native.RtmLinkOperationNetworkChange
)

// LinkReason preserves the native RTM_LINK_STATE_CHANGE_REASON enumeration.
type LinkReason = native.RtmLinkStateChangeReason

const (
	// LinkReasonUnknown is RTM_LINK_STATE_CHANGE_REASON_UNKNOWN.
	LinkReasonUnknown LinkReason = native.RtmLinkStateChangeReasonUnknown
	// LinkReasonLogin is RTM_LINK_STATE_CHANGE_REASON_LOGIN.
	LinkReasonLogin LinkReason = native.RtmLinkStateChangeReasonLogin
	// LinkReasonLoginSuccess is RTM_LINK_STATE_CHANGE_REASON_LOGIN_SUCCESS.
	LinkReasonLoginSuccess LinkReason = native.RtmLinkStateChangeReasonLoginSuccess
	// LinkReasonLoginTimeout is RTM_LINK_STATE_CHANGE_REASON_LOGIN_TIMEOUT.
	LinkReasonLoginTimeout LinkReason = native.RtmLinkStateChangeReasonLoginTimeout
	// LinkReasonLoginNotAuthorized is RTM_LINK_STATE_CHANGE_REASON_LOGIN_NOT_AUTHORIZED.
	LinkReasonLoginNotAuthorized LinkReason = native.RtmLinkStateChangeReasonLoginNotAuthorized
	// LinkReasonLoginRejected is RTM_LINK_STATE_CHANGE_REASON_LOGIN_REJECTED.
	LinkReasonLoginRejected LinkReason = native.RtmLinkStateChangeReasonLoginRejected
	// LinkReasonRelogin is RTM_LINK_STATE_CHANGE_REASON_RELOGIN.
	LinkReasonRelogin LinkReason = native.RtmLinkStateChangeReasonRelogin
	// LinkReasonLogout is RTM_LINK_STATE_CHANGE_REASON_LOGOUT.
	LinkReasonLogout LinkReason = native.RtmLinkStateChangeReasonLogout
	// LinkReasonAutoReconnect is RTM_LINK_STATE_CHANGE_REASON_AUTO_RECONNECT.
	LinkReasonAutoReconnect LinkReason = native.RtmLinkStateChangeReasonAutoReconnect
	// LinkReasonReconnectTimeout is RTM_LINK_STATE_CHANGE_REASON_RECONNECT_TIMEOUT.
	LinkReasonReconnectTimeout LinkReason = native.RtmLinkStateChangeReasonReconnectTimeout
	// LinkReasonReconnectSuccess is RTM_LINK_STATE_CHANGE_REASON_RECONNECT_SUCCESS.
	LinkReasonReconnectSuccess LinkReason = native.RtmLinkStateChangeReasonReconnectSuccess
	// LinkReasonJoin is RTM_LINK_STATE_CHANGE_REASON_JOIN.
	LinkReasonJoin LinkReason = native.RtmLinkStateChangeReasonJoin
	// LinkReasonJoinSuccess is RTM_LINK_STATE_CHANGE_REASON_JOIN_SUCCESS.
	LinkReasonJoinSuccess LinkReason = native.RtmLinkStateChangeReasonJoinSuccess
	// LinkReasonJoinFailed is RTM_LINK_STATE_CHANGE_REASON_JOIN_FAILED.
	LinkReasonJoinFailed LinkReason = native.RtmLinkStateChangeReasonJoinFailed
	// LinkReasonRejoin is RTM_LINK_STATE_CHANGE_REASON_REJOIN.
	LinkReasonRejoin LinkReason = native.RtmLinkStateChangeReasonRejoin
	// LinkReasonLeave is RTM_LINK_STATE_CHANGE_REASON_LEAVE.
	LinkReasonLeave LinkReason = native.RtmLinkStateChangeReasonLeave
	// LinkReasonInvalidToken is RTM_LINK_STATE_CHANGE_REASON_INVALID_TOKEN.
	LinkReasonInvalidToken LinkReason = native.RtmLinkStateChangeReasonInvalidToken
	// LinkReasonTokenExpired is RTM_LINK_STATE_CHANGE_REASON_TOKEN_EXPIRED.
	LinkReasonTokenExpired LinkReason = native.RtmLinkStateChangeReasonTokenExpired
	// LinkReasonInconsistentAppID is RTM_LINK_STATE_CHANGE_REASON_INCONSISTENT_APP_ID.
	LinkReasonInconsistentAppID LinkReason = native.RtmLinkStateChangeReasonInconsistentAppID
	// LinkReasonInvalidChannelName is RTM_LINK_STATE_CHANGE_REASON_INVALID_CHANNEL_NAME.
	LinkReasonInvalidChannelName LinkReason = native.RtmLinkStateChangeReasonInvalidChannelName
	// LinkReasonInvalidUserID is RTM_LINK_STATE_CHANGE_REASON_INVALID_USER_ID.
	LinkReasonInvalidUserID LinkReason = native.RtmLinkStateChangeReasonInvalidUserID
	// LinkReasonNotInitialized is RTM_LINK_STATE_CHANGE_REASON_NOT_INITIALIZED.
	LinkReasonNotInitialized LinkReason = native.RtmLinkStateChangeReasonNotInitialized
	// LinkReasonRTMServiceNotConnected is RTM_LINK_STATE_CHANGE_REASON_RTM_SERVICE_NOT_CONNECTED.
	LinkReasonRTMServiceNotConnected LinkReason = native.RtmLinkStateChangeReasonRTMServiceNotConnected
	// LinkReasonChannelInstanceExceedLimitation is RTM_LINK_STATE_CHANGE_REASON_CHANNEL_INSTANCE_EXCEED_LIMITATION.
	LinkReasonChannelInstanceExceedLimitation LinkReason = native.RtmLinkStateChangeReasonChannelInstanceExceedLimitation
	// LinkReasonOperationRateExceedLimitation is RTM_LINK_STATE_CHANGE_REASON_OPERATION_RATE_EXCEED_LIMITATION.
	LinkReasonOperationRateExceedLimitation LinkReason = native.RtmLinkStateChangeReasonOperationRateExceedLimitation
	// LinkReasonChannelInErrorState is RTM_LINK_STATE_CHANGE_REASON_CHANNEL_IN_ERROR_STATE.
	LinkReasonChannelInErrorState LinkReason = native.RtmLinkStateChangeReasonChannelInErrorState
	// LinkReasonPresenceNotConnected is RTM_LINK_STATE_CHANGE_REASON_PRESENCE_NOT_CONNECTED.
	LinkReasonPresenceNotConnected LinkReason = native.RtmLinkStateChangeReasonPresenceNotConnected
	// LinkReasonSameUidLogin is RTM_LINK_STATE_CHANGE_REASON_SAME_UID_LOGIN.
	LinkReasonSameUidLogin LinkReason = native.RtmLinkStateChangeReasonSameUidLogin
	// LinkReasonKickedOutByServer is RTM_LINK_STATE_CHANGE_REASON_KICKED_OUT_BY_SERVER.
	LinkReasonKickedOutByServer LinkReason = native.RtmLinkStateChangeReasonKickedOutByServer
	// LinkReasonKeepAliveTimeout is RTM_LINK_STATE_CHANGE_REASON_KEEP_ALIVE_TIMEOUT.
	LinkReasonKeepAliveTimeout LinkReason = native.RtmLinkStateChangeReasonKeepAliveTimeout
	// LinkReasonConnectionError is RTM_LINK_STATE_CHANGE_REASON_CONNECTION_ERROR.
	LinkReasonConnectionError LinkReason = native.RtmLinkStateChangeReasonConnectionError
	// LinkReasonPresenceNotReady is RTM_LINK_STATE_CHANGE_REASON_PRESENCE_NOT_READY.
	LinkReasonPresenceNotReady LinkReason = native.RtmLinkStateChangeReasonPresenceNotReady
	// LinkReasonNetworkChange is RTM_LINK_STATE_CHANGE_REASON_NETWORK_CHANGE.
	LinkReasonNetworkChange LinkReason = native.RtmLinkStateChangeReasonNetworkChange
	// LinkReasonServiceNotSupported is RTM_LINK_STATE_CHANGE_REASON_SERVICE_NOT_SUPPORTED.
	LinkReasonServiceNotSupported LinkReason = native.RtmLinkStateChangeReasonServiceNotSupported
	// LinkReasonStreamChannelNotAvailable is RTM_LINK_STATE_CHANGE_REASON_STREAM_CHANNEL_NOT_AVAILABLE.
	LinkReasonStreamChannelNotAvailable LinkReason = native.RtmLinkStateChangeReasonStreamChannelNotAvailable
	// LinkReasonStorageNotAvailable is RTM_LINK_STATE_CHANGE_REASON_STORAGE_NOT_AVAILABLE.
	LinkReasonStorageNotAvailable LinkReason = native.RtmLinkStateChangeReasonStorageNotAvailable
	// LinkReasonLockNotAvailable is RTM_LINK_STATE_CHANGE_REASON_LOCK_NOT_AVAILABLE.
	LinkReasonLockNotAvailable LinkReason = native.RtmLinkStateChangeReasonLockNotAvailable
	// LinkReasonLoginTooFrequent is RTM_LINK_STATE_CHANGE_REASON_LOGIN_TOO_FREQUENT.
	LinkReasonLoginTooFrequent LinkReason = native.RtmLinkStateChangeReasonLoginTooFrequent
)

// ProtocolType preserves the native RTM_PROTOCOL_TYPE enumeration.
type ProtocolType = native.RtmProtocolType

const (
	// ProtocolTCPUDP is RTM_PROTOCOL_TYPE_TCP_UDP.
	ProtocolTCPUDP ProtocolType = native.RtmProtocolTypeTCPUDP
	// ProtocolTCPOnly is RTM_PROTOCOL_TYPE_TCP_ONLY.
	ProtocolTCPOnly ProtocolType = native.RtmProtocolTypeTCPOnly
)

// ErrorCode preserves the native RTM_ERROR_CODE enumeration.
type ErrorCode = native.RtmErrorCode

const (
	// CodeOK is RTM_ERROR_OK.
	CodeOK ErrorCode = native.RtmErrorCodeOK
	// CodeNotInitialized is RTM_ERROR_NOT_INITIALIZED.
	CodeNotInitialized ErrorCode = native.RtmErrorCodeNotInitialized
	// CodeNotLogin is RTM_ERROR_NOT_LOGIN.
	CodeNotLogin ErrorCode = native.RtmErrorCodeNotLogin
	// CodeInvalidAppID is RTM_ERROR_INVALID_APP_ID.
	CodeInvalidAppID ErrorCode = native.RtmErrorCodeInvalidAppID
	// CodeInvalidEventHandler is RTM_ERROR_INVALID_EVENT_HANDLER.
	CodeInvalidEventHandler ErrorCode = native.RtmErrorCodeInvalidEventHandler
	// CodeInvalidToken is RTM_ERROR_INVALID_TOKEN.
	CodeInvalidToken ErrorCode = native.RtmErrorCodeInvalidToken
	// CodeInvalidUserID is RTM_ERROR_INVALID_USER_ID.
	CodeInvalidUserID ErrorCode = native.RtmErrorCodeInvalidUserID
	// CodeInitServiceFailed is RTM_ERROR_INIT_SERVICE_FAILED.
	CodeInitServiceFailed ErrorCode = native.RtmErrorCodeInitServiceFailed
	// CodeInvalidChannelName is RTM_ERROR_INVALID_CHANNEL_NAME.
	CodeInvalidChannelName ErrorCode = native.RtmErrorCodeInvalidChannelName
	// CodeTokenExpired is RTM_ERROR_TOKEN_EXPIRED.
	CodeTokenExpired ErrorCode = native.RtmErrorCodeTokenExpired
	// CodeLoginNoServerResources is RTM_ERROR_LOGIN_NO_SERVER_RESOURCES.
	CodeLoginNoServerResources ErrorCode = native.RtmErrorCodeLoginNoServerResources
	// CodeLoginTimeout is RTM_ERROR_LOGIN_TIMEOUT.
	CodeLoginTimeout ErrorCode = native.RtmErrorCodeLoginTimeout
	// CodeLoginRejected is RTM_ERROR_LOGIN_REJECTED.
	CodeLoginRejected ErrorCode = native.RtmErrorCodeLoginRejected
	// CodeLoginAborted is RTM_ERROR_LOGIN_ABORTED.
	CodeLoginAborted ErrorCode = native.RtmErrorCodeLoginAborted
	// CodeInvalidParameter is RTM_ERROR_INVALID_PARAMETER.
	CodeInvalidParameter ErrorCode = native.RtmErrorCodeInvalidParameter
	// CodeLoginNotAuthorized is RTM_ERROR_LOGIN_NOT_AUTHORIZED.
	CodeLoginNotAuthorized ErrorCode = native.RtmErrorCodeLoginNotAuthorized
	// CodeInconsistentAppid is RTM_ERROR_INCONSISTENT_APPID.
	CodeInconsistentAppid ErrorCode = native.RtmErrorCodeInconsistentAppid
	// CodeDuplicateOperation is RTM_ERROR_DUPLICATE_OPERATION.
	CodeDuplicateOperation ErrorCode = native.RtmErrorCodeDuplicateOperation
	// CodeInstanceAlreadyReleased is RTM_ERROR_INSTANCE_ALREADY_RELEASED.
	CodeInstanceAlreadyReleased ErrorCode = native.RtmErrorCodeInstanceAlreadyReleased
	// CodeInvalidChannelType is RTM_ERROR_INVALID_CHANNEL_TYPE.
	CodeInvalidChannelType ErrorCode = native.RtmErrorCodeInvalidChannelType
	// CodeInvalidEncryptionParameter is RTM_ERROR_INVALID_ENCRYPTION_PARAMETER.
	CodeInvalidEncryptionParameter ErrorCode = native.RtmErrorCodeInvalidEncryptionParameter
	// CodeOperationRateExceedLimitation is RTM_ERROR_OPERATION_RATE_EXCEED_LIMITATION.
	CodeOperationRateExceedLimitation ErrorCode = native.RtmErrorCodeOperationRateExceedLimitation
	// CodeDuplicateUserID is RTM_ERROR_DUPLICATE_USER_ID.
	CodeDuplicateUserID ErrorCode = native.RtmErrorCodeDuplicateUserID
	// CodeChannelNotJoined is RTM_ERROR_CHANNEL_NOT_JOINED.
	CodeChannelNotJoined ErrorCode = native.RtmErrorCodeChannelNotJoined
	// CodeChannelNotSubscribed is RTM_ERROR_CHANNEL_NOT_SUBSCRIBED.
	CodeChannelNotSubscribed ErrorCode = native.RtmErrorCodeChannelNotSubscribed
	// CodeChannelExceedTopicUserLimitation is RTM_ERROR_CHANNEL_EXCEED_TOPIC_USER_LIMITATION.
	CodeChannelExceedTopicUserLimitation ErrorCode = native.RtmErrorCodeChannelExceedTopicUserLimitation
	// CodeChannelInReuse is RTM_ERROR_CHANNEL_IN_REUSE.
	CodeChannelInReuse ErrorCode = native.RtmErrorCodeChannelInReuse
	// CodeChannelInstanceExceedLimitation is RTM_ERROR_CHANNEL_INSTANCE_EXCEED_LIMITATION.
	CodeChannelInstanceExceedLimitation ErrorCode = native.RtmErrorCodeChannelInstanceExceedLimitation
	// CodeChannelInErrorState is RTM_ERROR_CHANNEL_IN_ERROR_STATE.
	CodeChannelInErrorState ErrorCode = native.RtmErrorCodeChannelInErrorState
	// CodeChannelJoinFailed is RTM_ERROR_CHANNEL_JOIN_FAILED.
	CodeChannelJoinFailed ErrorCode = native.RtmErrorCodeChannelJoinFailed
	// CodeChannelInvalidTopicName is RTM_ERROR_CHANNEL_INVALID_TOPIC_NAME.
	CodeChannelInvalidTopicName ErrorCode = native.RtmErrorCodeChannelInvalidTopicName
	// CodeChannelInvalidMessage is RTM_ERROR_CHANNEL_INVALID_MESSAGE.
	CodeChannelInvalidMessage ErrorCode = native.RtmErrorCodeChannelInvalidMessage
	// CodeChannelMessageLengthExceedLimitation is RTM_ERROR_CHANNEL_MESSAGE_LENGTH_EXCEED_LIMITATION.
	CodeChannelMessageLengthExceedLimitation ErrorCode = native.RtmErrorCodeChannelMessageLengthExceedLimitation
	// CodeChannelInvalidUserList is RTM_ERROR_CHANNEL_INVALID_USER_LIST.
	CodeChannelInvalidUserList ErrorCode = native.RtmErrorCodeChannelInvalidUserList
	// CodeChannelNotAvailable is RTM_ERROR_CHANNEL_NOT_AVAILABLE.
	CodeChannelNotAvailable ErrorCode = native.RtmErrorCodeChannelNotAvailable
	// CodeChannelTopicNotSubscribed is RTM_ERROR_CHANNEL_TOPIC_NOT_SUBSCRIBED.
	CodeChannelTopicNotSubscribed ErrorCode = native.RtmErrorCodeChannelTopicNotSubscribed
	// CodeChannelExceedTopicLimitation is RTM_ERROR_CHANNEL_EXCEED_TOPIC_LIMITATION.
	CodeChannelExceedTopicLimitation ErrorCode = native.RtmErrorCodeChannelExceedTopicLimitation
	// CodeChannelJoinTopicFailed is RTM_ERROR_CHANNEL_JOIN_TOPIC_FAILED.
	CodeChannelJoinTopicFailed ErrorCode = native.RtmErrorCodeChannelJoinTopicFailed
	// CodeChannelTopicNotJoined is RTM_ERROR_CHANNEL_TOPIC_NOT_JOINED.
	CodeChannelTopicNotJoined ErrorCode = native.RtmErrorCodeChannelTopicNotJoined
	// CodeChannelTopicNotExist is RTM_ERROR_CHANNEL_TOPIC_NOT_EXIST.
	CodeChannelTopicNotExist ErrorCode = native.RtmErrorCodeChannelTopicNotExist
	// CodeChannelInvalidTopicMeta is RTM_ERROR_CHANNEL_INVALID_TOPIC_META.
	CodeChannelInvalidTopicMeta ErrorCode = native.RtmErrorCodeChannelInvalidTopicMeta
	// CodeChannelSubscribeTimeout is RTM_ERROR_CHANNEL_SUBSCRIBE_TIMEOUT.
	CodeChannelSubscribeTimeout ErrorCode = native.RtmErrorCodeChannelSubscribeTimeout
	// CodeChannelSubscribeTooFrequent is RTM_ERROR_CHANNEL_SUBSCRIBE_TOO_FREQUENT.
	CodeChannelSubscribeTooFrequent ErrorCode = native.RtmErrorCodeChannelSubscribeTooFrequent
	// CodeChannelSubscribeFailed is RTM_ERROR_CHANNEL_SUBSCRIBE_FAILED.
	CodeChannelSubscribeFailed ErrorCode = native.RtmErrorCodeChannelSubscribeFailed
	// CodeChannelUnsubscribeFailed is RTM_ERROR_CHANNEL_UNSUBSCRIBE_FAILED.
	CodeChannelUnsubscribeFailed ErrorCode = native.RtmErrorCodeChannelUnsubscribeFailed
	// CodeChannelEncryptMessageFailed is RTM_ERROR_CHANNEL_ENCRYPT_MESSAGE_FAILED.
	CodeChannelEncryptMessageFailed ErrorCode = native.RtmErrorCodeChannelEncryptMessageFailed
	// CodeChannelPublishMessageFailed is RTM_ERROR_CHANNEL_PUBLISH_MESSAGE_FAILED.
	CodeChannelPublishMessageFailed ErrorCode = native.RtmErrorCodeChannelPublishMessageFailed
	// CodeChannelPublishMessageTooFrequent is RTM_ERROR_CHANNEL_PUBLISH_MESSAGE_TOO_FREQUENT.
	CodeChannelPublishMessageTooFrequent ErrorCode = native.RtmErrorCodeChannelPublishMessageTooFrequent
	// CodeChannelPublishMessageTimeout is RTM_ERROR_CHANNEL_PUBLISH_MESSAGE_TIMEOUT.
	CodeChannelPublishMessageTimeout ErrorCode = native.RtmErrorCodeChannelPublishMessageTimeout
	// CodeChannelNotConnected is RTM_ERROR_CHANNEL_NOT_CONNECTED.
	CodeChannelNotConnected ErrorCode = native.RtmErrorCodeChannelNotConnected
	// CodeChannelLeaveFailed is RTM_ERROR_CHANNEL_LEAVE_FAILED.
	CodeChannelLeaveFailed ErrorCode = native.RtmErrorCodeChannelLeaveFailed
	// CodeChannelCustomTypeLengthOverflow is RTM_ERROR_CHANNEL_CUSTOM_TYPE_LENGTH_OVERFLOW.
	CodeChannelCustomTypeLengthOverflow ErrorCode = native.RtmErrorCodeChannelCustomTypeLengthOverflow
	// CodeChannelInvalidCustomType is RTM_ERROR_CHANNEL_INVALID_CUSTOM_TYPE.
	CodeChannelInvalidCustomType ErrorCode = native.RtmErrorCodeChannelInvalidCustomType
	// CodeChannelUnsupportedMessageType is RTM_ERROR_CHANNEL_UNSUPPORTED_MESSAGE_TYPE.
	CodeChannelUnsupportedMessageType ErrorCode = native.RtmErrorCodeChannelUnsupportedMessageType
	// CodeChannelPresenceNotReady is RTM_ERROR_CHANNEL_PRESENCE_NOT_READY.
	CodeChannelPresenceNotReady ErrorCode = native.RtmErrorCodeChannelPresenceNotReady
	// CodeChannelReceiverOffline is RTM_ERROR_CHANNEL_RECEIVER_OFFLINE.
	CodeChannelReceiverOffline ErrorCode = native.RtmErrorCodeChannelReceiverOffline
	// CodeChannelSubscribePermissionDenied is RTM_ERROR_CHANNEL_SUBSCRIBE_PERMISSION_DENIED.
	CodeChannelSubscribePermissionDenied ErrorCode = native.RtmErrorCodeChannelSubscribePermissionDenied
	// CodeChannelPublishPermissionDenied is RTM_ERROR_CHANNEL_PUBLISH_PERMISSION_DENIED.
	CodeChannelPublishPermissionDenied ErrorCode = native.RtmErrorCodeChannelPublishPermissionDenied
	// CodeChannelSubscribeCanceled is RTM_ERROR_CHANNEL_SUBSCRIBE_CANCELED.
	CodeChannelSubscribeCanceled ErrorCode = native.RtmErrorCodeChannelSubscribeCanceled
	// CodeStorageOperationFailed is RTM_ERROR_STORAGE_OPERATION_FAILED.
	CodeStorageOperationFailed ErrorCode = native.RtmErrorCodeStorageOperationFailed
	// CodeStorageMetadataItemExceedLimitation is RTM_ERROR_STORAGE_METADATA_ITEM_EXCEED_LIMITATION.
	CodeStorageMetadataItemExceedLimitation ErrorCode = native.RtmErrorCodeStorageMetadataItemExceedLimitation
	// CodeStorageInvalidMetadataItem is RTM_ERROR_STORAGE_INVALID_METADATA_ITEM.
	CodeStorageInvalidMetadataItem ErrorCode = native.RtmErrorCodeStorageInvalidMetadataItem
	// CodeStorageInvalidArgument is RTM_ERROR_STORAGE_INVALID_ARGUMENT.
	CodeStorageInvalidArgument ErrorCode = native.RtmErrorCodeStorageInvalidArgument
	// CodeStorageInvalidRevision is RTM_ERROR_STORAGE_INVALID_REVISION.
	CodeStorageInvalidRevision ErrorCode = native.RtmErrorCodeStorageInvalidRevision
	// CodeStorageMetadataLengthOverflow is RTM_ERROR_STORAGE_METADATA_LENGTH_OVERFLOW.
	CodeStorageMetadataLengthOverflow ErrorCode = native.RtmErrorCodeStorageMetadataLengthOverflow
	// CodeStorageInvalidLockName is RTM_ERROR_STORAGE_INVALID_LOCK_NAME.
	CodeStorageInvalidLockName ErrorCode = native.RtmErrorCodeStorageInvalidLockName
	// CodeStorageLockNotAcquired is RTM_ERROR_STORAGE_LOCK_NOT_ACQUIRED.
	CodeStorageLockNotAcquired ErrorCode = native.RtmErrorCodeStorageLockNotAcquired
	// CodeStorageInvalidKey is RTM_ERROR_STORAGE_INVALID_KEY.
	CodeStorageInvalidKey ErrorCode = native.RtmErrorCodeStorageInvalidKey
	// CodeStorageInvalidValue is RTM_ERROR_STORAGE_INVALID_VALUE.
	CodeStorageInvalidValue ErrorCode = native.RtmErrorCodeStorageInvalidValue
	// CodeStorageKeyLengthOverflow is RTM_ERROR_STORAGE_KEY_LENGTH_OVERFLOW.
	CodeStorageKeyLengthOverflow ErrorCode = native.RtmErrorCodeStorageKeyLengthOverflow
	// CodeStorageValueLengthOverflow is RTM_ERROR_STORAGE_VALUE_LENGTH_OVERFLOW.
	CodeStorageValueLengthOverflow ErrorCode = native.RtmErrorCodeStorageValueLengthOverflow
	// CodeStorageDuplicateKey is RTM_ERROR_STORAGE_DUPLICATE_KEY.
	CodeStorageDuplicateKey ErrorCode = native.RtmErrorCodeStorageDuplicateKey
	// CodeStorageOutdatedRevision is RTM_ERROR_STORAGE_OUTDATED_REVISION.
	CodeStorageOutdatedRevision ErrorCode = native.RtmErrorCodeStorageOutdatedRevision
	// CodeStorageNotSubscribe is RTM_ERROR_STORAGE_NOT_SUBSCRIBE.
	CodeStorageNotSubscribe ErrorCode = native.RtmErrorCodeStorageNotSubscribe
	// CodeStorageInvalidMetadataInstance is RTM_ERROR_STORAGE_INVALID_METADATA_INSTANCE.
	CodeStorageInvalidMetadataInstance ErrorCode = native.RtmErrorCodeStorageInvalidMetadataInstance
	// CodeStorageSubscribeUserExceedLimitation is RTM_ERROR_STORAGE_SUBSCRIBE_USER_EXCEED_LIMITATION.
	CodeStorageSubscribeUserExceedLimitation ErrorCode = native.RtmErrorCodeStorageSubscribeUserExceedLimitation
	// CodeStorageOperationTimeout is RTM_ERROR_STORAGE_OPERATION_TIMEOUT.
	CodeStorageOperationTimeout ErrorCode = native.RtmErrorCodeStorageOperationTimeout
	// CodeStorageNotAvailable is RTM_ERROR_STORAGE_NOT_AVAILABLE.
	CodeStorageNotAvailable ErrorCode = native.RtmErrorCodeStorageNotAvailable
	// CodeStoragePermissionDenied is RTM_ERROR_STORAGE_PERMISSION_DENIED.
	CodeStoragePermissionDenied ErrorCode = native.RtmErrorCodeStoragePermissionDenied
	// CodePresenceNotConnected is RTM_ERROR_PRESENCE_NOT_CONNECTED.
	CodePresenceNotConnected ErrorCode = native.RtmErrorCodePresenceNotConnected
	// CodePresenceNotWritable is RTM_ERROR_PRESENCE_NOT_WRITABLE.
	CodePresenceNotWritable ErrorCode = native.RtmErrorCodePresenceNotWritable
	// CodePresenceInvalidArgument is RTM_ERROR_PRESENCE_INVALID_ARGUMENT.
	CodePresenceInvalidArgument ErrorCode = native.RtmErrorCodePresenceInvalidArgument
	// CodePresenceCachedTooManyStates is RTM_ERROR_PRESENCE_CACHED_TOO_MANY_STATES.
	CodePresenceCachedTooManyStates ErrorCode = native.RtmErrorCodePresenceCachedTooManyStates
	// CodePresenceStateCountOverflow is RTM_ERROR_PRESENCE_STATE_COUNT_OVERFLOW.
	CodePresenceStateCountOverflow ErrorCode = native.RtmErrorCodePresenceStateCountOverflow
	// CodePresenceInvalidStateKey is RTM_ERROR_PRESENCE_INVALID_STATE_KEY.
	CodePresenceInvalidStateKey ErrorCode = native.RtmErrorCodePresenceInvalidStateKey
	// CodePresenceInvalidStateValue is RTM_ERROR_PRESENCE_INVALID_STATE_VALUE.
	CodePresenceInvalidStateValue ErrorCode = native.RtmErrorCodePresenceInvalidStateValue
	// CodePresenceStateKeySizeOverflow is RTM_ERROR_PRESENCE_STATE_KEY_SIZE_OVERFLOW.
	CodePresenceStateKeySizeOverflow ErrorCode = native.RtmErrorCodePresenceStateKeySizeOverflow
	// CodePresenceStateValueSizeOverflow is RTM_ERROR_PRESENCE_STATE_VALUE_SIZE_OVERFLOW.
	CodePresenceStateValueSizeOverflow ErrorCode = native.RtmErrorCodePresenceStateValueSizeOverflow
	// CodePresenceStateDuplicateKey is RTM_ERROR_PRESENCE_STATE_DUPLICATE_KEY.
	CodePresenceStateDuplicateKey ErrorCode = native.RtmErrorCodePresenceStateDuplicateKey
	// CodePresenceUserNotExist is RTM_ERROR_PRESENCE_USER_NOT_EXIST.
	CodePresenceUserNotExist ErrorCode = native.RtmErrorCodePresenceUserNotExist
	// CodePresenceOperationTimeout is RTM_ERROR_PRESENCE_OPERATION_TIMEOUT.
	CodePresenceOperationTimeout ErrorCode = native.RtmErrorCodePresenceOperationTimeout
	// CodePresenceOperationFailed is RTM_ERROR_PRESENCE_OPERATION_FAILED.
	CodePresenceOperationFailed ErrorCode = native.RtmErrorCodePresenceOperationFailed
	// CodePresenceInactive is RTM_ERROR_PRESENCE_INACTIVE.
	CodePresenceInactive ErrorCode = native.RtmErrorCodePresenceInactive
	// CodePresencePendingRequestCancelled is RTM_ERROR_PRESENCE_PENDING_REQUEST_CANCELLED.
	CodePresencePendingRequestCancelled ErrorCode = native.RtmErrorCodePresencePendingRequestCancelled
	// CodePresencePendingRequestExceedLimitation is RTM_ERROR_PRESENCE_PENDING_REQUEST_EXCEED_LIMITATION.
	CodePresencePendingRequestExceedLimitation ErrorCode = native.RtmErrorCodePresencePendingRequestExceedLimitation
	// CodeLockOperationFailed is RTM_ERROR_LOCK_OPERATION_FAILED.
	CodeLockOperationFailed ErrorCode = native.RtmErrorCodeLockOperationFailed
	// CodeLockOperationTimeout is RTM_ERROR_LOCK_OPERATION_TIMEOUT.
	CodeLockOperationTimeout ErrorCode = native.RtmErrorCodeLockOperationTimeout
	// CodeLockOperationPerforming is RTM_ERROR_LOCK_OPERATION_PERFORMING.
	CodeLockOperationPerforming ErrorCode = native.RtmErrorCodeLockOperationPerforming
	// CodeLockAlreadyExist is RTM_ERROR_LOCK_ALREADY_EXIST.
	CodeLockAlreadyExist ErrorCode = native.RtmErrorCodeLockAlreadyExist
	// CodeLockInvalidName is RTM_ERROR_LOCK_INVALID_NAME.
	CodeLockInvalidName ErrorCode = native.RtmErrorCodeLockInvalidName
	// CodeLockNotAcquired is RTM_ERROR_LOCK_NOT_ACQUIRED.
	CodeLockNotAcquired ErrorCode = native.RtmErrorCodeLockNotAcquired
	// CodeLockAcquireFailed is RTM_ERROR_LOCK_ACQUIRE_FAILED.
	CodeLockAcquireFailed ErrorCode = native.RtmErrorCodeLockAcquireFailed
	// CodeLockNotExist is RTM_ERROR_LOCK_NOT_EXIST.
	CodeLockNotExist ErrorCode = native.RtmErrorCodeLockNotExist
	// CodeLockNotAvailable is RTM_ERROR_LOCK_NOT_AVAILABLE.
	CodeLockNotAvailable ErrorCode = native.RtmErrorCodeLockNotAvailable
	// CodeLockPermissionDenied is RTM_ERROR_LOCK_PERMISSION_DENIED.
	CodeLockPermissionDenied ErrorCode = native.RtmErrorCodeLockPermissionDenied
	// CodeHistoryPermissionDenied is RTM_ERROR_HISTORY_PERMISSION_DENIED.
	CodeHistoryPermissionDenied ErrorCode = native.RtmErrorCodeHistoryPermissionDenied
)

// StorageEventType preserves the native RTM_STORAGE_EVENT_TYPE enumeration.
type StorageEventType = native.RtmStorageEventType

const (
	// StorageEventNone is RTM_STORAGE_EVENT_TYPE_NONE.
	StorageEventNone StorageEventType = native.RtmStorageEventTypeNone
	// StorageEventSnapshot is RTM_STORAGE_EVENT_TYPE_SNAPSHOT.
	StorageEventSnapshot StorageEventType = native.RtmStorageEventTypeSnapshot
	// StorageEventSet is RTM_STORAGE_EVENT_TYPE_SET.
	StorageEventSet StorageEventType = native.RtmStorageEventTypeSet
	// StorageEventUpdate is RTM_STORAGE_EVENT_TYPE_UPDATE.
	StorageEventUpdate StorageEventType = native.RtmStorageEventTypeUpdate
	// StorageEventRemove is RTM_STORAGE_EVENT_TYPE_REMOVE.
	StorageEventRemove StorageEventType = native.RtmStorageEventTypeRemove
)

// LockEventType preserves the native RTM_LOCK_EVENT_TYPE enumeration.
type LockEventType = native.RtmLockEventType

const (
	// LockEventNone is RTM_LOCK_EVENT_TYPE_NONE.
	LockEventNone LockEventType = native.RtmLockEventTypeNone
	// LockEventSnapshot is RTM_LOCK_EVENT_TYPE_SNAPSHOT.
	LockEventSnapshot LockEventType = native.RtmLockEventTypeSnapshot
	// LockEventLockSet is RTM_LOCK_EVENT_TYPE_LOCK_SET.
	LockEventLockSet LockEventType = native.RtmLockEventTypeLockSet
	// LockEventLockRemoved is RTM_LOCK_EVENT_TYPE_LOCK_REMOVED.
	LockEventLockRemoved LockEventType = native.RtmLockEventTypeLockRemoved
	// LockEventLockAcquired is RTM_LOCK_EVENT_TYPE_LOCK_ACQUIRED.
	LockEventLockAcquired LockEventType = native.RtmLockEventTypeLockAcquired
	// LockEventLockReleased is RTM_LOCK_EVENT_TYPE_LOCK_RELEASED.
	LockEventLockReleased LockEventType = native.RtmLockEventTypeLockReleased
	// LockEventLockExpired is RTM_LOCK_EVENT_TYPE_LOCK_EXPIRED.
	LockEventLockExpired LockEventType = native.RtmLockEventTypeLockExpired
)

// TopicEventType preserves the native RTM_TOPIC_EVENT_TYPE enumeration.
type TopicEventType = native.RtmTopicEventType

const (
	// TopicEventNone is RTM_TOPIC_EVENT_TYPE_NONE.
	TopicEventNone TopicEventType = native.RtmTopicEventTypeNone
	// TopicEventSnapshot is RTM_TOPIC_EVENT_TYPE_SNAPSHOT.
	TopicEventSnapshot TopicEventType = native.RtmTopicEventTypeSnapshot
	// TopicEventRemoteJoinTopic is RTM_TOPIC_EVENT_TYPE_REMOTE_JOIN_TOPIC.
	TopicEventRemoteJoinTopic TopicEventType = native.RtmTopicEventTypeRemoteJoinTopic
	// TopicEventRemoteLeaveTopic is RTM_TOPIC_EVENT_TYPE_REMOTE_LEAVE_TOPIC.
	TopicEventRemoteLeaveTopic TopicEventType = native.RtmTopicEventTypeRemoteLeaveTopic
)

// PresenceEventType preserves the native RTM_PRESENCE_EVENT_TYPE enumeration.
type PresenceEventType = native.RtmPresenceEventType

const (
	// PresenceEventNone is RTM_PRESENCE_EVENT_TYPE_NONE.
	PresenceEventNone PresenceEventType = native.RtmPresenceEventTypeNone
	// PresenceEventSnapshot is RTM_PRESENCE_EVENT_TYPE_SNAPSHOT.
	PresenceEventSnapshot PresenceEventType = native.RtmPresenceEventTypeSnapshot
	// PresenceEventInterval is RTM_PRESENCE_EVENT_TYPE_INTERVAL.
	PresenceEventInterval PresenceEventType = native.RtmPresenceEventTypeInterval
	// PresenceEventRemoteJoinChannel is RTM_PRESENCE_EVENT_TYPE_REMOTE_JOIN_CHANNEL.
	PresenceEventRemoteJoinChannel PresenceEventType = native.RtmPresenceEventTypeRemoteJoinChannel
	// PresenceEventRemoteLeaveChannel is RTM_PRESENCE_EVENT_TYPE_REMOTE_LEAVE_CHANNEL.
	PresenceEventRemoteLeaveChannel PresenceEventType = native.RtmPresenceEventTypeRemoteLeaveChannel
	// PresenceEventRemoteTimeout is RTM_PRESENCE_EVENT_TYPE_REMOTE_TIMEOUT.
	PresenceEventRemoteTimeout PresenceEventType = native.RtmPresenceEventTypeRemoteTimeout
	// PresenceEventRemoteStateChanged is RTM_PRESENCE_EVENT_TYPE_REMOTE_STATE_CHANGED.
	PresenceEventRemoteStateChanged PresenceEventType = native.RtmPresenceEventTypeRemoteStateChanged
	// PresenceEventErrorOutOfService is RTM_PRESENCE_EVENT_TYPE_ERROR_OUT_OF_SERVICE.
	PresenceEventErrorOutOfService PresenceEventType = native.RtmPresenceEventTypeErrorOutOfService
)
