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
	OJ                 OJConfig
}

type OJConfig struct {
	Server   OJServerConfig
	Redis    OJRedisConfig
	ES       OJESConfig
	COS      OJCOSConfig
	WX       OJWXConfig
	Judge    OJJudgeConfig
	RabbitMQ OJRabbitMQConfig
	Blog     OJBlogConfig
	Agent    OJAgentConfig
}

type OJServerConfig struct {
	Port string
}

type OJRedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type OJESConfig struct {
	Enabled bool
	URL     string
	Index   string
}

type OJCOSConfig struct {
	Enabled   bool
	Host      string
	SecretID  string
	SecretKey string
	BucketURL string
}

type OJWXConfig struct {
	MpToken    string
	OpenAppID  string
	OpenSecret string
}

type OJJudgeConfig struct {
	SandboxType string
	SandboxURL  string
	AuthSecret  string
}

type OJRabbitMQConfig struct {
	Enabled    bool
	URL        string
	JudgeQueue string
}

type OJBlogConfig struct {
	BaseURL           string
	AgentAccount      string
	AgentPassword     string
	AgentEmail        string
	AgentDisplayName  string
	InternalAuthToken string
}

type OJAgentConfig struct {
	Enabled     bool
	BaseURL     string
	APIKey      string
	Model       string
	MaxRound    int
	Temperature float64
}
