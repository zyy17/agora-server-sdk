package agorasdk

import (
	"errors"
	"fmt"

	agoraservice "github.com/zyy17/agora-server-sdk/agora/rtc"
)

var (
	// ErrEmptyPCMQueue is the error returned when the pcm queue is empty.
	ErrEmptyPCMQueue = errors.New("the pcm queue is empty")
)

// AudioChannelType represents the audio channel type.
type AudioChannelType int32

const (
	// AudioChannelTypeMono represents the mono audio channel type.
	AudioChannelTypeMono AudioChannelType = 1

	// AudioChannelTypeStereo represents the stereo audio channel type.
	AudioChannelTypeStereo AudioChannelType = 2
)

// AudioMode represents the audio mode.
type AudioMode int32

const (
	// Push audio frames directly to the RTC connection.
	AudioModeDirect AudioMode = 0

	// Enqueue audio frames to a queue and process them in a separate routine.
	AudioModeChannel AudioMode = 1
)

// RTCConnection represents an Agora RTC connection instance.
type RTCConnection struct {
	sampleRate              int
	channels                AudioChannelType
	audioMode               AudioMode
	enableReceiveAudioFrame bool

	// The PCM queue. It's used to receive audio frames from the RTC connection.
	pcmQueue *agoraservice.Queue

	// The underlying RTC connection.
	rtcConn *agoraservice.RtcConnection
}

// RTCConnectionConfig represents the configuration for an Agora RTC connection.
type RTCConnectionConfig struct {
	sampleRate       int
	audioChannelType AudioChannelType

	// The size of the PCM queue. It's used to receive audio frames from the RTC connection.
	pcmQueueSize int
	audioMode    AudioMode

	// Whether to enable receiving audio frames from the RTC connection.
	enableReceiveAudioFrame bool

	connCfg    *agoraservice.RtcConnectionConfig
	publishCfg *agoraservice.RtcConnectionPublishConfig

	// The following callbacks are called when the corresponding event occurs in the RTC connection.
	onConnected              OnConnected
	onDisconnected           OnDisconnected
	onConnecting             OnConnecting
	onReconnecting           OnReconnecting
	onReconnected            OnReconnected
	onConnectionLost         OnConnectionLost
	onConnectionFailure      OnConnectionFailure
	onUserJoined             OnUserJoined
	onUserLeft               OnUserLeft
	onAIQoSCapabilityMissing OnAIQoSCapabilityMissing

	// The following callbacks are called when the corresponding event occurs in the local user.
	onStreamMessage              OnStreamMessage
	onUserInfoUpdated            OnUserInfoUpdated
	onUserAudioTrackSubscribed   OnUserAudioTrackSubscribed
	onUserVideoTrackSubscribed   OnUserVideoTrackSubscribed
	onUserAudioTrackStateChanged OnUserAudioTrackStateChanged
	onUserVideoTrackStateChanged OnUserVideoTrackStateChanged
	onAudioPublishStateChanged   OnAudioPublishStateChanged
	onAudioVolumeIndication      OnAudioVolumeIndication
	onAudioMetaDataReceived      OnAudioMetaDataReceived
	onLocalAudioTrackStatistics  OnLocalAudioTrackStatistics
	onRemoteAudioTrackStatistics OnRemoteAudioTrackStatistics
	onLocalVideoTrackStatistics  OnLocalVideoTrackStatistics
	onRemoteVideoTrackStatistics OnRemoteVideoTrackStatistics
	onAudioTrackPublishSuccess   OnAudioTrackPublishSuccess
	onAudioTrackUnpublished      OnAudioTrackUnpublished
	onIntraRequestReceived       OnIntraRequestReceived

	// The following callbacks are called when the corresponding event occurs for handling the audio frame.
	onRecordAudioFrame                OnRecordAudioFrame
	onPlaybackAudioFrame              OnPlaybackAudioFrame
	onMixedAudioFrame                 OnMixedAudioFrame
	onEarMonitoringAudioFrame         OnEarMonitoringAudioFrame
	onPlaybackAudioFrameBeforeMixing  OnPlaybackAudioFrameBeforeMixing
	onGetAudioFramePosition           OnGetAudioFramePosition
	onGetPlaybackAudioFrameParam      OnGetPlaybackAudioFrameParam
	onGetRecordAudioFrameParam        OnGetRecordAudioFrameParam
	onGetMixedAudioFrameParam         OnGetMixedAudioFrameParam
	onGetEarMonitoringAudioFrameParam OnGetEarMonitoringAudioFrameParam
}

