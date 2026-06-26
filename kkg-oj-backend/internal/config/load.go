package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config.yaml: %w", err)
	}
	v.SetConfigFile("config.local.yaml")
	_ = v.MergeInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", "8121")
	v.SetDefault("mysql.dsn", "root:123456@tcp(127.0.0.1:3306)/kkgoj?charset=utf8mb4&parseTime=True&loc=Local")
	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 1)
	v.SetDefault("es.enabled", false)
	v.SetDefault("es.url", "http://127.0.0.1:9200")
	v.SetDefault("es.index", "post_v1")
	v.SetDefault("cos.enabled", false)
	v.SetDefault("wx.mp_token", "")
	v.SetDefault("wx.open_app_id", "")
	v.SetDefault("wx.open_secret", "")
	v.SetDefault("judge.sandbox_type", "example")
	v.SetDefault("judge.sandbox_url", "")
	v.SetDefault("judge.auth_secret", "secretKey")
	v.SetDefault("rabbitmq.enabled", false)
	v.SetDefault("rabbitmq.url", "amqp://guest:guest@127.0.0.1:5672/")
	v.SetDefault("rabbitmq.judge_queue", "oj.judge")
	v.SetDefault("blog.base_url", "http://127.0.0.1:8080")
	v.SetDefault("blog.agent_account", "kkg_agent")
	v.SetDefault("blog.agent_password", "")
	v.SetDefault("blog.agent_email", "kkg-agent@example.com")
	v.SetDefault("blog.agent_display_name", "KKG Agent")
	v.SetDefault("blog.internal_auth_token", "")
	v.SetDefault("agent.enabled", false)
	v.SetDefault("agent.base_url", "https://api.openai.com/v1")
	v.SetDefault("agent.api_key", "")
	v.SetDefault("agent.model", "gpt-4o-mini")
	v.SetDefault("agent.max_round", 3)
	v.SetDefault("agent.temperature", 0.2)
	v.SetDefault("jwt_secret", "")
}
