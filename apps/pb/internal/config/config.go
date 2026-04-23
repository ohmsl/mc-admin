package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRCONHost           = "minecraft"
	defaultRCONPort           = 25575
	defaultRCONTimeoutSeconds = 5
	defaultRCONRetries        = 1
	defaultAuditCollection    = "mc_audit_logs"
	defaultRoleField          = "role"
)

type Config struct {
	RCON        RCONConfig
	Permissions PermissionConfig
	Collections CollectionConfig
}

type RCONConfig struct {
	Host       string
	Port       int
	Password   string
	Timeout    time.Duration
	RetryCount int
}

type PermissionConfig struct {
	AllowRawCommand bool
	RoleField       string
}

type CollectionConfig struct {
	AuditLogs string
}

func Load() (Config, error) {
	port, err := getenvInt("MC_RCON_PORT", defaultRCONPort)
	if err != nil {
		return Config{}, err
	}

	timeoutSeconds, err := getenvInt("MC_RCON_TIMEOUT_SECONDS", defaultRCONTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}

	retryCount, err := getenvInt("MC_RCON_RETRY_COUNT", defaultRCONRetries)
	if err != nil {
		return Config{}, err
	}

	allowRaw, err := getenvBool("MC_ALLOW_RAW_COMMAND", false)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		RCON: RCONConfig{
			Host:       getenvString("MC_RCON_HOST", defaultRCONHost),
			Port:       port,
			Password:   os.Getenv("MC_RCON_PASSWORD"),
			Timeout:    time.Duration(timeoutSeconds) * time.Second,
			RetryCount: retryCount,
		},
		Permissions: PermissionConfig{
			AllowRawCommand: allowRaw,
			RoleField:       getenvString("MC_ROLE_FIELD", defaultRoleField),
		},
		Collections: CollectionConfig{
			AuditLogs: getenvString("MC_AUDIT_COLLECTION", defaultAuditCollection),
		},
	}

	if cfg.RCON.Host == "" {
		return Config{}, fmt.Errorf("MC_RCON_HOST cannot be empty")
	}

	if cfg.RCON.Port < 1 || cfg.RCON.Port > 65535 {
		return Config{}, fmt.Errorf("MC_RCON_PORT must be between 1 and 65535")
	}

	if cfg.RCON.Timeout <= 0 {
		return Config{}, fmt.Errorf("MC_RCON_TIMEOUT_SECONDS must be a positive integer")
	}

	if cfg.RCON.RetryCount < 0 || cfg.RCON.RetryCount > 10 {
		return Config{}, fmt.Errorf("MC_RCON_RETRY_COUNT must be between 0 and 10")
	}

	if cfg.Permissions.RoleField == "" {
		return Config{}, fmt.Errorf("MC_ROLE_FIELD cannot be empty")
	}

	if cfg.Collections.AuditLogs == "" {
		return Config{}, fmt.Errorf("MC_AUDIT_COLLECTION cannot be empty")
	}

	return cfg, nil
}

func (c RCONConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getenvString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return parsed, nil
}

func getenvBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}

	return parsed, nil
}
