package agorartm

//note: MUST dec cgo_xxx_xxx in this file and include C_IAgoraRtmClient.h, or the callback will not be called
//like the following:

/*
#include "C_IAgoraRtmClient.h"
void cgo_RtmEventHandlerBridge_onTokenEvent(struct C_IRtmEventHandler *, struct C_TokenEvent *);

void cgo_RtmEventHandlerBridge_onMessageEvent(struct C_IRtmEventHandler *this,struct C_MessageEvent *event);

void cgo_RtmEventHandlerBridge_onPresenceEvent(struct C_IRtmEventHandler *this,
	struct C_PresenceEvent *event);

void cgo_RtmEventHandlerBridge_onTopicEvent(struct C_IRtmEventHandler *this,
	struct C_TopicEvent *event);

void cgo_RtmEventHandlerBridge_onLockEvent(struct C_IRtmEventHandler *this,
	struct C_LockEvent *event);

void cgo_RtmEventHandlerBridge_onStorageEvent(struct C_IRtmEventHandler *this,
	struct C_StorageEvent *event);

void cgo_RtmEventHandlerBridge_onJoinResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *userId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onLeaveResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *userId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onJoinTopicResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *userId, char *topic, char *meta, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onLeaveTopicResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *userId, char *topic, char *meta, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onSubscribeTopicResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *userId, char *topic, struct C_UserList succeedUsers, struct C_UserList failedUsers, enum C_RTM_ERROR_CODE errorCode);


void cgo_RtmEventHandlerBridge_onTokenPrivilegeWillExpire(struct C_IRtmEventHandler *this,
	char *channelName);

void cgo_RtmEventHandlerBridge_onSubscribeResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_ERROR_CODE errorCode);
void cgo_RtmEventHandlerBridge_onUnsubscribeResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_ERROR_CODE errorCode);

	void cgo_RtmEventHandlerBridge_onPublishResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onLoginResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onSetChannelMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onUpdateChannelMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onRemoveChannelMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onGetChannelMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, struct C_Metadata *data, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onSetUserMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *userId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onUpdateUserMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *userId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onRemoveUserMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *userId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onGetUserMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *userId, struct C_Metadata *data, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onSubscribeUserMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *userId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onSetLockResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, char *lockName, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onRemoveLockResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, char *lockName, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onReleaseLockResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, char *lockName, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onAcquireLockResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, char *lockName, enum C_RTM_ERROR_CODE errorCode, char *errorDetails);

void cgo_RtmEventHandlerBridge_onRevokeLockResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, char *lockName, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onGetLocksResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, enum C_RTM_CHANNEL_TYPE channelType, struct C_LockDetail *lockDetailList, size_t count, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onWhoNowResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, struct C_UserState *userStateList, size_t count, char *nextPage, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onGetOnlineUsersResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, struct C_UserState *userStateList, size_t count, char *nextPage, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onWhereNowResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, struct C_ChannelInfo *channels, size_t count, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onGetUserChannelsResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, struct C_ChannelInfo *channels, size_t count, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onPresenceSetStateResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onPresenceRemoveStateResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onPresenceGetStateResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, struct C_UserState *state, enum C_RTM_ERROR_CODE errorCode);

// newly added callback functions
void cgo_RtmEventHandlerBridge_onLinkStateEvent(struct C_IRtmEventHandler *this,
	struct C_LinkStateEvent *event);

void cgo_RtmEventHandlerBridge_onLogoutResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onRenewTokenResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, enum C_RTM_SERVICE_TYPE serverType, char *channelName, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onPublishTopicMessageResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *topic, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onUnsubscribeTopicResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *topic, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onGetSubscribedUserListResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *channelName, char *topic, struct C_UserList users, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onGetHistoryMessagesResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, struct C_HistoryMessage *messageList, size_t count, uint64_t newStart, enum C_RTM_ERROR_CODE errorCode);

void cgo_RtmEventHandlerBridge_onUnsubscribeUserMetadataResult(struct C_IRtmEventHandler *this,
	uint64_t requestId, char *userId, enum C_RTM_ERROR_CODE errorCode);

*/
import "C"

