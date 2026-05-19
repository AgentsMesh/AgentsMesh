package config

import "fmt"

type ServerConfig struct {
	Address            string
	Debug              bool
	CORSAllowedOrigins []string
	InternalAPISecret  string // Secret for internal API authentication (Relay communication)
}

type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	ReplicaDSNs []string // Read replica DSNs for read-write separation
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func (c DatabaseConfig) HasReplicas() bool {
	return len(c.ReplicaDSNs) > 0
}

type RedisConfig struct {
	URL      string
	Host     string
	Port     int
	Password string
	DB       int
}

func (c RedisConfig) IsConfigured() bool {
	return c.URL != "" || c.Host != ""
}

type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

type WebhookConfig struct {
	GitLabSecret string
	GitHubSecret string
	GiteeSecret  string
}

type LogConfig struct {
	Level      string // debug, info, warn, error
	Format     string // json, text
	FilePath   string // path to log file, empty means stdout only
	MaxSizeMB  int    // max size in MB before rotation
	MaxBackups int    // max number of backup files
}
