package agorartm

/*
#include "C_IAgoraRtmClient.h"
#include "C_AgoraRtmBase.h"
#include <stdlib.h>
*/
import "C"
import (
	"sync"
	"unsafe"
)

// #region agora

// #region agora::rtm

/**
 *  Configurations for RTM Client.
 */
type RtmConfig struct {
	AppId             string
	UserId            string
	AreaCode          RtmAreaCode
	ProtocolType      uint32
	PresenceTimeout   uint32
	HeartbeatInterval uint32
	ReconnectTimeout  uint32
	Context           unsafe.Pointer
	UseStringUserId   bool
	Multipath         bool
	EventHandler      *RtmEventHandler
	LogConfig         *RtmLogConfig
	ProxyConfig       *RtmProxyConfig
	EncryptionConfig  *RtmEncryptionConfig
	PrivateConfig     *RtmPrivateConfig
}

/**
 * - For Android, it is the context of Activity or Application.
 * - For Windows, it is the window handle of app. Once set, this parameter enables you to plug
 * or unplug the video devices while they are powered.
 */

func NewRtmConfig() *RtmConfig {
	config := &RtmConfig{
		AppId:             "",
		UserId:            "",
		AreaCode:          RtmAreaCodeGLOB,
		ProtocolType:      0,
		HeartbeatInterval: 0,
		Context:           nil,
		UseStringUserId:   false,
		Multipath:         false,
		EventHandler:      nil,
		LogConfig:         nil,
		ProxyConfig:       nil,
		EncryptionConfig:  nil,
		PrivateConfig:     nil,
		PresenceTimeout:   30,
	}

	return config
}

// #region MessageEvent
type MessageEvent struct {
	ChannelType  RtmChannelType
	MessageType  RtmMessageType
	ChannelName  string
	ChannelTopic string
	Message      []byte
	Publisher    string
	CustomType   string
}

func NewMessageEvent() *MessageEvent {
	event := &MessageEvent{
		ChannelType:  RtmChannelTypeNONE,
		MessageType:  RtmMessageTypeSTRING,
		ChannelName:  "",
		ChannelTopic: "",
		Message:      make([]byte, 0),
		Publisher:    "",
		CustomType:   "",
	}

	return event
}

func (this_ *MessageEvent) fromC(cEvent *C.struct_C_MessageEvent) {
	if cEvent == nil {
		return
	}
	*this_ = MessageEvent{ChannelType: RtmChannelType(cEvent.channelType), MessageType: RtmMessageType(cEvent.messageType), ChannelName: C.GoString(cEvent.channelName), ChannelTopic: C.GoString(cEvent.channelTopic), Message: copyMessageBytes(cEvent.message, cEvent.messageLength), Publisher: C.GoString(cEvent.publisher), CustomType: C.GoString(cEvent.customType)}
}

// #endregion MessageEvent

type IntervalInfo struct {
	JoinUserList    *UserList
	LeaveUserList   *UserList
	TimeoutUserList *UserList
	UserStateList   []*UserState
	UserStateCount  uint
}

func NewIntervalInfo() *IntervalInfo {
	info := &IntervalInfo{
		JoinUserList:    NewUserList(),
		LeaveUserList:   NewUserList(),
		TimeoutUserList: NewUserList(),
		UserStateList:   make([]*UserState, 0),
		UserStateCount:  0,
	}

	return info
}

// #endregion IntervalInfo

type SnapshotInfo struct {
	UserStateList []*UserState
	UserCount     uint
}

func NewSnapshotInfo() *SnapshotInfo {
	info := &SnapshotInfo{
		UserStateList: make([]*UserState, 0),
		UserCount:     0,
	}

	return info
}

// #endregion SnapshotInfo

type PresenceEvent struct {
	Type           int
	ChannelType    RtmChannelType
	ChannelName    string
	Publisher      string
	StateItems     []*StateItem
	StateItemCount uint
	Interval       *IntervalInfo
	Snapshot       *SnapshotInfo
}

func NewPresenceEvent() *PresenceEvent {
	event := &PresenceEvent{
		Type:           0,
		ChannelType:    0,
		ChannelName:    "",
		Publisher:      "",
		StateItems:     make([]*StateItem, 0),
		StateItemCount: 0,
		Interval:       nil,
		Snapshot:       nil,
	}

	return event
}