// RTCConnectionOption is a function that configures a RTC connection.
type RTCConnectionOption func(*RTCConnectionConfig)

// WithSampleRate sets the sample rate for the RTC connection.
func WithSampleRate(sampleRate int) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.sampleRate = sampleRate
	}
}

// WithEnableReceiveAudioFrame sets the enable receive audio frame option.
func WithEnableReceiveAudioFrame(enableReceiveAudioFrame bool) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.enableReceiveAudioFrame = enableReceiveAudioFrame
	}
}

// WithAudioMode sets the audio mode for the RTC connection.
func WithAudioMode(audioMode AudioMode) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.audioMode = audioMode
	}
}

// WithPCMQueueSize sets the pcm queue size option.
func WithPCMQueueSize(queueSize int) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.pcmQueueSize = queueSize
	}
}

// WithAudioChannelType sets the audio channel type option.
func WithAudioChannelType(audioChannelType AudioChannelType) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.audioChannelType = audioChannelType
	}
}

// WithAutoSubscribeAudio sets the auto subscribe audio option.
func WithAutoSubscribeAudio(autoSubscribe bool) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.connCfg.AutoSubscribeAudio = autoSubscribe
	}
}

// WithAutoSubscribeVideo sets the auto subscribe video option.
func WithAutoSubscribeVideo(autoSubscribe bool) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.connCfg.AutoSubscribeVideo = autoSubscribe
	}
}

// WithClientRole sets the client role option.
func WithClientRole(clientRole agoraservice.ClientRole) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.connCfg.ClientRole = clientRole
	}
}

// WithChannelProfile sets the channel profile option.
func WithChannelProfile(channelProfile agoraservice.ChannelProfile) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.connCfg.ChannelProfile = channelProfile
	}
}

// WithAudioPublishType sets the audio publish type option.
func WithAudioPublishType(audioPublishType agoraservice.AudioPublishType) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.publishCfg.AudioPublishType = audioPublishType
	}
}

// WithAudioScenario sets the audio scenario option.
func WithAudioScenario(audioScenario agoraservice.AudioScenario) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.publishCfg.AudioScenario = audioScenario
	}
}

// WithIsPublishAudio sets the is publish audio option.
func WithIsPublishAudio(isPublishAudio bool) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.publishCfg.IsPublishAudio = isPublishAudio
	}
}

// WithIsPublishVideo sets the is publish video option.
func WithIsPublishVideo(isPublishVideo bool) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.publishCfg.IsPublishVideo = isPublishVideo
	}
}

// WithAudioProfile sets the audio profile option.
func WithAudioProfile(audioProfile agoraservice.AudioProfile) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.publishCfg.AudioProfile = audioProfile
	}
}

// WithOnConnected sets the on connected callback.
func WithOnConnected(onConnected OnConnected) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onConnected = onConnected
	}
}

// WithOnDisconnected sets the on disconnected callback.
func WithOnDisconnected(onDisconnected OnDisconnected) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onDisconnected = onDisconnected
	}
}

// WithOnStreamMessage sets the on stream message callback.
func WithOnStreamMessage(onStreamMessage OnStreamMessage) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onStreamMessage = onStreamMessage
	}
}

// WithOnUserInfoUpdated sets the on user info updated callback.
func WithOnUserInfoUpdated(onUserInfoUpdated OnUserInfoUpdated) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onUserInfoUpdated = onUserInfoUpdated
	}
}

// WithOnUserAudioTrackSubscribed sets the on user audio track subscribed callback.
func WithOnUserAudioTrackSubscribed(onUserAudioTrackSubscribed OnUserAudioTrackSubscribed) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onUserAudioTrackSubscribed = onUserAudioTrackSubscribed
	}
}

// WithOnUserVideoTrackSubscribed sets the on user video track subscribed callback.
func WithOnUserVideoTrackSubscribed(onUserVideoTrackSubscribed OnUserVideoTrackSubscribed) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onUserVideoTrackSubscribed = onUserVideoTrackSubscribed
	}
}

// WithOnUserAudioTrackStateChanged sets the on user audio track state changed callback.
func WithOnUserAudioTrackStateChanged(onUserAudioTrackStateChanged OnUserAudioTrackStateChanged) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onUserAudioTrackStateChanged = onUserAudioTrackStateChanged
	}
}

