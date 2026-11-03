package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

type osResponse struct {
	Uuid      string `json:"uuid"`
	Status    string `json:"status"`
	Aviso     string `json:"aviso"`
	Orcamento struct {
		Status        string `json:"status"`
		TotalCentavos int64  `json:"total_centavos"`
	} `json:"orcamento"`
	Servicos          []struct{ Uuid string } `json:"servicos"`
	Pecas             []struct{ Uuid string } `json:"pecas"`
	ReparosAdicionais []struct {
		Uuid   string `json:"uuid"`
		Status string `json:"status"`
	} `json:"reparos_adicionais"`
	Historico []struct {
		StatusNovo string `json:"status_novo"`
	} `json:"historico"`
	TempoExecucaoSegundos *int64 `json:"tempo_execucao_segundos"`
}

type cenario struct {
	admin, atendente, mecanico string
	documento                  string
	placa                      string
	servicoUuid                string
	servicoPreco               int64
	pecaUuid                   string
	pecaPreco                  int64
}

// prepararCenario cadastra cliente, veículo, serviço e peça novos para o teste.
func prepararCenario(t *testing.T, saldoPeca int32) cenario {
	c := cenario{
		admin:        login(t, emailAdmin, senhaAdmin),
		atendente:    login(t, emailAtendente, senhaAtendente),
		mecanico:     login(t, emailMecanico, senhaMecanico),
		documento:    util.RandomCPF(),
		placa:        util.RandomPlaca(),
		servicoPreco: 10000,
		pecaPreco:    2500,
	}

	rsp := call(t, http.MethodPost, "/clientes", c.atendente, map[string]any{
		"nome": "Cliente Integração", "documento": c.documento, "email": util.RandomEmail(),
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var cliente struct{ Uuid string }
	rsp.decode(t, &cliente)

	rsp = call(t, http.MethodPost, "/veiculos", c.atendente, map[string]any{
		"cliente_uuid": cliente.Uuid, "placa": c.placa, "marca": "Honda", "modelo": "Civic", "ano": 2020,
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))

	rsp = call(t, http.MethodPost, "/servicos", c.admin, map[string]any{
		"nome": "Serviço " + util.RandomString(8), "preco_centavos": c.servicoPreco, "tempo_estimado_minutos": 60,
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var servico struct{ Uuid string }
	rsp.decode(t, &servico)
	c.servicoUuid = servico.Uuid

	rsp = call(t, http.MethodPost, "/pecas", c.admin, map[string]any{
		"codigo": "INT-" + util.RandomString(8), "nome": "Peça integração", "tipo": "PECA",
		"preco_centavos": c.pecaPreco, "quantidade_estoque": saldoPeca, "estoque_minimo": 1,
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var peca struct{ Uuid string }
	rsp.decode(t, &peca)
	c.pecaUuid = peca.Uuid

	return c
}

func abrirOS(t *testing.T, c cenario, quantidadePeca int) osResponse {
	rsp := call(t, http.MethodPost, "/ordens-servico", c.atendente, map[string]any{
		"documento":          c.documento,
		"placa":              c.placa,
		"descricao_problema": "Barulho ao frear",
		"servicos":           []map[string]any{{"servico_uuid": c.servicoUuid}},
		"pecas":              []map[string]any{{"peca_uuid": c.pecaUuid, "quantidade": quantidadePeca}},
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))

	var os osResponse
	rsp.decode(t, &os)
	return os
}

func saldoPeca(t *testing.T, c cenario) int32 {
	rsp := call(t, http.MethodGet, "/pecas/"+c.pecaUuid, c.admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var peca struct {
		QuantidadeEstoque int32 `json:"quantidade_estoque"`
	}
	rsp.decode(t, &peca)
	return peca.QuantidadeEstoque
}

func TestFluxoCompletoOrdemServico(t *testing.T) {
	c := prepararCenario(t, 5)

	// Criação: OS nasce Recebida, com orçamento calculado automaticamente.
	os := abrirOS(t, c, 2)
	require.Equal(t, "RECEBIDA", os.Status)
	require.Equal(t, c.servicoPreco+2*c.pecaPreco, os.Orcamento.TotalCentavos)
	base := "/ordens-servico/" + os.Uuid

	// Transição fora da ordem é recusada.
	rsp := call(t, http.MethodPost, base+"/finalizar", c.mecanico, nil)
	require.Equal(t, http.StatusConflict, rsp.Code)

	// Atendente não pode iniciar diagnóstico (papel do mecânico).
	rsp = call(t, http.MethodPost, base+"/diagnostico", c.atendente, nil)
	require.Equal(t, http.StatusForbidden, rsp.Code)

	rsp = call(t, http.MethodPost, base+"/diagnostico", c.mecanico, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	rsp = call(t, http.MethodPost, base+"/orcamento/enviar", c.mecanico, map[string]string{"diagnostico": "Pastilhas gastas"})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "AGUARDANDO_APROVACAO", os.Status)
	require.Equal(t, "AGUARDANDO", os.Orcamento.Status)

	// Itens não mudam depois do envio do orçamento.
	rsp = call(t, http.MethodPost, base+"/servicos", c.atendente, map[string]any{"servico_uuid": c.servicoUuid})
	require.Equal(t, http.StatusConflict, rsp.Code)

	// Consulta pública: sem token, com o documento certo; documento errado = 404.
	rsp = call(t, http.MethodGet, base+"/andamento?documento="+c.documento, "", nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp = call(t, http.MethodGet, base+"/andamento?documento=52998224725", "", nil)
	require.Equal(t, http.StatusNotFound, rsp.Code)

	// Aprovação pelo cliente: sistema inicia a execução e baixa o estoque.
	rsp = call(t, http.MethodPost, base+"/orcamento/aprovar", "", map[string]string{"documento": c.documento})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "EM_EXECUCAO", os.Status)
	require.Equal(t, "APROVADO", os.Orcamento.Status)
	require.Equal(t, int32(3), saldoPeca(t, c))

	// Reparo adicional: só entra no orçamento depois da autorização.
	totalAntes := os.Orcamento.TotalCentavos
	rsp = call(t, http.MethodPost, base+"/reparos-adicionais", c.mecanico, map[string]any{
		"descricao": "Disco de freio empenado",
		"pecas":     []map[string]any{{"peca_uuid": c.pecaUuid, "quantidade": 1}},
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Len(t, os.ReparosAdicionais, 1)
	require.Equal(t, totalAntes, os.Orcamento.TotalCentavos)
	reparo := os.ReparosAdicionais[0].Uuid

	// Com reparo pendente a OS não finaliza.
	rsp = call(t, http.MethodPost, base+"/finalizar", c.mecanico, nil)
	require.Equal(t, http.StatusConflict, rsp.Code)

	rsp = call(t, http.MethodPost, fmt.Sprintf("%s/reparos-adicionais/%s/autorizar", base, reparo), "", map[string]string{"documento": c.documento})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, totalAntes+c.pecaPreco, os.Orcamento.TotalCentavos)
	require.Equal(t, "AUTORIZADO", os.ReparosAdicionais[0].Status)
	require.Equal(t, int32(2), saldoPeca(t, c))

	// Decidir de novo o mesmo reparo é recusado.
	rsp = call(t, http.MethodPost, fmt.Sprintf("%s/reparos-adicionais/%s/recusar", base, reparo), "", map[string]string{"documento": c.documento})
	require.Equal(t, http.StatusConflict, rsp.Code)

	rsp = call(t, http.MethodPost, base+"/finalizar", c.mecanico, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "FINALIZADA", os.Status)
	require.NotNil(t, os.TempoExecucaoSegundos)

	rsp = call(t, http.MethodPost, base+"/entregar", c.atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "ENTREGUE", os.Status)

	statusHistorico := make([]string, 0, len(os.Historico))
	for _, h := range os.Historico {
		statusHistorico = append(statusHistorico, h.StatusNovo)
	}
	require.Contains(t, statusHistorico, "EM_DIAGNOSTICO")
	require.Contains(t, statusHistorico, "EM_EXECUCAO")
	require.Contains(t, statusHistorico, "ENTREGUE")

	// Movimentos de estoque: saldo inicial + 2 baixas.
	rsp = call(t, http.MethodGet, "/pecas/"+c.pecaUuid+"/movimentos", c.admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var movimentos []map[string]any
	rsp.decode(t, &movimentos)
	require.Len(t, movimentos, 3)

	// Tempo médio de execução (somente administrador).
	rsp = call(t, http.MethodGet, "/ordens-servico/tempo-medio", c.atendente, nil)
	require.Equal(t, http.StatusForbidden, rsp.Code)
	rsp = call(t, http.MethodGet, "/ordens-servico/tempo-medio", c.admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	var tempo struct {
		TotalOrdens int64 `json:"total_ordens"`
	}
	rsp.decode(t, &tempo)
	require.GreaterOrEqual(t, tempo.TotalOrdens, int64(1))

	// Listagem administrativa filtrada por status.
	rsp = call(t, http.MethodGet, "/ordens-servico?status=ENTREGUE", c.atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
}

func TestFluxoOrcamentoRejeitado(t *testing.T) {
	c := prepararCenario(t, 5)
	os := abrirOS(t, c, 1)
	base := "/ordens-servico/" + os.Uuid

	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/diagnostico", c.mecanico, nil).Code)
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/orcamento/enviar", c.mecanico, nil).Code)

	// Antes da decisão do cliente o veículo não pode ser entregue.
	require.Equal(t, http.StatusConflict, call(t, http.MethodPost, base+"/entregar", c.atendente, nil).Code)

	rsp := call(t, http.MethodPost, base+"/orcamento/rejeitar", "", map[string]string{"documento": c.documento})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "EM_DIAGNOSTICO", os.Status)
	require.Equal(t, "REJEITADO", os.Orcamento.Status)

	// Rejeição não mexe no estoque.
	require.Equal(t, int32(5), saldoPeca(t, c))

	// O veículo é entregue sem execução.
	rsp = call(t, http.MethodPost, base+"/entregar", c.atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "ENTREGUE", os.Status)
}

func TestAprovacaoSemEstoqueAguardaReposicao(t *testing.T) {
	c := prepararCenario(t, 2)
	os := abrirOS(t, c, 2)
	base := "/ordens-servico/" + os.Uuid

	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/diagnostico", c.mecanico, nil).Code)
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/orcamento/enviar", c.mecanico, nil).Code)

	// O estoque cai depois da inclusão da peça na OS (não há reserva).
	rsp := call(t, http.MethodPost, "/pecas/"+c.pecaUuid+"/ajuste", c.admin, map[string]any{"quantidade": 1, "observacao": "Avaria"})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	rsp = call(t, http.MethodPost, base+"/orcamento/aprovar", "", map[string]string{"documento": c.documento})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "AGUARDANDO_APROVACAO", os.Status)
	require.Equal(t, "APROVADO", os.Orcamento.Status)
	require.NotEmpty(t, os.Aviso)
	require.Equal(t, int32(1), saldoPeca(t, c), "nada foi baixado")

	// Depois da reposição a execução começa.
	rsp = call(t, http.MethodPost, "/pecas/"+c.pecaUuid+"/reposicao", c.admin, map[string]any{"quantidade": 5})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	rsp = call(t, http.MethodPost, base+"/execucao", c.mecanico, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "EM_EXECUCAO", os.Status)
	require.Equal(t, int32(4), saldoPeca(t, c))
}

func TestValidacoesESeguranca(t *testing.T) {
	atendente := login(t, emailAtendente, senhaAtendente)

	// Sem token as rotas administrativas respondem 401.
	require.Equal(t, http.StatusUnauthorized, call(t, http.MethodGet, "/clientes", "", nil).Code)
	require.Equal(t, http.StatusUnauthorized, call(t, http.MethodGet, "/clientes", "token-invalido", nil).Code)

	// Login com senha errada.
	rsp := call(t, http.MethodPost, "/usuarios/login", "", map[string]string{"email": emailAdmin, "senha": "errada123"})
	require.Equal(t, http.StatusUnauthorized, rsp.Code)

	// CPF inválido e placa inválida.
	rsp = call(t, http.MethodPost, "/clientes", atendente, map[string]any{"nome": "X", "documento": "123.456.789-00", "email": util.RandomEmail()})
	require.Equal(t, http.StatusBadRequest, rsp.Code)
	require.Contains(t, string(rsp.Body), "DOCUMENTO_INVALIDO")

	rsp = call(t, http.MethodGet, "/clientes/documento/52998224725", atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code, "cliente do seed")
	var cliente struct{ Uuid string }
	rsp.decode(t, &cliente)

	rsp = call(t, http.MethodPost, "/veiculos", atendente, map[string]any{
		"cliente_uuid": cliente.Uuid, "placa": "AB-12345", "marca": "Fiat", "modelo": "Uno", "ano": 2010,
	})
	require.Equal(t, http.StatusBadRequest, rsp.Code)
	require.Contains(t, string(rsp.Body), "PLACA_INVALIDA")

	// Documento duplicado.
	rsp = call(t, http.MethodPost, "/clientes", atendente, map[string]any{"nome": "Duplicado", "documento": "529.982.247-25", "email": util.RandomEmail()})
	require.Equal(t, http.StatusConflict, rsp.Code)

	// Atendente não gerencia o estoque.
	rsp = call(t, http.MethodPost, "/pecas", atendente, map[string]any{"codigo": "X", "nome": "X", "tipo": "PECA"})
	require.Equal(t, http.StatusForbidden, rsp.Code)

	// Renovação do token com refresh token.
	rsp = call(t, http.MethodPost, "/usuarios/login", "", map[string]string{"email": emailAdmin, "senha": senhaAdmin})
	var sessao struct {
		RefreshToken string `json:"refresh_token"`
		SessionID    string `json:"session_id"`
		AccessToken  string `json:"access_token"`
	}
	rsp.decode(t, &sessao)
	rsp = call(t, http.MethodPost, "/tokens/refresh", "", map[string]string{"refresh_token": sessao.RefreshToken})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	// Depois do logout o refresh token deixa de valer.
	rsp = call(t, http.MethodPost, "/usuarios/logout", sessao.AccessToken, map[string]string{"session_id": sessao.SessionID})
	require.Equal(t, http.StatusNoContent, rsp.Code, string(rsp.Body))
	rsp = call(t, http.MethodPost, "/tokens/refresh", "", map[string]string{"refresh_token": sessao.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, rsp.Code)
}
