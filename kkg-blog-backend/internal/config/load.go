package config

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
		InternalAuthToken:  getEnv("BLOG_INTERNAL_AUTH_TOKEN", ""),
		SuperAdminUsername: getEnv("SUPER_ADMIN_USERNAME", ""),
		SuperAdminEmail:    getEnv("SUPER_ADMIN_EMAIL", ""),
		SuperAdminPassword: getEnv("SUPER_ADMIN_PASSWORD", ""),
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