//"github.com/AgoraIO-Extensions/Agora-RTM-Server-SDK-Go/pkg/agora"

type RtmEventHandler struct {
	OnTokenEvent                  func(event *TokenEvent)
	OnMessageEvent                func(event *MessageEvent)
	OnPresenceEvent               func(event *PresenceEvent)
	OnTopicEvent                  func(event *TopicEvent)
	OnLockEvent                   func(event *LockEvent)
	OnStorageEvent                func(event *StorageEvent)
	OnJoinResult                  func(requestId uint64, channelName string, userId string, errorCode int)
	OnLeaveResult                 func(requestId uint64, channelName string, userId string, errorCode int)
	OnJoinTopicResult             func(requestId uint64, channelName string, userId string, topic string, meta string, errorCode int)
	OnLeaveTopicResult            func(requestId uint64, channelName string, userId string, topic string, meta string, errorCode int)
	OnSubscribeTopicResult        func(requestId uint64, channelName string, userId string, topic string, succeedUsers UserList, failedUsers UserList, errorCode int)
	OnTokenPrivilegeWillExpire    func(channelName string)
	OnSubscribeResult             func(requestId uint64, channelName string, errorCode int)
	OnUnsubscribeResult           func(requestId uint64, channelName string, errorCode int)
	OnPublishResult               func(requestId uint64, errorCode int)
	OnLoginResult                 func(requestId uint64, errorCode int)
	OnSetChannelMetadataResult    func(requestId uint64, channelName string, channelType RtmChannelType, errorCode int)
	OnUpdateChannelMetadataResult func(requestId uint64, channelName string, channelType RtmChannelType, errorCode int)
	OnRemoveChannelMetadataResult func(requestId uint64, channelName string, channelType RtmChannelType, errorCode int)
	OnGetChannelMetadataResult    func(requestId uint64, channelName string, channelType RtmChannelType, data *IMetadata, errorCode int)
	OnSetUserMetadataResult       func(requestId uint64, userId string, errorCode int)
	OnUpdateUserMetadataResult    func(requestId uint64, userId string, errorCode int)
	OnRemoveUserMetadataResult    func(requestId uint64, userId string, errorCode int)
	OnGetUserMetadataResult       func(requestId uint64, userId string, data *IMetadata, errorCode int)
	OnSubscribeUserMetadataResult func(requestId uint64, userId string, errorCode int)
	OnSetLockResult               func(requestId uint64, channelName string, channelType RtmChannelType, lockName string, errorCode int)
	OnRemoveLockResult            func(requestId uint64, channelName string, channelType RtmChannelType, lockName string, errorCode int)
	OnReleaseLockResult           func(requestId uint64, channelName string, channelType RtmChannelType, lockName string, errorCode int)
	OnAcquireLockResult           func(requestId uint64, channelName string, channelType RtmChannelType, lockName string, errorCode int, errorDetails string)
	OnRevokeLockResult            func(requestId uint64, channelName string, channelType RtmChannelType, lockName string, errorCode int)
	OnGetLocksResult              func(requestId uint64, channelName string, channelType RtmChannelType, lockDetailList []LockDetail, errorCode int)
	OnWhoNowResult                func(requestId uint64, userStateList []UserState, nextPage string, errorCode int)
	OnGetOnlineUsersResult        func(requestId uint64, userStateList []UserState, nextPage string, errorCode int)
	OnWhereNowResult              func(requestId uint64, channels []ChannelInfo, errorCode int)
	OnGetUserChannelsResult       func(requestId uint64, channels []ChannelInfo, errorCode int)
	OnPresenceSetStateResult      func(requestId uint64, errorCode int)
	OnPresenceRemoveStateResult   func(requestId uint64, errorCode int)
	OnPresenceGetStateResult      func(requestId uint64, state *UserState, errorCode int)
	// newly added callback functions
	OnLinkStateEvent                func(event *LinkStateEvent)
	OnLogoutResult                  func(requestId uint64, errorCode int)
	OnRenewTokenResult              func(requestId uint64, serverType RtmServiceType, channelName string, errorCode int)
	OnPublishTopicMessageResult     func(requestId uint64, channelName string, topic string, errorCode int)
	OnUnsubscribeTopicResult        func(requestId uint64, channelName string, topic string, errorCode int)
	OnGetSubscribedUserListResult   func(requestId uint64, channelName string, topic string, user *UserList, errorCode int)
	OnGetHistoryMessagesResult      func(requestId uint64, messageList []HistoryMessage, newStart uint64, errorCode int)
	OnUnsubscribeUserMetadataResult func(requestId uint64, userId string, errorCode int)
}

