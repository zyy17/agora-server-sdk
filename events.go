package agorasdk

import (
	"slices"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// EventKind identifies a native callback translated into the session event queue.
// Events can be dropped under backpressure; consume Session.Events promptly
// and inspect Stream.Stats rather than treating it as a durable state log.
type EventKind string

const (
	// Connected reports initial native connection success.
	Connected EventKind = "connected"
	// Disconnected reports a native disconnection; it does not itself call Session.Close.
	Disconnected EventKind = "disconnected"
	// Connecting reports that the native connection is being established.
	Connecting EventKind = "connecting"
	// Reconnecting reports that the native SDK is trying to restore a connection.
	Reconnecting EventKind = "reconnecting"
	// Reconnected reports native reconnection success.
	Reconnected EventKind = "reconnected"
	// ConnectionLost reports that the native SDK lost the connection.
	ConnectionLost EventKind = "connection_lost"
	// ConnectionFailed reports a native connection failure, with its reason code.
	ConnectionFailed EventKind = "connection_failed"
	// UserJoined reports a remote user's arrival in UserID.
	UserJoined EventKind = "user_joined"
	// UserLeft reports a remote user's departure in UserID, with a native Reason.
	UserLeft EventKind = "user_left"
	// TokenExpiring requests timely RTC token renewal; the event never includes the token.
	TokenExpiring EventKind = "token_expiring"
	// TokenExpired reports expired RTC privileges; obtain and submit a matching replacement token.
	TokenExpired EventKind = "token_expired"
	// SDKError carries a native error Code and diagnostic Message.
	SDKError EventKind = "error"
	// StreamError carries a data-stream failure Code and StreamFailure in Data.
	StreamError EventKind = "stream_error"
	// EncryptionError carries a native media-encryption error Code.
	EncryptionError EventKind = "encryption_error"
	// AudioMetadata carries copied audio metadata as []byte in Data.
	AudioMetadata EventKind = "audio_metadata"
	// AudioVolume carries []Volume in Data and native total volume in State.
	AudioVolume EventKind = "audio_volume"
	// AudioPublished reports native audio-track publication success.
	AudioPublished EventKind = "audio_published"
	// VideoPublished reports native video-track publication success.
	VideoPublished EventKind = "video_published"
	// AudioUnpublished reports that the native audio track was unpublished.
	AudioUnpublished EventKind = "audio_unpublished"
	// VideoUnpublished reports that the native video track was unpublished.
	VideoUnpublished EventKind = "video_unpublished"
	// AudioSubscribed reports native subscription to a remote audio track.
	AudioSubscribed EventKind = "audio_subscribed"
	// VideoSubscribed reports native subscription to a remote video track.
	VideoSubscribed EventKind = "video_subscribed"
	// AudioStateChanged carries native remote-audio State and Reason.
	AudioStateChanged EventKind = "audio_state_changed"
	// VideoStateChanged carries native remote-video State and Reason.
	VideoStateChanged EventKind = "video_state_changed"
	// PublishStateChanged carries the new native audio publication State and old state in Reason.
	PublishStateChanged EventKind = "publish_state_changed"
	// UserInfoUpdated carries native media-type information in Reason and its value in State.
	UserInfoUpdated EventKind = "user_info_updated"
	// Statistics carries a copied native statistics struct; Message identifies its kind.
	Statistics EventKind = "statistics"
	// KeyFrameRequested asks an encoded-video publisher to provide an intra frame.
	KeyFrameRequested EventKind = "keyframe_requested"
	// CapabilityFallback reports a missing AI-QoS capability and native fallback Reason.
	// The wrapper rejects the synchronous fallback callback; choose a scenario
	// explicitly with Session.SetAudioScenario if appropriate.
	CapabilityFallback EventKind = "capability_fallback"
)

// Event contains Go-owned callback data without native handles or tokens.
// Interpret State, Reason, Code, Message, and Data according to Kind; fields
// not supplied by that callback retain their zero values.
type Event struct {
	// Kind identifies the callback represented by this event.
	Kind EventKind
	// Channel is the session's channel unless a callback supplies another value.
	// UserID identifies a remote user when the callback includes one.
	Channel, UserID string
	// Reason, State, and Code preserve native values. Their meaning depends on
	// Kind; for PublishStateChanged, Reason holds the previous state.
	Reason, State, Code int
	// Message is a native error message or a statistics discriminator.
	// Statistics uses "local_audio", "remote_audio", "local_video", or "remote_video".
	Message string
	// Data is nil, []Volume for AudioVolume, []byte for AudioMetadata,
	// StreamFailure for StreamError, or a copied rtc.LocalAudioTrackStats,
	// rtc.RemoteAudioTrackStats, rtc.LocalVideoTrackStats, or
	// rtc.RemoteVideoTrackStats value for Statistics. Use a checked type assertion.
	Data any
}

// Volume is a copied native per-user audio-volume indication.
// The native SDK determines reporting cadence and telemetry scales.
type Volume struct {
	// UserID identifies the user reported by the native volume callback.
	UserID string
	// Level is the native volume level and VAD is its native voice-activity indicator.
	Level, VAD uint32
	// Pitch is the native-reported voice pitch; support depends on the processing path.
	Pitch float64
}

// Message is a copied received RTC data-stream payload, independent of RTM.
// The session's data stream is unordered and unreliable; successful sends and
// local queueing do not guarantee application processing by the recipient.
type Message struct {
	// Channel and UserID identify the remote message source.
	Channel, UserID string
	// StreamID is the native data-stream identifier.
	StreamID int
	// Data owns the copied message bytes and remains valid after Session.Close.
	Data []byte
	// SentAtMillis preserves the native send timestamp in milliseconds.
	// Its clock origin is not guaranteed to match the local wall clock.
	SentAtMillis int64
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

// StreamFailure supplies native counters accompanying a StreamError event.
// It does not provide missing message payloads or an automatic retransmission mechanism.
type StreamFailure struct {
	// StreamID identifies the native data stream that reported a failure.
	StreamID int
	// Missed is the native count of missing stream messages.
	Missed int
	// Cached is the native count of cached stream messages.
	Cached int
}

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
