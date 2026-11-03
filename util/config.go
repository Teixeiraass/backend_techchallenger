package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ChaveExemplo é o valor de TOKEN_SYMMETRIC_KEY do app.env.example e do
// docker-compose.yaml. Só serve para desenvolvimento local.
const ChaveExemplo = "dev-only-change-me-0123456789abc"

const tamanhoMinimoChave = 32

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
	// Limite de requisições por IP nas rotas de login, renovação de token e
	// acompanhamento público: RATE_LIMIT_RPS por segundo, com rajada de
	// RATE_LIMIT_BURST.
	RateLimitRPS   float64 `mapstructure:"RATE_LIMIT_RPS"`
	RateLimitBurst int     `mapstructure:"RATE_LIMIT_BURST"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("DB_DRIVER", "postgres")
	viper.SetDefault("SERVER_ADDRESS", "0.0.0.0:8080")
	viper.SetDefault("TOKEN_TYPE", "jwt")
	viper.SetDefault("ACCESS_TOKEN_DURATION", "15m")
	viper.SetDefault("REFRESH_TOKEN_DURATION", "24h")
	viper.SetDefault("ADMIN_NOTIFICATION_TO", "admin@oficina.com")
	viper.SetDefault("RATE_LIMIT_RPS", 1.0)
	viper.SetDefault("RATE_LIMIT_BURST", 10)

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
		"RATE_LIMIT_RPS",
		"RATE_LIMIT_BURST",
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

	if err = viper.Unmarshal(&config); err != nil {
		return config, err
	}

	return config, config.validarSegredo()
}

// validarSegredo impede que a aplicação suba fora do ambiente de
// desenvolvimento com a chave de exemplo ou com uma chave curta demais.
func (config Config) validarSegredo() error {
	if strings.EqualFold(config.Environment, "development") {
		return nil
	}
	if config.TokenSymmetricKey == ChaveExemplo {
		return fmt.Errorf("TOKEN_SYMMETRIC_KEY usa o valor de exemplo; defina uma chave própria para ENVIRONMENT=%s", config.Environment)
	}
	if len(config.TokenSymmetricKey) < tamanhoMinimoChave {
		return fmt.Errorf("TOKEN_SYMMETRIC_KEY precisa ter pelo menos %d caracteres em ENVIRONMENT=%s", tamanhoMinimoChave, config.Environment)
	}
	return nil
}
