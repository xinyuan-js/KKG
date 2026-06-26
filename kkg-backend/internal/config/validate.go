package config

import (
	"fmt"
	"strings"
)

func validate(cfg *Config) error {
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
	if cfg.OJ.Agent.Enabled {
		if strings.TrimSpace(cfg.OJ.Agent.BaseURL) == "" {
			return fmt.Errorf("invalid config: OJ agent base URL is required when agent is enabled")
		}
		if strings.TrimSpace(cfg.OJ.Agent.APIKey) == "" {
			return fmt.Errorf("invalid config: OJ agent API key is required when agent is enabled")
		}
		if strings.TrimSpace(cfg.OJ.Agent.Model) == "" {
			return fmt.Errorf("invalid config: OJ agent model is required when agent is enabled")
		}
	}
	if cfg.OJ.RabbitMQ.Enabled && strings.TrimSpace(cfg.OJ.RabbitMQ.URL) == "" {
		return fmt.Errorf("invalid config: RABBITMQ_URL is required when OJ_RABBITMQ_ENABLED=true")
	}
	if strings.TrimSpace(cfg.OJ.Blog.AgentAccount) != "" && strings.TrimSpace(cfg.OJ.Blog.AgentPassword) == "" {
		return fmt.Errorf("invalid config: OJ_BLOG_AGENT_PASSWORD or BLOG_AGENT_PASSWORD is required when blog agent account is set")
	}
	return nil
}
