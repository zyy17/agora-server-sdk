package agoraservice

/*
#cgo CFLAGS: -I${SRCDIR}/../headers/include/c/api2 -I${SRCDIR}/../headers/include/c/base
#include <stdlib.h>
#include <string.h>
#include "agora_local_user.h"
#include "agora_rtc_conn.h"
#include "agora_service.h"
#include "agora_media_base.h"
#include "agora_parameter.h"
#include "audio_observer_cgo.h"
*/
import "C"
import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type RtcConnectionInfo struct {
	ConnectionId uint
	/**
	 * ID of the target channel. NULL if you did not call the connect
	 * method.
	 */
	ChannelId string
	/**
	 * The state of the current connection: #CONNECTION_STATE_TYPE.
	 */
	State int
	/**
	 * ID of the local user.
	 */
	LocalUserId string
	/**
	 * Internal use only.
	 */
	InternalUid uint
}

type AudioFrameObserverAudioParams struct {
	SampleRate     int
	Channels       int
	Mode           RawAudioFrameOpModeType
	SamplesPerCall int
}

type RtcConnectionObserver struct {
	OnConnected                func(con *RtcConnection, conInfo *RtcConnectionInfo, reason int)
	OnDisconnected             func(con *RtcConnection, conInfo *RtcConnectionInfo, reason int)
	OnConnecting               func(con *RtcConnection, conInfo *RtcConnectionInfo, reason int)
	OnReconnecting             func(con *RtcConnection, conInfo *RtcConnectionInfo, reason int)
	OnReconnected              func(con *RtcConnection, conInfo *RtcConnectionInfo, reason int)
	OnConnectionLost           func(con *RtcConnection, conInfo *RtcConnectionInfo)
	OnConnectionFailure        func(con *RtcConnection, conInfo *RtcConnectionInfo, errCode int)
	OnTokenPrivilegeWillExpire func(con *RtcConnection, token string)
	OnTokenPrivilegeDidExpire  func(con *RtcConnection)
	OnUserJoined               func(con *RtcConnection, uid string)
	OnUserLeft                 func(con *RtcConnection, uid string, reason int)
	OnError                    func(con *RtcConnection, err int, msg string)
	OnStreamMessageError       func(con *RtcConnection, uid string, streamId int, errCode int, missed int, cached int)
	OnEncryptionError          func(con *RtcConnection, err int)
	//date: 2025-06-27
	// Triggered when the following two conditions are both met:
	// 1. The developer sets the connection's scenario to AudioScenarioAiServer.
	// 2. The version of the SDK on the client side does not support aiqos.
	// If triggered, it means the client-side version does not support aiqos, and the developer needs to decide to reset the server-side scenario.
	// How should the developer handle it when the callback is triggered?
	// 1. If set return value to -1, it means the SDK internally does not handle the scenario incompatibility.
	// 2. If set return value to a valid scenario, it means the SDK internally automatically falls back to the scenario returned, ensuring compatibility.
	// how to use it: can ref to examples/ai_send_recv_pcm/ai_send_recv_pcm.go
	OnAIQoSCapabilityMissing func(con *RtcConnection, defaultFallbackSenario int) int
}

// struct for local audio track statistics
type LocalAudioTrackStats struct {
	/**
	 * The number of channels.
	 */
	NumChannels int
	/**
	 * The sample rate (Hz).
	 */
	SentSampleRate int
	/**
	 * The average sending bitrate (Kbps).
	 */
	SentBitrate int
	/**
	 * The internal payload type
	 */
	InternalCodec int
	/**
	 * Voice pitch frequency in Hz
	 */
	VoicePitch float64
}
type RemoteAudioTrackStats struct {
	/**
	 * User ID of the remote user sending the audio streams.
	 */
	Uid uint
	/**
	 * Audio quality received by the user: #QUALITY_TYPE.
	 */
	Quality int
	/**
	 * @return Network delay (ms) from the sender to the receiver.
	 */
	NetworkTransportDelay int
	/**
	 * @return Delay (ms) from the receiver to the jitter buffer.
	 */
	JitterBufferDelay int
	/**
	 * The audio frame loss rate in the reported interval.
	 */
	AudioLossRate int
	/**
	 * The number of channels.
	 */
	NumChannels int
	/**
	 * The sample rate (Hz) of the received audio stream in the reported interval.
	 */
	ReceivedSampleRate int
	/**
	 * The average bitrate (Kbps) of the received audio stream in the reported interval.
	 * */
	ReceivedBitrate int
	/**
	 * The total freeze time (ms) of the remote audio stream after the remote user joins the channel.
	 * In a session, audio freeze occurs when the audio frame loss rate reaches 4%.
	 * Agora uses 2 seconds as an audio piece unit to calculate the audio freeze time.
	 * The total audio freeze time = The audio freeze number &times; 2 seconds
	 */
	TotalFrozenTime int
	/**
	 * The total audio freeze time as a percentage (%) of the total time when the audio is available.
	 * */
	FrozenRate int
	/**
	 * The number of audio bytes received.
	 */
	ReceivedBytes int64
}
type LocalVideoTrackStats struct {
	NumberOfStreams           uint64
	BytesMajorStream          uint64
	BytesMinorStream          uint64
	FramesEncoded             uint32
	SSRCMajorStream           uint32
	SSRCMinorStream           uint32
	CaptureFrameRate          int
	RegulatedCaptureFrameRate int
	InputFrameRate            int
	EncodeFrameRate           int
	RenderFrameRate           int
	TargetMediaBitrateBps     int
	MediaBitrateBps           int
	TotalBitrateBps           int
	CaptureWidth              int
	CaptureHeight             int
	RegulatedCaptureWidth     int
	RegulatedCaptureHeight    int
	Width                     int
	Height                    int
	EncoderType               uint32
	UplinkCostTimeMs          int
	QualityAdaptIndication    int
}

type RemoteVideoTrackStats struct {
	/**
	  User ID of the remote user sending the video streams.
	*/
	Uid uint
	/** **DEPRECATED** Time delay (ms).
	 */
	Delay int
	/**
	  Width (pixels) of the video stream.
	*/
	Width int
	/**
	Height (pixels) of the video stream.
	*/
	Height int
	/**
	Bitrate (Kbps) received since the last count.
	*/
	ReceivedBitrate int
	/** The decoder output frame rate (fps) of the remote video.
	 */
	DecoderOutputFrameRate int
	/** The render output frame rate (fps) of the remote video.
	 */
	RendererOutputFrameRate int
	/** The video frame loss rate (%) of the remote video stream in the reported interval.
	 */
	FrameLossRate int
	/** Packet loss rate (%) of the remote video stream after using the anti-packet-loss method.
	 */
	PacketLossRate int
	RxStreamType   int
	/**
	  The total freeze time (ms) of the remote video stream after the remote user joins the channel.
	  In a video session where the frame rate is set to no less than 5 fps, video freeze occurs when
	  the time interval between two adjacent renderable video frames is more than 500 ms.
	*/
	TotalFrozenTime int
	/**
	  The total video freeze time as a percentage (%) of the total time when the video is available.
	*/
	FrozenRate int
	/**
	  The total video decoded frames.
	*/
	TotalDecodedFrames uint32
	/**
	   The offset (ms) between audio and video stream. A positive value indicates the audio leads the
	  video, and a negative value indicates the audio lags the video.
	*/
	AvSyncTimeMs int
	/**
	   The average offset(ms) between receive first packet which composite the frame and  the frame
	  ready to render.
	*/
	DownlinkProcessTimeMs int
	/**
	  The average time cost in renderer.
	*/
	FrameRenderDelayMs int
	/**
	   The total time (ms) when the remote user neither stops sending the video
	  stream nor disables the video module after joining the channel.
	*/
	TotalActiveTime uint64
	/**
	  The total publish duration (ms) of the remote video stream.
	*/
	PublishDuration uint64
}

type LocalUserObserver struct {
	OnStreamMessage func(localUser *LocalUser, uid string, streamId int, data []byte, sendTs int64)
	// userMediaInfo: UserMediaInfoXxx
	// val: 0 for false, 1 for true
	OnUserInfoUpdated          func(localUser *LocalUser, uid string, userMediaInfo int, val int)
	OnUserAudioTrackSubscribed func(localUser *LocalUser, uid string, remoteAudioTrack *RemoteAudioTrack)
	OnUserVideoTrackSubscribed func(localUser *LocalUser, uid string, info *VideoTrackInfo, remoteVideoTrack *RemoteVideoTrack)
	/*
		for Mute/Unmute
		(state== 0 and reason == 5): mute
		(state== 2 and reason == 6): unmute
	*/
	OnUserAudioTrackStateChanged func(localUser *LocalUser, uid string, remoteAudioTrack *RemoteAudioTrack, state int, reason int, elapsed int)
	OnUserVideoTrackStateChanged func(localUser *LocalUser, uid string, remoteAudioTrack *RemoteVideoTrack, state int, reason int, elapsed int)
	//  void (*on_audio_publish_state_changed)(AGORA_HANDLE agora_local_user, const char* channel, int old_state, int new_state, int elapse_since_last_state);
	OnAudioPublishStateChanged func(localUser *LocalUser, channelId string, oldState int, newState int, elapsed int)
	OnAudioVolumeIndication    func(localUser *LocalUser, audioVolumeInfo []*AudioVolumeInfo, speakerNumber int, totalVolume int)
	OnAudioMetaDataReceived    func(localUser *LocalUser, uid string, metaData []byte)
	// for version 2.2.2
	OnLocalAudioTrackStatistics  func(localUser *LocalUser, stats *LocalAudioTrackStats)
	OnRemoteAudioTrackStatistics func(localUser *LocalUser, uid string, stats *RemoteAudioTrackStats)
	OnLocalVideoTrackStatistics  func(localUser *LocalUser, stats *LocalVideoTrackStats)
	OnRemoteVideoTrackStatistics func(localUser *LocalUser, uid string, stats *RemoteVideoTrackStats)
	// added on 2025-06-09
	OnAudioTrackPublishSuccess func(localUser *LocalUser, audioTrack *LocalAudioTrack)
	OnAudioTrackUnpublished    func(localUser *LocalUser, audioTrack *LocalAudioTrack)
	OnIntraRequestReceived     func(localUser *LocalUser)
	// added on 2026-03-05
	OnVideoTrackPublishSuccess func(localUser *LocalUser, localVideoTrack *LocalVideoTrack)
	OnVideoTrackUnpublished    func(localUser *LocalUser, localVideoTrack *LocalVideoTrack)
}

