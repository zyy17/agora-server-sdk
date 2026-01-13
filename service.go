package agorasdk

import (
	"errors"
	"fmt"
	"os"

	rtctokenbuilder "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/rtctokenbuilder2"
	agoraservice "github.com/zyy17/agora-server-sdk/agora/rtc"
)

const (
	// AGORA_APP_ID_ENV_VAR is the environment variable for the Agora app ID.
	AGORA_APP_ID_ENV_VAR = "AGORA_APP_ID"

	// AGORA_APP_CERT_ENV_VAR is the environment variable for the Agora app certificate.
	AGORA_APP_CERT_ENV_VAR = "AGORA_APP_CERT"
)

const (
	// DefaultTokenExpirationInSeconds is the default token expiration time in seconds.
	DefaultTokenExpirationInSeconds = 3600

	// DefaultPrivilegeExpirationInSeconds is the default privilege expiration time in seconds.
	DefaultPrivilegeExpirationInSeconds = 3600
)

var (
	// ErrEmptyAppID is the error returned when the app ID is empty.
	ErrEmptyAppID = errors.New("empty app id")

	// ErrEmptyAppCert is the error returned when the app certificate is empty.
	ErrEmptyAppCert = errors.New("empty app cert")

	// ErrEmptyUserID is the error returned when the user ID is empty.
	ErrEmptyUserID = errors.New("empty user id")

	// ErrEmptyChannelName is the error returned when the channel name is empty.
	ErrEmptyChannelName = errors.New("empty channel name")
)

// Service represents an Agora RTC service instance.
type Service struct {
	channelName string
	userID      string
	token       string
}

// ServiceConfig represents the configuration for an Agora RTC service.
type ServiceConfig struct {
	appCert     string
	userID      string
	channelName string
	token       string
	svcCfg      *agoraservice.AgoraServiceConfig
}

// NewService creates a new Agora RTC service instance.
func NewService(opts ...ServiceOption) (*Service, error) {
	cfg := &ServiceConfig{
		svcCfg: agoraservice.NewAgoraServiceConfig(),
	}

	// Apply the service options.
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.svcCfg.AppId == "" {
		return nil, ErrEmptyAppID
	}

	if cfg.appCert == "" {
		return nil, ErrEmptyAppCert
	}

	if cfg.userID == "" {
		return nil, ErrEmptyUserID
	}

	if cfg.channelName == "" {
		return nil, ErrEmptyChannelName
	}

	if cfg.token == "" {
		if err := cfg.generateToken(); err != nil {
			return nil, fmt.Errorf("failed to generate token, %w", err)
		}
	}

	// Initialize the Agora RTC service.
	if ret := agoraservice.Initialize(cfg.svcCfg); ret != 0 {
		return nil, fmt.Errorf("failed to initialize Agora RTC service, return %d", ret)
	}

	return &Service{
		userID:      cfg.userID,
		channelName: cfg.channelName,
		token:       cfg.token,
	}, nil
}

// ServiceOption is a function that configures a service.
type ServiceOption func(*ServiceConfig)

// WithAppID sets the app ID for the service.
func WithAppID(appID string) ServiceOption {
	return func(cfg *ServiceConfig) {
		if appID == "" {
			appID = os.Getenv(AGORA_APP_ID_ENV_VAR)
		}
		cfg.svcCfg.AppId = appID
	}
}

// WithAppCert sets the app certificate for the service.
func WithAppCert(appCert string) ServiceOption {
	return func(cfg *ServiceConfig) {
		if appCert == "" {
			appCert = os.Getenv(AGORA_APP_CERT_ENV_VAR)
		}
		cfg.appCert = appCert
	}
}

// WithUserID sets the user ID for the service.
func WithUserID(userID string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.userID = userID
	}
}

// WithChannelName sets the channel name for the service.
func WithChannelName(channelName string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.channelName = channelName
	}
}

// WithToken sets the token for the service.
func WithToken(token string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.token = token
	}
}

// WithLogPath sets the log path for the service.
func WithLogPath(path string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.svcCfg.LogPath = path
	}
}

// WithConfigDir sets the config directory for the service.
func WithConfigDir(dir string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.svcCfg.ConfigDir = dir
	}
}

// WithDataDir sets the data directory for the service.
func WithDataDir(dir string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.svcCfg.DataDir = dir
	}
}

// Release releases the service resources.
func (s *Service) Release() {
	agoraservice.Release()
}

func (cfg *ServiceConfig) generateToken() error {
	token, err := rtctokenbuilder.BuildTokenWithUserAccount(cfg.svcCfg.AppId, cfg.appCert, cfg.channelName, cfg.userID,
		rtctokenbuilder.RolePublisher, DefaultTokenExpirationInSeconds, DefaultPrivilegeExpirationInSeconds)
	if err != nil {
		return err
	}

	// Set the token that requested from the Agora.
	cfg.token = token

	return nil
}