func (this_ *PresenceEvent) fromC(cEvent *C.struct_C_PresenceEvent) {
	if cEvent == nil {
		return
	}
	*this_ = PresenceEvent{Type: int(cEvent._type), ChannelType: RtmChannelType(cEvent.channelType), ChannelName: C.GoString(cEvent.channelName), Publisher: C.GoString(cEvent.publisher)}
	states := copyStateItems(cEvent.stateItems, cEvent.stateItemCount)
	this_.StateItems = make([]*StateItem, len(states))
	for i := range states {
		this_.StateItems[i] = &states[i]
	}
	this_.StateItemCount = uint(len(states))
	intervalUsers := copyUserStates(cEvent.interval.userStateList, cEvent.interval.userStateCount)
	this_.Interval = &IntervalInfo{JoinUserList: CUserListToUserList(&cEvent.interval.joinUserList), LeaveUserList: CUserListToUserList(&cEvent.interval.leaveUserList), TimeoutUserList: CUserListToUserList(&cEvent.interval.timeoutUserList), UserStateList: make([]*UserState, len(intervalUsers)), UserStateCount: uint(len(intervalUsers))}
	for i := range intervalUsers {
		this_.Interval.UserStateList[i] = &intervalUsers[i]
	}
	snapshotUsers := copyUserStates(cEvent.snapshot.userStateList, cEvent.snapshot.userCount)
	this_.Snapshot = &SnapshotInfo{UserStateList: make([]*UserState, len(snapshotUsers)), UserCount: uint(len(snapshotUsers))}
	for i := range snapshotUsers {
		this_.Snapshot.UserStateList[i] = &snapshotUsers[i]
	}
}

// #endregion PresenceEvent

type TopicEvent struct {
	Type           int
	ChannelName    string
	Publisher      string
	TopicInfos     []*TopicInfo
	TopicInfoCount uint
}

func NewTopicEvent() *TopicEvent {
	event := &TopicEvent{
		Type:           0,
		ChannelName:    "",
		Publisher:      "",
		TopicInfos:     make([]*TopicInfo, 0),
		TopicInfoCount: 0,
	}

	return event
}

func (this_ *TopicEvent) fromC(cEvent *C.struct_C_TopicEvent) {
	if cEvent == nil {
		return
	}
	*this_ = TopicEvent{Type: int(cEvent._type), ChannelName: C.GoString(cEvent.channelName), Publisher: C.GoString(cEvent.publisher)}
	infos := nativeSlice(cEvent.topicInfos, cEvent.topicInfoCount)
	this_.TopicInfos = make([]*TopicInfo, len(infos))
	this_.TopicInfoCount = uint(len(infos))
	for i := range infos {
		src := &infos[i]
		publishers := nativeSlice(src.publishers, src.publisherCount)
		info := &TopicInfo{Topic: C.GoString(src.topic), Publishers: make([]PublisherInfo, len(publishers))}
		for j := range publishers {
			info.Publishers[j] = PublisherInfo{UserId: C.GoString(publishers[j].publisherUserId), Meta: C.GoString(publishers[j].publisherMeta)}
		}
		this_.TopicInfos[i] = info
	}
}

// #endregion TopicEvent

type LockEvent struct {
	ChannelType    RtmChannelType
	EventType      int
	ChannelName    string
	LockDetailList []*LockDetail
	Count          uint
}

func NewLockEvent() *LockEvent {
	event := &LockEvent{
		ChannelType:    RtmChannelTypeNONE,
		EventType:      0,
		ChannelName:    "",
		LockDetailList: make([]*LockDetail, 0),
		Count:          0,
	}

	return event
}

func (this_ *LockEvent) fromC(cEvent *C.struct_C_LockEvent) {
	if cEvent == nil {
		return
	}
	*this_ = LockEvent{ChannelType: RtmChannelType(cEvent.channelType), EventType: int(cEvent.eventType), ChannelName: C.GoString(cEvent.channelName)}
	details := copyLockDetails(cEvent.lockDetailList, cEvent.count)
	this_.LockDetailList = make([]*LockDetail, len(details))
	this_.Count = uint(len(details))
	for i := range details {
		this_.LockDetailList[i] = &details[i]
	}
}

// #endregion LockEvent

type StorageEvent struct {
	ChannelType RtmChannelType
	StorageType RtmStorageType
	EventType   int
	Target      string
	Data        *IMetadata
}

func NewStorageEvent() *StorageEvent {
	event := &StorageEvent{
		ChannelType: RtmChannelTypeNONE,
		StorageType: RtmStorageTypeNONE,
		EventType:   0,
		Target:      "",
		Data:        nil,
	}

	return event
}

