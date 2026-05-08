package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppEnv             string
	ServerPort         string
	MySQLHost          string
	MySQLPort          string
	MySQLDB            string
	MySQLUser          string
	MySQLPass          string
	RedisHost          string
	RedisPort          string
	RedisPass          string
	JWTSecretKey       string
	MinIOEndpoint      string
	MinIOAccessKey     string
	MinIOSecretKey     string
	MinIOBucket        string
	MinIOPublicBaseURL string
	MinIOUseSSL        bool
	ElasticsearchURL   string
	ElasticsearchIndex string
	ESPostIndex        string
	ESUserIndex        string
	SuperAdminUsername string
	SuperAdminEmail    string
	SuperAdminPassword string
}

func Load() *Config {
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
		JWTSecretKey:       getEnv("JWT_SECRET", ""),
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
		SuperAdminUsername: getEnv("SUPER_ADMIN_USERNAME", ""),
		SuperAdminEmail:    getEnv("SUPER_ADMIN_EMAIL", ""),
		SuperAdminPassword: getEnv("SUPER_ADMIN_PASSWORD", ""),
	}
	if err := validate(cfg); err != nil {
		panic(err)
	}
	return cfg
}

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

func getEnv(key string, defaultVal string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	return v
}

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	switch v {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	default:
		return false
	}
}
