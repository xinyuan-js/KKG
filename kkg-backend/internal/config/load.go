package config

import "fmt"

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "dev"),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		MySQLHost:          getEnv("MYSQL_HOST", "127.0.0.1"),
		MySQLPort:          getEnv("MYSQL_PORT", "3307"),
		MySQLDB:            getEnv("MYSQL_DATABASE", "kkgoj"),
		MySQLUser:          getEnv("MYSQL_USER", "blog"),
		MySQLPass:          getEnv("MYSQL_PASSWORD", ""),
		RedisHost:          getEnv("REDIS_HOST", "127.0.0.1"),
		RedisPort:          getEnv("REDIS_PORT", "6379"),
		RedisPass:          getEnv("REDIS_PASSWORD", ""),
		MinIOEndpoint:      getEnv("MINIO_ENDPOINT", "127.0.0.1:9000"),
		MinIOAccessKey:     getEnv("MINIO_ACCESS_KEY", getEnv("MINIO_ROOT_USER", "")),
		MinIOSecretKey:     getEnv("MINIO_SECRET_KEY", getEnv("MINIO_ROOT_PASSWORD", "")),
		MinIOBucket:        getEnv("MINIO_BUCKET", "blog-images"),
		MinIOPublicBaseURL: getEnv("MINIO_PUBLIC_BASE_URL", "http://127.0.0.1:9000"),
		MinIOUseSSL:        getEnvBool("MINIO_USE_SSL", false),
		ElasticsearchURL:   getEnv("ELASTICSEARCH_URL", "http://127.0.0.1:9200"),
		ElasticsearchIndex: getEnv("ELASTICSEARCH_TWEET_INDEX", "tweets"),
		ESPostIndex:        getEnv("ELASTICSEARCH_POST_INDEX", "posts"),
		ESUserIndex:        getEnv("ELASTICSEARCH_USER_INDEX", "users"),
		InternalAuthToken:  getEnv("BLOG_INTERNAL_AUTH_TOKEN", ""),
		SuperAdminUsername: getEnv("SUPER_ADMIN_USERNAME", ""),
		SuperAdminEmail:    getEnv("SUPER_ADMIN_EMAIL", ""),
		SuperAdminPassword: getEnv("SUPER_ADMIN_PASSWORD", ""),
	}
	cfg.OJ = loadOJConfig(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadOJConfig(cfg *Config) OJConfig {
	return OJConfig{
		Server: OJServerConfig{
			Port: cfg.ServerPort,
		},
		Redis: OJRedisConfig{
			Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
			Password: cfg.RedisPass,
			DB:       0,
		},
		ES: OJESConfig{
			Enabled: getEnvBool("OJ_ES_ENABLED", false),
			URL:     cfg.ElasticsearchURL,
			Index:   getEnv("OJ_ES_INDEX", cfg.ESPostIndex),
		},
		COS: OJCOSConfig{
			Enabled:   getEnvBool("OJ_COS_ENABLED", false),
			Host:      getEnv("OJ_COS_HOST", ""),
			SecretID:  getEnv("OJ_COS_SECRET_ID", ""),
			SecretKey: getEnv("OJ_COS_SECRET_KEY", ""),
			BucketURL: getEnv("OJ_COS_BUCKET_URL", ""),
		},
		WX: OJWXConfig{
			MpToken:    getEnv("WX_MP_TOKEN", ""),
			OpenAppID:  getEnv("WX_OPEN_APP_ID", ""),
			OpenSecret: getEnv("WX_OPEN_SECRET", ""),
		},
		Judge: OJJudgeConfig{
			SandboxType: getEnv("JUDGE_SANDBOX_TYPE", "example"),
			SandboxURL:  getEnv("JUDGE_SANDBOX_URL", ""),
			AuthSecret:  getEnv("JUDGE_AUTH_SECRET", "secretKey"),
		},
		RabbitMQ: OJRabbitMQConfig{
			Enabled:    getEnvBool("OJ_RABBITMQ_ENABLED", false),
			URL:        getEnv("RABBITMQ_URL", ""),
			JudgeQueue: getEnv("RABBITMQ_JUDGE_QUEUE", "oj.judge"),
		},
		Blog: OJBlogConfig{
			BaseURL:           getEnv("BLOG_BASE_URL", ""),
			AgentAccount:      getEnv("BLOG_AGENT_ACCOUNT", "kkg_agent"),
			AgentPassword:     getEnv("BLOG_AGENT_PASSWORD", getEnv("OJ_BLOG_AGENT_PASSWORD", "")),
			AgentEmail:        getEnv("BLOG_AGENT_EMAIL", "kkg-agent@example.com"),
			AgentDisplayName:  getEnv("BLOG_AGENT_DISPLAY_NAME", "KKG Agent"),
			InternalAuthToken: cfg.InternalAuthToken,
		},
		Agent: OJAgentConfig{
			Enabled:     getEnvBool("AGENT_ENABLED", getEnvBool("OJ_AGENT_ENABLED", false)),
			BaseURL:     getEnv("AGENT_BASE_URL", getEnv("OJ_AGENT_BASE_URL", "https://api.openai.com/v1")),
			APIKey:      getEnv("AGENT_API_KEY", getEnv("OJ_AGENT_API_KEY", "")),
			Model:       getEnv("AGENT_MODEL", getEnv("OJ_AGENT_MODEL", "gpt-4o-mini")),
			MaxRound:    getEnvInt("AGENT_MAX_ROUND", getEnvInt("OJ_AGENT_MAX_ROUND", 3)),
			Temperature: getEnvFloat("AGENT_TEMPERATURE", getEnvFloat("OJ_AGENT_TEMPERATURE", 0.2)),
		},
	}
}