// WithOnUserVideoTrackStateChanged sets the on user video track state changed callback.
func WithOnUserVideoTrackStateChanged(onUserVideoTrackStateChanged OnUserVideoTrackStateChanged) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onUserVideoTrackStateChanged = onUserVideoTrackStateChanged
	}
}

// WithOnAudioPublishStateChanged sets the on audio publish state changed callback.
func WithOnAudioPublishStateChanged(onAudioPublishStateChanged OnAudioPublishStateChanged) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onAudioPublishStateChanged = onAudioPublishStateChanged
	}
}

// WithOnAudioVolumeIndication sets the on audio volume indication callback.
func WithOnAudioVolumeIndication(onAudioVolumeIndication OnAudioVolumeIndication) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onAudioVolumeIndication = onAudioVolumeIndication
	}
}

// WithOnAudioMetaDataReceived sets the on audio meta data received callback.
func WithOnAudioMetaDataReceived(onAudioMetaDataReceived OnAudioMetaDataReceived) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onAudioMetaDataReceived = onAudioMetaDataReceived
	}
}

// WithOnLocalAudioTrackStatistics sets the on local audio track statistics callback.
func WithOnLocalAudioTrackStatistics(onLocalAudioTrackStatistics OnLocalAudioTrackStatistics) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onLocalAudioTrackStatistics = onLocalAudioTrackStatistics
	}
}

// WithOnRemoteAudioTrackStatistics sets the on remote audio track statistics callback.
func WithOnRemoteAudioTrackStatistics(onRemoteAudioTrackStatistics OnRemoteAudioTrackStatistics) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onRemoteAudioTrackStatistics = onRemoteAudioTrackStatistics
	}
}

// WithOnLocalVideoTrackStatistics sets the on local video track statistics callback.
func WithOnLocalVideoTrackStatistics(onLocalVideoTrackStatistics OnLocalVideoTrackStatistics) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onLocalVideoTrackStatistics = onLocalVideoTrackStatistics
	}
}

// WithOnRemoteVideoTrackStatistics sets the on remote video track statistics callback.
func WithOnRemoteVideoTrackStatistics(onRemoteVideoTrackStatistics OnRemoteVideoTrackStatistics) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onRemoteVideoTrackStatistics = onRemoteVideoTrackStatistics
	}
}

// WithOnAudioTrackPublishSuccess sets the on audio track publish success callback.
func WithOnAudioTrackPublishSuccess(onAudioTrackPublishSuccess OnAudioTrackPublishSuccess) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onAudioTrackPublishSuccess = onAudioTrackPublishSuccess
	}
}

// WithOnAudioTrackUnpublished sets the on audio track unpublished callback.
func WithOnAudioTrackUnpublished(onAudioTrackUnpublished OnAudioTrackUnpublished) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onAudioTrackUnpublished = onAudioTrackUnpublished
	}
}

// WithOnIntraRequestReceived sets the on intra request received callback.
func WithOnIntraRequestReceived(onIntraRequestReceived OnIntraRequestReceived) RTCConnectionOption {
	return func(cfg *RTCConnectionConfig) {
		cfg.onIntraRequestReceived = onIntraRequestReceived
	}
}

// NewRTCConnection creates a new Agora RTC connection instance.
func (s *Service) NewRTCConnection(opts ...RTCConnectionOption) (*RTCConnection, error) {
	cfg := &RTCConnectionConfig{
		connCfg: &agoraservice.RtcConnectionConfig{
			AutoSubscribeAudio: true,
			AutoSubscribeVideo: false,
			ClientRole:         agoraservice.ClientRoleBroadcaster,
			ChannelProfile:     agoraservice.ChannelProfileLiveBroadcasting,
		},
		publishCfg: s.buildPublishConfig(),
	}

	// Apply the options.
	for _, opt := range opts {
		opt(cfg)
	}

	rtcConn := agoraservice.NewRtcConnection(cfg.connCfg, cfg.publishCfg)
	conn := &RTCConnection{
		sampleRate:              cfg.sampleRate,
		channels:                cfg.audioChannelType,
		audioMode:               cfg.audioMode,
		enableReceiveAudioFrame: cfg.enableReceiveAudioFrame,
		rtcConn:                 rtcConn,
	}

	if cfg.pcmQueueSize > 0 {
		conn.pcmQueue = agoraservice.NewQueue(cfg.pcmQueueSize)
	}

	// Setup the channels and sample rate. You should setup it before registering the observers.
	if err := conn.setupChannelsAndSampleRate(cfg.sampleRate, cfg.audioChannelType); err != nil {
		return nil, fmt.Errorf("failed to setup channels and sample rate, %w", err)
	}

	// Create a channel to notify when the connection is established.
	// When the onConnectedCallback is called, the channel will be notified.
	connectedCh := make(chan struct{})

	// Register the connection observer.
	conn.registerConnectionObserver(cfg, connectedCh)

	// Register the local user observer.
	conn.registerLocalUserObserver(cfg)

	// Register the audio frame observer.
	conn.registerAudioFrameObserver(cfg)

	// Connect to the channel.
	if ret := conn.rtcConn.Connect(s.token, s.channelName, s.userID); ret != 0 {
		return nil, fmt.Errorf("failed to connect to the channel, return %d", ret)
	}

	// Wait for the connection to be established.
	<-connectedCh

	return conn, nil
}