func CRtmEventHandler() *C.struct_C_IRtmEventHandler {

	ret := (*C.struct_C_IRtmEventHandler)(C.C_IRtmEventHandler_New(nil))
	if ret == nil {
		return nil
	}
	ret.userData = nil
	ret.onTokenEvent = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onTokenEvent)
	ret.onMessageEvent = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onMessageEvent)
	ret.onPresenceEvent = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onPresenceEvent)
	ret.onTopicEvent = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onTopicEvent)
	ret.onLockEvent = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onLockEvent)
	ret.onStorageEvent = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onStorageEvent)
	ret.onJoinResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onJoinResult)
	ret.onLeaveResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onLeaveResult)
	ret.onJoinTopicResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onJoinTopicResult)
	ret.onLeaveTopicResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onLeaveTopicResult)
	ret.onSubscribeTopicResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onSubscribeTopicResult)
	ret.onTokenPrivilegeWillExpire = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onTokenPrivilegeWillExpire)
	ret.onSubscribeResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onSubscribeResult)
	ret.onUnsubscribeResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onUnsubscribeResult)
	ret.onPublishResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onPublishResult)
	ret.onLoginResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onLoginResult)
	ret.onSetChannelMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onSetChannelMetadataResult)
	ret.onUpdateChannelMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onUpdateChannelMetadataResult)
	ret.onRemoveChannelMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onRemoveChannelMetadataResult)
	ret.onGetChannelMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onGetChannelMetadataResult)
	ret.onSetUserMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onSetUserMetadataResult)
	ret.onUpdateUserMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onUpdateUserMetadataResult)
	ret.onRemoveUserMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onRemoveUserMetadataResult)
	ret.onGetUserMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onGetUserMetadataResult)
	ret.onSubscribeUserMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onSubscribeUserMetadataResult)
	ret.onSetLockResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onSetLockResult)
	ret.onRemoveLockResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onRemoveLockResult)
	ret.onReleaseLockResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onReleaseLockResult)
	ret.onAcquireLockResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onAcquireLockResult)
	ret.onRevokeLockResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onRevokeLockResult)
	ret.onGetLocksResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onGetLocksResult)
	ret.onWhoNowResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onWhoNowResult)
	ret.onGetOnlineUsersResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onGetOnlineUsersResult)
	ret.onWhereNowResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onWhereNowResult)
	ret.onGetUserChannelsResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onGetUserChannelsResult)
	ret.onPresenceSetStateResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onPresenceSetStateResult)
	ret.onPresenceRemoveStateResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onPresenceRemoveStateResult)
	ret.onPresenceGetStateResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onPresenceGetStateResult)
	ret.onLinkStateEvent = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onLinkStateEvent)
	ret.onLogoutResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onLogoutResult)
	ret.onRenewTokenResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onRenewTokenResult)
	ret.onPublishTopicMessageResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onPublishTopicMessageResult)
	ret.onUnsubscribeTopicResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onUnsubscribeTopicResult)
	ret.onGetSubscribedUserListResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onGetSubscribedUserListResult)
	ret.onGetHistoryMessagesResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onGetHistoryMessagesResult)
	ret.onUnsubscribeUserMetadataResult = (*[0]byte)(C.cgo_RtmEventHandlerBridge_onUnsubscribeUserMetadataResult)
	return ret
}

