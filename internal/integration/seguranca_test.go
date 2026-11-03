package integration

import (
	"bytes"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type sessaoResponse struct {
	SessionID    string `json:"session_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func abrirSessao(t *testing.T) sessaoResponse {
	t.Helper()
	rsp := call(t, http.MethodPost, "/usuarios/login", "", map[string]string{"email": emailAdmin, "senha": senhaAdmin})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	var sessao sessaoResponse
	rsp.decode(t, &sessao)
	return sessao
}

// O refresh token só renova a sessão: nunca autentica uma rota protegida, e
// o access token nunca renova. Depois do logout nenhum dos dois vale.
func TestTiposDeTokenNaoSeMisturam(t *testing.T) {
	sessao := abrirSessao(t)

	// Access token funciona nas rotas protegidas; refresh token não.
	require.Equal(t, http.StatusOK, call(t, http.MethodGet, "/clientes", sessao.AccessToken, nil).Code)
	rsp := call(t, http.MethodGet, "/clientes", sessao.RefreshToken, nil)
	require.Equal(t, http.StatusUnauthorized, rsp.Code, string(rsp.Body))
	require.Contains(t, string(rsp.Body), "TOKEN_TIPO_INVALIDO")

	// Access token não renova a sessão.
	rsp = call(t, http.MethodPost, "/tokens/refresh", "", map[string]string{"refresh_token": sessao.AccessToken})
	require.Equal(t, http.StatusUnauthorized, rsp.Code, string(rsp.Body))
	require.Contains(t, string(rsp.Body), "SESSAO_INVALIDA")

	// Refresh token renova.
	rsp = call(t, http.MethodPost, "/tokens/refresh", "", map[string]string{"refresh_token": sessao.RefreshToken})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	// Depois do logout o refresh token não renova nem autentica.
	rsp = call(t, http.MethodPost, "/usuarios/logout", sessao.AccessToken, map[string]string{"session_id": sessao.SessionID})
	require.Equal(t, http.StatusNoContent, rsp.Code, string(rsp.Body))
	rsp = call(t, http.MethodPost, "/tokens/refresh", "", map[string]string{"refresh_token": sessao.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, rsp.Code, string(rsp.Body))
	require.Equal(t, http.StatusUnauthorized, call(t, http.MethodGet, "/clientes", sessao.RefreshToken, nil).Code)
}

func TestMensagensDoMiddlewareEmPortugues(t *testing.T) {
	rsp := call(t, http.MethodGet, "/clientes", "", nil)
	require.Equal(t, http.StatusUnauthorized, rsp.Code)
	require.Contains(t, string(rsp.Body), "TOKEN_AUSENTE")

	rsp = call(t, http.MethodGet, "/clientes", "lixo", nil)
	require.Equal(t, http.StatusUnauthorized, rsp.Code)
	require.Contains(t, string(rsp.Body), "TOKEN_INVALIDO")
}

func TestLimiteDeTentativas(t *testing.T) {
	credenciais := map[string]string{"email": emailAdmin, "senha": "errada123"}
	var ultimo apiResponse
	for i := 0; i < 3; i++ {
		ultimo = callCom(t, testRouterLimitado, http.MethodPost, "/usuarios/login", "", credenciais, nil)
		require.Equal(t, http.StatusUnauthorized, ultimo.Code, "tentativa %d", i+1)
	}
	ultimo = callCom(t, testRouterLimitado, http.MethodPost, "/usuarios/login", "", credenciais, nil)
	require.Equal(t, http.StatusTooManyRequests, ultimo.Code, string(ultimo.Body))
	require.Contains(t, string(ultimo.Body), "MUITAS_TENTATIVAS")

	// A consulta pública entra no mesmo limite (mesmo IP).
	rsp := callCom(t, testRouterLimitado, http.MethodGet, "/ordens-servico/00000000-0000-0000-0000-000000000000/andamento", "", nil, map[string]string{"X-Documento": "52998224725"})
	require.Equal(t, http.StatusTooManyRequests, rsp.Code)
}

func TestSenhaCurtaNaoCriaUsuario(t *testing.T) {
	admin := login(t, emailAdmin, senhaAdmin)
	rsp := call(t, http.MethodPost, "/usuarios", admin, map[string]any{"nome": "Curta", "email": "curta@oficina.com", "senha": "1234567", "papel": "MECANICO"})
	require.Equal(t, http.StatusBadRequest, rsp.Code, string(rsp.Body))
	require.Contains(t, string(rsp.Body), `"campo":"senha"`)
	require.Contains(t, string(rsp.Body), `"regra":"min"`)
	require.NotContains(t, string(rsp.Body), "CriarUsuarioRequest")
}

// O log de acesso não pode conter o CPF/CNPJ, nem quando ele vem pela
// query string obsoleta.
func TestDocumentoNaoVaiParaOLog(t *testing.T) {
	c := prepararCenario(t, 1)
	os := abrirOS(t, c, 1)

	var saida bytes.Buffer
	anterior := log.Writer()
	log.SetOutput(&saida)
	defer log.SetOutput(anterior)

	require.Equal(t, http.StatusOK, callDocumento(t, http.MethodGet, "/ordens-servico/"+os.Uuid+"/andamento", "", c.documento, nil).Code)
	require.Equal(t, http.StatusOK, call(t, http.MethodGet, "/ordens-servico/"+os.Uuid+"/andamento?documento="+c.documento, "", nil).Code)
	require.Equal(t, http.StatusOK, callDocumento(t, http.MethodGet, "/clientes/por-documento", c.atendente, c.documento, nil).Code)

	require.Contains(t, saida.String(), "/andamento -> 200")
	require.NotContains(t, saida.String(), c.documento)
}

func TestCorpoGrandeDemais(t *testing.T) {
	atendente := login(t, emailAtendente, senhaAtendente)
	grande := map[string]any{"nome": string(bytes.Repeat([]byte("a"), 2<<20)), "documento": "52998224725", "email": "x@x.com"}
	rsp := call(t, http.MethodPost, "/clientes", atendente, grande)
	require.Equal(t, http.StatusRequestEntityTooLarge, rsp.Code, string(rsp.Body))
	require.Contains(t, string(rsp.Body), "CORPO_GRANDE_DEMAIS")
}
