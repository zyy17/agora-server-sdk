package messaging

import (
	"time"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

func copyState(s *native.UserState) UserState {
	out := UserState{State: map[string]string{}}
	if s == nil {
		return out
	}
	out.UserID = s.UserId
	for _, item := range s.States {
		out.State[item.Key] = item.Value
	}
	return out
}

func copyStates(in []native.UserState) []UserState {
	out := make([]UserState, len(in))
	for i := range in {
		out[i] = copyState(&in[i])
	}
	return out
}

func copyStatePointers(in []*native.UserState) []UserState {
	out := make([]UserState, 0, len(in))
	for _, s := range in {
		if s != nil {
			out = append(out, copyState(s))
		}
	}
	return out
}

func copyUsers(in *native.UserList) []string {
	if in == nil {
		return nil
	}
	return append([]string(nil), in.Users...)
}

func copyLock(in native.LockDetail) Lock {
	return Lock{Name: in.LockName, Owner: in.Owner, TTL: time.Duration(in.Ttl) * time.Second}
}

func copyLocks(in []native.LockDetail) []Lock {
	out := make([]Lock, len(in))
	for i, v := range in {
		out[i] = copyLock(v)
	}
	return out
}

func copyMetadata(in *native.IMetadata) Metadata {
	if in == nil {
		return Metadata{}
	}
	out := Metadata{Revision: in.GetMajorRevision()}
	for _, v := range in.Items() {
		out.Items = append(out.Items, MetadataItem{Key: v.Key, Value: v.Value, AuthorUserID: v.AuthorUserId, Revision: int64(v.Revision), UpdatedAt: int64(v.UpdateTs)})
	}
	return out
}

func (c *Client) handler() *native.RtmEventHandler {
	h := &native.RtmEventHandler{}
	h.OnLoginResult = func(id uint64, code int) {
		event := RequestResult{RequestID: id, Operation: RequestLogin}
		c.completeResult(event, nil, code, "")
	}
	h.OnLogoutResult = func(id uint64, code int) {
		event := RequestResult{RequestID: id, Operation: RequestLogout}
		c.completeResult(event, nil, code, "")
	}
	h.OnPublishResult = func(id uint64, code int) {
		event := RequestResult{RequestID: id, Operation: RequestPublish}
		c.completeResult(event, nil, code, "")
	}
	h.OnSubscribeResult = func(id uint64, target string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestSubscribe}
		event.Channel.Name = target
		c.completeResult(event, nil, code, "")
	}
	h.OnUnsubscribeResult = func(id uint64, target string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestUnsubscribe}
		event.Channel.Name = target
		c.completeResult(event, nil, code, "")
	}
	h.OnRenewTokenResult = func(id uint64, service native.RtmServiceType, channel string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestRenewToken}
		event.Service = service
		event.Channel.Name = channel
		c.completeResult(event, nil, code, "")
	}
	h.OnPresenceSetStateResult = func(id uint64, code int) {
		event := RequestResult{RequestID: id, Operation: RequestPresenceSetState}
		c.completeResult(event, nil, code, "")
	}
	h.OnPresenceRemoveStateResult = func(id uint64, code int) {
		event := RequestResult{RequestID: id, Operation: RequestPresenceRemoveState}
		c.completeResult(event, nil, code, "")
	}
	h.OnPresenceGetStateResult = func(id uint64, state *native.UserState, code int) {
		event := RequestResult{RequestID: id, Operation: RequestPresenceGetState}
		data := copyState(state)
		event.State = &data
		c.completeResult(event, data, code, "")
	}
	h.OnGetOnlineUsersResult = func(id uint64, users []native.UserState, page string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestGetOnlineUsers}
		data := PresencePage{Users: copyStates(users), NextPage: page}
		event.Presence = &data
		c.completeResult(event, data, code, "")
	}
	h.OnWhoNowResult = func(id uint64, users []native.UserState, page string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestWhoNow}
		data := PresencePage{Users: copyStates(users), NextPage: page}
		event.Presence = &data
		c.completeResult(event, data, code, "")
	}
	h.OnGetUserChannelsResult = func(id uint64, channels []native.ChannelInfo, code int) {
		event := RequestResult{RequestID: id, Operation: RequestGetUserChannels}
		data := make([]Channel, len(channels))
		for i, channel := range channels {
			data[i] = Channel{Name: channel.ChannelName, Type: channel.ChannelType}
		}
		event.Channels = data
		c.completeResult(event, data, code, "")
	}
	h.OnWhereNowResult = func(id uint64, channels []native.ChannelInfo, code int) {
		event := RequestResult{RequestID: id, Operation: RequestWhereNow}
		data := make([]Channel, len(channels))
		for i, channel := range channels {
			data[i] = Channel{Name: channel.ChannelName, Type: channel.ChannelType}
		}
		event.Channels = data
		c.completeResult(event, data, code, "")
	}
	h.OnSetChannelMetadataResult = func(id uint64, channel string, kind native.RtmChannelType, code int) {
		event := RequestResult{RequestID: id, Operation: RequestSetChannelMetadata}
		event.Channel = Channel{Name: channel, Type: kind}
		c.completeResult(event, nil, code, "")
	}
	h.OnUpdateChannelMetadataResult = func(id uint64, channel string, kind native.RtmChannelType, code int) {
		event := RequestResult{RequestID: id, Operation: RequestUpdateChannelMetadata}
		event.Channel = Channel{Name: channel, Type: kind}
		c.completeResult(event, nil, code, "")
	}
	h.OnRemoveChannelMetadataResult = func(id uint64, channel string, kind native.RtmChannelType, code int) {
		event := RequestResult{RequestID: id, Operation: RequestRemoveChannelMetadata}
		event.Channel = Channel{Name: channel, Type: kind}
		c.completeResult(event, nil, code, "")
	}
	h.OnGetChannelMetadataResult = func(id uint64, channel string, kind native.RtmChannelType, metadata *native.IMetadata, code int) {
		event := RequestResult{RequestID: id, Operation: RequestGetChannelMetadata}
		event.Channel = Channel{Name: channel, Type: kind}
		data := copyMetadata(metadata)
		event.Metadata = &data
		c.completeResult(event, data, code, "")
	}
	h.OnSetUserMetadataResult = func(id uint64, target string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestSetUserMetadata}
		event.UserID = target
		c.completeResult(event, nil, code, "")
	}
	h.OnUpdateUserMetadataResult = func(id uint64, target string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestUpdateUserMetadata}
		event.UserID = target
		c.completeResult(event, nil, code, "")
	}
	h.OnRemoveUserMetadataResult = func(id uint64, target string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestRemoveUserMetadata}
		event.UserID = target
		c.completeResult(event, nil, code, "")
	}
	h.OnGetUserMetadataResult = func(id uint64, user string, metadata *native.IMetadata, code int) {
		event := RequestResult{RequestID: id, Operation: RequestGetUserMetadata}
		event.UserID = user
		data := copyMetadata(metadata)
		event.Metadata = &data
		c.completeResult(event, data, code, "")
	}
	h.OnSubscribeUserMetadataResult = func(id uint64, target string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestSubscribeUserMetadata}
		event.UserID = target
		c.completeResult(event, nil, code, "")
	}
	h.OnUnsubscribeUserMetadataResult = func(id uint64, target string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestUnsubscribeUserMetadata}
		event.UserID = target
		c.completeResult(event, nil, code, "")
	}
	h.OnSetLockResult = func(id uint64, channel string, kind native.RtmChannelType, lock string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestSetLock}
		event.Channel = Channel{Name: channel, Type: kind}
		event.LockName = lock
		c.completeResult(event, nil, code, "")
	}
	h.OnRemoveLockResult = func(id uint64, channel string, kind native.RtmChannelType, lock string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestRemoveLock}
		event.Channel = Channel{Name: channel, Type: kind}
		event.LockName = lock
		c.completeResult(event, nil, code, "")
	}
	h.OnReleaseLockResult = func(id uint64, channel string, kind native.RtmChannelType, lock string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestReleaseLock}
		event.Channel = Channel{Name: channel, Type: kind}
		event.LockName = lock
		c.completeResult(event, nil, code, "")
	}
	h.OnRevokeLockResult = func(id uint64, channel string, kind native.RtmChannelType, lock string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestRevokeLock}
		event.Channel = Channel{Name: channel, Type: kind}
		event.LockName = lock
		c.completeResult(event, nil, code, "")
	}
	h.OnAcquireLockResult = func(id uint64, channel string, kind native.RtmChannelType, lock string, code int, details string) {
		event := RequestResult{RequestID: id, Operation: RequestAcquireLock}
		event.Channel = Channel{Name: channel, Type: kind}
		event.LockName = lock
		c.completeResult(event, nil, code, details)
	}
	h.OnGetLocksResult = func(id uint64, channel string, kind native.RtmChannelType, locks []native.LockDetail, code int) {
		event := RequestResult{RequestID: id, Operation: RequestGetLocks}
		event.Channel = Channel{Name: channel, Type: kind}
		data := copyLocks(locks)
		event.Locks = data
		c.completeResult(event, data, code, "")
	}
	h.OnJoinResult = func(id uint64, channel, user string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestJoin}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.UserID = user
		c.completeResult(event, nil, code, "")
	}
	h.OnLeaveResult = func(id uint64, channel, user string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestLeave}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.UserID = user
		c.completeResult(event, nil, code, "")
	}
	h.OnJoinTopicResult = func(id uint64, channel, user string, topic, metadata string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestJoinTopic}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.UserID = user
		event.Topic = topic
		event.TopicMetadata = metadata
		c.completeResult(event, nil, code, "")
	}
	h.OnLeaveTopicResult = func(id uint64, channel, user string, topic, metadata string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestLeaveTopic}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.UserID = user
		event.Topic = topic
		event.TopicMetadata = metadata
		c.completeResult(event, nil, code, "")
	}
	h.OnPublishTopicMessageResult = func(id uint64, channel, topic string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestPublishTopicMessage}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.Topic = topic
		c.completeResult(event, nil, code, "")
	}
	h.OnUnsubscribeTopicResult = func(id uint64, channel, topic string, code int) {
		event := RequestResult{RequestID: id, Operation: RequestUnsubscribeTopic}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.Topic = topic
		c.completeResult(event, nil, code, "")
	}
	h.OnSubscribeTopicResult = func(id uint64, channel, user, topic string, succeeded, failed native.UserList, code int) {
		event := RequestResult{RequestID: id, Operation: RequestSubscribeTopic}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.UserID = user
		event.Topic = topic
		data := TopicSubscription{Succeeded: copyUsers(&succeeded), Failed: copyUsers(&failed)}
		event.Subscription = &data
		c.completeResult(event, data, code, "")
	}
	h.OnGetSubscribedUserListResult = func(id uint64, channel, topic string, users *native.UserList, code int) {
		event := RequestResult{RequestID: id, Operation: RequestGetSubscribedUserList}
		event.Channel = Channel{Name: channel, Type: StreamChannel}
		event.Topic = topic
		data := copyUsers(users)
		event.Subscribers = data
		c.completeResult(event, data, code, "")
	}
	h.OnGetHistoryMessagesResult = func(id uint64, items []native.HistoryMessage, next uint64, code int) {
		event := RequestResult{RequestID: id, Operation: RequestGetHistoryMessages}
		messages := make([]HistoryMessage, len(items))
		for i, item := range items {
			messages[i] = HistoryMessage{Type: item.MessageType, Publisher: item.Publisher, Data: []byte(item.Message), CustomType: item.CustomType, Timestamp: item.Timestamp}
		}
		data := HistoryPage{Messages: messages, NextStart: next}
		event.History = &data
		c.completeResult(event, data, code, "")
	}
	h.OnMessageEvent = func(in *native.MessageEvent) {
		if in == nil {
			return
		}
		out := &Message{Channel: Channel{Name: in.ChannelName, Type: in.ChannelType}, Topic: in.ChannelTopic, Type: in.MessageType, Data: append([]byte(nil), in.Message...), Publisher: in.Publisher, CustomType: in.CustomType}
		c.emit(Event{Kind: EventMessage, Message: out})
	}
	h.OnPresenceEvent = func(in *native.PresenceEvent) {
		if in == nil {
			return
		}
		out := &PresenceEvent{Type: PresenceEventType(in.Type), Channel: Channel{Name: in.ChannelName, Type: in.ChannelType}, Publisher: in.Publisher, State: map[string]string{}}
		for _, v := range in.StateItems {
			if v != nil {
				out.State[v.Key] = v.Value
			}
		}
		if i := in.Interval; i != nil {
			out.Joined = copyUsers(i.JoinUserList)
			out.Left = copyUsers(i.LeaveUserList)
			out.TimedOut = copyUsers(i.TimeoutUserList)
			out.Updated = copyStatePointers(i.UserStateList)
		}
		if in.Snapshot != nil {
			out.Snapshot = copyStatePointers(in.Snapshot.UserStateList)
		}
		c.emit(Event{Kind: EventPresence, Presence: out})
	}
	h.OnTopicEvent = func(in *native.TopicEvent) {
		if in == nil {
			return
		}
		out := &TopicEvent{Type: TopicEventType(in.Type), Channel: in.ChannelName, Publisher: in.Publisher}
		for _, t := range in.TopicInfos {
			if t == nil {
				continue
			}
			info := TopicInfo{Name: t.Topic}
			for _, p := range t.Publishers {
				info.Publishers = append(info.Publishers, TopicPublisher{UserID: p.UserId, Metadata: p.Meta})
			}
			out.Topics = append(out.Topics, info)
		}
		c.emit(Event{Kind: EventTopic, Topic: out})
	}
	h.OnLockEvent = func(in *native.LockEvent) {
		if in == nil {
			return
		}
		out := &LockEvent{Type: LockEventType(in.EventType), Channel: Channel{Name: in.ChannelName, Type: in.ChannelType}}
		for _, l := range in.LockDetailList {
			if l != nil {
				out.Locks = append(out.Locks, copyLock(*l))
			}
		}
		c.emit(Event{Kind: EventLock, Lock: out})
	}
	h.OnStorageEvent = func(in *native.StorageEvent) {
		if in == nil {
			return
		}
		out := &StorageEvent{Type: StorageEventType(in.EventType), StorageType: in.StorageType, ChannelType: in.ChannelType, Target: in.Target, Metadata: copyMetadata(in.Data)}
		c.emit(Event{Kind: EventStorage, Storage: out})
	}
	h.OnLinkStateEvent = func(in *native.LinkStateEvent) {
		if in == nil {
			return
		}
		out := &LinkEvent{CurrentState: LinkState(in.CurrentState), PreviousState: LinkState(in.PreviousState), ServiceType: in.ServiceType, Operation: LinkOperation(in.Operation), ReasonCode: LinkReason(in.ReasonCode), Reason: in.Reason, AffectedChannels: append([]string(nil), in.AffectedChannels...), UnrestoredChannels: append([]string(nil), in.UnrestoredChannels...), Resumed: in.IsResumed, Timestamp: in.Timestamp}
		c.emit(Event{Kind: EventLink, Link: out})
	}
	h.OnTokenPrivilegeWillExpire = func(channel string) { c.emit(Event{Kind: EventTokenExpiring, Channel: channel}) }
	h.OnTokenEvent = func(in *native.TokenEvent) {
		if in == nil {
			return
		}
		out := &TokenEvent{Type: TokenEventType(in.EventType), Reason: in.Reason, MessageChannels: append([]string(nil), in.MessageChannels...), Timestamp: in.Timestamp}
		c.emit(Event{Kind: EventToken, Token: out})
	}
	return h
}