func (this_ *StorageEvent) fromC(cEvent *C.struct_C_StorageEvent) {
	if cEvent == nil {
		return
	}
	*this_ = StorageEvent{ChannelType: RtmChannelType(cEvent.channelType), StorageType: RtmStorageType(cEvent.storageType), EventType: int(cEvent.eventType), Target: C.GoString(cEvent.target), Data: CMetadataToIMetadata(cEvent.data)}
}

type IRtmClient struct {
	rtmClient     unsafe.Pointer
	handler       *RtmEventHandler
	history       *IRtmHistory
	presence      *IRtmPresence
	lock          *IRtmLock
	storage       *IRtmStorage
	releaseOnce   sync.Once
	releaseResult int
	cEventHandler *C.struct_C_IRtmEventHandler
}

/**
 * Initializes the rtm client instance.
 *
 * @param [in] config The configurations for RTM Client.
 * @param [in] eventHandler .
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
/**
 * Creates the rtm client object and returns the pointer.
 *
 * @return Pointer of the rtm client object.
 */
func NewRtmClient(config *RtmConfig) *IRtmClient {
	client, _ := NewRtmClientWithError(config)
	return client
}

// NewRtmClientWithError preserves the native initialization error for callers.
func NewRtmClientWithError(config *RtmConfig) (*IRtmClient, int) {
	if config == nil {
		return nil, int(C.RTM_ERROR_INVALID_PARAMETER)
	}

	cConfig := C.C_RtmConfig_New()
	if cConfig == nil {
		return nil, int(C.RTM_ERROR_NOT_INITIALIZED)
	}
	defer C.C_RtmConfig_Delete(cConfig)

	cConfig.appId = C.CString(config.AppId)
	defer C.free(unsafe.Pointer(cConfig.appId))
	cConfig.userId = C.CString(config.UserId)
	defer C.free(unsafe.Pointer(cConfig.userId))
	cConfig.areaCode = C.enum_C_RTM_AREA_CODE(config.AreaCode)
	cConfig.protocolType = config.ProtocolType
	cConfig.presenceTimeout = C.uint32_t(config.PresenceTimeout)
	cConfig.heartbeatInterval = C.uint32_t(config.HeartbeatInterval)
	cConfig.reconnectTimeout = C.uint32_t(config.ReconnectTimeout)
	cConfig.multipath = C.bool(config.Multipath)
	cConfig.context = config.Context
	cConfig.useStringUserId = C.bool(config.UseStringUserId)

	// add log config
	cLogConfig := (*C.struct_C_RtmLogConfig)(nil)
	if config.LogConfig != nil {
		cLogConfig = (*C.struct_C_RtmLogConfig)(config.LogConfig.toC())
		cConfig.logConfig.filePath = cLogConfig.filePath
		cConfig.logConfig.fileSizeInKB = cLogConfig.fileSizeInKB
		cConfig.logConfig.level = cLogConfig.level
	}

	defer freeRtmLogConfig(unsafe.Pointer(cLogConfig))

	// add encryption config
	cEncryptionConfig := (*C.struct_C_RtmEncryptionConfig)(nil)
	if config.EncryptionConfig != nil {
		cEncryptionConfig = (*C.struct_C_RtmEncryptionConfig)(config.EncryptionConfig.toC())
		cConfig.encryptionConfig.encryptionMode = cEncryptionConfig.encryptionMode
		cConfig.encryptionConfig.encryptionKey = cEncryptionConfig.encryptionKey
		cConfig.encryptionConfig.encryptionSalt = cEncryptionConfig.encryptionSalt
	}
	defer freeRtmEncryptionConfig(unsafe.Pointer(cEncryptionConfig))

	// add proxy config
	cProxyConfig := (*C.struct_C_RtmProxyConfig)(nil)
	if config.ProxyConfig != nil {
		cProxyConfig = (*C.struct_C_RtmProxyConfig)(config.ProxyConfig.toC())
		cConfig.proxyConfig.proxyType = cProxyConfig.proxyType
		cConfig.proxyConfig.server = cProxyConfig.server
		cConfig.proxyConfig.port = cProxyConfig.port
		cConfig.proxyConfig.account = cProxyConfig.account
		cConfig.proxyConfig.password = cProxyConfig.password
	}
	defer freeRtmProxyConfig(unsafe.Pointer(cProxyConfig))

	// add private config
	cPrivateConfig := (*C.struct_C_RtmPrivateConfig)(nil)
	if config.PrivateConfig != nil {
		cPrivateConfig = (*C.struct_C_RtmPrivateConfig)(config.PrivateConfig.toC())
		cConfig.privateConfig.serviceType = cPrivateConfig.serviceType
		cConfig.privateConfig.accessPointHosts = cPrivateConfig.accessPointHosts
		cConfig.privateConfig.accessPointHostsCount = cPrivateConfig.accessPointHostsCount
	}
	defer freeRtmPrivateConfig(unsafe.Pointer(cPrivateConfig))

	// The native SDK requires a handler even when callers do not want events.
	client := &IRtmClient{handler: &RtmEventHandler{}}
	if config.EventHandler != nil {
		handler := *config.EventHandler // Immutable for the native client's lifetime.
		client.handler = &handler
	}
	client.cEventHandler = CRtmEventHandler()
	if client.cEventHandler == nil {
		return nil, int(C.RTM_ERROR_NOT_INITIALIZED)
	}
	cConfig.eventHandler = client.cEventHandler
	registerRtmClient(client)

	success := false
	defer func() {
		if !success {
			client.Release()
		}
	}()

	var errorCode C.int
	rtmClient := C.agora_rtm_client_create(cConfig, &errorCode)

	if rtmClient == nil {
		if errorCode == 0 {
			errorCode = C.RTM_ERROR_NOT_INITIALIZED
		}
		return nil, int(errorCode)
	}

	client.rtmClient = unsafe.Pointer(rtmClient)
	if errorCode != 0 {
		return nil, int(errorCode)
	}

	// get storage
	cStorage := C.agora_rtm_client_get_storage(client.rtmClient)
	if cStorage == nil {
		return nil, int(C.RTM_ERROR_NOT_INITIALIZED)
	}
	client.storage = &IRtmStorage{rtmStorage: unsafe.Pointer(cStorage)}

	// get lock
	cLock := C.agora_rtm_client_get_lock(client.rtmClient)
	if cLock == nil {
		return nil, int(C.RTM_ERROR_NOT_INITIALIZED)
	}
	client.lock = &IRtmLock{rtmLock: unsafe.Pointer(cLock)}

	// get presence
	cPresence := C.agora_rtm_client_get_presence(client.rtmClient)
	if cPresence == nil {
		return nil, int(C.RTM_ERROR_NOT_INITIALIZED)
	}
	client.presence = &IRtmPresence{rtmPresence: unsafe.Pointer(cPresence)}

	// get history
	cHistory := C.agora_rtm_client_get_history(client.rtmClient)
	if cHistory == nil {
		return nil, int(C.RTM_ERROR_NOT_INITIALIZED)
	}
	client.history = &IRtmHistory{rtmHistory: unsafe.Pointer(cHistory)}

	success = true
	return client, 0
}

