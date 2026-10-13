package agorasdk

import (
	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
	"strings"
	"time"
)

// Config owns process-wide settings. One Service may be open at a time.
// Authentication is per channel; an App Certificate is never needed to join.
type Config struct {
	AppID string
	Video bool
	// AudioProcessing enables native APM extensions for external processors.
	AudioProcessing             bool
	LogPath, ConfigDir, DataDir string
	// ConfigureNative runs before initialization. Do not retain the pointer.
	ConfigureNative func(*rtc.AgoraServiceConfig) error
}

type Role uint8

const (
	Broadcaster Role = iota
	Audience
)

type ChannelMode uint8

const (
	Standard ChannelMode = iota
	LargeScale
)

type AudioMode uint8

const (
	PCM AudioMode = iota
	EncodedAudio
	NoAudio
)

type VideoMode uint8

const (
	NoVideo VideoMode = iota
	RawVideo
	EncodedVideo
)

// AudioScenario selects native audio processing. Auto uses the upstream AI-server default.
type AudioScenario uint8

const (
	ScenarioAuto AudioScenario = iota
	ScenarioDefault
	ScenarioGameStreaming
	ScenarioChatRoom
	ScenarioChorus
	ScenarioMeeting
	ScenarioAIServer
	ScenarioAIClient
)

func (s AudioScenario) native() rtc.AudioScenario {
	return [...]rtc.AudioScenario{rtc.AudioScenarioAiServer, rtc.AudioScenarioDefault, rtc.AudioScenarioGameStreaming, rtc.AudioScenarioChatRoom, rtc.AudioScenarioChorus, rtc.AudioScenarioMeeting, rtc.AudioScenarioAiServer, rtc.AudioScenarioAiClient}[s]
}

// AudioConfig configures sending and receiving PCM16, or encoded audio.
// Zero values select 16 kHz mono PCM with the upstream AI-server scenario.
type AudioConfig struct {
	Mode                 AudioMode
	SampleRate, Channels int
	Scenario             AudioScenario
	Profile              rtc.AudioProfile
	// VAD enables upstream VAD v2 on 16 kHz mono received PCM.
	VAD       bool
	VADConfig *rtc.AudioVadConfigV2
}

// VideoConfig selects a publish format. Receiving is configured separately.
type VideoConfig struct {
	Mode          VideoMode
	Encoder       *rtc.VideoEncoderConfiguration
	EncodedSender *rtc.VideoEncodedImageSenderOptions
}

// ReceiveConfig selects copied media streams. Audio defaults to enabled unless
// Disabled is true. Video and encoded reception are opt-in.
type AudioPosition uint32

const (
	AudioPlayback      AudioPosition = 1
	AudioRecord        AudioPosition = 2
	AudioMixed         AudioPosition = 4
	AudioBeforeMixing  AudioPosition = 8
	AudioEarMonitoring AudioPosition = 16
)

type ReceiveConfig struct {
	// AudioPositions selects observation points; zero means per-user before mixing.
	AudioPositions                    AudioPosition
	Disabled                          bool
	ManualSubscriptions               bool
	Video, EncodedAudio, EncodedVideo bool
}

// ChannelConfig is independent for each join, including identity and token.
type ChannelConfig struct {
	Channel, UserID, Token string
	// Info is the native connection info payload, normally empty.
	Info    string
	Role    Role
	Mode    ChannelMode
	Audio   AudioConfig
	Video   VideoConfig
	Receive ReceiveConfig
	// QueueCapacity bounds each receive stream. Default: 64 items.
	QueueCapacity int
	// JoinTimeout applies in addition to the caller's context. Default: 30 s.
	JoinTimeout time.Duration
	Encryption  *rtc.EncryptionConfig
	// ConfigureNative is the advanced configuration escape hatch, before allocation.
	ConfigureNative func(*rtc.RtcConnectionConfig, *rtc.RtcConnectionPublishConfig) error
}

