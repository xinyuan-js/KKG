package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig   `mapstructure:"server"`
	MySQL     MySQLConfig    `mapstructure:"mysql"`
	Redis     RedisConfig    `mapstructure:"redis"`
	ES        ESConfig       `mapstructure:"es"`
	COS       COSConfig      `mapstructure:"cos"`
	WX        WXConfig       `mapstructure:"wx"`
	Judge     JudgeConfig    `mapstructure:"judge"`
	RabbitMQ  RabbitMQConfig `mapstructure:"rabbitmq"`
	Blog      BlogConfig     `mapstructure:"blog"`
	Agent     AgentConfig    `mapstructure:"agent"`
	JWTSecret string         `mapstructure:"jwt_secret"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type MySQLConfig struct {
	DSN string `mapstructure:"dsn"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ESConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
	Index   string `mapstructure:"index"`
}

type COSConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Host      string `mapstructure:"host"`
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
	BucketURL string `mapstructure:"bucket_url"`
}

type WXConfig struct {
	MpToken    string `mapstructure:"mp_token"`
	OpenAppID  string `mapstructure:"open_app_id"`
	OpenSecret string `mapstructure:"open_secret"`
}

type JudgeConfig struct {
	SandboxType string `mapstructure:"sandbox_type"`
	SandboxURL  string `mapstructure:"sandbox_url"`
	AuthSecret  string `mapstructure:"auth_secret"`
}

type RabbitMQConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	URL        string `mapstructure:"url"`
	JudgeQueue string `mapstructure:"judge_queue"`
}

type BlogConfig struct {
	BaseURL           string `mapstructure:"base_url"`
	AgentAccount      string `mapstructure:"agent_account"`
	AgentPassword     string `mapstructure:"agent_password"`
	AgentEmail        string `mapstructure:"agent_email"`
	AgentDisplayName  string `mapstructure:"agent_display_name"`
	InternalAuthToken string `mapstructure:"internal_auth_token"`
}

type AgentConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxRound    int     `mapstructure:"max_round"`
	Temperature float64 `mapstructure:"temperature"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
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
