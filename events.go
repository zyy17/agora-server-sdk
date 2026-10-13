package agorasdk

import (
	"slices"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

type EventKind string

const (
	Connected           EventKind = "connected"
	Disconnected        EventKind = "disconnected"
	Connecting          EventKind = "connecting"
	Reconnecting        EventKind = "reconnecting"
	Reconnected         EventKind = "reconnected"
	ConnectionLost      EventKind = "connection_lost"
	ConnectionFailed    EventKind = "connection_failed"
	UserJoined          EventKind = "user_joined"
	UserLeft            EventKind = "user_left"
	TokenExpiring       EventKind = "token_expiring"
	TokenExpired        EventKind = "token_expired"
	SDKError            EventKind = "error"
	StreamError         EventKind = "stream_error"
	EncryptionError     EventKind = "encryption_error"
	AudioMetadata       EventKind = "audio_metadata"
	AudioVolume         EventKind = "audio_volume"
	AudioPublished      EventKind = "audio_published"
	VideoPublished      EventKind = "video_published"
	AudioUnpublished    EventKind = "audio_unpublished"
	VideoUnpublished    EventKind = "video_unpublished"
	AudioSubscribed     EventKind = "audio_subscribed"
	VideoSubscribed     EventKind = "video_subscribed"
	AudioStateChanged   EventKind = "audio_state_changed"
	VideoStateChanged   EventKind = "video_state_changed"
	PublishStateChanged EventKind = "publish_state_changed"
	UserInfoUpdated     EventKind = "user_info_updated"
	Statistics          EventKind = "statistics"
	KeyFrameRequested   EventKind = "keyframe_requested"
	CapabilityFallback  EventKind = "capability_fallback"
)

// Event contains only Go-owned data. Data is a copied native statistics value,
// []Volume, []byte (metadata), or nil. Tokens and native handles are never exposed.
type Event struct {
	Kind                EventKind
	Channel, UserID     string
	Reason, State, Code int
	Message             string
	Data                any
}
type Volume struct {
	UserID     string
	Level, VAD uint32
	Pitch      float64
}
type Message struct {
	Channel, UserID string
	StreamID        int
	Data            []byte
	SentAtMillis    int64
}

func (s *Session) event(e Event) {
	if e.Channel == "" {
		e.Channel = s.cfg.Channel
	}
	s.events.push(e)
}
func (s *Session) state(kind EventKind, info *rtc.RtcConnectionInfo, reason int) {
	copy := s.Info()
	copy.Reason = reason
	if info != nil {
		copy.Channel = info.ChannelId
		copy.UserID = info.LocalUserId
		copy.State = info.State
		copy.ConnectionID = info.ConnectionId
	}
	s.info.Store(&copy)
	s.event(Event{Kind: kind, Reason: reason, State: copy.State})
}
func (s *Session) connectionObserver() *rtc.RtcConnectionObserver {
	return &rtc.RtcConnectionObserver{
		OnConnected: func(_ *rtc.RtcConnection, i *rtc.RtcConnectionInfo, r int) {
			s.state(Connected, i, r)
			s.signalReady(nil)
		},
		OnDisconnected: func(_ *rtc.RtcConnection, i *rtc.RtcConnectionInfo, r int) {
			s.state(Disconnected, i, r)
			s.signalReady(eventError("disconnected before join", r))
		},
		OnConnecting:   func(_ *rtc.RtcConnection, i *rtc.RtcConnectionInfo, r int) { s.state(Connecting, i, r) },
		OnReconnecting: func(_ *rtc.RtcConnection, i *rtc.RtcConnectionInfo, r int) { s.state(Reconnecting, i, r) },
		OnReconnected: func(_ *rtc.RtcConnection, i *rtc.RtcConnectionInfo, r int) {
			s.state(Reconnected, i, r)
			s.signalReady(nil)
		},
		OnConnectionLost: func(_ *rtc.RtcConnection, i *rtc.RtcConnectionInfo) { s.state(ConnectionLost, i, 0) },
		OnConnectionFailure: func(_ *rtc.RtcConnection, i *rtc.RtcConnectionInfo, r int) {
			s.state(ConnectionFailed, i, r)
			s.signalReady(eventError("connection failed", r))
		},
		OnTokenPrivilegeWillExpire: func(_ *rtc.RtcConnection, _ string) { s.event(Event{Kind: TokenExpiring}) },
		OnTokenPrivilegeDidExpire:  func(_ *rtc.RtcConnection) { s.event(Event{Kind: TokenExpired}) },
		OnUserJoined:               func(_ *rtc.RtcConnection, uid string) { s.event(Event{Kind: UserJoined, UserID: uid}) },
		OnUserLeft:                 func(_ *rtc.RtcConnection, uid string, r int) { s.event(Event{Kind: UserLeft, UserID: uid, Reason: r}) },
		OnError: func(_ *rtc.RtcConnection, code int, msg string) {
			s.event(Event{Kind: SDKError, Code: code, Message: msg})
		},
		OnStreamMessageError: func(_ *rtc.RtcConnection, uid string, id, code, missed, cached int) {
			s.event(Event{Kind: StreamError, UserID: uid, Code: code, Data: StreamFailure{id, missed, cached}})
		},
		OnEncryptionError: func(_ *rtc.RtcConnection, code int) { s.event(Event{Kind: EncryptionError, Code: code}) },
		OnAIQoSCapabilityMissing: func(_ *rtc.RtcConnection, fallback int) int {
			s.event(Event{Kind: CapabilityFallback, Reason: fallback})
			return -1
		},
	}
}

type StreamFailure struct{ StreamID, Missed, Cached int }

func (s *Session) localObserver() *rtc.LocalUserObserver {
	return &rtc.LocalUserObserver{
		OnStreamMessage: func(_ *rtc.LocalUser, uid string, id int, data []byte, ts int64) {
			s.messages.push(Message{s.cfg.Channel, uid, id, slices.Clone(data), ts})
		},
		OnUserInfoUpdated: func(_ *rtc.LocalUser, uid string, media, val int) {
			s.event(Event{Kind: UserInfoUpdated, UserID: uid, Reason: media, State: val})
		},
		OnUserAudioTrackSubscribed: func(_ *rtc.LocalUser, uid string, _ *rtc.RemoteAudioTrack) {
			s.event(Event{Kind: AudioSubscribed, UserID: uid})
		},
		OnUserVideoTrackSubscribed: func(_ *rtc.LocalUser, uid string, _ *rtc.VideoTrackInfo, _ *rtc.RemoteVideoTrack) {
			s.event(Event{Kind: VideoSubscribed, UserID: uid})
		},
		OnUserAudioTrackStateChanged: func(_ *rtc.LocalUser, uid string, _ *rtc.RemoteAudioTrack, state, reason, elapsed int) {
			s.event(Event{Kind: AudioStateChanged, UserID: uid, State: state, Reason: reason})
		},
		OnUserVideoTrackStateChanged: func(_ *rtc.LocalUser, uid string, _ *rtc.RemoteVideoTrack, state, reason, elapsed int) {
			s.event(Event{Kind: VideoStateChanged, UserID: uid, State: state, Reason: reason})
		},
		OnAudioPublishStateChanged: func(_ *rtc.LocalUser, channel string, old, new, elapsed int) {
			s.event(Event{Kind: PublishStateChanged, Channel: channel, State: new, Reason: old})
		},
		OnAudioVolumeIndication: func(_ *rtc.LocalUser, info []*rtc.AudioVolumeInfo, _ int, total int) {
			v := make([]Volume, 0, len(info))
			for _, i := range info {
				if i != nil {
					v = append(v, Volume{i.UserId, i.Volume, i.VAD, i.VoicePitch})
				}
			}
			s.event(Event{Kind: AudioVolume, Data: v, State: total})
		},
		OnAudioMetaDataReceived: func(_ *rtc.LocalUser, uid string, data []byte) {
			s.event(Event{Kind: AudioMetadata, UserID: uid, Data: slices.Clone(data)})
		},
		OnLocalAudioTrackStatistics: func(_ *rtc.LocalUser, v *rtc.LocalAudioTrackStats) {
			if v != nil {
				s.event(Event{Kind: Statistics, Message: "local_audio", Data: *v})
			}
		},
		OnRemoteAudioTrackStatistics: func(_ *rtc.LocalUser, uid string, v *rtc.RemoteAudioTrackStats) {
			if v != nil {
				s.event(Event{Kind: Statistics, UserID: uid, Message: "remote_audio", Data: *v})
			}
		},
		OnLocalVideoTrackStatistics: func(_ *rtc.LocalUser, v *rtc.LocalVideoTrackStats) {
			if v != nil {
				s.event(Event{Kind: Statistics, Message: "local_video", Data: *v})
			}
		},
		OnRemoteVideoTrackStatistics: func(_ *rtc.LocalUser, uid string, v *rtc.RemoteVideoTrackStats) {
			if v != nil {
				s.event(Event{Kind: Statistics, UserID: uid, Message: "remote_video", Data: *v})
			}
		},
		OnAudioTrackPublishSuccess: func(_ *rtc.LocalUser, _ *rtc.LocalAudioTrack) { s.event(Event{Kind: AudioPublished}) },
		OnAudioTrackUnpublished:    func(_ *rtc.LocalUser, _ *rtc.LocalAudioTrack) { s.event(Event{Kind: AudioUnpublished}) },
		OnVideoTrackPublishSuccess: func(_ *rtc.LocalUser, _ *rtc.LocalVideoTrack) { s.event(Event{Kind: VideoPublished}) },
		OnVideoTrackUnpublished:    func(_ *rtc.LocalUser, _ *rtc.LocalVideoTrack) { s.event(Event{Kind: VideoUnpublished}) },
		OnIntraRequestReceived:     func(_ *rtc.LocalUser) { s.event(Event{Kind: KeyFrameRequested}) },
	}
}