func (s *Service) buildPublishConfig() *agoraservice.RtcConnectionPublishConfig {
	publishCfg := agoraservice.NewRtcConPublishConfig()
	publishCfg.AudioPublishType = agoraservice.AudioPublishTypePcm
	publishCfg.AudioScenario = agoraservice.AudioScenarioDefault
	publishCfg.IsPublishAudio = true
	publishCfg.IsPublishVideo = false
	publishCfg.AudioProfile = agoraservice.AudioProfileDefault
	return publishCfg
}

func (c *RTCConnection) PushAudioPCMData(data []byte, startPtsInMs int64) error {
	if ret := c.rtcConn.PushAudioPcmData(data, int(c.sampleRate), int(c.channels), startPtsInMs); ret != 0 {
		return fmt.Errorf("failed to push audio PCM data, return %d", ret)
	}
	return nil
}

func (c *RTCConnection) FetchAudioFrame() (*agoraservice.AudioFrame, error) {
	if c.pcmQueue == nil {
		return nil, fmt.Errorf("the pcm queue is not initialized")
	}

	if c.pcmQueue.IsEmpty() {
		return nil, ErrEmptyPCMQueue
	}

	if frame := c.pcmQueue.Dequeue(); frame != nil {
		if audioFrame, ok := frame.(*agoraservice.AudioFrame); ok {
			return audioFrame, nil
		}
		return nil, fmt.Errorf("unexpected type of the frame, %T", frame)
	}

	return nil, nil
}

func (c *RTCConnection) IsPushToRtcCompleted() bool {
	return c.rtcConn.IsPushToRtcCompleted()
}

func (c *RTCConnection) Disconnect() error {
	if ret := c.rtcConn.Disconnect(); ret != 0 {
		return fmt.Errorf("failed to disconnect, return %d", ret)
	}
	return nil
}

func (c *RTCConnection) PublishAudio() error {
	if ret := c.rtcConn.PublishAudio(); ret != 0 {
		return fmt.Errorf("failed to publish audio, return %d", ret)
	}
	return nil
}

func (c *RTCConnection) PublishVideo() error {
	if ret := c.rtcConn.PublishVideo(); ret != 0 {
		return fmt.Errorf("failed to publish video, return %d", ret)
	}
	return nil
}

// Release releases the RTC connection resources.
func (c *RTCConnection) Release() {
	c.rtcConn.Release()
}

func (c *RTCConnection) registerConnectionObserver(cfg *RTCConnectionConfig, connectedCh chan<- struct{}) {
	observer := &agoraservice.RtcConnectionObserver{
		OnConnected: func(rtcConn *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			if cfg.onConnected != nil {
				cfg.onConnected(rtcConn, info, reason)
			}
			// Notify the channel that the connection is established.
			connectedCh <- struct{}{}
		},
		OnDisconnected: func(rtcConn *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			if cfg.onDisconnected != nil {
				cfg.onDisconnected(rtcConn, info, reason)
			}
		},
		OnConnecting: func(rtcConn *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			if cfg.onConnecting != nil {
				cfg.onConnecting(rtcConn, info, reason)
			}
		},
		OnReconnecting: func(rtcConn *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			if cfg.onReconnecting != nil {
				cfg.onReconnecting(rtcConn, info, reason)
			}
		},
		OnReconnected: func(rtcConn *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			if cfg.onReconnected != nil {
				cfg.onReconnected(rtcConn, info, reason)
			}
		},
		OnConnectionLost: func(rtcConn *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo) {
			if cfg.onConnectionLost != nil {
				cfg.onConnectionLost(rtcConn, info)
			}
		},
		OnConnectionFailure: func(rtcConn *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, errCode int) {
			if cfg.onConnectionFailure != nil {
				cfg.onConnectionFailure(rtcConn, info, errCode)
			}
		},
		OnUserJoined: func(rtcConn *agoraservice.RtcConnection, uid string) {
			if cfg.onUserJoined != nil {
				cfg.onUserJoined(rtcConn, uid)
			}
		},
		OnUserLeft: func(rtcConn *agoraservice.RtcConnection, uid string, reason int) {
			if cfg.onUserLeft != nil {
				cfg.onUserLeft(rtcConn, uid, reason)
			}
		},
		OnAIQoSCapabilityMissing: func(rtcConn *agoraservice.RtcConnection, defaultFallbackScenario int) int {
			if cfg.onAIQoSCapabilityMissing != nil {
				return cfg.onAIQoSCapabilityMissing(rtcConn, defaultFallbackScenario)
			}
			return defaultFallbackScenario
		},
	}

	c.rtcConn.RegisterObserver(observer)
}