type AudioFrameObserver struct {
	OnRecordAudioFrame                func(localUser *LocalUser, channelId string, frame *AudioFrame) bool
	OnPlaybackAudioFrame              func(localUser *LocalUser, channelId string, frame *AudioFrame) bool
	OnMixedAudioFrame                 func(localUser *LocalUser, channelId string, frame *AudioFrame) bool
	OnEarMonitoringAudioFrame         func(localUser *LocalUser, frame *AudioFrame) bool
	OnPlaybackAudioFrameBeforeMixing  func(localUser *LocalUser, channelId string, uid string, frame *AudioFrame, vadResultStat VadState, vadResultFrame *AudioFrame) bool
	OnGetAudioFramePosition           func(localUser *LocalUser) int
	OnGetPlaybackAudioFrameParam      func(localUser *LocalUser) AudioFrameObserverAudioParams
	OnGetRecordAudioFrameParam        func(localUser *LocalUser) AudioFrameObserverAudioParams
	OnGetMixedAudioFrameParam         func(localUser *LocalUser) AudioFrameObserverAudioParams
	OnGetEarMonitoringAudioFrameParam func(localUser *LocalUser) AudioFrameObserverAudioParams
}

type VideoFrameObserver struct {
	OnFrame func(channelId string, userId string, frame *VideoFrame)
}

type VideoEncodedFrameObserver struct {
	OnEncodedVideoFrame func(channelId string, uid string, imageBuffer []byte,
		frameInfo *EncodedVideoFrameInfo) bool
}

type AudioEncoderConfiguration struct {
	AudioProfile int
}

// type SubscribeAudioConfig struct {
// 	SampleRate int
// 	Channels   int
// }

/**
 * Configurations for the RTC connection.
 */
type RtcConnectionConfig struct {
	/**
	 * Determines whether to subscribe to all audio streams automatically.
	 * - true: (Default) Subscribe to all audio streams automatically.
	 * - false: Do not subscribe to any audio stream automatically.
	 */
	AutoSubscribeAudio bool
	/**
	 * Determines whether to subscribe to all video streams automatically.
	 * - true: (Default) Subscribe to all video streams automatically.
	 * - false: Do not subscribe to any video stream automatically.
	 */
	AutoSubscribeVideo bool
	/**
	 * Determines whether to enable audio recording or playout.
	 * - true: It's used to publish audio and mix microphone, or subscribe audio and playout
	 * - false: It's used to publish extenal audio frame only without mixing microphone, or no need audio device to playout audio either
	 */
	EnableAudioRecordingOrPlayout bool
	/**
	 * The maximum sending bitrate.
	 */
	MaxSendBitrate int
	/**
	 * The minimum port.
	 */
	MinPort int
	/**
	 * The maximum port.
	 */
	MaxPort int
	/**
	 * The role of the user. The default user role is ClientRoleAudience.
	 */
	ClientRole     ClientRole
	ChannelProfile ChannelProfile

	/**
	 * The channel type. The default channel type is ChannelTypeStandard.
	 */
	ChannelType ChannelType
	/**
	 * Determines whether to receive audio media packet or not.
	 */
	AudioRecvMediaPacket bool
	/**
	 * Determines whether to receive audio encoded frame or not.
	 */
	AudioRecvEncodedFrame bool
	/**
	 * Determines whether to receive video media packet or not.
	 */
	VideoRecvMediaPacket bool
}

type RtcConnectionPublishConfig struct {
	AudioProfile                   AudioProfile
	AudioScenario                  AudioScenario
	IsPublishAudio                 bool             //default to true
	IsPublishVideo                 bool             // default to false
	AudioPublishType               AudioPublishType // 0: no publish, 1: pcm, 2: encoded pcm. default to 1
	VideoPublishType               VideoPublishType // 0: no publish, 1: yuv, 2: encoded image. default to 0
	VideoEncodedImageSenderOptions *VideoEncodedImageSenderOptions
	//only for send external audio and limited send speek for ai scenario . default to false
	// if want to use this feature, should set the IsSendExternalAudioForAI to true, and set the UnlimitedMs and Speed
	// and consult tech support for the details.
	// for this case, recommend to ExternalAudioSendMs to 500, and Speed to 2
	SendExternalAudioParameters *SendExternalAudioParameters
}

// note: DeliverMuteDataForFakeAdm can only set to rtc engine level, can not
// set to connection level
// so if once a connection has set to true, wihich will affect all the connections,
type SendExternalAudioParameters struct {
	Enabled                   bool
	SendMs                    int
	SendSpeed                 int
	DeliverMuteDataForFakeAdm bool
}

type RtcConnection struct {
	callbacks         callbackGate
	remoteTracks      remoteTrackRegistry
	audioMu           sync.Mutex
	infoMu            sync.RWMutex
	encodedObserverMu sync.Mutex
	retiredObservers  []func()
	cConnection       unsafe.Pointer
	connInfo          RtcConnectionInfo
	localUser         *LocalUser
	parameter         *AgoraParameter
	// cLocalUser  unsafe.Pointer
	// subAudioConfig     *SubscribeAudioConfig
	handler               *RtcConnectionObserver
	cHandler              *C.struct__rtc_conn_observer
	localUserObserver     *LocalUserObserver
	cLocalUserObserver    *C.struct__local_user_observer
	audioObserver         *AudioFrameObserver
	cAudioObserver        *C.struct__audio_frame_observer
	videoObserver         *VideoFrameObserver
	cVideoObserver        unsafe.Pointer
	encodedVideoObserver  *VideoEncodedFrameObserver
	cEncodedVideoObserver unsafe.Pointer

	// remoteVideoRWMutex          *sync.RWMutex
	// remoteEncodedVideoReceivers map[*VideoEncodedImageReceiver]*videoEncodedImageReceiverInner
	// vad related for the connection
	enableVad       int
	audioVadManager *AudioVadManager

	// capabilities observer
	cCapObserverHandle    unsafe.Pointer
	cCapabilitiesObserver *C.struct__capabilites_observer

	// audio scenario
	audioScenario      AudioScenario
	audioScenarioDirty bool
	audioProfile       AudioProfile

	// publish option
	publishConfig *RtcConnectionPublishConfig

	// track & sender
	audioSettings            audioTrackSettings
	audioPublicationRecovery bool
	audioTrack               *LocalAudioTrack
	videoTrack               *LocalVideoTrack
	audioSender              *AudioPcmDataSender
	videoSender              *VideoFrameSender
	encodedAudioSender       *AudioEncodedFrameSender
	encodedVideoSender       *VideoEncodedImageSender

	// pcm consumption stats for raw pcm data only
	pcmConsumeStats *PcmConsumeStats

	// stream id for data stream： no need to call createDataStream manually, it is created by the sdk automatically
	// and just use it for sendStreamMessage
	dataStreamId int

	// for ai scenario send external audio parameters,default to nil
	sendExternalAudioParameters *SendExternalAudioParameters

	// for transcoding
	transcodingWorker *TranscodingWorker
	// for video encoder configuration
	videoEncoderConfiguration *VideoEncoderConfiguration

	// for encoded audio frame observer
	encodedAudioFrameObserver *AudioEncodedFrameObserver

	// for audio transcoding
	audioTranscodingWorker *AudioTranscodingWorker
}

// for pcm consumption stats
type PcmConsumeStats struct {
	startTime   int64 // in ms
	totalLength int64 // in bytes
	duration    int   // in ms
}

func NewRtcConPublishConfig() *RtcConnectionPublishConfig {
	return &RtcConnectionPublishConfig{
		AudioProfile:     AudioProfileDefault,
		AudioScenario:    AudioScenarioAiServer,
		IsPublishAudio:   true,
		IsPublishVideo:   false,
		AudioPublishType: AudioPublishTypePcm,
		VideoPublishType: VideoPublishTypeNoPublish,
		VideoEncodedImageSenderOptions: &VideoEncodedImageSenderOptions{
			CcMode:        VideoSendCcEnabled, // should check todo???
			CodecType:     VideoCodecTypeH264,
			TargetBitrate: 5000,
		},
		SendExternalAudioParameters: &SendExternalAudioParameters{
			Enabled:                   false,
			SendMs:                    0,
			SendSpeed:                 0,
			DeliverMuteDataForFakeAdm: false,
		},
	}
}

