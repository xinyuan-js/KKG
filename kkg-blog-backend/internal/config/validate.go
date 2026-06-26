package config

import (
	"fmt"
	"strings"
)

func validate(cfg *Config) error {
	if strings.TrimSpace(cfg.JWTSecretKey) == "" || cfg.JWTSecretKey == "replace-with-strong-secret" {
		return fmt.Errorf("invalid config: JWT_SECRET is required and must not use placeholder")
	}
	if strings.TrimSpace(cfg.MySQLPass) == "" {
		return fmt.Errorf("invalid config: MYSQL_PASSWORD is required")
	}
	if strings.TrimSpace(cfg.RedisPass) == "" {
		return fmt.Errorf("invalid config: REDIS_PASSWORD is required")
	}
	if strings.TrimSpace(cfg.MinIOAccessKey) == "" || strings.TrimSpace(cfg.MinIOSecretKey) == "" {
		return fmt.Errorf("invalid config: MINIO credentials are required")
	}
	if strings.TrimSpace(cfg.SuperAdminUsername) != "" ||
		strings.TrimSpace(cfg.SuperAdminEmail) != "" ||
		strings.TrimSpace(cfg.SuperAdminPassword) != "" {
		if len(strings.TrimSpace(cfg.SuperAdminPassword)) < 8 {
			return fmt.Errorf("invalid config: SUPER_ADMIN_PASSWORD must be at least 8 characters when bootstrap account is enabled")
		}
	}
	return nil
}
