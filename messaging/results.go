package messaging

// RequestOperation identifies the native asynchronous operation producing a
// RequestResult. These values map one-to-one to the pinned result callbacks.
type RequestOperation string

const (
	// RequestLogin identifies the native onLoginResult callback.
	RequestLogin RequestOperation = "login"
	// RequestLogout identifies the native onLogoutResult callback.
	RequestLogout RequestOperation = "logout"
	// RequestPublish identifies the native onPublishResult callback.
	RequestPublish RequestOperation = "publish"
	// RequestSubscribe identifies the native onSubscribeResult callback.
	RequestSubscribe RequestOperation = "subscribe"
	// RequestUnsubscribe identifies the native onUnsubscribeResult callback.
	RequestUnsubscribe RequestOperation = "unsubscribe"
	// RequestRenewToken identifies the native onRenewTokenResult callback.
	RequestRenewToken RequestOperation = "renew_token"
	// RequestPresenceSetState identifies the native onPresenceSetStateResult callback.
	RequestPresenceSetState RequestOperation = "presence_set_state"
	// RequestPresenceRemoveState identifies the native onPresenceRemoveStateResult callback.
	RequestPresenceRemoveState RequestOperation = "presence_remove_state"
	// RequestPresenceGetState identifies the native onPresenceGetStateResult callback.
	RequestPresenceGetState RequestOperation = "presence_get_state"
	// RequestGetOnlineUsers identifies the native onGetOnlineUsersResult callback.
	RequestGetOnlineUsers RequestOperation = "get_online_users"
	// RequestWhoNow identifies the native onWhoNowResult callback.
	RequestWhoNow RequestOperation = "who_now"
	// RequestGetUserChannels identifies the native onGetUserChannelsResult callback.
	RequestGetUserChannels RequestOperation = "get_user_channels"
	// RequestWhereNow identifies the native onWhereNowResult callback.
	RequestWhereNow RequestOperation = "where_now"
	// RequestSetChannelMetadata identifies the native onSetChannelMetadataResult callback.
	RequestSetChannelMetadata RequestOperation = "set_channel_metadata"
	// RequestUpdateChannelMetadata identifies the native onUpdateChannelMetadataResult callback.
	RequestUpdateChannelMetadata RequestOperation = "update_channel_metadata"
	// RequestRemoveChannelMetadata identifies the native onRemoveChannelMetadataResult callback.
	RequestRemoveChannelMetadata RequestOperation = "remove_channel_metadata"
	// RequestGetChannelMetadata identifies the native onGetChannelMetadataResult callback.
	RequestGetChannelMetadata RequestOperation = "get_channel_metadata"
	// RequestSetUserMetadata identifies the native onSetUserMetadataResult callback.
	RequestSetUserMetadata RequestOperation = "set_user_metadata"
	// RequestUpdateUserMetadata identifies the native onUpdateUserMetadataResult callback.
	RequestUpdateUserMetadata RequestOperation = "update_user_metadata"
	// RequestRemoveUserMetadata identifies the native onRemoveUserMetadataResult callback.
	RequestRemoveUserMetadata RequestOperation = "remove_user_metadata"
	// RequestGetUserMetadata identifies the native onGetUserMetadataResult callback.
	RequestGetUserMetadata RequestOperation = "get_user_metadata"
	// RequestSubscribeUserMetadata identifies the native onSubscribeUserMetadataResult callback.
	RequestSubscribeUserMetadata RequestOperation = "subscribe_user_metadata"
	// RequestUnsubscribeUserMetadata identifies the native onUnsubscribeUserMetadataResult callback.
	RequestUnsubscribeUserMetadata RequestOperation = "unsubscribe_user_metadata"
	// RequestSetLock identifies the native onSetLockResult callback.
	RequestSetLock RequestOperation = "set_lock"
	// RequestRemoveLock identifies the native onRemoveLockResult callback.
	RequestRemoveLock RequestOperation = "remove_lock"
	// RequestReleaseLock identifies the native onReleaseLockResult callback.
	RequestReleaseLock RequestOperation = "release_lock"
	// RequestRevokeLock identifies the native onRevokeLockResult callback.
	RequestRevokeLock RequestOperation = "revoke_lock"
	// RequestAcquireLock identifies the native onAcquireLockResult callback.
	RequestAcquireLock RequestOperation = "acquire_lock"
	// RequestGetLocks identifies the native onGetLocksResult callback.
	RequestGetLocks RequestOperation = "get_locks"
	// RequestJoin identifies the native onJoinResult callback.
	RequestJoin RequestOperation = "join"
	// RequestLeave identifies the native onLeaveResult callback.
	RequestLeave RequestOperation = "leave"
	// RequestJoinTopic identifies the native onJoinTopicResult callback.
	RequestJoinTopic RequestOperation = "join_topic"
	// RequestLeaveTopic identifies the native onLeaveTopicResult callback.
	RequestLeaveTopic RequestOperation = "leave_topic"
	// RequestPublishTopicMessage identifies the native onPublishTopicMessageResult callback.
	RequestPublishTopicMessage RequestOperation = "publish_topic_message"
	// RequestUnsubscribeTopic identifies the native onUnsubscribeTopicResult callback.
	RequestUnsubscribeTopic RequestOperation = "unsubscribe_topic"
	// RequestSubscribeTopic identifies the native onSubscribeTopicResult callback.
	RequestSubscribeTopic RequestOperation = "subscribe_topic"
	// RequestGetSubscribedUserList identifies the native onGetSubscribedUserListResult callback.
	RequestGetSubscribedUserList RequestOperation = "get_subscribed_user_list"
	// RequestGetHistoryMessages identifies the native onGetHistoryMessagesResult callback.
	RequestGetHistoryMessages RequestOperation = "get_history_messages"
)

