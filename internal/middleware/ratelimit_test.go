package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRateLimiterBloqueiaDepoisDoBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(0.001, 3)

	router := gin.New()
	router.GET("/x", rl.Handler(), func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	chamar := func(ip string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/x", nil)
		request.RemoteAddr = ip + ":1234"
		router.ServeHTTP(recorder, request)
		return recorder
	}

	for i := 0; i < 3; i++ {
		require.Equal(t, http.StatusOK, chamar("10.0.0.1").Code)
	}
	bloqueado := chamar("10.0.0.1")
	require.Equal(t, http.StatusTooManyRequests, bloqueado.Code)
	require.NotEmpty(t, bloqueado.Header().Get("Retry-After"))
	require.Contains(t, bloqueado.Body.String(), "MUITAS_TENTATIVAS")

	// Outro IP tem o próprio balde.
	require.Equal(t, http.StatusOK, chamar("10.0.0.2").Code)
}

func TestRateLimiterLimpaClientesInativos(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	agora := time.Now()
	rl.now = func() time.Time { return agora }

	require.True(t, rl.permitir("1.1.1.1"))
	require.Len(t, rl.clientes, 1)

	rl.now = func() time.Time { return agora.Add(2 * time.Minute) }
	rl.limpar()
	require.Len(t, rl.clientes, 1, "ainda dentro do ttl")

	rl.now = func() time.Time { return agora.Add(10 * time.Minute) }
	rl.limpar()
	require.Empty(t, rl.clientes)
}

func TestRateLimiterRetryAfter(t *testing.T) {
	require.Equal(t, 1, NewRateLimiter(1, 1).segundosParaProximo())
	require.Equal(t, 2, NewRateLimiter(0.5, 1).segundosParaProximo())
	require.Equal(t, 60, NewRateLimiter(0, 1).segundosParaProximo())
}
