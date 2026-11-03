package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter limita requisições por IP de origem (token bucket). Protege o
// login, a renovação de token e as rotas públicas de acompanhamento contra
// tentativas em massa (adivinhação de senha ou de CPF/CNPJ).
type RateLimiter struct {
	rps   rate.Limit
	burst int
	ttl   time.Duration

	mu       sync.Mutex
	clientes map[string]*clienteLimitado
	now      func() time.Time
}

type clienteLimitado struct {
	limiter   *rate.Limiter
	ultimoUso time.Time
}

// NewRateLimiter cria um limitador com rps requisições por segundo e burst
// de rajada por IP. Entradas sem uso há mais de 3 minutos são descartadas na
// limpeza periódica.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rps:      rate.Limit(rps),
		burst:    burst,
		ttl:      3 * time.Minute,
		clientes: make(map[string]*clienteLimitado),
		now:      time.Now,
	}
	go rl.limparPeriodicamente(time.Minute)
	return rl
}

// Handler é o middleware Gin. Responde 429 com Retry-After quando o IP
// estoura o limite.
func (rl *RateLimiter) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !rl.permitir(ctx.ClientIP()) {
			ctx.Header("Retry-After", strconv.Itoa(rl.segundosParaProximo()))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "muitas tentativas, aguarde",
				"code":  "MUITAS_TENTATIVAS",
			})
			return
		}
		ctx.Next()
	}
}

func (rl *RateLimiter) permitir(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, ok := rl.clientes[ip]
	if !ok {
		c = &clienteLimitado{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.clientes[ip] = c
	}
	c.ultimoUso = rl.now()
	return c.limiter.Allow()
}

func (rl *RateLimiter) segundosParaProximo() int {
	if rl.rps <= 0 {
		return 60
	}
	segundos := int(1 / float64(rl.rps))
	if segundos < 1 {
		segundos = 1
	}
	return segundos
}

func (rl *RateLimiter) limparPeriodicamente(intervalo time.Duration) {
	ticker := time.NewTicker(intervalo)
	defer ticker.Stop()
	for range ticker.C {
		rl.limpar()
	}
}

// limpar remove os IPs sem uso há mais de ttl, para o mapa não crescer sem limite.
func (rl *RateLimiter) limpar() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	limite := rl.now().Add(-rl.ttl)
	for ip, c := range rl.clientes {
		if c.ultimoUso.Before(limite) {
			delete(rl.clientes, ip)
		}
	}
}
