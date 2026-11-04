package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLoggerNaoRegistraQueryString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var linhas []string
	router := gin.New()
	router.Use(LoggerCom(func(format string, args ...any) { linhas = append(linhas, fmt.Sprintf(format, args...)) }))
	router.GET("/x", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/x?documento=52998224725", nil))

	require.Len(t, linhas, 1)
	require.Contains(t, linhas[0], "GET /x -> 200")
	require.NotContains(t, linhas[0], "52998224725")
}

func TestLimiteCorpo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LimiteCorpo(8))
	router.POST("/x", func(ctx *gin.Context) {
		var body map[string]any
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		ctx.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/x", bytesReader(`{"a":1}`)))
	require.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/x", bytesReader(`{"a":"grande demais"}`)))
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
}

func bytesReader(s string) *strings.Reader { return strings.NewReader(s) }
