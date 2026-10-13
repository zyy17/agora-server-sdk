package messaging

import (
	"encoding/json"
	"math"
	"time"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

// IdentityMode selects how RTM interprets Config.UserID for RTC interoperability.
// Numeric IDs are still supplied as decimal strings; the SDK validates their range.
type IdentityMode uint8

const (
	// IdentityDefault uses string identities unless Config.Native was supplied,
	// in which case its UseStringUserId setting is retained.
	IdentityDefault IdentityMode = iota
	// IdentityString explicitly selects string identities, including numeric-looking strings.
	IdentityString
	// IdentityNumeric selects numeric RTC-compatible identities. Use the same
	// numeric identity when generating the associated RTC and RTM tokens.
	IdentityNumeric
)

// AreaCode selects the native region or combination of regions.
type AreaCode = native.RtmAreaCode

const (
	// AreaGlobal allows the SDK to select a global region.
	AreaGlobal = native.RtmAreaCodeGLOB
	// AreaChina selects mainland China.
	AreaChina = native.RtmAreaCodeCN
	// AreaNorthAmerica selects North America.
	AreaNorthAmerica = native.RtmAreaCodeNA
	// AreaEurope selects Europe.
	AreaEurope = native.RtmAreaCodeEU
	// AreaAsia selects Asia excluding mainland China.
	AreaAsia = native.RtmAreaCodeAS
	// AreaJapan selects Japan.
	AreaJapan = native.RtmAreaCodeJP
	// AreaIndia selects India.
	AreaIndia = native.RtmAreaCodeIN
)

// ConnectionConfig controls all native transport options. When supplied, its
// fields replace the corresponding Native settings. Durations must be nonnegative
// whole seconds fitting uint32; zero is passed unchanged. Area zero selects global.
// The SDK validates service-specific timer ranges. A nil Config.Connection retains
// the pinned Go binding's defaults (presence 30 seconds, heartbeat and reconnect zero).
type ConnectionConfig struct {
	// Area restricts connection regions; flags can be combined with bitwise OR.
	Area AreaCode
	// Protocol selects TCP/UDP or TCP-only transport.
	Protocol ProtocolType
	// PresenceTimeout is the native presence-disconnection timeout.
	PresenceTimeout time.Duration
	// HeartbeatInterval is the native heartbeat interval.
	HeartbeatInterval time.Duration
	// ReconnectTimeout is the native login/reconnection timeout. Zero retries
	// indefinitely. This is independent of Config.RequestTimeout.
	ReconnectTimeout time.Duration
	// Multipath enables native multipath for stream channels.
	Multipath bool
	// ISPPolicyEnabled enables domain-based connection for ISP-restricted devices.
	ISPPolicyEnabled bool
}

// LogLevel is the native RTM logging level.
type LogLevel = native.RtmLogLevel

const (
	// LogNone disables logging.
	LogNone = native.RtmLogLevelNONE
	// LogInfo enables informational logging.
	LogInfo = native.RtmLogLevelINFO
	// LogWarning enables warning logging.
	LogWarning = native.RtmLogLevelWARN
	// LogError enables error logging.
	LogError = native.RtmLogLevelERROR
	// LogFatal enables critical-error logging.
	LogFatal = native.RtmLogLevelFATAL
)

// LogConfig controls native log files. Fields are passed unchanged; use nil in
// Config for native defaults. FilePath must be UTF-8 without NUL.
type LogConfig struct {
	// FilePath selects the native log file location.
	FilePath string
	// FileSizeKB is the native maximum file size in KiB.
	FileSizeKB uint32
	// Level selects native logging severity; zero disables logging.
	Level LogLevel
}

// ProxyType is the native RTM proxy selection.
type ProxyType = native.RtmProxyType

const (
	// ProxyNone disables proxying.
	ProxyNone = native.RtmProxyTypeNONE
	// ProxyHTTP selects an HTTP proxy.
	ProxyHTTP = native.RtmProxyTypeHTTP
	// ProxyCloudTCP selects Agora's TCP cloud proxy.
	ProxyCloudTCP = native.RtmProxyTypeCLOUD_TCP
)

// ProxyConfig controls native proxying. Strings must be UTF-8 without NUL;
// the SDK validates the combination of type, endpoint, and credentials.
type ProxyConfig struct {
	// Type selects the proxy transport.
	Type ProxyType
	// Server is the proxy host.
	Server string
	// Port is the proxy port.
	Port uint16
	// Account is the optional proxy login.
	Account string
	// Password is the optional proxy password. Do not log this configuration.
	Password string
}

// EncryptionMode selects native message encryption.
type EncryptionMode = native.RtmEncryptionMode

const (
	// EncryptionNone disables message encryption.
	EncryptionNone = native.RtmEncryptionModeNONE
	// EncryptionAES128GCM selects AES-128-GCM.
	EncryptionAES128GCM = native.RtmEncryptionModeAES128GCM
	// EncryptionAES256GCM selects AES-256-GCM.
	EncryptionAES256GCM = native.RtmEncryptionModeAES256GCM
)

// EncryptionConfig configures native message encryption. The fixed-size Salt
// avoids silent truncation or padding. All participating clients need compatible
// settings; key generation/distribution is the application's responsibility.
type EncryptionConfig struct {
	// Mode selects the encryption algorithm.
	Mode EncryptionMode
	// Key is the native encryption key and must be UTF-8 without NUL. Do not log it.
	Key string
	// Salt is the native 32-byte encryption salt.
	Salt [32]byte
}

// ServiceType identifies the RTM message or stream service.
type ServiceType = native.RtmServiceType

const (
	// ServiceNone is the unspecified service.
	ServiceNone = native.RtmServiceTypeNONE
	// ServiceMessage identifies the message service.
	ServiceMessage = native.RtmServiceTypeMESSAGE
	// ServiceStream identifies the stream service.
	ServiceStream = native.RtmServiceTypeSTREAM
)

// StorageType identifies user or channel metadata.
type StorageType = native.RtmStorageType

const (
	// StorageNone is unspecified metadata storage.
	StorageNone = native.RtmStorageTypeNONE
	// StorageUser identifies user metadata.
	StorageUser = native.RtmStorageTypeUSER
	// StorageChannel identifies channel metadata.
	StorageChannel = native.RtmStorageTypeCHANNEL
)

// PrivateConfig configures the native private deployment endpoints.
// It does not provision private service access; endpoints must be UTF-8 without NUL.
type PrivateConfig struct {
	// Service selects the service routed to these access points.
	Service ServiceType
	// AccessPointHosts contains the access-point host names, copied during New.
	AccessPointHosts []string
}

func nativeSeconds(name string, d time.Duration) (uint32, error) {
	if d < 0 || d%time.Second != 0 || d/time.Second > math.MaxUint32 {
		return 0, invalid(name + " must be nonnegative whole seconds fitting uint32")
	}
	return uint32(d / time.Second), nil
}

func validateParameters(parameters string) error {
	if err := validText("parameters", parameters, false); err != nil {
		return err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(parameters), &object); err != nil || object == nil {
		return invalid("parameters must be a JSON object")
	}
	return nil
}

func (cfg Config) nativeConfig() (*native.RtmConfig, error) {
	if err := validText("app ID", cfg.AppID, true); err != nil {
		return nil, err
	}
	if err := validText("user ID", cfg.UserID, true); err != nil {
		return nil, err
	}
	n := native.NewRtmConfig()
	if cfg.Native != nil {
		*n = *cfg.Native
	} else {
		n.UseStringUserId = true
	}
	n.AppId, n.UserId, n.EventHandler = cfg.AppID, cfg.UserID, nil
	switch cfg.Identity {
	case IdentityDefault:
	case IdentityString:
		n.UseStringUserId = true
	case IdentityNumeric:
		n.UseStringUserId = false
	default:
		return nil, invalid("identity mode")
	}
	if cfg.Connection != nil {
		v := cfg.Connection
		n.AreaCode = v.Area
		if n.AreaCode == 0 {
			n.AreaCode = AreaGlobal
		}
		if v.Protocol != ProtocolTCPUDP && v.Protocol != ProtocolTCPOnly {
			return nil, invalid("protocol")
		}
		n.ProtocolType = uint32(v.Protocol)
		var err error
		if n.PresenceTimeout, err = nativeSeconds("presence timeout", v.PresenceTimeout); err != nil {
			return nil, err
		}
		if n.HeartbeatInterval, err = nativeSeconds("heartbeat interval", v.HeartbeatInterval); err != nil {
			return nil, err
		}
		if n.ReconnectTimeout, err = nativeSeconds("reconnect timeout", v.ReconnectTimeout); err != nil {
			return nil, err
		}
		n.Multipath, n.ISPPolicyEnabled = v.Multipath, v.ISPPolicyEnabled
	}
	if cfg.Log != nil {
		n.LogConfig = &native.RtmLogConfig{FilePath: cfg.Log.FilePath, FileSizeInKB: cfg.Log.FileSizeKB, Level: cfg.Log.Level}
	}
	if cfg.Proxy != nil {
		n.ProxyConfig = &native.RtmProxyConfig{ProxyType: cfg.Proxy.Type, Server: cfg.Proxy.Server, Port: cfg.Proxy.Port, Account: cfg.Proxy.Account, Password: cfg.Proxy.Password}
	}
	if cfg.Encryption != nil {
		n.EncryptionConfig = &native.RtmEncryptionConfig{EncryptionMode: cfg.Encryption.Mode, EncryptionKey: cfg.Encryption.Key, EncryptionSalt: cfg.Encryption.Salt[:]}
	}
	if cfg.Private != nil {
		n.PrivateConfig = &native.RtmPrivateConfig{ServiceType: uint32(cfg.Private.Service), AccessPointHosts: cfg.Private.AccessPointHosts}
	}
	if cfg.Parameters != "" {
		n.Parameters = cfg.Parameters
	}
	if n.Parameters != "" {
		if err := validateParameters(n.Parameters); err != nil {
			return nil, err
		}
	}
	// Copy and validate nested configurations, including the compatibility path.
	if n.LogConfig != nil {
		v := *n.LogConfig
		n.LogConfig = &v
		if err := validText("log path", v.FilePath, false); err != nil {
			return nil, err
		}
	}
	if n.ProxyConfig != nil {
		v := *n.ProxyConfig
		n.ProxyConfig = &v
		for _, item := range []struct{ name, value string }{{"proxy server", v.Server}, {"proxy account", v.Account}, {"proxy password", v.Password}} {
			if err := validText(item.name, item.value, false); err != nil {
				return nil, err
			}
		}
	}
	if n.EncryptionConfig != nil {
		v := *n.EncryptionConfig
		n.EncryptionConfig = &v
		v.EncryptionSalt = append([]byte(nil), v.EncryptionSalt...)
		if err := validText("encryption key", v.EncryptionKey, false); err != nil {
			return nil, err
		}
		if len(v.EncryptionSalt) > 32 {
			return nil, invalid("encryption salt exceeds 32 bytes")
		}
	}
	if n.PrivateConfig != nil {
		v := *n.PrivateConfig
		n.PrivateConfig = &v
		v.AccessPointHosts = append([]string(nil), v.AccessPointHosts...)
		for _, host := range v.AccessPointHosts {
			if err := validText("private access point", host, true); err != nil {
				return nil, err
			}
		}
	}
	return n, nil
}