func (c *RTCConnection) registerLocalUserObserver(cfg *RTCConnectionConfig) {
	observer := &agoraservice.LocalUserObserver{
		OnStreamMessage: func(localUser *agoraservice.LocalUser, uid string, streamId int, data []byte) {
			if cfg.onStreamMessage != nil {
				cfg.onStreamMessage(localUser, uid, streamId, data)
			}
		},
		OnUserInfoUpdated: func(localUser *agoraservice.LocalUser, uid string, userMediaInfo int, val int) {
			if cfg.onUserInfoUpdated != nil {
				cfg.onUserInfoUpdated(localUser, uid, userMediaInfo, val)
			}
		},
		OnUserAudioTrackSubscribed: func(localUser *agoraservice.LocalUser, uid string, remoteAudioTrack *agoraservice.RemoteAudioTrack) {
			if cfg.onUserAudioTrackSubscribed != nil {
				cfg.onUserAudioTrackSubscribed(localUser, uid, remoteAudioTrack)
			}
		},
		OnUserVideoTrackSubscribed: func(localUser *agoraservice.LocalUser, uid string, info *agoraservice.VideoTrackInfo, remoteVideoTrack *agoraservice.RemoteVideoTrack) {
			if cfg.onUserVideoTrackSubscribed != nil {
				cfg.onUserVideoTrackSubscribed(localUser, uid, info, remoteVideoTrack)
			}
		},
		OnUserAudioTrackStateChanged: func(localUser *agoraservice.LocalUser, uid string, remoteAudioTrack *agoraservice.RemoteAudioTrack, state int, reason int, elapsed int) {
			if cfg.onUserAudioTrackStateChanged != nil {
				cfg.onUserAudioTrackStateChanged(localUser, uid, remoteAudioTrack, state, reason, elapsed)
			}
		},
		OnUserVideoTrackStateChanged: func(localUser *agoraservice.LocalUser, uid string, remoteVideoTrack *agoraservice.RemoteVideoTrack, state int, reason int, elapsed int) {
			if cfg.onUserVideoTrackStateChanged != nil {
				cfg.onUserVideoTrackStateChanged(localUser, uid, remoteVideoTrack, state, reason, elapsed)
			}
		},
		OnAudioPublishStateChanged: func(localUser *agoraservice.LocalUser, channelId string, oldState int, newState int, elapsed int) {
			if cfg.onAudioPublishStateChanged != nil {
				cfg.onAudioPublishStateChanged(localUser, channelId, oldState, newState, elapsed)
			}
		},
		OnAudioVolumeIndication: func(localUser *agoraservice.LocalUser, audioVolumeInfo []*agoraservice.AudioVolumeInfo, speakerNumber int, totalVolume int) {
			if cfg.onAudioVolumeIndication != nil {
				cfg.onAudioVolumeIndication(localUser, audioVolumeInfo, speakerNumber, totalVolume)
			}
		},
		OnAudioMetaDataReceived: func(localUser *agoraservice.LocalUser, uid string, metaData []byte) {
			if cfg.onAudioMetaDataReceived != nil {
				cfg.onAudioMetaDataReceived(localUser, uid, metaData)
			}
		},
		OnLocalAudioTrackStatistics: func(localUser *agoraservice.LocalUser, stats *agoraservice.LocalAudioTrackStats) {
			if cfg.onLocalAudioTrackStatistics != nil {
				cfg.onLocalAudioTrackStatistics(localUser, stats)
			}
		},
		OnRemoteAudioTrackStatistics: func(localUser *agoraservice.LocalUser, uid string, stats *agoraservice.RemoteAudioTrackStats) {
			if cfg.onRemoteAudioTrackStatistics != nil {
				cfg.onRemoteAudioTrackStatistics(localUser, uid, stats)
			}
		},
		OnLocalVideoTrackStatistics: func(localUser *agoraservice.LocalUser, stats *agoraservice.LocalVideoTrackStats) {
			if cfg.onLocalVideoTrackStatistics != nil {
				cfg.onLocalVideoTrackStatistics(localUser, stats)
			}
		},
		OnRemoteVideoTrackStatistics: func(localUser *agoraservice.LocalUser, uid string, stats *agoraservice.RemoteVideoTrackStats) {
			if cfg.onRemoteVideoTrackStatistics != nil {
				cfg.onRemoteVideoTrackStatistics(localUser, uid, stats)
			}
		},
		OnAudioTrackPublishSuccess: func(localUser *agoraservice.LocalUser, audioTrack *agoraservice.LocalAudioTrack) {
			if cfg.onAudioTrackPublishSuccess != nil {
				cfg.onAudioTrackPublishSuccess(localUser, audioTrack)
			}
		},
		OnAudioTrackUnpublished: func(localUser *agoraservice.LocalUser, audioTrack *agoraservice.LocalAudioTrack) {
			if cfg.onAudioTrackUnpublished != nil {
				cfg.onAudioTrackUnpublished(localUser, audioTrack)
			}
		},
		OnIntraRequestReceived: func(localUser *agoraservice.LocalUser) {
			if cfg.onIntraRequestReceived != nil {
				cfg.onIntraRequestReceived(localUser)
			}
		},
	}
	c.rtcConn.RegisterLocalUserObserver(observer)
}

