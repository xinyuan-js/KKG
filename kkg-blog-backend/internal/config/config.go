package config

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
	InternalAuthToken  string
	SuperAdminUsername string
	SuperAdminEmail    string
	SuperAdminPassword string
}