func normalized(c ChannelConfig) (ChannelConfig, error) {
	if strings.TrimSpace(c.Channel) == "" || strings.ContainsRune(c.Channel, 0) {
		return c, invalid("channel", "must be nonempty and contain no NUL")
	}
	if strings.TrimSpace(c.UserID) == "" || strings.ContainsRune(c.UserID, 0) {
		return c, invalid("user ID", "must be nonempty and contain no NUL")
	}
	if len(c.Channel) > 64 || len(c.UserID) > 255 {
		return c, invalid("identity", "exceeds channel (64) or user ID (255) byte limit")
	}
	if strings.ContainsRune(c.Token, 0) || strings.ContainsRune(c.Info, 0) {
		return c, invalid("token/info", "must contain no NUL")
	}
	if c.Role > Audience || c.Mode > LargeScale || c.Audio.Mode > NoAudio || c.Video.Mode > EncodedVideo {
		return c, invalid("mode", "is unknown")
	}
	if c.Audio.SampleRate == 0 {
		c.Audio.SampleRate = 16000
	}
	if c.Audio.Channels == 0 {
		c.Audio.Channels = 1
	}
	if err := validatePCMFormat(c.Audio.SampleRate, c.Audio.Channels); err != nil {
		return c, err
	}
	if c.Receive.AudioPositions == 0 {
		c.Receive.AudioPositions = AudioBeforeMixing
	}
	if c.Receive.AudioPositions & ^AudioPosition(31) != 0 {
		return c, invalid("audio positions", "contains unsupported bits")
	}
	if c.QueueCapacity == 0 {
		c.QueueCapacity = 64
	}
	if c.QueueCapacity < 1 || c.QueueCapacity > 65536 {
		return c, invalid("queue capacity", "must be between 1 and 65536")
	}
	if c.JoinTimeout == 0 {
		c.JoinTimeout = 30 * time.Second
	}
	if c.JoinTimeout < 0 {
		return c, invalid("join timeout", "must be positive")
	}
	if c.Mode == LargeScale && !c.Receive.ManualSubscriptions {
		return c, invalid("large-scale channel", "requires Receive.ManualSubscriptions=true")
	}
	if c.Audio.Scenario > ScenarioAIClient {
		return c, invalid("audio scenario", "is unknown")
	}
	if c.Audio.VAD && (c.Audio.SampleRate != 16000 || c.Audio.Channels != 1) {
		return c, invalid("VAD", "requires 16 kHz mono PCM")
	}
	if c.Audio.VADConfig != nil {
		if err := validateVAD(*c.Audio.VADConfig); err != nil {
			return c, err
		}
		v := *c.Audio.VADConfig
		c.Audio.VADConfig = &v
	}
	if c.Video.Encoder != nil {
		v := *c.Video.Encoder
		c.Video.Encoder = &v
	}
	if c.Video.EncodedSender != nil {
		v := *c.Video.EncodedSender
		c.Video.EncodedSender = &v
	}
	if c.Encryption != nil {
		v := *c.Encryption
		v.EncryptionKdfSalt = append([]byte(nil), v.EncryptionKdfSalt...)
		c.Encryption = &v
	}
	return c, nil
}
func validatePCMFormat(rate, channels int) error {
	switch rate {
	case 8000, 16000, 32000, 44100, 48000:
	default:
		return invalid("sample rate", "must be 8000, 16000, 32000, 44100, or 48000 Hz")
	}
	if channels != 1 && channels != 2 {
		return invalid("channels", "must be 1 or 2")
	}
	return nil
}

func validateVAD(v rtc.AudioVadConfigV2) error {
	if v.PreStartRecognizeCount < 0 || v.PreStartRecognizeCount > 10000 || v.StartRecognizeCount < 0 || v.StartRecognizeCount > 10000 || v.StopRecognizeCount < 0 || v.StopRecognizeCount > 10000 {
		return invalid("VAD counts", "must be between 0 and 10000 (zero uses defaults)")
	}
	if v.ActivePercent < 0 || v.ActivePercent > 1 || v.InactivePercent < 0 || v.InactivePercent > 1 || v.AdaptiveRmsThresholdFactor < 0 || v.AdaptiveRmsThresholdFactor > 1 {
		return invalid("VAD probabilities", "must be between 0 and 1")
	}
	if v.StartVoiceProb < 0 || v.StartVoiceProb > 100 || v.StopVoiceProb < 0 || v.StopVoiceProb > 100 || v.StartRms < -127 || v.StartRms > 0 || v.StopRms < -127 || v.StopRms > 0 {
		return invalid("VAD thresholds", "require voice probability 0–100 and RMS -127–0 dB")
	}
	return nil
}