// NOTE: date：2025-06-27
// add audioScenario param, to set the audio scenario for a connection
// and it can diff from the service config, and can diff from each other
func NewRtcConnection(cfg *RtcConnectionConfig, publishConfig *RtcConnectionPublishConfig) *RtcConnection {
	if cfg == nil || publishConfig == nil || !agoraService.inited || agoraService.service == nil || agoraService.mediaFactory == nil {
		return nil
	}
	cCfg := CRtcConnectionConfig(cfg)
	defer FreeCRtcConnectionConfig(cCfg)

	audioScenario := publishConfig.AudioScenario
	audioProfile := publishConfig.AudioProfile
	conn := &RtcConnection{
		cConnection:   C.agora_rtc_conn_create(agoraService.service, cCfg),
		audioScenario: audioScenario,
		audioProfile:  audioProfile,
		publishConfig: publishConfig,
		dataStreamId:  -1,
	}
	if conn.cConnection == nil {
		return nil
	}
	if isSupportExternalAudio(publishConfig) {
		parameters := *publishConfig.SendExternalAudioParameters
		conn.sendExternalAudioParameters = &parameters
	}
	if !conn.initializeLocalUser() {
		C.agora_rtc_conn_destroy(conn.cConnection)
		return nil
	}

	conn.localUser.role.Store(int32(cfg.ClientRole))

	// Make callback routing available before registering native observers.
	agoraService.setConFromHandle(conn.cConnection, conn, ConTypeCCon)
	agoraService.setConFromHandle(conn.localUser.cLocalUser, conn, ConTypeCLocalUser)
	conn.registerCapabilitiesObserver()

	conn.createAudioPublishTrack()
	conn.createVideoPublishTrack()
	if publishConfig.IsPublishAudio && publishConfig.AudioPublishType != AudioPublishTypeNoPublish && conn.audioTrack == nil {
		conn.Release()
		return nil
	}
	if publishConfig.IsPublishVideo && publishConfig.VideoPublishType != VideoPublishTypeNoPublish && conn.videoTrack == nil {
		conn.Release()
		return nil
	}

	if params := publishConfig.SendExternalAudioParameters; params != nil && params.Enabled {
		if code := conn.setExtraSendFrameSpeed(params); code != 0 {
			conn.Release()
			return nil
		}
	}
	fmt.Printf("______auto create data stream, id: %d\n", conn.dataStreamId)
	return conn
}

func (conn *RtcConnection) initializeLocalUser() bool {
	conn.localUser = &LocalUser{
		cLocalUser: C.agora_rtc_conn_get_local_user(conn.cConnection),
	}
	conn.parameter = &AgoraParameter{
		cParameter: C.agora_rtc_conn_get_agora_parameter(conn.cConnection),
	}
	if conn.localUser.cLocalUser == nil || conn.parameter.cParameter == nil {
		return false
	}

	conn.pcmConsumeStats = &PcmConsumeStats{}
	conn.localUser.SetAudioEncoderConfiguration(&AudioEncoderConfiguration{AudioProfile: int(conn.audioProfile)})
	conn.localUser.SetAudioScenario(conn.audioScenario)
	fmt.Printf("______set audio scenario to %d, audio profile to %d\n", conn.audioScenario, conn.audioProfile)
	return true
}

// The internal capabilities observer supports audio-scenario fallback for AI
// server connections and cannot be unregistered by the application.
func (conn *RtcConnection) registerCapabilitiesObserver() {
	if conn.audioScenario != AudioScenarioAiServer {
		return
	}
	conn.cCapabilitiesObserver = CCapatilitiesObserver()
	conn.cCapObserverHandle = C.agora_local_user_capabilities_observer_create(conn.cCapabilitiesObserver)
	agoraService.setConFromHandle(conn.cCapObserverHandle, conn, ConTypeCCapabilitiesObserver)
	C.agora_local_user_register_capabilities_observer(conn.localUser.cLocalUser, conn.cCapObserverHandle)
}

func (conn *RtcConnection) createAudioPublishTrack() {
	if !conn.publishConfig.IsPublishAudio {
		return
	}
	switch conn.publishConfig.AudioPublishType {
	case AudioPublishTypeNoPublish:
		fmt.Printf("WARN:publish audio is no publish, so no audio track created\n")
	case AudioPublishTypePcm:
		conn.audioSender = agoraService.mediaFactory.NewAudioPcmDataSender()
		externalAudio := conn.sendExternalAudioParameters != nil && conn.sendExternalAudioParameters.Enabled
		conn.audioTrack = NewCustomAudioTrackPcm(conn.audioSender, conn.audioScenario, externalAudio)
	case AudioPublishTypeEncodedPcm:
		conn.encodedAudioSender = agoraService.mediaFactory.NewAudioEncodedFrameSender()
		conn.audioTrack = NewCustomAudioTrackEncoded(conn.encodedAudioSender, AudioTrackMixDisabled)
	}
	if conn.audioTrack != nil {
		conn.audioTrack.SetEnabled(true)
		if code := conn.audioSettings.apply(conn.audioTrack); code != 0 {
			conn.audioTrack.Release()
			conn.audioTrack = nil
		}
	}
}

func (conn *RtcConnection) createVideoPublishTrack() {
	if !conn.publishConfig.IsPublishVideo {
		return
	}
	switch conn.publishConfig.VideoPublishType {
	case VideoPublishTypeNoPublish:
		fmt.Printf("WARN:publish video is no publish, so no video track created\n")
	case VideoPublishTypeYuv:
		conn.videoSender = agoraService.mediaFactory.NewVideoFrameSender()
		conn.videoTrack = NewCustomVideoTrackFrame(conn.videoSender)
	case VideoPublishTypeEncodedImage:
		conn.encodedVideoSender = agoraService.mediaFactory.NewVideoEncodedImageSender()
		conn.videoTrack = NewCustomVideoTrackEncoded(conn.encodedVideoSender, conn.publishConfig.VideoEncodedImageSenderOptions)
	}
	if conn.videoTrack != nil {
		conn.videoTrack.SetEnabled(true)
	}
}

func (conn *RtcConnection) Release() {
	if conn == nil || conn.cConnection == nil {
		return
	}
	conn.stopTranscodingWorker()
	conn.stopAudioTranscodingWorker()
	conn.callbacks.pause()
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	conn.remoteTracks.clear()
	conn.unregisterAudioEncodedFrameObserver()
	conn.unregisterAudioFrameObserver()
	conn.unregisterObserver()
	conn.removeCallbackMappings()
	conn.unregisterNativeObservers()
	conn.destroyNativeConnection()
	conn.releaseLocalState()
	conn.releasePublishTracks()
	conn.releasePublishSenders()
}

// Teardown helpers run with callbacks paused and audioMu held by Release.
func (conn *RtcConnection) removeCallbackMappings() {
	agoraService.deleteConFromHandle(conn.cConnection, ConTypeCCon)
	agoraService.deleteConFromHandle(conn.localUser.cLocalUser, ConTypeCLocalUser)
	if conn.cVideoObserver != nil {
		agoraService.deleteConFromHandle(conn.cVideoObserver, ConTypeCVideoObserver)
	}
	if conn.cEncodedVideoObserver != nil {
		agoraService.deleteConFromHandle(conn.cEncodedVideoObserver, ConTypeCEncodedVideoObserver)
	}
}

func (conn *RtcConnection) unregisterNativeObservers() {
	if conn.cCapObserverHandle != nil {
		C.agora_local_user_unregister_capabilities_observer(conn.localUser.cLocalUser, conn.cCapObserverHandle)
	}
	if conn.cAudioObserver != nil {
		C.agora_local_user_unregister_audio_frame_observer(conn.localUser.cLocalUser)
	}
	if conn.cVideoObserver != nil {
		C.agora_local_user_unregister_video_frame_observer(conn.localUser.cLocalUser, conn.cVideoObserver)
	}
	if conn.cEncodedVideoObserver != nil {
		C.agora_local_user_unregister_video_encoded_frame_observer(conn.localUser.cLocalUser, conn.cEncodedVideoObserver)
	}
	if conn.cLocalUserObserver != nil {
		C.agora_local_user_unregister_observer(conn.localUser.cLocalUser)
	}
	if conn.cHandler != nil {
		C.agora_rtc_conn_unregister_observer(conn.cConnection)
	}
}

// nativeObserverCleanup snapshots the native allocations before the connection's
// fields are cleared. Even failed unregistrations retain their observers until
// native connection destruction, including when idle mode delays that destruction.
func (conn *RtcConnection) nativeObserverCleanup() func() {
	audioObserver := conn.cAudioObserver
	videoObserver := conn.cVideoObserver
	encodedVideoObserver := conn.cEncodedVideoObserver
	localObserver := conn.cLocalUserObserver
	handler := conn.cHandler
	capabilities := conn.cCapabilitiesObserver
	capabilityHandle := conn.cCapObserverHandle
	retired := conn.retiredObservers
	agoraService.deleteConFromHandle(capabilityHandle, ConTypeCCapabilitiesObserver)
	return func() {
		if audioObserver != nil {
			FreeCAudioFrameObserver(audioObserver)
		}
		if videoObserver != nil {
			FreeCVideoFrameObserver(videoObserver)
		}
		if encodedVideoObserver != nil {
			FreeCEncodedVideoFrameObserver(encodedVideoObserver)
		}
		if localObserver != nil {
			FreeCLocalUserObserver(localObserver)
		}
		if handler != nil {
			FreeCRtcConnectionObserver(handler)
		}
		if capabilityHandle != nil {
			C.agora_local_user_capabilities_observer_destory(capabilityHandle)
		}
		if capabilities != nil {
			FreeCCapatilitiesObserver(capabilities)
		}
		for _, cleanup := range retired {
			cleanup()
		}
	}
}

func (conn *RtcConnection) destroyNativeConnection() {
	handle := conn.cConnection
	freeObservers := conn.nativeObserverCleanup()
	if agoraService.idleMode {
		addIdleItem(handle, 2000, freeObservers)
	} else {
		C.agora_rtc_conn_destroy(handle)
		freeObservers()
	}
	conn.cConnection = nil
	conn.cAudioObserver = nil
	conn.cVideoObserver = nil
	conn.cEncodedVideoObserver = nil
	conn.cLocalUserObserver = nil
	conn.cHandler = nil
	conn.cCapabilitiesObserver = nil
	conn.cCapObserverHandle = nil
	conn.retiredObservers = nil
}

func (conn *RtcConnection) releaseLocalState() {
	if conn.audioVadManager != nil {
		conn.audioVadManager.Release()
		conn.audioVadManager = nil
	}
	conn.parameter = nil
	localUser := conn.localUser
	conn.localUser = nil
	localUser.cLocalUser = nil
	conn.handler = nil
	conn.localUserObserver = nil
	conn.audioObserver = nil
	conn.videoObserver = nil
}