func (c *RTCConnection) registerAudioFrameObserver(cfg *RTCConnectionConfig) {
	observer := &agoraservice.AudioFrameObserver{
		OnRecordAudioFrame: func(localUser *agoraservice.LocalUser, channelId string, frame *agoraservice.AudioFrame) bool {
			if cfg.onRecordAudioFrame != nil {
				return cfg.onRecordAudioFrame(localUser, channelId, frame)
			}
			return true
		},
		OnPlaybackAudioFrame: func(localUser *agoraservice.LocalUser, channelId string, frame *agoraservice.AudioFrame) bool {
			if cfg.onPlaybackAudioFrame != nil {
				return cfg.onPlaybackAudioFrame(localUser, channelId, frame)
			}
			return true
		},
		OnMixedAudioFrame: func(localUser *agoraservice.LocalUser, channelId string, frame *agoraservice.AudioFrame) bool {
			if cfg.onMixedAudioFrame != nil {
				return cfg.onMixedAudioFrame(localUser, channelId, frame)
			}
			return true
		},
		OnEarMonitoringAudioFrame: func(localUser *agoraservice.LocalUser, frame *agoraservice.AudioFrame) bool {
			if cfg.onEarMonitoringAudioFrame != nil {
				return cfg.onEarMonitoringAudioFrame(localUser, frame)
			}
			return true
		},
		OnPlaybackAudioFrameBeforeMixing: func(localUser *agoraservice.LocalUser, channelId string, uid string, frame *agoraservice.AudioFrame,
			vadResultStat agoraservice.VadState, vadResultFrame *agoraservice.AudioFrame) bool {
			if c.enableReceiveAudioFrame {
				if err := c.recvAudioFrame(frame); err != nil {
					return false
				}
			}
			if cfg.onPlaybackAudioFrameBeforeMixing != nil {
				return cfg.onPlaybackAudioFrameBeforeMixing(localUser, channelId, uid, frame, vadResultStat, vadResultFrame)
			}
			return true
		},
		OnGetAudioFramePosition: func(localUser *agoraservice.LocalUser) int {
			if cfg.onGetAudioFramePosition != nil {
				return cfg.onGetAudioFramePosition(localUser)
			}
			return 0
		},
		OnGetPlaybackAudioFrameParam: func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams {
			if cfg.onGetPlaybackAudioFrameParam != nil {
				return cfg.onGetPlaybackAudioFrameParam(localUser)
			}
			return agoraservice.AudioFrameObserverAudioParams{}
		},
		OnGetRecordAudioFrameParam: func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams {
			if cfg.onGetRecordAudioFrameParam != nil {
				return cfg.onGetRecordAudioFrameParam(localUser)
			}
			return agoraservice.AudioFrameObserverAudioParams{}
		},
		OnGetMixedAudioFrameParam: func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams {
			if cfg.onGetMixedAudioFrameParam != nil {
				return cfg.onGetMixedAudioFrameParam(localUser)
			}
			return agoraservice.AudioFrameObserverAudioParams{}
		},
		OnGetEarMonitoringAudioFrameParam: func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams {
			if cfg.onGetEarMonitoringAudioFrameParam != nil {
				return cfg.onGetEarMonitoringAudioFrameParam(localUser)
			}
			return agoraservice.AudioFrameObserverAudioParams{}
		},
	}

	c.rtcConn.RegisterAudioFrameObserver(observer, 0, nil)
}

