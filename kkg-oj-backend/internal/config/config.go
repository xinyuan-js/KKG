package config

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
