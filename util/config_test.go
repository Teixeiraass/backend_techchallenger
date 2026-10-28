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
	dir := t.TempDir()
	content := "DB_SOURCE=postgresql://file@localhost/oficina\nSERVER_ADDRESS=127.0.0.1:9090\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(content), 0o600))

	config, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "postgresql://file@localhost/oficina", config.DBSource)
	require.Equal(t, "127.0.0.1:9090", config.ServerAddress)
}

func TestLoadConfigInvalidFile(t *testing.T) {
	viper.Reset()
	dir := t.TempDir()
	// Linha sem "=" não é um arquivo .env válido.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte("isto nao e env\n\"aberto"), 0o600))

	_, err := LoadConfig(dir)
	require.Error(t, err)
}