//export cgo_RtmEventHandlerBridge_onMessageEvent
func cgo_RtmEventHandlerBridge_onMessageEvent(handler *C.struct_C_IRtmEventHandler,
	event *C.struct_C_MessageEvent) {
	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnMessageEvent == nil {
		return
	}

	goEvent := NewMessageEvent()
	if goEvent != nil {
		goEvent.fromC(event)
		client.handler.OnMessageEvent(goEvent)
	}
}

//export cgo_RtmEventHandlerBridge_onPresenceEvent
func cgo_RtmEventHandlerBridge_onPresenceEvent(handler *C.struct_C_IRtmEventHandler,
	event *C.struct_C_PresenceEvent) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnPresenceEvent == nil {
		return
	}

	goEvent := NewPresenceEvent()
	if goEvent != nil {
		goEvent.fromC(event)
		client.handler.OnPresenceEvent(goEvent)
	}
}

//export cgo_RtmEventHandlerBridge_onTopicEvent
func cgo_RtmEventHandlerBridge_onTopicEvent(handler *C.struct_C_IRtmEventHandler,
	event *C.struct_C_TopicEvent) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnTopicEvent == nil {
		return
	}

	goEvent := NewTopicEvent()
	if goEvent != nil {
		goEvent.fromC(event)
		client.handler.OnTopicEvent(goEvent)
	}
}

//export cgo_RtmEventHandlerBridge_onLockEvent
func cgo_RtmEventHandlerBridge_onLockEvent(handler *C.struct_C_IRtmEventHandler,
	event *C.struct_C_LockEvent) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnLockEvent == nil {
		return
	}

	goEvent := NewLockEvent()
	if goEvent != nil {
		goEvent.fromC(event)
		client.handler.OnLockEvent(goEvent)
	}
}

//export cgo_RtmEventHandlerBridge_onStorageEvent
func cgo_RtmEventHandlerBridge_onStorageEvent(handler *C.struct_C_IRtmEventHandler,
	event *C.struct_C_StorageEvent) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnStorageEvent == nil {
		return
	}

	goEvent := NewStorageEvent()
	if goEvent != nil {
		goEvent.fromC(event)
		client.handler.OnStorageEvent(goEvent)
	}
}