// RequestResult is a Go-owned copy of one asynchronous native result callback.
// Enable Config.ResultEvents to receive it as EventRequestResult. Every callback
// is observable, including repeated callbacks and callbacks after the caller's
// context expires. A result event does not prove a method is still waiting.
//
// Callback context and data are preserved below; fields not provided by that
// callback are zero. Payloads are independent of both native memory and the value
// returned by a waiting method. Successful submission without a callback does not
// generate a result, nor do local validation or context errors. The shared event
// queue can overflow: DroppedEvents includes these events, but request waiters
// never use the queue and still receive their first matching completion.
type RequestResult struct {
	// RequestID is the native correlation ID, unchanged.
	RequestID uint64
	// Operation identifies the callback that produced this result.
	Operation RequestOperation
	// Error is nil for native success; otherwise it preserves code and details.
	Error *Error
	// Channel contains the callback's channel name and type. A callback that
	// provides only a name leaves Type zero, rather than inventing a channel kind.
	Channel Channel
	// UserID is the callback's identity, including the SDK-resolved join identity.
	UserID string
	// Topic is the topic associated with a stream result.
	Topic string
	// TopicMetadata is the metadata carried by join/leave-topic results.
	TopicMetadata string
	// LockName is the lock associated with a lock operation.
	LockName string
	// Service identifies the service supplied by token-renewal results.
	Service ServiceType
	// Presence contains GetOnlineUsers or WhoNow query data, even on errors.
	Presence *PresencePage
	// State contains the PresenceGetState query data, even on errors.
	State *UserState
	// Channels contains GetUserChannels or WhereNow query data.
	Channels []Channel
	// Metadata contains channel/user metadata query data.
	Metadata *Metadata
	// Locks contains GetLocks query data.
	Locks []Lock
	// Subscription contains successful and failed topic publisher subscriptions.
	Subscription *TopicSubscription
	// Subscribers contains GetSubscribedUserList query data.
	Subscribers []string
	// History contains the history messages and next-page cursor.
	History *HistoryPage
}

func cloneUserState(in UserState) UserState {
	out := UserState{UserID: in.UserID, State: make(map[string]string, len(in.State))}
	for key, value := range in.State {
		out.State[key] = value
	}
	return out
}

func cloneRequestResult(in RequestResult) RequestResult {
	out := in
	if in.Error != nil {
		value := *in.Error
		out.Error = &value
	}
	if in.Presence != nil {
		value := *in.Presence
		value.Users = make([]UserState, len(in.Presence.Users))
		for i, user := range in.Presence.Users {
			value.Users[i] = cloneUserState(user)
		}
		out.Presence = &value
	}
	if in.State != nil {
		value := cloneUserState(*in.State)
		out.State = &value
	}
	out.Channels = append([]Channel(nil), in.Channels...)
	if in.Metadata != nil {
		value := *in.Metadata
		value.Items = append([]MetadataItem(nil), in.Metadata.Items...)
		out.Metadata = &value
	}
	out.Locks = append([]Lock(nil), in.Locks...)
	if in.Subscription != nil {
		value := *in.Subscription
		value.Succeeded = append([]string(nil), in.Subscription.Succeeded...)
		value.Failed = append([]string(nil), in.Subscription.Failed...)
		out.Subscription = &value
	}
	out.Subscribers = append([]string(nil), in.Subscribers...)
	if in.History != nil {
		value := *in.History
		value.Messages = append([]HistoryMessage(nil), in.History.Messages...)
		for i := range value.Messages {
			value.Messages[i].Data = append([]byte(nil), in.History.Messages[i].Data...)
		}
		out.History = &value
	}
	return out
}

func (c *Client) completeResult(event RequestResult, value any, code int, details string) {
	if code != 0 {
		event.Error = &Error{Op: string(event.Operation), Code: ErrorCode(code), Details: details}
	}
	if c.resultEvents {
		event = cloneRequestResult(event)
	}
	// Copy before completion makes the event independent of mutations by a
	// caller immediately awakened by complete.
	c.complete(event.RequestID, result{value: value, code: code, details: details})
	if c.resultEvents {
		c.emit(Event{Kind: EventRequestResult, RequestResult: &event})
	}
}
