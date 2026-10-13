package agorasdk

import (
	"slices"
	"strings"

	rtc "github.com/zyy17/agora-server-sdk/agora/rtc"
)

// AudioFallbackPolicy selects the immediate native AI-QoS fallback decision.
// A nil policy rejects fallback. Native compatibility still depends on the peer.
type AudioFallbackPolicy struct {
	// UseRecommended selects the scenario supplied by the native callback.
	// If false, Scenario selects the fallback explicitly.
	UseRecommended bool
	// Scenario is used when UseRecommended is false. ScenarioAuto maps to AIServer
	// and is generally unsuitable when that capability is missing.
	Scenario AudioScenario
}

// AudioFallbackDecision is the copied policy decision accompanying CapabilityFallback.
// Selected is -1 when rejected. It reports the requested native scenario; it does
// not confirm that the SDK successfully changed its audio processing path.
type AudioFallbackDecision struct {
	// Recommended is the native callback's proposed audio scenario.
	Recommended rtc.AudioScenario
	// Selected is the requested native scenario, or -1 to reject fallback.
	Selected int
}

func (s *Session) storeAudioFallback(policy *AudioFallbackPolicy) {
	value := int32(-1)
	if policy != nil {
		value = int32(policy.Scenario.native())
		if policy.UseRecommended {
			value = -2
		}
	}
	s.fallback.Store(value)
}

// SetAudioFallback changes the immediate AI-QoS fallback decision without calling
// native code. Nil rejects fallback. It is safe to call from an audio callback;
// the decision itself never runs application code on the native callback thread.
func (s *Session) SetAudioFallback(policy *AudioFallbackPolicy) error {
	if policy != nil && policy.Scenario > ScenarioAIClient {
		return invalid("fallback scenario", "is unknown")
	}
	select {
	case <-s.done:
		return ErrClosed
	default:
	}
	s.storeAudioFallback(policy)
	return nil
}

// Role returns the most recently requested publication role, including changes
// made through WithNative. It is a local snapshot, not server acknowledgement,
// and remains readable after Close.
func (s *Session) Role() Role { return Role(s.role.Load()) }

// SetRole switches the native user role while serialized with sends and Close.
// Promotion lazily allocates the configured publication tracks; NoAudio and
// NoVideo remain disabled. Demotion first unpublishes active tracks. If an
// unpublish fails, the role is unchanged, but earlier unpublishes remain applied.
// A successful promotion does not publish media until Publish or Send is called.
// The native role setter has no server completion acknowledgement.
func (s *Session) SetRole(role Role) error {
	if role > Audience {
		return invalid("role", "is unknown")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.Role() == role {
		return nil
	}
	transport, ok := s.transport.(interface{ SetUserRole(int) int })
	if !ok {
		return ErrUnsupported
	}
	if role == Audience {
		if s.audioPublished {
			if err := check("unpublish audio", s.transport.UnpublishAudio()); err != nil {
				return err
			}
			s.audioPublished = false
		}
		if s.videoPublished {
			if err := check("unpublish video", s.native.UnpublishVideo()); err != nil {
				return err
			}
			s.videoPublished = false
		}
	}
	nativeRole := rtc.ClientRoleBroadcaster
	if role == Audience {
		nativeRole = rtc.ClientRoleAudience
	}
	if err := check("set role", transport.SetUserRole(int(nativeRole))); err != nil {
		return err
	}
	s.role.Store(uint32(role))
	return nil
}

func (s *Session) syncNativeState() {
	snapshot, ok := s.transport.(interface {
		GetUserRole() int
		GetPublicationState() (bool, bool)
	})
	if !ok {
		return
	}
	role := Broadcaster
	if snapshot.GetUserRole() == int(rtc.ClientRoleAudience) {
		role = Audience
	}
	s.role.Store(uint32(role))
	s.audioPublished, s.videoPublished = snapshot.GetPublicationState()
}

func validateNativeOwnership(cfg ChannelConfig, conn *rtc.RtcConnectionConfig, pub *rtc.RtcConnectionPublishConfig) error {
	role := rtc.ClientRoleBroadcaster
	if cfg.Role == Audience {
		role = rtc.ClientRoleAudience
	}
	audioType := rtc.AudioPublishTypePcm
	if cfg.Audio.Mode == EncodedAudio {
		audioType = rtc.AudioPublishTypeEncodedPcm
	} else if cfg.Audio.Mode == NoAudio {
		audioType = rtc.AudioPublishTypeNoPublish
	}
	if conn.ClientRole != role || pub.IsPublishAudio != (cfg.Audio.Mode != NoAudio && cfg.Role == Broadcaster) || pub.IsPublishVideo != (cfg.Video.Mode != NoVideo && cfg.Role == Broadcaster) || pub.AudioPublishType != audioType || pub.VideoPublishType != rtc.VideoPublishType(cfg.Video.Mode) {
		return invalid("native configuration", "role and publication modes must match ChannelConfig")
	}
	if conn.ChannelType != rtc.ChannelTypeStandard && conn.ChannelType != rtc.ChannelTypeLargeScale {
		return invalid("native channel type", "is unknown")
	}
	return nil
}

func copyLocalAccessPoint(config *rtc.LocalAccessPointConfiguration) *rtc.LocalAccessPointConfiguration {
	if config == nil {
		return nil
	}
	copy := *config
	copy.IPList = slices.Clone(config.IPList)
	copy.DomainList = slices.Clone(config.DomainList)
	return &copy
}

func validateLocalAccessPoint(config rtc.LocalAccessPointConfiguration) error {
	if len(config.IPList)+len(config.DomainList) == 0 {
		return invalid("local access point", "requires at least one IP address or domain")
	}
	if config.Mode != rtc.LocalProxyModeConnectivityFirst && config.Mode != rtc.LocalProxyModeLocalOnly {
		return invalid("local proxy mode", "is unknown")
	}
	for _, list := range [][]string{config.IPList, config.DomainList} {
		for _, value := range list {
			if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
				return invalid("local access point", "addresses must be nonempty without NUL")
			}
		}
	}
	log := config.AdvancedConfig.LogUploadServerInfo
	for _, value := range []string{config.VerifyDomainName, log.ServerDomain, log.ServerPath} {
		if strings.ContainsRune(value, 0) {
			return invalid("local access point", "strings must not contain NUL")
		}
	}
	if log.ServerPort < 0 || log.ServerPort > 65535 {
		return invalid("log upload port", "must be between 0 and 65535")
	}
	return nil
}

