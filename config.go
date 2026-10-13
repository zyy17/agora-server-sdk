package agorasdk

import (
	"strconv"
	"strings"
	"time"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// Config owns process-wide settings. One Service may be open at a time.
// Authentication is per channel; an App Certificate is never needed to join.
type Config struct {
	AppID        string
	IdentityMode IdentityMode
	Video        bool
	// AudioProcessing enables native APM extensions for external processors.
	AudioProcessing             bool
	LogPath, ConfigDir, DataDir string
	// ConfigureNative runs before initialization. Do not retain the pointer.
	ConfigureNative func(*rtc.AgoraServiceConfig) error
}

// IdentityMode selects the native identity namespace for every session in a Service.
type IdentityMode uint8

const (
	// StringIdentity uses string user accounts, including all-digit accounts.
	StringIdentity IdentityMode = iota
	// NumericIdentity uses canonical decimal uint32 UIDs, never string accounts.
	NumericIdentity
)

func (m IdentityMode) validateUserID(uid string, allowAuto bool) error {
	if m > NumericIdentity {
		return invalid("identity mode", "is unknown")
	}
	if err := validateUserID(uid); err != nil {
		return err
	}
	if m == NumericIdentity {
		value, err := strconv.ParseUint(uid, 10, 32)
		if err != nil || strconv.FormatUint(value, 10) != uid || (!allowAuto && value == 0) {
			return invalid("numeric user ID", "must be a canonical decimal uint32; zero is allowed only when joining")
		}
	}
	return nil
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

// PCMFormat describes signed interleaved little-endian PCM16.
// Zero fields select 16 kHz mono independently for each direction.
type PCMFormat struct{ SampleRate, Channels int }

func (f PCMFormat) normalized() (PCMFormat, error) {
	if f.SampleRate == 0 {
		f.SampleRate = 16000
	}
	if f.Channels == 0 {
		f.Channels = 1
	}
	return f, validatePCMFormat(f.SampleRate, f.Channels)
}

// AudioConfig configures independent send/receive PCM16 formats, or encoded audio.
// Zero values select 16 kHz mono PCM with the upstream AI-server scenario.
type AudioConfig struct {
	Mode          AudioMode
	Send, Receive PCMFormat
	Scenario      AudioScenario
	Profile       rtc.AudioProfile
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
	Channel, Token string
	// UserID is a string account in StringIdentity mode or a canonical decimal
	// uint32 in NumericIdentity mode. Numeric "0" requests native UID allocation.
	UserID string
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
	if err := validateChannelIdentity(c); err != nil {
		return c, err
	}
	if c.Role > Audience || c.Mode > LargeScale || c.Audio.Mode > NoAudio || c.Video.Mode > EncodedVideo {
		return c, invalid("mode", "is unknown")
	}
	if err := c.normalizeDefaults(); err != nil {
		return c, err
	}
	if err := c.validateMediaOptions(); err != nil {
		return c, err
	}
	return c.copyNativeOptions(), nil
}

func validateChannelIdentity(c ChannelConfig) error {
	if strings.TrimSpace(c.Channel) == "" || strings.ContainsRune(c.Channel, 0) {
		return invalid("channel", "must be nonempty and contain no NUL")
	}
	if strings.TrimSpace(c.UserID) == "" || strings.ContainsRune(c.UserID, 0) {
		return invalid("user ID", "must be nonempty and contain no NUL")
	}
	if len(c.Channel) > 64 || len(c.UserID) > 255 {
		return invalid("identity", "exceeds channel (64) or user ID (255) byte limit")
	}
	if strings.ContainsRune(c.Token, 0) || strings.ContainsRune(c.Info, 0) {
		return invalid("token/info", "must contain no NUL")
	}
	return nil
}

func (c *ChannelConfig) normalizeDefaults() error {
	var err error
	if c.Audio.Send, err = c.Audio.Send.normalized(); err != nil {
		return err
	}
	if c.Audio.Receive, err = c.Audio.Receive.normalized(); err != nil {
		return err
	}
	if c.Receive.AudioPositions == 0 {
		c.Receive.AudioPositions = AudioBeforeMixing
	}
	if c.Receive.AudioPositions & ^AudioPosition(31) != 0 {
		return invalid("audio positions", "contains unsupported bits")
	}
	if c.QueueCapacity == 0 {
		c.QueueCapacity = 64
	}
	if c.QueueCapacity < 1 || c.QueueCapacity > 65536 {
		return invalid("queue capacity", "must be between 1 and 65536")
	}
	if c.JoinTimeout == 0 {
		c.JoinTimeout = 30 * time.Second
	}
	if c.JoinTimeout < 0 {
		return invalid("join timeout", "must be positive")
	}
	return nil
}

func (c ChannelConfig) validateMediaOptions() error {
	if c.Mode == LargeScale && !c.Receive.ManualSubscriptions {
		return invalid("large-scale channel", "requires Receive.ManualSubscriptions=true")
	}
	if c.Audio.Scenario > ScenarioAIClient {
		return invalid("audio scenario", "is unknown")
	}
	if c.Audio.VAD && (c.Audio.Receive.SampleRate != 16000 || c.Audio.Receive.Channels != 1) {
		return invalid("VAD", "requires 16 kHz mono received PCM")
	}
	if c.Audio.VAD && (c.Receive.Disabled || c.Receive.AudioPositions&AudioBeforeMixing == 0) {
		return invalid("VAD", "requires received PCM at AudioBeforeMixing")
	}
	if c.Audio.VADConfig != nil {
		return validateVAD(*c.Audio.VADConfig)
	}
	return nil
}

func (c ChannelConfig) copyNativeOptions() ChannelConfig {
	if c.Audio.VADConfig != nil {
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
	return c
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