func (conn *RtcConnection) releasePublishTracks() {
	if conn.audioTrack != nil {
		conn.audioTrack.Release()
		conn.audioTrack = nil
	}
	if conn.videoTrack != nil {
		conn.videoTrack.Release()
		conn.videoTrack = nil
	}
}

func (conn *RtcConnection) releasePublishSenders() {
	if conn.audioSender != nil {
		conn.audioSender.Release()
		conn.audioSender = nil
	}
	if conn.videoSender != nil {
		conn.videoSender.Release()
		conn.videoSender = nil
	}
	if conn.encodedAudioSender != nil {
		conn.encodedAudioSender.Release()
		conn.encodedAudioSender = nil
	}
	if conn.encodedVideoSender != nil {
		conn.encodedVideoSender.Release()
		conn.encodedVideoSender = nil
	}
}

func (conn *RtcConnection) GetLocalUser() *LocalUser {
	return conn.localUser
}

func (conn *RtcConnection) GetAgoraParameter() *AgoraParameter {
	return conn.parameter
}

func (conn *RtcConnection) GetConnectionInfo() *RtcConnectionInfo {
	if conn == nil {
		return nil
	}
	conn.infoMu.RLock()
	defer conn.infoMu.RUnlock()
	info := conn.connInfo
	return &info
}

func (conn *RtcConnection) Connect(token string, channel string, uid string, info string) int {
	if conn.cConnection == nil {
		return -1
	}
	// auto create data stream: from 0331 for datastream encryption support
	conn.dataStreamId, _ = conn.createDataStream(false, false)

	conn.infoMu.Lock()
	conn.connInfo.ChannelId = channel
	conn.connInfo.LocalUserId = uid
	uidInt, _ := strconv.Atoi(uid)
	conn.connInfo.InternalUid = uint(uidInt)
	conn.infoMu.Unlock()
	cChannel := C.CString(channel)
	cToken := C.CString(token)
	cUid := C.CString(uid)
	cInfo := C.CString(info)
	defer C.free(unsafe.Pointer(cChannel))
	defer C.free(unsafe.Pointer(cToken))
	defer C.free(unsafe.Pointer(cUid))
	defer C.free(unsafe.Pointer(cInfo))
	return int(C.agora_rtc_conn_connect(conn.cConnection, cToken, cChannel, cInfo, cUid))
}

// date: 2025-07-04
// add a function to disconnect the connection
// and it will unpublish all tracks and unregister all observers
// and then do really disconnect, no need to call unregister observer manually
func (conn *RtcConnection) Disconnect() int {
	if conn == nil || conn.cConnection == nil {
		return -1
	}

	// for transcoding worker, stop it
	conn.stopTranscodingWorker()

	// for audio transcoding worker, stop it
	conn.stopAudioTranscodingWorker()
	// date: 2025-07-04
	//1. unpublish all tracks
	conn.UnpublishAudio()
	conn.UnpublishVideo()

	conn.callbacks.pause()
	//2. unregister all observers, but except rtc connection observer
	conn.unregisterAudioFrameObserver()
	conn.unregisterVideoFrameObserver()
	conn.unregisterVideoEncodedFrameObserver()
	conn.unregisterAudioEncodedFrameObserver()

	conn.unregisterLocalUserObserver()

	conn.remoteTracks.clear()
	conn.callbacks.resume()
	//3 and then do really disconnect
	ret := int(C.agora_rtc_conn_disconnect(conn.cConnection))

	//3. unregister rtc connection observer，yeah

	return ret
}

func (conn *RtcConnection) RenewToken(token string) int {
	if conn.cConnection == nil {
		return -1
	}
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))
	return int(C.agora_rtc_conn_renew_token(conn.cConnection, cToken))
}

func (conn *RtcConnection) createDataStream(reliable bool, ordered bool) (int, int) {
	if conn.cConnection == nil {
		return -1, -1
	}
	// int* stream_id, int reliable, int ordered
	cStreamId := C.int(-1)
	ret := int(C.agora_rtc_conn_create_data_stream(conn.cConnection, &cStreamId, CIntFromBool(reliable), CIntFromBool(ordered)))
	return int(cStreamId), ret
}

func (conn *RtcConnection) SendStreamMessage(msg []byte) int {
	if conn == nil || conn.cConnection == nil {
		return -1
	}
	cMsg := C.CBytes(msg)
	defer C.free(cMsg)
	streamId := conn.dataStreamId
	return int(C.agora_rtc_conn_send_stream_message(conn.cConnection, C.int(streamId), (*C.char)(cMsg), C.uint32_t(len(msg))))
}

func (conn *RtcConnection) RegisterObserver(handler *RtcConnectionObserver) int {
	if conn == nil || conn.cConnection == nil || handler == nil {
		return -1
	}
	if ret := conn.unregisterObserver(); ret != 0 {
		return ret
	}
	conn.callbacks.pause()
	observerCopy := *handler
	conn.handler = &observerCopy

	conn.cHandler = CRtcConnectionObserver()
	if conn.cHandler == nil {
		conn.handler = nil
		conn.callbacks.resume()
		return -1
	}

	conn.callbacks.resume()
	ret := int(C.agora_rtc_conn_register_observer(conn.cConnection, conn.cHandler))
	if ret != 0 {
		conn.unregisterObserver()
	}
	return ret
}

func (conn *RtcConnection) unregisterObserver() int {
	if conn == nil {
		return 0
	}
	conn.callbacks.pause()
	defer conn.callbacks.resume()
	// check if need to unregister
	if conn.cConnection == nil {
		return 0
	}
	if conn.cHandler != nil {
		if ret := int(C.agora_rtc_conn_unregister_observer(conn.cConnection)); ret != 0 {
			return ret
		}
		FreeCRtcConnectionObserver(conn.cHandler)
	}
	conn.cHandler = nil
	conn.handler = nil
	return 0
}

func (conn *RtcConnection) RegisterLocalUserObserver(handler *LocalUserObserver) int {
	if conn == nil || conn.cConnection == nil || handler == nil {
		return -1
	}
	if ret := conn.unregisterLocalUserObserver(); ret != 0 {
		return ret
	}
	conn.callbacks.pause()
	observerCopy := *handler
	conn.localUserObserver = &observerCopy

	conn.cLocalUserObserver = CLocalUserObserver()
	if conn.cLocalUserObserver == nil {
		conn.localUserObserver = nil
		conn.callbacks.resume()
		return -1
	}

	conn.callbacks.resume()
	ret := int(C.agora_local_user_register_observer(conn.localUser.cLocalUser, conn.cLocalUserObserver))
	if ret != 0 {
		conn.unregisterLocalUserObserver()
	}
	return ret
}

func (conn *RtcConnection) unregisterLocalUserObserver() int {
	if conn == nil {
		return 0
	}
	conn.callbacks.pause()
	defer conn.callbacks.resume()
	// check if need to unregister
	if conn.cConnection == nil {
		return 0
	}
	if conn.cLocalUserObserver != nil {
		if ret := int(C.agora_local_user_unregister_observer(conn.localUser.cLocalUser)); ret != 0 {
			return ret
		}
		FreeCLocalUserObserver(conn.cLocalUserObserver)
	}
	conn.cLocalUserObserver = nil
	conn.localUserObserver = nil
	return 0
}

func (conn *RtcConnection) RegisterAudioFrameObserver(observer *AudioFrameObserver, enableVad int, vadConfigure *AudioVadConfigV2) int {
	if conn == nil || conn.cConnection == nil || observer == nil {
		return -1
	}
	if ret := conn.unregisterAudioFrameObserver(); ret != 0 {
		return ret
	}
	conn.callbacks.pause()
	observerCopy := *observer
	conn.audioObserver = &observerCopy
	conn.enableVad = enableVad
	if enableVad > 0 {
		conn.audioVadManager = NewAudioVadManager(vadConfigure)
	}
	conn.cAudioObserver = CAudioFrameObserver()
	if conn.cAudioObserver == nil {
		conn.audioObserver = nil
		conn.callbacks.resume()
		return -1
	}

	conn.callbacks.resume()
	ret := int(C.agora_local_user_register_audio_frame_observer(conn.localUser.cLocalUser, conn.cAudioObserver))
	if ret != 0 {
		conn.unregisterAudioFrameObserver()
	}
	return ret
}

func (conn *RtcConnection) unregisterAudioFrameObserver() int {
	if conn == nil {
		return 0
	}
	conn.callbacks.pause()
	defer conn.callbacks.resume()
	// check if need to unregister
	if conn.cConnection == nil {
		return 0
	}
	conn.enableVad = 0
	if conn.cAudioObserver != nil {
		if ret := int(C.agora_local_user_unregister_audio_frame_observer(conn.localUser.cLocalUser)); ret != 0 {
			return ret
		}
		FreeCAudioFrameObserver(conn.cAudioObserver)
	}
	conn.cAudioObserver = nil
	conn.audioObserver = nil
	if conn.audioVadManager != nil {
		conn.audioVadManager.Release()
		conn.audioVadManager = nil
	}

	conn.enableVad = 0
	return 0
}

func (conn *RtcConnection) RegisterVideoFrameObserver(observer *VideoFrameObserver) int {
	if conn == nil || conn.cConnection == nil || observer == nil {
		return -1
	}
	if ret := conn.unregisterVideoFrameObserver(); ret != 0 {
		return ret
	}
	conn.callbacks.pause()
	observerCopy := *observer
	conn.videoObserver = &observerCopy

	conn.cVideoObserver = CVideoFrameObserver()
	if conn.cVideoObserver == nil {
		conn.videoObserver = nil
		conn.callbacks.resume()
		return -1
	}
	agoraService.setConFromHandle(conn.cVideoObserver, conn, ConTypeCVideoObserver)
	conn.callbacks.resume()
	ret := int(C.agora_local_user_register_video_frame_observer(conn.localUser.cLocalUser, conn.cVideoObserver))
	if ret != 0 {
		conn.unregisterVideoFrameObserver()
	}
	return ret
}