/**
 * Release the rtm client instance.
 *
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
// Release stops accepting callbacks before destroying native resources. The caller must
// serialize Release with SDK methods and must not call it from an SDK callback.
func (this_ *IRtmClient) Release() int {
	if this_ == nil {
		return 0
	}
	this_.releaseOnce.Do(func() {
		unregisterRtmClient(this_)
		if this_.rtmClient != nil {
			this_.releaseResult = int(C.agora_rtm_client_release(this_.rtmClient))
		}
		if this_.cEventHandler != nil {
			C.C_IRtmEventHandler_Delete(this_.cEventHandler)
		}
		this_.rtmClient = nil
		this_.cEventHandler = nil
		if this_.history != nil {
			this_.history.rtmHistory = nil
		}
		if this_.presence != nil {
			this_.presence.rtmPresence = nil
		}
		if this_.lock != nil {
			this_.lock.rtmLock = nil
		}
		if this_.storage != nil {
			this_.storage.rtmStorage = nil
		}
		this_.history, this_.presence, this_.lock, this_.storage = nil, nil, nil, nil
		// handler is immutable: callbacks already admitted retain a safe Go reference.
	})
	return this_.releaseResult
}

/**
 * Login the Agora RTM service. The operation result will be notified by \ref agora::rtm::IRtmEventHandler::onLoginResult callback.
 *
 * @param [in] token Token used to login RTM service.
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) Login(token string) (int, uint64) {

	// Native callbacks determine login state.
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}

	// do really login
	var requestId uint64
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))
	ret := int(C.agora_rtm_client_login(this_.rtmClient,
		cToken,
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))

	return int(ret), requestId
}

/**
 * Logout the Agora RTM service. Be noticed that this method will break the rtm service including storage/lock/presence.
 *
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) Logout() (int, uint64) {
	// Native callbacks determine login state.
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}
	var requestId uint64
	ret := int(C.agora_rtm_client_logout(this_.rtmClient,
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))

	return ret, requestId
}

/**
 * Get the storage instance.
 *
 * @return
 * - return NULL if error occurred
 */
