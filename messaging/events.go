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
	simple := func(id uint64, code int) { c.complete(id, result{code: code}) }
	channel := func(id uint64, _ string, code int) { simple(id, code) }
	metadata := func(id uint64, _ string, _ native.RtmChannelType, code int) { simple(id, code) }
	lock := func(id uint64, _ string, _ native.RtmChannelType, _ string, code int) { simple(id, code) }
	join := func(id uint64, _ string, _ string, code int) { simple(id, code) }
	topic := func(id uint64, _ string, _ string, _ string, _ string, code int) { simple(id, code) }
	h.OnLoginResult = simple
	h.OnLogoutResult = simple
	h.OnPublishResult = simple
	h.OnSubscribeResult = channel
	h.OnUnsubscribeResult = channel
	h.OnRenewTokenResult = func(id uint64, _ native.RtmServiceType, _ string, code int) { simple(id, code) }
	h.OnPresenceSetStateResult = simple
	h.OnPresenceRemoveStateResult = simple
	h.OnPresenceGetStateResult = func(id uint64, s *native.UserState, code int) {
		c.complete(id, result{value: copyState(s), code: code})
	}
	users := func(id uint64, s []native.UserState, page string, code int) {
		c.complete(id, result{value: PresencePage{Users: copyStates(s), NextPage: page}, code: code})
	}
	h.OnGetOnlineUsersResult = users
	h.OnWhoNowResult = users
	channels := func(id uint64, list []native.ChannelInfo, code int) {
		out := make([]Channel, len(list))
		for i, v := range list {
			out[i] = Channel{Name: v.ChannelName, Type: v.ChannelType}
		}
		c.complete(id, result{value: out, code: code})
	}
	h.OnGetUserChannelsResult = channels
	h.OnWhereNowResult = channels
	h.OnSetChannelMetadataResult = metadata
	h.OnUpdateChannelMetadataResult = metadata
	h.OnRemoveChannelMetadataResult = metadata
	h.OnGetChannelMetadataResult = func(id uint64, _ string, _ native.RtmChannelType, data *native.IMetadata, code int) {
		c.complete(id, result{value: copyMetadata(data), code: code})
	}
	h.OnSetUserMetadataResult = channel
	h.OnUpdateUserMetadataResult = channel
	h.OnRemoveUserMetadataResult = channel
	h.OnGetUserMetadataResult = func(id uint64, _ string, data *native.IMetadata, code int) {
		c.complete(id, result{value: copyMetadata(data), code: code})
	}
	h.OnSubscribeUserMetadataResult = channel
	h.OnUnsubscribeUserMetadataResult = channel
	h.OnSetLockResult = lock
	h.OnRemoveLockResult = lock
	h.OnReleaseLockResult = lock
	h.OnRevokeLockResult = lock
	h.OnAcquireLockResult = func(id uint64, _ string, _ native.RtmChannelType, _ string, code int, details string) {
		c.complete(id, result{code: code, details: details})
	}
	h.OnGetLocksResult = func(id uint64, _ string, _ native.RtmChannelType, locks []native.LockDetail, code int) {
		c.complete(id, result{value: copyLocks(locks), code: code})
	}
	h.OnJoinResult = join
	h.OnLeaveResult = join
	h.OnJoinTopicResult = topic
	h.OnLeaveTopicResult = topic
	h.OnPublishTopicMessageResult = func(id uint64, _ string, _ string, code int) { simple(id, code) }
	h.OnUnsubscribeTopicResult = func(id uint64, _ string, _ string, code int) { simple(id, code) }
	h.OnSubscribeTopicResult = func(id uint64, _ string, _ string, _ string, ok native.UserList, failed native.UserList, code int) {
		c.complete(id, result{value: TopicSubscription{Succeeded: copyUsers(&ok), Failed: copyUsers(&failed)}, code: code})
	}
	h.OnGetSubscribedUserListResult = func(id uint64, _ string, _ string, users *native.UserList, code int) {
		c.complete(id, result{value: copyUsers(users), code: code})
	}
	h.OnGetHistoryMessagesResult = func(id uint64, items []native.HistoryMessage, next uint64, code int) {
		messages := make([]HistoryMessage, len(items))
		for i, v := range items {
			messages[i] = HistoryMessage{Type: v.MessageType, Publisher: v.Publisher, Data: []byte(v.Message), CustomType: v.CustomType, Timestamp: v.Timestamp}
		}
		c.complete(id, result{value: HistoryPage{Messages: messages, NextStart: next}, code: code})
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
		out := &PresenceEvent{Type: in.Type, Channel: Channel{Name: in.ChannelName, Type: in.ChannelType}, Publisher: in.Publisher, State: map[string]string{}}
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
		out := &TopicEvent{Type: in.Type, Channel: in.ChannelName, Publisher: in.Publisher}
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
		out := &LockEvent{Type: in.EventType, Channel: Channel{Name: in.ChannelName, Type: in.ChannelType}}
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
		out := &StorageEvent{Type: in.EventType, StorageType: int(in.StorageType), ChannelType: in.ChannelType, Target: in.Target, Metadata: copyMetadata(in.Data)}
		c.emit(Event{Kind: EventStorage, Storage: out})
	}
	h.OnLinkStateEvent = func(in *native.LinkStateEvent) {
		if in == nil {
			return
		}
		out := &LinkEvent{CurrentState: in.CurrentState, PreviousState: in.PreviousState, ServiceType: int(in.ServiceType), Operation: in.Operation, ReasonCode: in.ReasonCode, Reason: in.Reason, AffectedChannels: append([]string(nil), in.AffectedChannels...), UnrestoredChannels: append([]string(nil), in.UnrestoredChannels...), Resumed: in.IsResumed, Timestamp: in.Timestamp}
		c.emit(Event{Kind: EventLink, Link: out})
	}
	h.OnTokenPrivilegeWillExpire = func(channel string) { c.emit(Event{Kind: EventTokenExpiring, Channel: channel}) }
	h.OnTokenEvent = func(in *native.TokenEvent) {
		if in == nil {
			return
		}
		out := &TokenEvent{Type: in.EventType, Reason: in.Reason, MessageChannels: append([]string(nil), in.MessageChannels...), Timestamp: in.Timestamp}
		c.emit(Event{Kind: EventToken, Token: out})
	}
	return h
}