func (conn *RtcConnection) unregisterVideoFrameObserver() int {
	if conn == nil {
		return 0
	}
	conn.callbacks.pause()
	defer conn.callbacks.resume()
	// check if need to unregister
	if conn.cConnection == nil {
		return 0
	}
	if conn.cVideoObserver != nil {
		if ret := int(C.agora_local_user_unregister_video_frame_observer(conn.localUser.cLocalUser, conn.cVideoObserver)); ret != 0 {
			return ret
		}
		// delete from sync map
		agoraService.deleteConFromHandle(conn.cVideoObserver, ConTypeCVideoObserver)
		FreeCVideoFrameObserver(conn.cVideoObserver)
		conn.cVideoObserver = nil
	}
	conn.cVideoObserver = nil
	conn.videoObserver = nil
	return 0
}

func (conn *RtcConnection) RegisterVideoEncodedFrameObserver(observer *VideoEncodedFrameObserver) int {
	if conn == nil || conn.cConnection == nil || observer == nil {
		return -1
	}
	if ret := conn.unregisterVideoEncodedFrameObserver(); ret != 0 {
		return ret
	}
	conn.callbacks.pause()
	observerCopy := *observer
	conn.encodedVideoObserver = &observerCopy

	conn.cEncodedVideoObserver = CVideoEncodedFrameObserver()
	if conn.cEncodedVideoObserver == nil {
		conn.encodedVideoObserver = nil
		conn.callbacks.resume()
		return -1
	}
	agoraService.setConFromHandle(conn.cEncodedVideoObserver, conn, ConTypeCEncodedVideoObserver)
	conn.callbacks.resume()
	ret := int(C.agora_local_user_register_video_encoded_frame_observer(conn.localUser.cLocalUser, conn.cEncodedVideoObserver))
	if ret != 0 {
		conn.unregisterVideoEncodedFrameObserver()
	}
	return ret
}

func (conn *RtcConnection) unregisterVideoEncodedFrameObserver() int {
	if conn == nil {
		return 0
	}
	conn.callbacks.pause()
	defer conn.callbacks.resume()
	// check if need to unregister
	if conn.cConnection == nil {
		return 0
	}
	if conn.cEncodedVideoObserver != nil {
		if ret := int(C.agora_local_user_unregister_video_encoded_frame_observer(conn.localUser.cLocalUser, conn.cEncodedVideoObserver)); ret != 0 {
			return ret
		}
		// delete from sync map
		agoraService.deleteConFromHandle(conn.cEncodedVideoObserver, ConTypeCEncodedVideoObserver)
		FreeCEncodedVideoFrameObserver(conn.cEncodedVideoObserver)
		conn.cEncodedVideoObserver = nil
	}
	conn.cEncodedVideoObserver = nil
	conn.encodedVideoObserver = nil
	return 0
}

/*
* for stero encoded audio mode
* Must be called before con.connect
 */
func (conn *RtcConnection) enableSteroEncodeMode() int {
	if conn.cConnection == nil {
		return -1
	}
	//set private parameter
	localUser := conn.localUser

	// remove set senario to gs here, as it can pass senario as a parameter in NewRtcConnection
	// only force profile to stero encoding profile and force codec to opus here!
	// the default codec for musicstandstero is HAAC, no need for audio only senario
	// the HAAC is valid or better for muisic+audio combination senario
	// localUser.SetAudioScenario(AudioScenarioGameStreaming)
	localUser.SetAudioEncoderConfiguration(&AudioEncoderConfiguration{AudioProfile: int(AudioProfileMusicStandardStereo)})

	// fill pirvate parameter
	agoraParameterHandler := conn.parameter
	agoraParameterHandler.SetParameters("{\"che.audio.aec.enable\":false}")
	agoraParameterHandler.SetParameters("{\"che.audio.ans.enable\":false}")
	agoraParameterHandler.SetParameters("{\"che.audio.agc.enable\":false}")
	//  // "HEAAC_2ch" is case 78,but no need to set it for ai senario so disable it
	// in ai senario, we only want stero encoded audio, but really for speech so disable 78
	agoraParameterHandler.SetParameters("{\"che.audio.custom_payload_type\":122}")
	// and set bitrate to 32000,but not work in here! so move to other place
	//agoraParameterHandler.SetParameters("{\"che.audio.custom_bitrate\":32000}")
	return 0

}

type EncryptionConfig struct {
	EncryptionMode              int
	EncryptionKey               string
	EncryptionKdfSalt           []byte
	DatastreamEncryptionEnabled bool
}

// EnableEncryption enables or disables encryption for the RTC connection.
// It sets the encryption mode and configuration for the connection.
// Must be called before RtcConnection.Connect
//
// Parameters:
// - enable: An integer indicating whether to enable (1) or disable (0) encryption.
// - config: A pointer to an EncryptionConfig struct containing the encryption configuration.
//
// Returns:
// - An integer indicating the result of the operation. 0 indicates success, and negative value indicates failure.
func (conn *RtcConnection) EnableEncryption(enable int, config *EncryptionConfig) int {
	if conn.cConnection == nil || config == nil || enable == 0 {
		return -1
	}

	cConfig := C.struct__encryption_config{}
	C.memset(unsafe.Pointer(&cConfig), 0, C.sizeof_struct__encryption_config)
	cConfig.encryption_mode = C.int(config.EncryptionMode)
	if config.EncryptionKey != "" {
		ckey := C.CString(config.EncryptionKey)
		defer C.free(unsafe.Pointer(ckey))
		cConfig.encryption_key = ckey
	}
	saltlen := 0
	if config.EncryptionKdfSalt != nil {
		saltlen = len(config.EncryptionKdfSalt)
	}
	if saltlen > 0 {
		if saltlen > 32 {
			saltlen = 32
		}
		C.memcpy(unsafe.Pointer(&cConfig.encryption_kdf_salt[0]), unsafe.Pointer(&config.EncryptionKdfSalt[0]), C.size_t(saltlen))
	}

	// set datastream encryption enabled
	cConfig.datastream_encryption_enabled = C.bool(config.DatastreamEncryptionEnabled)

	ret := C.agora_rtc_conn_enable_encryption(conn.cConnection, C.int(enable), &cConfig)
	return int(ret)
}

func (conn *RtcConnection) handleCapabilitiesChanged(caps *C.struct__capabilities, size C.int) int {
	if conn == nil || conn.cConnection == nil || conn.cCapabilitiesObserver == nil {
		return -1
	}

	var fallback bool = true

	// 获取数组起始地址
	capsPtr := unsafe.Pointer(caps)
	// 计算每个元素的大小
	elemSize := unsafe.Sizeof(C.struct__capabilities{})

	// 遍历数组
	for i := 0; i < int(size); i++ {
		// 计算当前元素的地址
		curCapPtr := (*C.struct__capabilities)(unsafe.Pointer(uintptr(capsPtr) + uintptr(i)*elemSize))

		// 打印基本信息
		fmt.Printf("Capability[%d] - Type: %d\n", i, curCapPtr.capability_type)

		// 获取并打印 item_map 信息
		if curCapPtr.item_map != nil {
			itemMap := (*C.struct__capability_item_map)(unsafe.Pointer(curCapPtr.item_map))

			// 遍历 item_map 中的每个 item
			for j := 0; j < int(itemMap.size); j++ {
				item := (*C.struct__capability_item)(unsafe.Pointer(uintptr(unsafe.Pointer(itemMap.item)) + uintptr(j)*unsafe.Sizeof(C.struct__capability_item{})))

				// 打印 item 的详细信息
				itemName := ""
				if item.name != nil {
					itemName = C.GoString(item.name)
				}
				fmt.Printf("  Item[%d] - ID: %d, Name: %s\n", j, item.id, itemName)

			}
		}

		// check fabll back or not
		if curCapPtr.capability_type == 19 && curCapPtr.item_map != nil {
			itemMap := (*C.struct__capability_item_map)(unsafe.Pointer(curCapPtr.item_map))
			for j := 0; j < int(itemMap.size); j++ {
				item := (*C.struct__capability_item)(unsafe.Pointer(uintptr(unsafe.Pointer(itemMap.item)) + uintptr(j)*unsafe.Sizeof(C.struct__capability_item{})))

				itemName := ""
				if item.name != nil {
					itemName = C.GoString(item.name)
				}
				if itemName == "SUPPORT" && item.id == 0 {
					fallback = false
				}
			}
		}
	}
	fmt.Printf("*********fallback: %v\n", fallback)
	// call conectionobserver to notify
	if fallback && conn.cHandler != nil && conn.handler != nil && conn.handler.OnAIQoSCapabilityMissing != nil {
		userDefinedSenario := conn.handler.OnAIQoSCapabilityMissing(conn, int(AudioScenarioChorus))
		if userDefinedSenario >= 0 {
			fmt.Printf("onAIQoSCapabilityMissing, userDefinedSenario: %d\n", userDefinedSenario)
			if code := conn.UpdateAudioSenario(AudioScenario(userDefinedSenario)); code != 0 && conn.handler.OnError != nil {
				conn.handler.OnError(conn, code, "automatic audio scenario fallback failed")
			}
		}
	}
	return 0
}

// author: weihongqin
// description: push audio pcm data to agora sdk
// param: data: audio data
// param: sampleRate: sample rate
// param: channels: channels
// return: 0: success, -1: error, -2: invalid data
//
//date:2025-07-04 10:00:00
func (conn *RtcConnection) PublishAudio() int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn == nil || conn.cConnection == nil || conn.audioTrack == nil {
		return -2000
	}
	//conn.audioTrack.SetEnabled(true)
	ret := conn.localUser.publishAudio(conn.audioTrack)
	if ret == 0 {
		conn.audioPublicationRecovery = false
	}
	return int(ret)
}

