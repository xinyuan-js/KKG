package config

import (
	"fmt"
	"strings"
)

func validate(cfg *Config) error {
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return fmt.Errorf("invalid config: jwt_secret is required")
	}
	if cfg.Agent.Enabled {
		if strings.TrimSpace(cfg.Agent.BaseURL) == "" {
			return fmt.Errorf("invalid config: agent.base_url is required when agent.enabled=true")
		}
		if strings.TrimSpace(cfg.Agent.APIKey) == "" {
			return fmt.Errorf("invalid config: agent.api_key is required when agent.enabled=true")
		}
		if strings.TrimSpace(cfg.Agent.Model) == "" {
			return fmt.Errorf("invalid config: agent.model is required when agent.enabled=true")
		}
	}
	if cfg.RabbitMQ.Enabled && strings.TrimSpace(cfg.RabbitMQ.URL) == "" {
		return fmt.Errorf("invalid config: rabbitmq.url is required when rabbitmq.enabled=true")
	}
	if strings.TrimSpace(cfg.Blog.AgentAccount) != "" && strings.TrimSpace(cfg.Blog.AgentPassword) == "" {
		return fmt.Errorf("invalid config: blog.agent_password is required when blog.agent_account is set")
	}
	return nil
}
