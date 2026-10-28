package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config store all configuration of the application.
// The values are read by viper from a config file or environmnet variables.
type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	TokenType            string        `mapstructure:"TOKEN_TYPE"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	AdminNotificationTo  string        `mapstructure:"ADMIN_NOTIFICATION_TO"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("DB_DRIVER", "postgres")
	viper.SetDefault("SERVER_ADDRESS", "0.0.0.0:8080")
	viper.SetDefault("TOKEN_TYPE", "paseto")
	viper.SetDefault("ACCESS_TOKEN_DURATION", "15m")
	viper.SetDefault("REFRESH_TOKEN_DURATION", "24h")
	viper.SetDefault("ADMIN_NOTIFICATION_TO", "admin@oficina.com")

	for _, key := range []string{
		"ENVIRONMENT",
		"DB_DRIVER",
		"DB_SOURCE",
		"SERVER_ADDRESS",
		"TOKEN_TYPE",
		"TOKEN_SYMMETRIC_KEY",
		"ACCESS_TOKEN_DURATION",
		"REFRESH_TOKEN_DURATION",
		"ADMIN_NOTIFICATION_TO",
	} {
		if err := viper.BindEnv(key); err != nil {
			return config, err
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	err = viper.Unmarshal(&config)
	return
}