func (conn *RtcConnection) UnpublishAudio() int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn == nil || conn.cConnection == nil || conn.audioTrack == nil {
		return -2000
	}

	//reset consumer stats
	if conn.pcmConsumeStats != nil {
		conn.pcmConsumeStats.reset()
	}
	//conn.audioTrack.SetEnabled(false)
	ret := conn.localUser.unpublishAudio(conn.audioTrack)
	if ret == 0 {
		conn.audioPublicationRecovery = false
	}
	return int(ret)
}

func (conn *RtcConnection) PublishVideo() int {
	if conn == nil || conn.cConnection == nil || conn.videoTrack == nil {
		return -2000
	}
	//conn.videoTrack.SetEnabled(true)
	ret := conn.localUser.publishVideo(conn.videoTrack)
	return int(ret)
}

func (conn *RtcConnection) UnpublishVideo() int {
	if conn == nil || conn.cConnection == nil || conn.videoTrack == nil {
		return -2000
	}
	//conn.videoTrack.SetEnabled(false)
	ret := conn.localUser.unpublishVideo(conn.videoTrack)
	return int(ret)
}

func (conn *RtcConnection) InterruptAudio() int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn == nil || conn.cConnection == nil || conn.audioTrack == nil {
		return -2000
	}

	if conn.audioScenario == AudioScenarioAiServer {
		// for aiServer, we need to unpublish the track
		conn.localUser.unpublishAudio(conn.audioTrack)
		// and publish the track again
		conn.localUser.publishAudio(conn.audioTrack)

	} else {
		// and other scenarios, we need to clear the buffer of the track
		conn.audioTrack.ClearSenderBuffer()
	}

	// if has audio consumption, we need to reset the stats
	if conn.pcmConsumeStats != nil {
		conn.pcmConsumeStats.reset()
	}
	return 0
}

/*
date:2025-08-14
author: weihongqin
description: push audio pcm data to agora sdk
param: data: audio data
param: sampleRate: sample rate
param: channels: channels
param: startPtsMs: start present timestamp in ms. and it can pass to onPlaybackBeforeMixing's audioframe.
default value is 0. Note: be carefully to set the pstvalue
for server end, the AudioFrame.RenderTimeMs is the capture timestamp
and the presentTimeMs is really the present timestamp.
*/
func (conn *RtcConnection) PushAudioPcmData(data []byte, sampleRate int, channels int, startPtsInMs int64) int {
	if conn == nil {
		return -2000
	}
	frame, valid := makePCMFrame(data, sampleRate, channels, startPtsInMs)
	if !valid {
		return -2
	}
	return conn.PushAudioFrame(frame)
}

func (conn *RtcConnection) PushAudioEncodedData(data []byte, frameInfo *EncodedAudioFrameInfo) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	// If captureTimeMs is 0, set it to -1 to avoid audio-video desynchronization.
	// Otherwise, audio and video may be out of sync.
	if conn == nil || conn.cConnection == nil || conn.encodedAudioSender == nil {
		return -2000
	}
	if frameInfo != nil && frameInfo.CaptureTimeMs == 0 {
		frameInfo.CaptureTimeMs = -1
	}
	return conn.encodedAudioSender.SendEncodedAudioFrame(data, frameInfo)
}

func (conn *RtcConnection) PushVideoFrame(frame *ExternalVideoFrame) int {
	if conn == nil || conn.cConnection == nil || conn.videoSender == nil {
		return -2000
	}
	return conn.videoSender.SendVideoFrame(frame)
}

func (conn *RtcConnection) PushVideoEncodedData(data []byte, frameInfo *EncodedVideoFrameInfo) int {
	if conn == nil || conn.cConnection == nil || conn.encodedVideoSender == nil || frameInfo == nil || len(data) == 0 {
		return -2000
	}

	// check and process SEI if needed
	if frameInfo.SeiData != nil && len(frameInfo.SeiData) > 0 {
		result := InsertSEIToEncodedData(data, frameInfo.SeiData, frameInfo.CodecType)
		if result != nil {
			data = result
		}
		//fmt.Printf("PushVideoEncodedData, sei data inserted, data size %d -> %d\n", len(data), len(result))
	}
	return conn.encodedVideoSender.SendEncodedVideoImage(data, frameInfo)
}

// to pudate connction's scenario
// UpdateAudioSenario preserves existing tracks if a scenario transition fails.
func (conn *RtcConnection) UpdateAudioSenario(scenario AudioScenario) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn.cConnection == nil || conn.localUser == nil {
		return -2000
	}
	if conn.audioScenario == scenario && !conn.audioScenarioDirty {
		return 0
	}
	oldScenario := conn.audioScenario
	// Encoded and receive-only connections do not own a PCM track to recreate.
	if conn.audioTrack == nil || conn.audioSender == nil {
		ret := conn.localUser.SetAudioScenario(scenario)
		if ret == 0 {
			conn.audioScenario = scenario
			conn.audioScenarioDirty = false
		}
		return ret
	}
	external := conn.sendExternalAudioParameters != nil && conn.sendExternalAudioParameters.Enabled
	replacement := NewCustomAudioTrackPcm(conn.audioSender, scenario, external)
	if replacement == nil {
		return -1
	}
	if code := conn.audioSettings.apply(replacement); code != 0 {
		conn.audioSender.audioScenario = oldScenario
		replacement.Release()
		return code
	}
	oldTrack := conn.audioTrack
	wasPublished := conn.localUser.publishFlag
	rollback := func() {
		conn.rollbackAudioScenario(oldScenario, wasPublished, conn.localUser.SetAudioScenario, func() int {
			return conn.localUser.publishAudio(oldTrack)
		})
		conn.audioSender.audioScenario = oldScenario
		replacement.Release()
	}
	if wasPublished {
		if ret := conn.localUser.unpublishAudio(oldTrack); ret != 0 {
			conn.audioSender.audioScenario = oldScenario
			replacement.Release()
			return ret
		}
	}
	if ret := conn.localUser.SetAudioScenario(scenario); ret != 0 {
		rollback()
		return ret
	}
	if wasPublished {
		if ret := conn.localUser.publishAudio(replacement); ret != 0 {
			rollback()
			return ret
		}
	}
	conn.audioTrack = replacement
	conn.audioScenario = scenario
	conn.audioScenarioDirty = false
	oldTrack.Release()
	if conn.pcmConsumeStats != nil {
		conn.pcmConsumeStats.reset()
	}
	return 0
}