func (c *RTCConnection) setupChannelsAndSampleRate(sampleRate int, channels AudioChannelType) error {
	if ret := c.rtcConn.GetLocalUser().SetPlaybackAudioFrameBeforeMixingParameters(int(channels), sampleRate); ret != 0 {
		return fmt.Errorf("failed to setup channels and sample rate, return %d", ret)
	}
	return nil
}

func (c *RTCConnection) recvAudioFrame(frame *agoraservice.AudioFrame) error {
	switch c.audioMode {
	case AudioModeDirect:
		// Push the audio frame directly to the RTC connection.
		if ret := c.rtcConn.PushAudioPcmData(frame.Buffer, frame.SamplesPerSec, frame.Channels, 0); ret != 0 {
			return fmt.Errorf("failed to push audio PCM data, return %d", ret)
		}
	case AudioModeChannel:
		// No need to enqueue the audio frame if the queue is not set.
		if c.pcmQueue == nil {
			return nil
		}

		// Enqueue the audio frame to the queue.
		c.pcmQueue.Enqueue(frame)
	default:
		return fmt.Errorf("invalid audio mode, %d", c.audioMode)
	}

	return nil
}

// OnConnected is the callback function for the connection established event.
type OnConnected func(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, reason int)

// OnDisconnected is the callback function for the connection disconnected event.
type OnDisconnected func(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, reason int)

// OnConnecting is the callback function for the connection connecting event.
type OnConnecting func(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, reason int)

// OnReconnecting is the callback function for the connection reconnecting event.
type OnReconnecting func(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, reason int)

// OnReconnected is the callback function for the connection reconnected event.
type OnReconnected func(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, reason int)

// OnConnectionLost is the callback function for the connection lost event.
type OnConnectionLost func(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo)

// OnConnectionFailure is the callback function for the connection failure event.
type OnConnectionFailure func(rtcConn *agoraservice.RtcConnection, rtcConnInfo *agoraservice.RtcConnectionInfo, errCode int)

// OnUserJoined is the callback function for the user joined event.
type OnUserJoined func(rtcConn *agoraservice.RtcConnection, uid string)

// OnUserLeft is the callback function for the user left event.
type OnUserLeft func(rtcConn *agoraservice.RtcConnection, uid string, reason int)

// OnAIQoSCapabilityMissing is the callback function for the AIQoSCapabilityMissing event.
type OnAIQoSCapabilityMissing func(rtcConn *agoraservice.RtcConnection, defaultFallbackScenario int) int

// OnStreamMessage is the callback function for the stream message event.
type OnStreamMessage func(localUser *agoraservice.LocalUser, uid string, streamId int, data []byte)

// OnUserInfoUpdated is the callback function for the user info updated event.
type OnUserInfoUpdated func(localUser *agoraservice.LocalUser, uid string, userMediaInfo int, val int)

// OnUserAudioTrackSubscribed is the callback function for the user audio track subscribed event.
type OnUserAudioTrackSubscribed func(localUser *agoraservice.LocalUser, uid string, remoteAudioTrack *agoraservice.RemoteAudioTrack)

// OnUserVideoTrackSubscribed is the callback function for the user video track subscribed event.
type OnUserVideoTrackSubscribed func(localUser *agoraservice.LocalUser, uid string, info *agoraservice.VideoTrackInfo, remoteVideoTrack *agoraservice.RemoteVideoTrack)

// OnUserAudioTrackStateChanged is the callback function for the user audio track state changed event.
type OnUserAudioTrackStateChanged func(localUser *agoraservice.LocalUser, uid string, remoteAudioTrack *agoraservice.RemoteAudioTrack, state int, reason int, elapsed int)