// LoadExtensionProvider loads a native extension from path under the Service
// lifecycle lock. Use only a trusted library matching the SDK/platform ABI.
// unloadAfterUse requests the native provider's unloading behavior.
func (s *Service) LoadExtensionProvider(path string, unloadAfterUse bool) error {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) {
		return invalid("extension path", "must be nonempty without NUL")
	}
	return s.serviceControl("load extension provider", func() int { return rtc.LoadExtensionProvider(path, unloadAfterUse) })
}

// EnableExtension enables a provider's extension for the native track identifier.
// Empty trackID uses the SDK's default scope. Set autoEnableOnTrack to request
// automatic enablement for subsequently created tracks. Native support varies.
func (s *Service) EnableExtension(provider, extension, trackID string, autoEnableOnTrack bool) error {
	if err := validateExtension(provider, extension, trackID); err != nil {
		return err
	}
	return s.serviceControl("enable extension", func() int { return rtc.EnableExtension(provider, extension, trackID, autoEnableOnTrack) })
}

// DisableExtension disables an extension in the specified native track scope.
// Empty trackID uses the SDK's default scope.
func (s *Service) DisableExtension(provider, extension, trackID string) error {
	if err := validateExtension(provider, extension, trackID); err != nil {
		return err
	}
	return s.serviceControl("disable extension", func() int { return rtc.DisableExtension(provider, extension, trackID) })
}

func validateExtension(provider, extension, track string) error {
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(extension) == "" || strings.ContainsRune(provider+extension+track, 0) {
		return invalid("extension", "provider and extension must be nonempty; names must not contain NUL")
	}
	return nil
}

func (s *Service) serviceControl(op string, call func() int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	return check(op, call())
}

// SessionID returns the native diagnostic session identifier. It may be empty
// before connection establishment. The copied string may be retained after Close;
// calling this method after Close returns ErrClosed.
func (s *Session) SessionID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", ErrClosed
	}
	if s.native == nil {
		return "", ErrUnsupported
	}
	return s.native.GetSid(), nil
}

// SetAudioProfile changes the native audio encoder profile. It does not switch
// between PCM and encoded publication. Profile and codec support remain subject
// to the installed native SDK and peer configuration.
func (s *Session) SetAudioProfile(profile rtc.AudioProfile) error {
	if profile < rtc.AudioProfileDefault || profile > rtc.AudioProfileIot {
		return invalid("audio profile", "is unknown")
	}
	return s.withNative("set audio profile", func(c *rtc.RtcConnection) int {
		return c.GetLocalUser().SetAudioEncoderConfiguration(&rtc.AudioEncoderConfiguration{AudioProfile: int(profile)})
	})
}