func (conn *RtcConnection) IsPushToRtcCompleted() bool {
	if conn == nil {
		return false
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn == nil || conn.pcmConsumeStats == nil {
		return false
	}
	return conn.pcmConsumeStats.isPushCompleted(conn.audioScenario)
}

func (consumer *PcmConsumeStats) addPcmData(len int, samplerate int, channels int) {
	isNewRound := consumer.isNewRound(samplerate, channels)
	if isNewRound {
		consumer.startTime = time.Now().UnixMilli()
		consumer.totalLength = 0
	}

	consumer.totalLength += int64(len)

	// update duration
	consumer.duration = int(consumer.totalLength * 1000 / (int64(samplerate) * int64(channels) * 2))
	//fmt.Printf("addPcmData, duration: %d, totalLength: %d, startTime: %d\n", consumer.duration, consumer.totalLength, consumer.startTime)
}

func (consumer *PcmConsumeStats) isNewRound(samplerate int, channels int) bool {
	if consumer.startTime == 0 {
		return true
	}
	now := time.Now().UnixMilli()
	diff := now - consumer.startTime

	if diff > int64(consumer.duration) {
		return true
	}
	return false
}

func (consumer *PcmConsumeStats) getCurrentPosition() int {
	if consumer == nil || consumer.startTime == 0 {
		return 0
	}
	now := time.Now().UnixMilli()
	diff := now - consumer.startTime
	return int(diff)
}

const E2E_DELAY_MS = 180

func (consumer *PcmConsumeStats) isPushCompleted(scenario AudioScenario) bool {
	now := time.Now().UnixMilli()
	diff := now - consumer.startTime
	delay := E2E_DELAY_MS
	/* 	if scenario == AudioScenarioAiServer {
		delay = E2E_DELAY_MS
	} */
	if diff > int64(consumer.duration+delay) {
		return true
	}
	return false
}

func (consumer *PcmConsumeStats) reset() {
	consumer.startTime = 0
	consumer.totalLength = 0
	consumer.duration = 0
}

// SetVideoEncoderConfiguration applies an encoder configuration to the owned
// video track. An audience connection with configured video stores a copy until
// promotion creates the track; the caller may reuse cfg after this call.
func (conn *RtcConnection) SetVideoEncoderConfiguration(cfg *VideoEncoderConfiguration) int {
	// validate the connection
	if conn == nil || conn.cConnection == nil || cfg == nil {
		return -1000
	}
	// Audience sessions retain a copied encoder setting for lazy promotion.
	copy := *cfg
	conn.videoEncoderConfiguration = &copy
	if conn.videoTrack == nil || conn.videoTrack.cTrack == nil {
		if conn.publishConfig != nil && conn.publishConfig.VideoPublishType != VideoPublishTypeNoPublish {
			return 0
		}
		return -1000
	}

	// validate the cfg

	cCfg := C.struct__video_encoder_config{}
	C.memset(unsafe.Pointer(&cCfg), 0, C.sizeof_struct__video_encoder_config)
	cCfg.codec_type = C.int(cfg.CodecType)
	cCfg.dimensions.width = C.int(cfg.Width)
	cCfg.dimensions.height = C.int(cfg.Height)
	cCfg.frame_rate = C.int(cfg.Framerate)
	cCfg.bitrate = C.int(cfg.Bitrate * 1000)
	cCfg.min_bitrate = C.int(cfg.MinBitrate * 1000)
	cCfg.orientation_mode = C.int(cfg.OrientationMode)
	cCfg.degradation_preference = C.int(cfg.DegradePreference)
	cCfg.mirror_mode = C.int(cfg.MirrorMode)
	cCfg.encode_alpha = CIntFromBool(cfg.EncodeAlpha)
	return int(C.agora_local_video_track_set_video_encoder_config(conn.videoTrack.cTrack, &cCfg))
}

func (conn *RtcConnection) SendAudioMetaData(metaData []byte) int {
	if conn == nil {
		return -2000
	}
	conn.audioMu.Lock()
	defer conn.audioMu.Unlock()
	if conn == nil || conn.cConnection == nil || conn.localUser == nil || conn.localUser.cLocalUser == nil {
		return -2000
	}
	return conn.localUser.sendAudioMetaData(metaData)
}

// apm related: date 20251028
// at present, only support service-level apm filter related config, not support connection-level apm filter related config
// later, we will support connection-level apm filter related config,i.e, add a new connection-level api like:
// conneciton::SetApmConfigure(apmconfigure *ApmConf) which can set the apm filter related config for the connection
// but should call before connection.connect
func (conn *RtcConnection) isEnable3A() bool {
	if conn == nil || conn.cConnection == nil || conn.localUser == nil || conn.localUser.cLocalUser == nil {
		return false
	}
	if agoraService.apmConfig == nil {
		return false
	}
	return true
}

func (conn *RtcConnection) setApmFilterProperties(uid *C.char, cRemoteAudioTrack unsafe.Pointer) int {
	if conn == nil || conn.cConnection == nil || conn.localUser == nil || conn.localUser.cLocalUser == nil {
		return -2000
	}
	apmEnabled := conn.isEnable3A()
	fmt.Printf("------setApmFilterProperties, uid: %s, apmEnabled: %t\n", C.GoString(uid), apmEnabled)
	if !apmEnabled {
		return 0
	}

	// Load AINS resource
	ret := setFilterPropertyByTrack(cRemoteAudioTrack, "audio_processing_remote_playback", "apm_load_resource", "ains", false)
	if ret != 0 {
		fmt.Printf("Failed to set apm_load_resource, error: %d\n", ret)
	}

	// Set APM configuration
	configJSON := agoraService.apmConfig.toJson()
	ret = setFilterPropertyByTrack(cRemoteAudioTrack, "audio_processing_remote_playback", "apm_config", configJSON, false)
	if ret != 0 {
		fmt.Printf("Failed to set apm_config, error: %d\n", ret)
		return -1
	}
	fmt.Printf("*****APM config set:%s\n", configJSON)

	// Enable dump
	if agoraService.apmConfig.EnableDump {
		ret = setFilterPropertyByTrack(cRemoteAudioTrack, "audio_processing_remote_playback", "apm_dump", "true", false)
		if ret != 0 {
			fmt.Printf("Failed to enable apm_dump, error: %d\n", ret)
		}
	}

	return 0
}

// should call before connection is connected
func (conn *RtcConnection) setExtraSendFrameSpeed(sendExternalAudioParameters *SendExternalAudioParameters) int {
	if conn == nil || conn.cConnection == nil || conn.localUser == nil || conn.localUser.cLocalUser == nil {
		return -2000
	}
	if code := configureExtraSendFrameSpeed(sendExternalAudioParameters, conn.GetAgoraParameter().SetParameters); code != 0 {
		return code
	}

	// deliver mute data for fake adm: can only effect at service level, no effect for connection level
	setDeliverMuteData(sendExternalAudioParameters.DeliverMuteDataForFakeAdm)
	return 0
}

func (conn *RtcConnection) setTotalExtraSendMs() int {
	// note: for internal call only, so do not need to check the connection
	// only check the model from parameters
	if conn.sendExternalAudioParameters == nil || conn.sendExternalAudioParameters.Enabled == false {
		return -2001
	}

	ret := C.agora_local_audio_track_set_total_extra_send_ms(conn.audioTrack.cTrack, C.uint64_t(conn.sendExternalAudioParameters.SendMs))

	return int(ret)
}

func isSupportExternalAudio(publishConfig *RtcConnectionPublishConfig) bool {
	if publishConfig.SendExternalAudioParameters != nil && publishConfig.SendExternalAudioParameters.Enabled == true && publishConfig.SendExternalAudioParameters.SendMs > 0 && publishConfig.SendExternalAudioParameters.SendSpeed > 1 {
		return true
	}
	return false
}

// add a closer function to set only once the extra audio send ms the agora parameter
var deliverMuteDataHasSet bool = false

func setDeliverMuteData(deliverMuteData bool) bool {

	fmt.Printf("createDeliverMuteDataSetter, deliverMuteData: %t\n", deliverMuteDataHasSet)

	if !deliverMuteData && !deliverMuteDataHasSet {
		agoraParameterHandler := GetAgoraParameter()
		agoraParameterHandler.SetParameters("{\"che.audio.deliver_mute_data_for_fake_adm\":false}")
		deliverMuteDataHasSet = true // set the flag to ensure only once
		fmt.Printf("createDeliverMuteDataSetter, deliverMuteData: %t, deliverMuteDataHasSet: %t\n", deliverMuteData, deliverMuteDataHasSet)
	}

	return deliverMuteDataHasSet
}

// send intra request to the remote user to request a new key frame
// limitation:
// 1. if remtoe user is fixed-interval key frame encoding, the api call will be ignored
// 2. the min-frequency of the api call is 1s, otherwise, it will be ignored  or the vos will do converge
func (conn *RtcConnection) SendIntraRequest(remoteUid string) int {
	if conn == nil || conn.cConnection == nil || conn.localUser == nil || conn.localUser.cLocalUser == nil {
		return -2000
	}
	cUid := C.CString(remoteUid)
	defer C.free(unsafe.Pointer(cUid))
	ret := int(C.agora_local_user_send_intra_request(conn.localUser.cLocalUser, cUid))
	return ret
}

// SetSimulcastStream: to enable or disalbe dual video stream when publishing video through yuv
// and can be called anytime after connection is connected
// normally the call sequence should be:
// 1. set the video encoder configuration
// 2. set the simulcast stream config
// 3. publish video
// 4. during the publishing, can call the api to update the simulcast stream config without stopping the publishing

// if want disable the simulcast stream, just call the api with enable=false, and config=nil
func (conn *RtcConnection) SetSimulcastStream(enable bool, config *SimulcastStreamConfig) int {
	if conn == nil || conn.cConnection == nil || conn.videoTrack == nil || conn.videoTrack.cTrack == nil {
		return -1000
	}
	if conn.videoTrack == nil || conn.videoTrack.cTrack == nil {
		return -1001
	}

	if enable == false && config == nil {
		config = &SimulcastStreamConfig{
			Width:     0,
			Height:    0,
			Bitrate:   0,
			Framerate: 15,
		}
	}
	return conn.videoTrack.setSimulcastStream(enable, config)
}

func (conn *RtcConnection) PushVideoEncodedDataForTranscode(data []byte, frameInfo *EncodedVideoFrameInfo) int {
	if conn == nil || conn.cConnection == nil || conn.videoSender == nil {
		return -2000
	}
	if conn.transcodingWorker == nil {
		if ret := conn.startTranscodingWorker(); ret != 0 {
			return ret
		}
	}

	conn.transcodingWorker.PushEncodedData(data, frameInfo)

	return 0
}

func (conn *RtcConnection) handleDecodedVideoFrameForTranscode(frame *ExternalVideoFrame) int {
	if conn == nil {
		return -2000
	}
	ret := conn.PushVideoFrame(frame)

	return ret
}

func (conn *RtcConnection) startTranscodingWorker() int {
	if conn.transcodingWorker != nil {
		return -1000
	}
	fps := 15
	if conn.videoEncoderConfiguration != nil {
		fps = conn.videoEncoderConfiguration.Framerate
	}
	// range validation
	if fps < 1 {
		fps = 5
	}
	if fps > 30 {
		fps = 30
	}
	fmt.Printf("startTranscodingWorker, fps: %d\n", fps)
	conn.transcodingWorker = NewTranscodingWorker(conn, fps)
	if conn.transcodingWorker == nil {
		return -7
	}
	return conn.transcodingWorker.Start()
}

func (conn *RtcConnection) stopTranscodingWorker() int {
	if conn.transcodingWorker == nil {
		return -1000
	}
	conn.transcodingWorker.Stop()
	conn.transcodingWorker = nil
	return 0
}

func (conn *RtcConnection) RegisterAudioEncodedFrameObserver(observer *AudioEncodedFrameObserver) int {
	if conn == nil {
		return -2000
	}
	conn.callbacks.pause()
	defer conn.callbacks.resume()
	if conn == nil || conn.cConnection == nil {
		return -2000
	}
	if observer == nil {
		return -2001
	}
	// 'hold' the observer
	observerCopy := *observer
	conn.encodedAudioFrameObserver = &observerCopy

	return 0
}

func (conn *RtcConnection) unregAudioEncFrameObserverItem(item *EncAudioFrameObserverItem) int {
	if item == nil {
		return -2000
	}

	ret := C.agora_remote_audio_track_unregister_encoded_frame_rev_observer(item.CTrack, item.CObserverHandle)
	fmt.Printf("unregAudioEncFrameObserverItem, uid: %s, cTrack: %p, cObserverHandle: %p\n", item.Uid, item.CTrack, item.CObserverHandle)
	return int(ret)
}

func (conn *RtcConnection) unregisterAudioEncodedFrameObserver() int {
	conn.callbacks.pause()
	defer conn.callbacks.resume()
	if conn == nil || conn.cConnection == nil || conn.localUser == nil || conn.localUser.cLocalUser == nil {
		return -2000
	}
	if conn.encodedAudioFrameObserver == nil {
		return -2001
	}

	//iterate agoraService.encAudioFrameObserverItemsMap, when item.Con == conn, delete the item, and free the handle
	agoraService.encAudioFrameObserverItemsMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*EncAudioFrameObserverItem); ok {
			if item.Con == conn {
				if _, loaded := agoraService.encAudioFrameObserverItemsMap.LoadAndDelete(item.CObserverHandle); !loaded {
					return true
				}
				if ret := conn.unregAudioEncFrameObserverItem(item); ret == 0 {
					freeAudioEncodedFrameObserverItem(item)
				} else {
					retainedItem := item
					conn.retiredObservers = append(conn.retiredObservers, func() { freeAudioEncodedFrameObserverItem(retainedItem) })
				}
				item = nil
				// continue to iterate!
				return true
			}
		}
		return true
	})
	conn.encodedAudioFrameObserver = nil
	return 0
}

