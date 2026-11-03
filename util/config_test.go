package util

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigFromEnv(t *testing.T) {
	viper.Reset()
	t.Setenv("DB_SOURCE", "postgresql://user:pass@localhost:5432/oficina?sslmode=disable")
	t.Setenv("TOKEN_SYMMETRIC_KEY", "12345678901234567890123456789012")
	t.Setenv("ACCESS_TOKEN_DURATION", "30m")
	t.Setenv("TOKEN_TYPE", "jwt")

	config, err := LoadConfig(t.TempDir())
	require.NoError(t, err)
	require.Equal(t, "postgresql://user:pass@localhost:5432/oficina?sslmode=disable", config.DBSource)
	require.Equal(t, "12345678901234567890123456789012", config.TokenSymmetricKey)
	require.Equal(t, 30*time.Minute, config.AccessTokenDuration)
	require.Equal(t, "jwt", config.TokenType)

	// defaults
	require.Equal(t, "postgres", config.DBDriver)
	require.Equal(t, "0.0.0.0:8080", config.ServerAddress)
	require.Equal(t, 24*time.Hour, config.RefreshTokenDuration)
	require.Equal(t, "admin@oficina.com", config.AdminNotificationTo)
}

func TestLoadConfigFromFile(t *testing.T) {
	viper.Reset()
	// Variáveis de ambiente têm prioridade sobre o arquivo; o "make test"
	// define DB_SOURCE, então este teste esvazia as duas chaves que confere.
	t.Setenv("DB_SOURCE", "")
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("TOKEN_TYPE", "")
	dir := t.TempDir()
	content := "DB_SOURCE=postgresql://file@localhost/oficina\nSERVER_ADDRESS=127.0.0.1:9090\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(content), 0o600))

	config, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "postgresql://file@localhost/oficina", config.DBSource)
	require.Equal(t, "127.0.0.1:9090", config.ServerAddress)
	// JWT é o padrão (decisão DC6); PASETO fica como opção.
	require.Equal(t, "jwt", config.TokenType)
}

func TestLoadConfigInvalidFile(t *testing.T) {
	viper.Reset()
	dir := t.TempDir()
	// Linha sem "=" não é um arquivo .env válido.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte("isto nao e env\n\"aberto"), 0o600))

	_, err := LoadConfig(dir)
	require.Error(t, err)
}

func TestLoadConfigRecusaChaveDeExemploForaDoDesenvolvimento(t *testing.T) {
	viper.Reset()
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("TOKEN_SYMMETRIC_KEY", ChaveExemplo)

	_, err := LoadConfig(t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "valor de exemplo")
}

func TestLoadConfigRecusaChaveCurtaForaDoDesenvolvimento(t *testing.T) {
	viper.Reset()
	t.Setenv("ENVIRONMENT", "staging")
	t.Setenv("TOKEN_SYMMETRIC_KEY", "curta")

	_, err := LoadConfig(t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "32 caracteres")
}

func TestLoadConfigAceitaChaveDeExemploEmDesenvolvimento(t *testing.T) {
	viper.Reset()
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("TOKEN_SYMMETRIC_KEY", ChaveExemplo)

	config, err := LoadConfig(t.TempDir())
	require.NoError(t, err)
	require.Equal(t, 1.0, config.RateLimitRPS)
	require.Equal(t, 10, config.RateLimitBurst)
}

func TestLoadConfigAceitaChavePropriaEmProducao(t *testing.T) {
	viper.Reset()
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("TOKEN_SYMMETRIC_KEY", "12345678901234567890123456789012")
	t.Setenv("RATE_LIMIT_RPS", "5")
	t.Setenv("RATE_LIMIT_BURST", "50")

	config, err := LoadConfig(t.TempDir())
	require.NoError(t, err)
	require.Equal(t, 5.0, config.RateLimitRPS)
	require.Equal(t, 50, config.RateLimitBurst)
}
