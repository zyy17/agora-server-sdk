package agorasdk

import (
	"errors"
	"math"
	"slices"
	"sync"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// VADConfig is the standalone native detector's complete configuration. Nil in
// NewAudioVAD or NewStereoVAD selects native-binding defaults; a non-nil value
// replaces every default, including zeros. Counts are 10 ms frames, at most 10000.
// See rtc.AudioVadConfig for threshold ranges and individual defaults.
type VADConfig = rtc.AudioVadConfig

// TelemetryVADConfig configures the SDK's Go-side telemetry segmenter. It does not
// classify PCM samples; input needs native voice/RMS telemetry. See
// rtc.AudioVadConfigV2 for defaults and fields retained only for compatibility.
type TelemetryVADConfig = rtc.AudioVadConfigV2

// VADState is the detector's speech state. State values match the raw SDK.
type VADState = rtc.VadState

const (
	// VADInvalid indicates an error or an unavailable detector state.
	VADInvalid VADState = rtc.VadStateInvalid
	// VADSilent indicates no selected speech.
	VADSilent VADState = rtc.VadStateNoSpeeking
	// VADSpeechStart indicates speech onset, possibly with pre-roll.
	VADSpeechStart VADState = rtc.VadStateStartSpeeking
	// VADSpeaking indicates ongoing selected speech.
	VADSpeaking VADState = rtc.VadStateSpeeking
	// VADSpeechEnd indicates the end of the current speech segment.
	VADSpeechEnd VADState = rtc.VadStateStopSpeeking
)

// VADResult contains a state and optional selected speech. Speech and its Data
// are independently Go-owned and survive future processing and Close. Speech
// may include pre-roll and may be nil or empty even without an error.
type VADResult struct {
	State  VADState
	Speech *AudioFrame
}

type nativeMonoVAD interface {
	ProcessPcmFrame(*rtc.AudioFrame) (*rtc.AudioFrame, int)
	Release()
}

// AudioVAD owns a standalone native mono detector independently of Service and
// channel connections. Use one detector per speaker, constructed by NewAudioVAD.
// Processing and Close serialize; do not copy a detector after first use.
type AudioVAD struct {
	mu     sync.Mutex
	native nativeMonoVAD
}

// NewAudioVAD creates a standalone detector using a copied configuration. The
// native VAD extension must be installed. Invalid settings wrap ErrInvalidConfig;
// allocation failure returns *Error. Close the detector when finished.
func NewAudioVAD(cfg *VADConfig) (*AudioVAD, error) {
	settings, err := copiedVADConfig(cfg)
	if err != nil {
		return nil, err
	}
	native := rtc.NewAudioVad(settings)
	if native == nil {
		return nil, &Error{Op: "create mono VAD", Code: -1}
	}
	return &AudioVAD{native: native}, nil
}

// ProcessPCM classifies exactly 320 bytes: 10 ms of 16 kHz mono PCM16 little-endian.
// Keep data unchanged until return. Speech timestamps are cumulative selected
// output duration, not wall time or the input's timeline. Native failures return
// *Error; processing after Close returns ErrClosed. A native exception (-2002)
// retires the detector: close it and create a new one rather than reusing it.
func (v *AudioVAD) ProcessPCM(data []byte) (VADResult, error) {
	if err := validateStandaloneVADPCM(data, 1); err != nil {
		return VADResult{State: VADInvalid}, err
	}
	if v == nil {
		return VADResult{State: VADInvalid}, ErrClosed
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.native == nil {
		return VADResult{State: VADInvalid}, ErrClosed
	}
	frame, state := v.native.ProcessPcmFrame(standaloneVADFrame(data, 1))
	return copiedVADResult(frame, state, "process mono VAD")
}

// Process validates and classifies one 10 ms, 16 kHz mono PCM16 AudioFrame.
// SamplesPerChannel must be 160 and BytesPerSample must be 2. Input timestamps
// and telemetry are not used by the native classifier. Results follow ProcessPCM.
func (v *AudioVAD) Process(frame AudioFrame) (VADResult, error) {
	if err := validateStandaloneVADFrame(frame, 1); err != nil {
		return VADResult{State: VADInvalid}, err
	}
	return v.ProcessPCM(frame.Data)
}

// Close releases the detector and waits for any in-flight processing call. It
// is idempotent, concurrency-safe, and accepts nil and zero-value receivers.
func (v *AudioVAD) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.native != nil {
		v.native.Release()
		v.native = nil
	}
	return nil
}

// StereoVADResult preserves independent left/right channel states and mono speech.
// On a native error, the other channel's successful result remains available.
type StereoVADResult struct {
	Left, Right VADResult
}

type nativeStereoVAD interface {
	ProcessAudioFrame(*rtc.AudioFrame) (*rtc.AudioFrame, int, *rtc.AudioFrame, int)
	Release()
}

// StereoVAD owns independent native left/right detectors. Construct it with
// NewStereoVAD and Close it when finished. Processing and Close serialize;
// do not copy it after first use. It requires no Service or channel connection.
type StereoVAD struct {
	mu     sync.Mutex
	native nativeStereoVAD
}

// NewStereoVAD creates left and right detectors with independently copied
// configurations. Either may be nil for defaults. Invalid settings wrap
// ErrInvalidConfig; allocation failure returns *Error and releases partial work.
func NewStereoVAD(left, right *VADConfig) (*StereoVAD, error) {
	leftCopy, err := copiedVADConfig(left)
	if err != nil {
		return nil, err
	}
	rightCopy, err := copiedVADConfig(right)
	if err != nil {
		return nil, err
	}
	native := rtc.NewSteroVad(leftCopy, rightCopy)
	if native == nil {
		return nil, &Error{Op: "create stereo VAD", Code: -1}
	}
	return &StereoVAD{native: native}, nil
}

// ProcessPCM classifies exactly 640 bytes: 10 ms of 16 kHz interleaved stereo
// PCM16 little-endian. Keep data unchanged until return. Each channel follows
// AudioVAD.ProcessPCM's state, speech-timestamp, and native-error contracts.
// If one channel fails, the error names it and the other result is preserved;
// errors.Join preserves both errors when both fail.
func (v *StereoVAD) ProcessPCM(data []byte) (StereoVADResult, error) {
	invalidResult := StereoVADResult{Left: VADResult{State: VADInvalid}, Right: VADResult{State: VADInvalid}}
	if err := validateStandaloneVADPCM(data, 2); err != nil {
		return invalidResult, err
	}
	if v == nil {
		return invalidResult, ErrClosed
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.native == nil {
		return invalidResult, ErrClosed
	}
	left, leftState, right, rightState := v.native.ProcessAudioFrame(standaloneVADFrame(data, 2))
	l, lerr := copiedVADResult(left, leftState, "process left VAD")
	r, rerr := copiedVADResult(right, rightState, "process right VAD")
	return StereoVADResult{Left: l, Right: r}, errors.Join(lerr, rerr)
}

// Process validates and classifies one 10 ms, 16 kHz stereo PCM16 AudioFrame.
// SamplesPerChannel must be 160 and BytesPerSample must be 2. Input timestamps
// and telemetry are not used by the native classifiers. Results follow ProcessPCM.
func (v *StereoVAD) Process(frame AudioFrame) (StereoVADResult, error) {
	if err := validateStandaloneVADFrame(frame, 2); err != nil {
		return StereoVADResult{Left: VADResult{State: VADInvalid}, Right: VADResult{State: VADInvalid}}, err
	}
	return v.ProcessPCM(frame.Data)
}

// Close releases both owned detectors and serializes with processing. It is
// idempotent, concurrency-safe, and accepts nil and zero-value receivers.
func (v *StereoVAD) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.native != nil {
		v.native.Release()
		v.native = nil
	}
	return nil
}

// TelemetryVAD wraps the Go-side VAD v2 segmenter with copied input/output and
// concurrency-safe cleanup. It consumes AudioFrame voice/RMS telemetry, rather
// than classifying PCM samples. Use one instance per stream; the first accepted
// frame fixes the PCM format. Construct it with NewTelemetryVAD, then Close it.
type TelemetryVAD struct {
	mu     sync.Mutex
	native *rtc.AudioVadV2
	format PCMFormat
}

// NewTelemetryVAD copies cfg and creates a telemetry segmenter without a Service
// or channel connection. Nil uses raw defaults; a non-nil zero configuration
// differs from nil. Invalid settings wrap ErrInvalidConfig. The raw segmenter
// may print diagnostic configuration and state lines to stdout.
func NewTelemetryVAD(cfg *TelemetryVADConfig) (*TelemetryVAD, error) {
	if cfg != nil {
		if err := validateTelemetryVADConfig(*cfg); err != nil {
			return nil, err
		}
		copy := *cfg
		cfg = &copy
	}
	return &TelemetryVAD{native: rtc.NewAudioVadV2(cfg)}, nil
}

// Process consumes a consistent, nonempty PCM16 AudioFrame with native telemetry.
// Input duration need not be 10 ms, but configuration counts count calls; use
// consistent frame durations. Keep input unchanged until return; internal copies
// permit immediate reuse afterward. Format changes and invalid shapes wrap
// ErrInvalidConfig; a closed or zero-value segmenter returns ErrClosed.
// State selection is exactly rtc.AudioVadV2's current algorithm, including its
// adaptive RMS behavior and compatibility-only threshold fields.
func (v *TelemetryVAD) Process(frame AudioFrame) (VADResult, error) {
	if err := validatePCM(frame.Data, frame.SampleRate, frame.Channels); err != nil {
		return VADResult{State: VADInvalid}, err
	}
	if frame.BytesPerSample != 2 || frame.SamplesPerChannel != len(frame.Data)/(2*frame.Channels) {
		return VADResult{State: VADInvalid}, invalid("telemetry VAD frame", "requires PCM16 with matching sample count")
	}
	if v == nil {
		return VADResult{State: VADInvalid}, ErrClosed
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.native == nil {
		return VADResult{State: VADInvalid}, ErrClosed
	}
	format := PCMFormat{SampleRate: frame.SampleRate, Channels: frame.Channels}
	if v.format != (PCMFormat{}) && v.format != format {
		return VADResult{State: VADInvalid}, invalid("telemetry VAD format", "must match the first accepted frame")
	}
	v.format = format
	raw := &rtc.AudioFrame{Type: rtc.AudioFrameTypePCM16, Buffer: slices.Clone(frame.Data),
		SamplesPerSec: frame.SampleRate, Channels: frame.Channels, SamplesPerChannel: frame.SamplesPerChannel,
		BytesPerSample: frame.BytesPerSample, RenderTimeMs: frame.TimestampMillis, PresentTimeMs: frame.PresentationMillis,
		AvsyncType: frame.AVSyncType, FarFieldFlag: frame.FarField, VoiceProb: frame.VoiceProbability,
		MusicProb: frame.MusicProbability, Rms: frame.RMS, Pitch: frame.Pitch}
	speech, state := v.native.Process(raw)
	result, err := copiedVADResult(speech, int(state), "process telemetry VAD")
	if result.Speech != nil {
		result.Speech.Position, result.Speech.Channel, result.Speech.UserID = frame.Position, frame.Channel, frame.UserID
	}
	return result, err
}

// Close drops retained frames and retires the segmenter. It is idempotent,
// concurrency-safe, and accepts nil and zero-value receivers.
func (v *TelemetryVAD) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.native != nil {
		v.native.Release()
		v.native = nil
	}
	return nil
}

func copiedVADConfig(cfg *VADConfig) (*VADConfig, error) {
	if cfg == nil {
		return nil, nil
	}
	copy := *cfg
	for _, count := range []int{copy.StartRecognizeCount, copy.StopRecognizeCount, copy.PreStartRecognizeCount} {
		if count < 0 || count > 10000 {
			return nil, invalid("standalone VAD counts", "must be between 0 and 10000")
		}
	}
	for _, value := range []float32{copy.ActivePercent, copy.InactivePercent, copy.VoiceProb, copy.JointThr} {
		if math.IsNaN(float64(value)) || value < 0 || value > 1 {
			return nil, invalid("standalone VAD probabilities", "must be finite and between 0 and 1")
		}
	}
	if math.IsNaN(float64(copy.RmsThr)) || copy.RmsThr < -100 || copy.RmsThr > 0 || math.IsNaN(float64(copy.Aggressive)) || copy.Aggressive < 0 || copy.Aggressive > 3 {
		return nil, invalid("standalone VAD thresholds", "require finite RMS -100–0 dB and aggressiveness 0–3")
	}
	return &copy, nil
}

func validateTelemetryVADConfig(cfg TelemetryVADConfig) error {
	if err := validateVAD(cfg); err != nil {
		return err
	}
	for _, value := range []float32{cfg.ActivePercent, cfg.InactivePercent, cfg.AdaptiveRmsThresholdFactor} {
		if math.IsNaN(float64(value)) {
			return invalid("VAD probabilities", "must be finite")
		}
	}
	return nil
}

func validateStandaloneVADPCM(data []byte, channels int) error {
	if len(data) != 320*channels {
		return invalid("standalone VAD PCM", "requires exactly one 10 ms, 16 kHz PCM16 frame")
	}
	return nil
}

func validateStandaloneVADFrame(frame AudioFrame, channels int) error {
	if err := validateStandaloneVADPCM(frame.Data, channels); err != nil {
		return err
	}
	if frame.SampleRate != 16000 || frame.Channels != channels || frame.SamplesPerChannel != 160 || frame.BytesPerSample != 2 {
		return invalid("standalone VAD frame", "requires 160 samples per channel at 16 kHz, PCM16, and the detector's channel count")
	}
	return nil
}

func standaloneVADFrame(data []byte, channels int) *rtc.AudioFrame {
	return &rtc.AudioFrame{Type: rtc.AudioFrameTypePCM16, Buffer: data, SamplesPerSec: 16000,
		Channels: channels, SamplesPerChannel: 160, BytesPerSample: 2}
}

func copiedVADResult(frame *rtc.AudioFrame, state int, op string) (VADResult, error) {
	if state < 0 {
		return VADResult{State: VADInvalid}, &Error{Op: op, Code: state}
	}
	result := VADResult{State: VADState(state)}
	if frame != nil {
		speech := copyAudioFrame(frame)
		speech.VAD = result.State
		result.Speech = &speech
	}
	return result, nil
}