// OnUserVideoTrackStateChanged is the callback function for the user video track state changed event.
type OnUserVideoTrackStateChanged func(localUser *agoraservice.LocalUser, uid string, remoteVideoTrack *agoraservice.RemoteVideoTrack, state int, reason int, elapsed int)

// OnAudioPublishStateChanged is the callback function for the audio publish state changed event.
type OnAudioPublishStateChanged func(localUser *agoraservice.LocalUser, channelId string, oldState int, newState int, elapsed int)

// OnAudioVolumeIndication is the callback function for the audio volume indication event.
type OnAudioVolumeIndication func(localUser *agoraservice.LocalUser, audioVolumeInfo []*agoraservice.AudioVolumeInfo, speakerNumber int, totalVolume int)

// OnAudioMetaDataReceived is the callback function for the audio meta data received event.
type OnAudioMetaDataReceived func(localUser *agoraservice.LocalUser, uid string, metaData []byte)

// OnLocalAudioTrackStatistics is the callback function for the local audio track statistics event.
type OnLocalAudioTrackStatistics func(localUser *agoraservice.LocalUser, stats *agoraservice.LocalAudioTrackStats)

// OnRemoteAudioTrackStatistics is the callback function for the remote audio track statistics event.
type OnRemoteAudioTrackStatistics func(localUser *agoraservice.LocalUser, uid string, stats *agoraservice.RemoteAudioTrackStats)

// OnLocalVideoTrackStatistics is the callback function for the local video track statistics event.
type OnLocalVideoTrackStatistics func(localUser *agoraservice.LocalUser, stats *agoraservice.LocalVideoTrackStats)

// OnRemoteVideoTrackStatistics is the callback function for the remote video track statistics event.
type OnRemoteVideoTrackStatistics func(localUser *agoraservice.LocalUser, uid string, stats *agoraservice.RemoteVideoTrackStats)

// OnAudioTrackPublishSuccess is the callback function for the audio track publish success event.
type OnAudioTrackPublishSuccess func(localUser *agoraservice.LocalUser, audioTrack *agoraservice.LocalAudioTrack)

// OnAudioTrackUnpublished is the callback function for the audio track unpublished event.
type OnAudioTrackUnpublished func(localUser *agoraservice.LocalUser, audioTrack *agoraservice.LocalAudioTrack)

// OnIntraRequestReceived is the callback function for the intra request received event.
type OnIntraRequestReceived func(localUser *agoraservice.LocalUser)

// OnRecordAudioFrame is the callback function for the record audio frame event.
type OnRecordAudioFrame func(localUser *agoraservice.LocalUser, channelId string, frame *agoraservice.AudioFrame) bool

// OnPlaybackAudioFrame is the callback function for the playback audio frame event.
type OnPlaybackAudioFrame func(localUser *agoraservice.LocalUser, channelId string, frame *agoraservice.AudioFrame) bool

// OnMixedAudioFrame is the callback function for the mixed audio frame event.
type OnMixedAudioFrame func(localUser *agoraservice.LocalUser, channelId string, frame *agoraservice.AudioFrame) bool

// OnEarMonitoringAudioFrame is the callback function for the ear monitoring audio frame event.
type OnEarMonitoringAudioFrame func(localUser *agoraservice.LocalUser, frame *agoraservice.AudioFrame) bool

// OnPlaybackAudioFrameBeforeMixing is the callback function for the playback audio frame before mixing event.
type OnPlaybackAudioFrameBeforeMixing func(localUser *agoraservice.LocalUser, channelId string, uid string, frame *agoraservice.AudioFrame, vadResultStat agoraservice.VadState, vadResultFrame *agoraservice.AudioFrame) bool

// OnGetAudioFramePosition is the callback function for getting the audio frame position.
type OnGetAudioFramePosition func(localUser *agoraservice.LocalUser) int

// OnGetPlaybackAudioFrameParam is the callback function for getting the playback audio frame parameters.
type OnGetPlaybackAudioFrameParam func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams

// OnGetRecordAudioFrameParam is the callback function for getting the record audio frame parameters.
type OnGetRecordAudioFrameParam func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams

// OnGetMixedAudioFrameParam is the callback function for getting the mixed audio frame parameters.
type OnGetMixedAudioFrameParam func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams

// OnGetEarMonitoringAudioFrameParam is the callback function for getting the ear monitoring audio frame parameters.
type OnGetEarMonitoringAudioFrameParam func(localUser *agoraservice.LocalUser) agoraservice.AudioFrameObserverAudioParams