func (this_ *IRtmClient) GetStorage() *IRtmStorage {
	// validity check
	if this_ == nil || this_.storage == nil || this_.rtmClient == nil {
		return nil
	}
	return this_.storage
}

/**
 * Get the lock instance.
 *
 * @return
 * - return NULL if error occurred
 */
func (this_ *IRtmClient) GetLock() *IRtmLock {
	// validity check
	if this_ == nil || this_.lock == nil || this_.rtmClient == nil {
		return nil
	}
	return this_.lock
}

/**
 * Get the presence instance.
 *
 * @return
 * - return NULL if error occurred
 */
func (this_ *IRtmClient) GetPresence() *IRtmPresence {
	// validity check
	if this_ == nil || this_.presence == nil || this_.rtmClient == nil {
		return nil
	}
	return this_.presence
}

/**
 * Get the history instance.
 *
 * @return
 * - return NULL if error occurred
 */
func (this_ *IRtmClient) GetHistory() *IRtmHistory {
	// validity check
	if this_ == nil || this_.history == nil || this_.rtmClient == nil {
		return nil
	}
	return this_.history
}

/**
 * Renews the token. Once a token is enabled and used, it expires after a certain period of time.
 * You should generate a new token on your server, call this method to renew it.
 *
 * @param [in] token Token used renew.
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) RenewToken(token string) (int, uint64) {
	// validity check
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}
	// do really renew token
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))
	var requestId uint64
	ret := int(C.agora_rtm_client_renew_token(this_.rtmClient,
		cToken,
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))
	return int(ret), requestId
}

/**
 * Publish a message in the channel.
 *
 * @param [in] channelName The name of the channel.
 * @param [in] message The content of the message.
 * @param [in] length The length of the message.
 * @param [in] option The option of the message.
 * @param [out] requestId The related request id of this operation.
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) Publish(channelName string, message []byte, option *PublishOptions) (int, uint64) {

	// Reject released handles; native code validates login state.
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}

	// do really publish
	length := int(len(message))
	cChannelName := C.CString(channelName)
	defer C.free(unsafe.Pointer(cChannelName))
	cMessage := C.CBytes(message)
	defer C.free(unsafe.Pointer(cMessage))
	var cOption unsafe.Pointer
	if option != nil {
		cOption = option.toC()
		defer freePublishOptions(cOption)
	} else {
		cOption = NewPublishOptions().toC()
		defer freePublishOptions(cOption)
	}

	var requestId uint64
	ret := int(C.agora_rtm_client_publish(this_.rtmClient,
		cChannelName,
		(*C.char)(cMessage),
		C.size_t(length),
		(*C.struct_C_PublishOptions)(cOption),
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))
	return ret, requestId
}

/**
 * Send a message to a channel.
 *
 * @param [in] channelName The name of the channel.
 * @param [in] message The content of the message.
 * @param [in] length The length of the message.
 * @param [out] requestId The related request id of this operation.
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) SendChannelMessage(channelName string, message []byte) (int, uint64) {
	// Reject released handles; native code validates login state.
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}

	// do really send channel message
	length := int(len(message))
	var requestId uint64
	cChannelName := C.CString(channelName)
	defer C.free(unsafe.Pointer(cChannelName))
	cMessage := C.CBytes(message)
	defer C.free(unsafe.Pointer(cMessage))
	opt := NewPublishOptions()
	opt.ChannelType = RtmChannelTypeMESSAGE
	opt.MessageType = RtmMessageTypeBINARY

	cOption := opt.toC()
	defer freePublishOptions(cOption)

	ret := int(C.agora_rtm_client_publish(this_.rtmClient,
		cChannelName,
		(*C.char)(cMessage),
		C.size_t(length),
		(*C.struct_C_PublishOptions)(cOption),
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))
	return ret, requestId
}

/**
 * Send a message to a user.
 *
 * @param [in] userId The id of the user.
 * @param [in] message The content of the message.
 * @param [in] length The length of the message.
 * @param [out] requestId The related request id of this operation.
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) SendUserMessage(userId string, message []byte) (int, uint64) {
	// Reject released handles; native code validates login state.
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}

	// do really send user message
	length := int(len(message))
	cUserId := C.CString(userId)
	defer C.free(unsafe.Pointer(cUserId))
	cMessage := C.CBytes(message)
	defer C.free(unsafe.Pointer(cMessage))
	opt := NewPublishOptions()
	opt.ChannelType = RtmChannelTypeUSER
	opt.MessageType = RtmMessageTypeBINARY

	cOption := opt.toC()
	defer freePublishOptions(cOption)
	var requestId uint64

	ret := int(C.agora_rtm_client_publish(this_.rtmClient,
		cUserId,
		(*C.char)(cMessage),
		C.size_t(length),
		(*C.struct_C_PublishOptions)(cOption),
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))
	return ret, requestId
}

/**
 * Subscribe a channel.
 *
 * @param [in] channelName The name of the channel.
 * @param [in] options The options of subscribe the channel.
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) Subscribe(channelName string, option *SubscribeOptions) (int, uint64) {
	// Reject released handles; native code validates login state.
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}
	if option == nil {
		option = NewSubscribeOptions()
	}

	// do really subscribe
	cChannelName := C.CString(channelName)
	defer C.free(unsafe.Pointer(cChannelName))
	cOption := option.toC()
	defer freeSubscribeOptions(cOption)
	var requestId uint64
	ret := int(C.agora_rtm_client_subscribe(this_.rtmClient,
		cChannelName,
		(*C.struct_C_SubscribeOptions)(cOption),
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))
	return ret, requestId
}

/**
 * Unsubscribe a channel.
 *
 * @param [in] channelName The name of the channel.
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) Unsubscribe(channelName string) (int, uint64) {
	// Reject released handles; native code validates login state.
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED), 0
	}

	// do really unsubscribe
	cChannelName := C.CString(channelName)
	defer C.free(unsafe.Pointer(cChannelName))
	var requestId uint64
	ret := int(C.agora_rtm_client_unsubscribe(this_.rtmClient,
		cChannelName,
		(*C.uint64_t)(unsafe.Pointer(&requestId)),
	))
	return int(ret), requestId
}

/**
 * Create a stream channel instance.
 *
 * @param [in] channelName The Name of the channel.
 * @return
 * - return NULL if error occurred
 */
