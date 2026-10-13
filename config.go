package agorasdk

import (
	"math"
	"strconv"
	"strings"
	"time"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// Config configures the process-wide RTC runtime. Only one Service may be open
// at a time. Authentication is configured per channel; joining does not require
// an App Certificate. The zero Config is invalid because AppID is required.
type Config struct {
	// AppID identifies the Agora project and must contain exactly 32 bytes without NUL.
	AppID string
	// IdentityMode selects the identity namespace for every session. Its zero value
	// uses string accounts; a numeric-looking string is still a string account.
	IdentityMode IdentityMode
	// Video enables the native video subsystem. Set it before creating sessions
	// that send or receive either raw or encoded video.
	Video bool
	// AudioProcessing enables native APM support for NewAudioProcessor.
	// The installed native package must include the requested extensions.
	AudioProcessing bool
	// LogPath, ConfigDir, and DataDir are native SDK filesystem paths.
	// Empty values leave path selection to the native SDK.
	LogPath, ConfigDir, DataDir string
	// IdleMode defers destruction of closed native connection handles briefly.
	// Session.Close still stops callbacks and releases Go resources immediately;
	// Service.Close drains all deferred handles before releasing the runtime.
	IdleMode bool
	// LocalAccessPoint optionally selects the native local proxy configuration.
	// The value and its lists are copied. It is applied once after the first connection
	// is created and before Connect, as required by the pinned SDK.
	LocalAccessPoint *rtc.LocalAccessPointConfiguration
	// ConfigureNative optionally adjusts native settings before initialization.
	// It runs synchronously; return an error to abort creation. Do not retain its
	// pointer. UseStringUid must agree with IdentityMode.
	ConfigureNative func(*rtc.AgoraServiceConfig) error
}

// IdentityMode selects the native identity namespace for all sessions of a Service.
// Tokens and remote user IDs must use the same namespace.
type IdentityMode uint8

const (
	// StringIdentity uses string user accounts, including all-digit accounts.
	// It is the default; it is not interchangeable with NumericIdentity.
	StringIdentity IdentityMode = iota
	// NumericIdentity uses canonical base-10 uint32 UIDs, without leading zeroes.
	// UserID "0" requests allocation when joining; remote user IDs must be nonzero.
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

// Role controls whether a session may publish media. The zero value is Broadcaster.
type Role uint8

const (
	// Broadcaster permits publication of the configured media types.
	Broadcaster Role = iota
	// Audience receives media but cannot publish audio or video.
	Audience
)

// ChannelMode selects a native RTC channel type. The zero value is Standard.
type ChannelMode uint8

const (
	// Standard supports automatic subscriptions and subscribe-all operations.
	Standard ChannelMode = iota
	// LargeScale requires ReceiveConfig.ManualSubscriptions and explicit per-user subscriptions.
	LargeScale
)

// AudioMode selects the audio publication path, independently of reception.
type AudioMode uint8

const (
	// PCM publishes signed interleaved PCM16 little-endian audio. It is the default.
	PCM AudioMode = iota
	// EncodedAudio publishes encoded packets through Session.SendEncodedAudio.
	EncodedAudio
	// NoAudio disables audio publication without disabling received audio.
	NoAudio
)

// VideoMode selects the video publication path, independently of reception.
type VideoMode uint8

const (
	// NoVideo disables video publication. It is the default.
	NoVideo VideoMode = iota
	// RawVideo publishes tightly packed CPU pixels through Session.SendVideo.
	RawVideo
	// EncodedVideo publishes H.264 or H.265 packets through Session.SendEncodedVideo.
	EncodedVideo
)

// AudioScenario selects native audio processing behavior. ScenarioAuto maps to
// the upstream AI-server scenario. Peer compatibility and processing support
// depend on the installed SDK; handle CapabilityFallback events explicitly.
type AudioScenario uint8

const (
	// ScenarioAuto selects ScenarioAIServer; it does not negotiate a fallback automatically.
	ScenarioAuto AudioScenario = iota
	// ScenarioDefault selects the native default audio scenario.
	ScenarioDefault
	// ScenarioGameStreaming selects native game-streaming audio processing.
	ScenarioGameStreaming
	// ScenarioChatRoom selects native chat-room audio processing.
	ScenarioChatRoom
	// ScenarioChorus selects native chorus audio processing.
	ScenarioChorus
	// ScenarioMeeting selects native meeting audio processing.
	ScenarioMeeting
	// ScenarioAIServer selects the AI-server path, which requires compatible peers.
	ScenarioAIServer
	// ScenarioAIClient selects the corresponding native AI-client path.
	ScenarioAIClient
)

func (s AudioScenario) native() rtc.AudioScenario {
	return [...]rtc.AudioScenario{rtc.AudioScenarioAiServer, rtc.AudioScenarioDefault, rtc.AudioScenarioGameStreaming, rtc.AudioScenarioChatRoom, rtc.AudioScenarioChorus, rtc.AudioScenarioMeeting, rtc.AudioScenarioAiServer, rtc.AudioScenarioAiClient}[s]
}

// PCMFormat describes signed interleaved little-endian PCM16.
// SampleRate is in Hz and must be 8000, 16000, 32000, 44100, or 48000;
// Channels must be 1 or 2. Zero fields select 16000 Hz and mono independently.
// Sending and receiving use separate formats; no implicit resampling is performed
// by the Go wrapper.
type PCMFormat struct {
	// SampleRate is the sampling frequency in Hz; zero selects 16000.
	SampleRate int
	// Channels is 1 for mono or 2 for interleaved stereo; zero selects mono.
	Channels int
}

func (f PCMFormat) normalized() (PCMFormat, error) {
	if f.SampleRate == 0 {
		f.SampleRate = 16000
	}
	if f.Channels == 0 {
		f.Channels = 1
	}
	return f, validatePCMFormat(f.SampleRate, f.Channels)
}

// AudioConfig configures publication and independent send/receive PCM formats.
// Its zero value selects PCM, 16 kHz mono in both directions, and ScenarioAuto.
type AudioConfig struct {
	// Mode selects the publication path; it does not disable receive observers.
	Mode AudioMode
	// Send and Receive describe outgoing and incoming PCM respectively.
	// Each zero field uses the defaults described by PCMFormat.
	Send, Receive PCMFormat
	// Scenario selects native processing and peer compatibility behavior.
	Scenario AudioScenario
	// Profile is the native audio encoding profile, passed through unchanged.
	Profile rtc.AudioProfile
	// VAD enables the SDK's Go VAD v2 segmenter for received 16 kHz mono PCM at
	// AudioBeforeMixing. It requires enabled PCM reception. This is distinct from
	// the standalone native rtc.AudioVad detector.
	VAD bool
	// VADConfig optionally overrides the SDK VAD v2 settings; nil uses its defaults.
	// Join copies the value. Frame counts are validated in 0–10000, probabilities
	// in 0–1, voice thresholds in 0–100, and RMS thresholds in -127–0 dB.
	// A non-nil zero configuration is not identical to nil; consult rtc.AudioVadConfigV2.
	VADConfig *rtc.AudioVadConfigV2
	// Fallback selects the native synchronous AI-QoS compatibility decision.
	// Nil rejects fallback. The policy is copied; no application function runs
	// on a native callback thread. SetAudioFallback can change it later.
	Fallback *AudioFallbackPolicy
	// ReceiveParameters optionally overrides each mixed/local PCM observation
	// point independently. Nil entries inherit Receive with 10 ms frames.
	ReceiveParameters *AudioReceiveParameters
	// Callbacks optionally handles synchronous native PCM accept/reject decisions.
	// Prefer Audio() for ordinary asynchronous processing.
	Callbacks *AudioCallbacks
}

// VideoConfig selects the video publication format. Receiving is configured
// separately by ReceiveConfig. All video paths require Config.Video.
type VideoConfig struct {
	// Mode selects no video, raw pixels, or encoded packets.
	Mode VideoMode
	// Encoder optionally configures the native video encoder. Join copies the value.
	Encoder *rtc.VideoEncoderConfiguration
	// EncodedSender optionally configures the encoded-video sender. Join copies the value.
	EncodedSender *rtc.VideoEncodedImageSenderOptions
}

// AudioPosition is a bitmask of native PCM observation points.
// Combine values with bitwise OR. ReceiveConfig defaults to AudioBeforeMixing.
type AudioPosition uint32

const (
	// AudioPlayback observes playback PCM after remote users have been mixed.
	AudioPlayback AudioPosition = 1
	// AudioRecord observes the native recording path, when available.
	AudioRecord AudioPosition = 2
	// AudioMixed observes the native mixed recording/playback path, when available.
	AudioMixed AudioPosition = 4
	// AudioBeforeMixing observes each remote user separately and supports SDK VAD.
	AudioBeforeMixing AudioPosition = 8
	// AudioEarMonitoring observes the native ear-monitoring path, when available.
	AudioEarMonitoring AudioPosition = 16
)

// ReceiveConfig selects receive observers and subscription behavior.
// The zero value enables PCM audio before mixing with automatic audio subscriptions.
// Video and encoded-audio observers are opt-in. Native feature availability still
// depends on the media path, platform, and peer SDK.
type ReceiveConfig struct {
	// AudioPositions selects PCM observation points; zero means AudioBeforeMixing.
	// Only the five declared bit flags are accepted.
	AudioPositions AudioPosition
	// Disabled disables PCM observation and automatic audio subscription.
	// An explicitly enabled encoded-audio observer still exists, but requires a
	// subscription established by the application when automatic subscription is off.
	Disabled bool
	// ManualSubscriptions disables automatic audio and video subscriptions.
	// Subscribe to each desired user explicitly. LargeScale requires this option.
	ManualSubscriptions bool
	// Video, EncodedAudio, and EncodedVideo enable the corresponding receive queues.
	// Video flags also enable automatic video subscription unless ManualSubscriptions is set.
	Video, EncodedAudio, EncodedVideo bool
}

// ChannelConfig describes one independently owned RTC session.
// Join validates and copies supported native option values. The caller must not
// mutate configuration data concurrently with Join.
type ChannelConfig struct {
	// Channel is a nonblank channel name of at most 64 bytes without NUL.
	// Token is the matching RTC credential; an empty token is allowed for unsecured
	// projects. The wrapper does not mint or automatically renew tokens.
	Channel, Token string
	// UserID is a nonblank string account of at most 255 bytes without NUL in
	// StringIdentity mode, or a canonical decimal uint32 in NumericIdentity mode.
	// Numeric "0" requests native allocation; Session.Info reports the assigned UID.
	UserID string
	// Info is the optional native connection-information payload. It must not contain NUL.
	Info string
	// Role selects initial publication permission. Zero is Broadcaster.
	// Session.SetRole can change it after joining.
	Role Role
	// Mode selects Standard or LargeScale. LargeScale requires manual subscriptions.
	Mode ChannelMode
	// Audio controls outgoing media and send/receive PCM formats.
	Audio AudioConfig
	// Video controls outgoing video and requires Config.Video when enabled.
	Video VideoConfig
	// Receive selects incoming observers and subscription policy.
	Receive ReceiveConfig
	// QueueCapacity bounds each independent receive queue in items, not bytes.
	// Zero uses 64; valid explicit values are 1–65536. Overflow drops the oldest item.
	QueueCapacity int
	// JoinTimeout bounds connection establishment in addition to the caller's context.
	// Zero uses 30 seconds; negative values are invalid. It does not limit session lifetime.
	JoinTimeout time.Duration
	// Encryption optionally enables native media encryption during setup.
	// Join copies the configuration and salt; project and peer settings must match.
	Encryption *rtc.EncryptionConfig
	// ConfigureNative optionally adjusts native connection and publication settings
	// before allocation. Return an error to abort Join. Do not retain the pointers
	// or change Role, publication modes, or publication enablement. LargeScale
	// cannot enable auto-subscription. Other native options are passed through.
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
	if err := c.validateAudioCallbacks(); err != nil {
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
	c.Audio = copyAudioOptions(c.Audio)
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
	if math.IsNaN(float64(v.ActivePercent)) || math.IsNaN(float64(v.InactivePercent)) || math.IsNaN(float64(v.AdaptiveRmsThresholdFactor)) {
		return invalid("VAD probabilities", "must be finite")
	}
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