//export cgo_RtmEventHandlerBridge_onJoinResult
func cgo_RtmEventHandlerBridge_onJoinResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, userId *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnJoinResult == nil {
		return
	}

	client.handler.OnJoinResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(userId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onLeaveResult
func cgo_RtmEventHandlerBridge_onLeaveResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, userId *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnLeaveResult == nil {
		return
	}

	client.handler.OnLeaveResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(userId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onJoinTopicResult
func cgo_RtmEventHandlerBridge_onJoinTopicResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, userId *C.char, topic *C.char, meta *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnJoinTopicResult == nil {
		return
	}

	client.handler.OnJoinTopicResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(userId),
		C.GoString(topic),
		C.GoString(meta),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onLeaveTopicResult
func cgo_RtmEventHandlerBridge_onLeaveTopicResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, userId *C.char, topic *C.char, meta *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnLeaveTopicResult == nil {
		return
	}

	client.handler.OnLeaveTopicResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(userId),
		C.GoString(topic),
		C.GoString(meta),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onSubscribeTopicResult
func cgo_RtmEventHandlerBridge_onSubscribeTopicResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, userId *C.char, topic *C.char, succeedUsers C.struct_C_UserList, failedUsers C.struct_C_UserList, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnSubscribeTopicResult == nil {
		return
	}
	goSucceedUsers := CUserListToUserList(&succeedUsers)
	goFailedUsers := CUserListToUserList(&failedUsers)

	// 安全地处理可能为 nil 的 UserList
	var safeSucceedUsers UserList
	var safeFailedUsers UserList

	if goSucceedUsers != nil {
		safeSucceedUsers = *goSucceedUsers
	}

	if goFailedUsers != nil {
		safeFailedUsers = *goFailedUsers
	}

	client.handler.OnSubscribeTopicResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(userId),
		C.GoString(topic),
		safeSucceedUsers,
		safeFailedUsers,
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onTokenPrivilegeWillExpire
func cgo_RtmEventHandlerBridge_onTokenPrivilegeWillExpire(handler *C.struct_C_IRtmEventHandler,
	channelName *C.char) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnTokenPrivilegeWillExpire == nil {
		return
	}

	client.handler.OnTokenPrivilegeWillExpire(
		C.GoString(channelName),
	)
}

//export cgo_RtmEventHandlerBridge_onSubscribeResult
func cgo_RtmEventHandlerBridge_onSubscribeResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnSubscribeResult == nil {
		return
	}

	client.handler.OnSubscribeResult(
		uint64(requestId),
		C.GoString(channelName),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onPublishResult
func cgo_RtmEventHandlerBridge_onPublishResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnPublishResult == nil {
		return
	}

	client.handler.OnPublishResult(
		uint64(requestId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onLoginResult
func cgo_RtmEventHandlerBridge_onLoginResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnLoginResult == nil {
		return
	}

	client.handler.OnLoginResult(
		uint64(requestId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onSetChannelMetadataResult
func cgo_RtmEventHandlerBridge_onSetChannelMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnSetChannelMetadataResult == nil {
		return
	}

	client.handler.OnSetChannelMetadataResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onUpdateChannelMetadataResult
func cgo_RtmEventHandlerBridge_onUpdateChannelMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnUpdateChannelMetadataResult == nil {
		return
	}

	client.handler.OnUpdateChannelMetadataResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onRemoveChannelMetadataResult
func cgo_RtmEventHandlerBridge_onRemoveChannelMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnRemoveChannelMetadataResult == nil {
		return
	}

	client.handler.OnRemoveChannelMetadataResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onGetChannelMetadataResult
func cgo_RtmEventHandlerBridge_onGetChannelMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, data *C.struct_C_Metadata, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnGetChannelMetadataResult == nil {
		return
	}
	goData := CMetadataToIMetadata(data)

	client.handler.OnGetChannelMetadataResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		goData,
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onSetUserMetadataResult
func cgo_RtmEventHandlerBridge_onSetUserMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userId *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnSetUserMetadataResult == nil {
		return
	}

	client.handler.OnSetUserMetadataResult(
		uint64(requestId),
		C.GoString(userId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onUpdateUserMetadataResult
func cgo_RtmEventHandlerBridge_onUpdateUserMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userId *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnUpdateUserMetadataResult == nil {
		return
	}

	client.handler.OnUpdateUserMetadataResult(
		uint64(requestId),
		C.GoString(userId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onRemoveUserMetadataResult
func cgo_RtmEventHandlerBridge_onRemoveUserMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userId *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnRemoveUserMetadataResult == nil {
		return
	}

	client.handler.OnRemoveUserMetadataResult(
		uint64(requestId),
		C.GoString(userId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onGetUserMetadataResult
func cgo_RtmEventHandlerBridge_onGetUserMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userId *C.char, data *C.struct_C_Metadata, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnGetUserMetadataResult == nil {
		return
	}

	client.handler.OnGetUserMetadataResult(
		uint64(requestId),
		C.GoString(userId),
		CMetadataToIMetadata(data),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onSubscribeUserMetadataResult
func cgo_RtmEventHandlerBridge_onSubscribeUserMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userId *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnSubscribeUserMetadataResult == nil {
		return
	}

	client.handler.OnSubscribeUserMetadataResult(
		uint64(requestId),
		C.GoString(userId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onSetLockResult
func cgo_RtmEventHandlerBridge_onSetLockResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, lockName *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnSetLockResult == nil {
		return
	}

	client.handler.OnSetLockResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		C.GoString(lockName),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onRemoveLockResult
func cgo_RtmEventHandlerBridge_onRemoveLockResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, lockName *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnRemoveLockResult == nil {
		return
	}

	client.handler.OnRemoveLockResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		C.GoString(lockName),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onReleaseLockResult
func cgo_RtmEventHandlerBridge_onReleaseLockResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, lockName *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnReleaseLockResult == nil {
		return
	}

	client.handler.OnReleaseLockResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		C.GoString(lockName),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onAcquireLockResult
func cgo_RtmEventHandlerBridge_onAcquireLockResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, lockName *C.char, errorCode C.enum_C_RTM_ERROR_CODE, errorDetails *C.char) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnAcquireLockResult == nil {
		return
	}

	client.handler.OnAcquireLockResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		C.GoString(lockName),
		int(errorCode),
		C.GoString(errorDetails),
	)
}

//export cgo_RtmEventHandlerBridge_onRevokeLockResult
func cgo_RtmEventHandlerBridge_onRevokeLockResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, lockName *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnRevokeLockResult == nil {
		return
	}

	client.handler.OnRevokeLockResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		C.GoString(lockName),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onGetLocksResult
func cgo_RtmEventHandlerBridge_onGetLocksResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, channelType C.enum_C_RTM_CHANNEL_TYPE, lockDetailList *C.struct_C_LockDetail, count C.size_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnGetLocksResult == nil {
		return
	}
	goLockDetail := copyLockDetails(lockDetailList, count)

	client.handler.OnGetLocksResult(
		uint64(requestId),
		C.GoString(channelName),
		RtmChannelType(channelType),
		goLockDetail,
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onWhoNowResult
func cgo_RtmEventHandlerBridge_onWhoNowResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userStateList *C.struct_C_UserState, count C.size_t, nextPage *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnWhoNowResult == nil {
		return
	}
	goUserState := copyUserStates(userStateList, count)

	client.handler.OnWhoNowResult(
		uint64(requestId),
		goUserState,
		C.GoString(nextPage),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onGetOnlineUsersResult
func cgo_RtmEventHandlerBridge_onGetOnlineUsersResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userStateList *C.struct_C_UserState, count C.size_t, nextPage *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnGetOnlineUsersResult == nil {
		return
	}
	goUserState := copyUserStates(userStateList, count)

	client.handler.OnGetOnlineUsersResult(
		uint64(requestId),
		goUserState,
		C.GoString(nextPage),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onWhereNowResult
func cgo_RtmEventHandlerBridge_onWhereNowResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channels *C.struct_C_ChannelInfo, count C.size_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnWhereNowResult == nil {
		return
	}
	goChannelInfo := copyChannelInfos(channels, count)

	client.handler.OnWhereNowResult(
		uint64(requestId),
		goChannelInfo,
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onGetUserChannelsResult
func cgo_RtmEventHandlerBridge_onGetUserChannelsResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channels *C.struct_C_ChannelInfo, count C.size_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnGetUserChannelsResult == nil {
		return
	}
	goChannelInfo := copyChannelInfos(channels, count)

	client.handler.OnGetUserChannelsResult(
		uint64(requestId),
		goChannelInfo,
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onPresenceSetStateResult
func cgo_RtmEventHandlerBridge_onPresenceSetStateResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnPresenceSetStateResult == nil {
		return
	}

	client.handler.OnPresenceSetStateResult(
		uint64(requestId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onPresenceRemoveStateResult
func cgo_RtmEventHandlerBridge_onPresenceRemoveStateResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnPresenceRemoveStateResult == nil {
		return
	}

	client.handler.OnPresenceRemoveStateResult(
		uint64(requestId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onPresenceGetStateResult
func cgo_RtmEventHandlerBridge_onPresenceGetStateResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, state *C.struct_C_UserState, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnPresenceGetStateResult == nil {
		return
	}
	goUserState := CUserStateToUserState(state)

	client.handler.OnPresenceGetStateResult(
		uint64(requestId),
		goUserState,
		int(errorCode),
	)
}

// newly added callback functions

//export cgo_RtmEventHandlerBridge_onLinkStateEvent
func cgo_RtmEventHandlerBridge_onLinkStateEvent(handler *C.struct_C_IRtmEventHandler,
	event *C.struct_C_LinkStateEvent) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnLinkStateEvent == nil {
		return
	}
	goLinkStateEvent := CLinkStateEventToLinkStateEvent(event)
	client.handler.OnLinkStateEvent(
		goLinkStateEvent,
	)
}

//export cgo_RtmEventHandlerBridge_onLogoutResult
func cgo_RtmEventHandlerBridge_onLogoutResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnLogoutResult == nil {
		return
	}

	client.handler.OnLogoutResult(
		uint64(requestId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onRenewTokenResult
func cgo_RtmEventHandlerBridge_onRenewTokenResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, serverType C.enum_C_RTM_SERVICE_TYPE, channelName *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnRenewTokenResult == nil {
		return
	}

	client.handler.OnRenewTokenResult(
		uint64(requestId),
		RtmServiceType(serverType),
		C.GoString(channelName),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onPublishTopicMessageResult
func cgo_RtmEventHandlerBridge_onPublishTopicMessageResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, topic *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnPublishTopicMessageResult == nil {
		return
	}

	client.handler.OnPublishTopicMessageResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(topic),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onUnsubscribeTopicResult
func cgo_RtmEventHandlerBridge_onUnsubscribeTopicResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, topic *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnUnsubscribeTopicResult == nil {
		return
	}

	client.handler.OnUnsubscribeTopicResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(topic),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onGetSubscribedUserListResult
func cgo_RtmEventHandlerBridge_onGetSubscribedUserListResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, topic *C.char, users C.struct_C_UserList, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnGetSubscribedUserListResult == nil {
		return
	}

	goUserList := CUserListToUserList(&users)

	client.handler.OnGetSubscribedUserListResult(
		uint64(requestId),
		C.GoString(channelName),
		C.GoString(topic),
		goUserList,
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onGetHistoryMessagesResult
func cgo_RtmEventHandlerBridge_onGetHistoryMessagesResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, messageList *C.struct_C_HistoryMessage, count C.size_t, newStart C.uint64_t, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnGetHistoryMessagesResult == nil {
		return
	}

	messages := copyHistoryMessages(messageList, count)

	client.handler.OnGetHistoryMessagesResult(
		uint64(requestId),
		messages,
		uint64(newStart),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onUnsubscribeUserMetadataResult
func cgo_RtmEventHandlerBridge_onUnsubscribeUserMetadataResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, userId *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnUnsubscribeUserMetadataResult == nil {
		return
	}

	client.handler.OnUnsubscribeUserMetadataResult(
		uint64(requestId),
		C.GoString(userId),
		int(errorCode),
	)
}

//export cgo_RtmEventHandlerBridge_onUnsubscribeResult
func cgo_RtmEventHandlerBridge_onUnsubscribeResult(handler *C.struct_C_IRtmEventHandler,
	requestId C.uint64_t, channelName *C.char, errorCode C.enum_C_RTM_ERROR_CODE) {

	if handler == nil {
		return
	}

	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnUnsubscribeResult == nil {
		return
	}

	client.handler.OnUnsubscribeResult(
		uint64(requestId),
		C.GoString(channelName),
		int(errorCode),
	)
}

// TokenEvent describes token expiry and affected message channels in SDK 2.6.5.
type TokenEvent struct {
	EventType       int
	Reason          string
	MessageChannels []string
	Timestamp       uint64
}

//export cgo_RtmEventHandlerBridge_onTokenEvent
func cgo_RtmEventHandlerBridge_onTokenEvent(handler *C.struct_C_IRtmEventHandler, event *C.struct_C_TokenEvent) {
	client := lookupRtmClient(handler)
	if client == nil || client.handler == nil || client.handler.OnTokenEvent == nil || event == nil {
		return
	}
	client.handler.OnTokenEvent(&TokenEvent{EventType: int(event.eventType), Reason: C.GoString(event.reason), MessageChannels: copyStrings(event.affectedResources.messageChannels.channels, event.affectedResources.messageChannels.channelCount), Timestamp: uint64(event.timestamp)})
}