func (this_ *IRtmClient) CreateStreamChannel(channelName string) *IStreamChannel {
	channel, _ := this_.CreateStreamChannelWithError(channelName)
	return channel
}

// CreateStreamChannelWithError preserves the native channel initialization error.
func (this_ *IRtmClient) CreateStreamChannelWithError(channelName string) (*IStreamChannel, int) {
	if this_ == nil || this_.rtmClient == nil {
		return nil, int(C.RTM_ERROR_NOT_INITIALIZED)
	}
	cChannelName := C.CString(channelName)
	defer C.free(unsafe.Pointer(cChannelName))
	var errorCode C.int
	ptr := C.agora_rtm_client_create_stream_channel(this_.rtmClient, cChannelName, &errorCode)
	if ptr == nil {
		if errorCode == 0 {
			errorCode = C.RTM_ERROR_NOT_INITIALIZED
		}
		return nil, int(errorCode)
	}
	if errorCode != 0 {
		C.agora_rtm_stream_channel_release(ptr)
		return nil, int(errorCode)
	}
	return &IStreamChannel{streamChannel: unsafe.Pointer(ptr)}, 0
}

/**
 * Set parameters of the sdk or engine
 *
 * @param [in] parameters The parameters in json format
 * @return
 * - 0: Success.
 * - < 0: Failure.
 */
func (this_ *IRtmClient) SetParameters(parameters string) int {

	// validity check: can't set parameters if rtm client is not created
	if this_ == nil || this_.rtmClient == nil {
		return int(C.RTM_ERROR_NOT_INITIALIZED)
	}

	// do really set parameters
	cParameters := C.CString(parameters)
	defer C.free(unsafe.Pointer(cParameters))

	ret := int(C.agora_rtm_client_set_parameters(this_.rtmClient,
		cParameters,
	))
	return ret
}

// #endregion IRtmClient

/**
 * Convert error code to error string
 *
 * @param [in] errorCode Received error code
 * @return The error reason
 */
func GetErrorReason(errorCode int) string {
	return C.GoString(C.agora_rtm_client_get_error_reason(C.int(errorCode)))
}

/**
 * Get the version info of the Agora RTM SDK.
 *
 * @return The version info of the Agora RTM SDK.
 */
func GetVersion() string {
	return C.GoString(C.agora_rtm_client_get_version())
}

// #endregion agora::rtm

// #endregion agora