func (conn *RtcConnection) doUnregisterAudioEncodedFrameObserver(uid string) int {
	agoraService.encAudioFrameObserverItemsMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*EncAudioFrameObserverItem); ok {
			if item.Con == conn && item.Uid == uid {
				if _, loaded := agoraService.encAudioFrameObserverItemsMap.LoadAndDelete(item.CObserverHandle); !loaded {
					return true
				}
				// This path runs inside a subscription/state callback, while other
				// encoded callbacks may still be in flight. Keep native storage
				// alive until connection destruction even after unregistration.
				conn.unregAudioEncFrameObserverItem(item)
				retainedItem := item
				conn.retiredObservers = append(conn.retiredObservers, func() { freeAudioEncodedFrameObserverItem(retainedItem) })
				item = nil

				fmt.Printf("doUnregisterAudioEncodedFrameObserver, uid: %s", uid)

				// continue to iterate!
				return true
			}
		}
		return true
	})
	return 0
}

func (conn *RtcConnection) initEncodedAudioFrameReceived(uid *C.char, cRemoteAudioTrack unsafe.Pointer) int {
	conn.encodedObserverMu.Lock()
	defer conn.encodedObserverMu.Unlock()
	if conn == nil || conn.cConnection == nil || conn.localUser == nil || conn.localUser.cLocalUser == nil {
		return -2000
	}
	if conn.encodedAudioFrameObserver == nil {
		return -2001
	}
	// create a map , key is uid, value is cRemoteAudioTrack
	uidStr := C.GoString(uid)

	// iterate agoraService.encAudioFrameObserverItemsMap, find all items where item.Uid == uidStr, delete the item, and free the handle

	conn.doUnregisterAudioEncodedFrameObserver(uidStr)

	// create a new item
	item := createAudioEncodedFrameObserverItem(uidStr, cRemoteAudioTrack, conn)
	if item == nil {
		return -3001
	}

	agoraService.setAudioEncObserToMap(item)

	persistTrack := item.CTrack

	cObserverHandle := item.CObserverHandle
	ret := C.agora_remote_audio_track_register_encoded_frame_rev_observer(persistTrack, cObserverHandle)

	// store to sync map
	fmt.Printf("initEncodedAudioFrameReceived, uid: %s, ret: %d, persistTrack: %p, cObserverHandle: %p\n", uidStr, ret, persistTrack, cObserverHandle)

	if ret != 0 {
		agoraService.encAudioFrameObserverItemsMap.Delete(item.CObserverHandle)
		// Registration may have started delivering callbacks before failing.
		// As above, retain native storage until the connection is destroyed.
		conn.retiredObservers = append(conn.retiredObservers, func() { freeAudioEncodedFrameObserverItem(item) })
	}
	return int(ret)
}

func (conn *RtcConnection) doAudioTrackStateChanged(uid *C.char, cRemoteAudioTrack unsafe.Pointer, state int, reason int) int {
	conn.encodedObserverMu.Lock()
	defer conn.encodedObserverMu.Unlock()

	//1. if has audio encoded frame observer, then unregister the observer
	if conn.encodedAudioFrameObserver != nil {
		// if state is 0, and reason is 5 or 7,
		if state == 0 && (reason == 5 || reason == 7) {
			conn.doUnregisterAudioEncodedFrameObserver(C.GoString(uid))
		}
	}

	return 0
}

// date: 20260408 add get sid api
// note: only after connection is connected, the sid is valid(not empty string),otherwise return empty string
func (conn *RtcConnection) GetSid() string {
	if conn == nil || conn.cConnection == nil {
		return ""
	}
	c_sid := C.agora_rtc_conn_get_sid(conn.cConnection)
	if c_sid == nil {
		return ""
	}
	//note: do not free the c_sid, it is a const string
	return C.GoString(c_sid)
}

func (conn *RtcConnection) setBestPractice(channel_type ChannelType) int {
	if conn == nil || conn.cConnection == nil {
		return -2000
	}
	if channel_type == ChannelTypeLargeScale {
		return conn.setBestPracticeForLargeScale(channel_type)
	}
	if channel_type == ChannelTypeStandard {
		return conn.setBestPracticeForStandard(channel_type)
	}
	return -2002
}

func (conn *RtcConnection) setBestPracticeForLargeScale(channel_type ChannelType) int {
	if conn == nil || conn.cConnection == nil {
		return -2000
	}
	// only for large scale channel
	if channel_type != ChannelTypeLargeScale {
		return -2001
	}

	conn.GetAgoraParameter().SetParameters("{\"rtc.stream_sub_remote_stats\":false}")
	conn.GetAgoraParameter().SetParameters("{\"rtc.stream_pub_local_stats\":false}")
	// set {"rtc.max_sub_peers":100}
	conn.GetAgoraParameter().SetParameters("{\"rtc.max_sub_peers\":50}")
	// vos topN : "rtc.audio.filter_streams":4
	conn.GetAgoraParameter().SetParameters("{\"rtc.audio.filter_streams\":4}")

	// ack flow control:
	// rtc.use_const_ack_delay: true
	// rtc.const_ack_delay: 100
	conn.GetAgoraParameter().SetParameters("{\"rtc.use_const_ack_delay\":true}")
	conn.GetAgoraParameter().SetParameters("{\"rtc.ack_delay\":100}")

	// api tracer report: rtc.metadata.enable_report: false
	conn.GetAgoraParameter().SetParameters("{\"rtc.metadata.enable_report\":false}")

	// rtc report configure:"rtc.report_config": {
	reportConfig := `{
		"rtc.report_config": {
			"data.report.event.19": {
				"id": 19,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.20": {
				"id": 20,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.5": {
				"id": 5,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.6": {
				"id": 6,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.262": {
				"id": 262,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.263": {
				"id": 263,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.341": {
				"id": 341,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.342": {
				"id": 342,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.8": {
				"id": 8,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.13": {
				"id": 13,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			},
			"data.report.event.136": {
				"id": 136,
				"report_count": 0,
				"report_interval": 6,
				"type": "event"
			}
		}
	}`
	conn.GetAgoraParameter().SetParameters(reportConfig)

	return 0
}

func (conn *RtcConnection) setBestPracticeForStandard(channel_type ChannelType) int {
	if conn == nil || conn.cConnection == nil {
		return -2000
	}
	if channel_type != ChannelTypeStandard {
		return -2001
	}

	conn.GetAgoraParameter().SetParameters("{\"rtc.max_sub_peers\":1000}")
	// vos topN : "rtc.audio.filter_streams":4
	conn.GetAgoraParameter().SetParameters("{\"rtc.audio.filter_streams\":4}")

	// ack flow control:
	// rtc.use_const_ack_delay: true
	// rtc.const_ack_delay: 100
	conn.GetAgoraParameter().SetParameters("{\"rtc.use_const_ack_delay\":false}")
	conn.GetAgoraParameter().SetParameters("{\"rtc.ack_delay\":16}")

	// api tracer report: rtc.metadata.enable_report: false
	conn.GetAgoraParameter().SetParameters("{\"rtc.metadata.enable_report\":true}")

	// rtc report configure:"rtc.report_config": {
	reportConfig := `{
		"rtc.report_config": {
			"data.report.event.19": {
				"id": 19,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.20": {
				"id": 20,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.5": {
				"id": 5,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.6": {
				"id": 6,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.262": {
				"id": 262,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.263": {
				"id": 263,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.341": {
				"id": 341,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.342": {
				"id": 342,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.8": {
				"id": 8,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.13": {
				"id": 13,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			},
			"data.report.event.136": {
				"id": 136,
				"report_count": 1,
				"report_interval": 0,
				"type": "event"
			}
		}
	}`
	conn.GetAgoraParameter().SetParameters(reportConfig)

	return 0
}

func (conn *RtcConnection) startAudioTranscodingWorker(in_channels int) int {
	if conn.audioTranscodingWorker != nil {
		return -1000
	}
	conn.audioTranscodingWorker = NewAudioTranscodingWorker(conn, in_channels)
	if conn.audioTranscodingWorker == nil {
		return -7
	}
	ret := conn.audioTranscodingWorker.Start()
	if ret != 0 {
		conn.audioTranscodingWorker = nil
	}
	return ret
}

func (conn *RtcConnection) stopAudioTranscodingWorker() int {
	if conn.audioTranscodingWorker == nil {
		return -1000
	}
	conn.audioTranscodingWorker.Stop()
	conn.audioTranscodingWorker = nil
	return 0
}

// push encoded audio data for transcoding
func (conn *RtcConnection) PushAudioEncodedDataForTranscode(data []byte, channels int, codec int) int {

	if conn == nil || conn.cConnection == nil || conn.audioSender == nil {
		return -2000
	}

	// only support opus codec
	if codec != int(AudioCodecOpus) {
		return -2001
	}

	if conn.audioTranscodingWorker == nil {
		if ret := conn.startAudioTranscodingWorker(channels); ret != 0 {
			return ret
		}
	}

	conn.audioTranscodingWorker.PushEncodedData(data, channels, codec)

	return 0
}
