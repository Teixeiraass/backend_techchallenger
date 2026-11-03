package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

// Gestão administrativa do enunciado: CRUD de clientes, veículos, serviços e
// peças (com controle de estoque), listagem e detalhamento das OS e gestão de
// usuários. Os fluxos da OS estão em fluxo_os_test.go.

type clienteResponse struct {
	Uuid               string `json:"uuid"`
	Nome               string `json:"nome"`
	Documento          string `json:"documento"`
	DocumentoFormatado string `json:"documento_formatado"`
	TipoDocumento      string `json:"tipo_documento"`
	Email              string `json:"email"`
}

type veiculoResponse struct {
	Uuid        string `json:"uuid"`
	ClienteUuid string `json:"cliente_uuid"`
	Placa       string `json:"placa"`
	Modelo      string `json:"modelo"`
	Ano         int32  `json:"ano"`
}

type servicoResponse struct {
	Uuid          string `json:"uuid"`
	PrecoCentavos int64  `json:"preco_centavos"`
	Ativo         bool   `json:"ativo"`
}

type pecaResponse struct {
	Uuid              string `json:"uuid"`
	Nome              string `json:"nome"`
	QuantidadeEstoque int32  `json:"quantidade_estoque"`
	EstoqueMinimo     int32  `json:"estoque_minimo"`
	EstoqueBaixo      bool   `json:"estoque_baixo"`
	Ativo             bool   `json:"ativo"`
}

type itemOSResponse struct {
	Uuid string `json:"uuid"`
}

type osDetalheResponse struct {
	Uuid      string `json:"uuid"`
	Status    string `json:"status"`
	Orcamento struct {
		TotalCentavos int64 `json:"total_centavos"`
	} `json:"orcamento"`
	Servicos []itemOSResponse `json:"servicos"`
	Pecas    []itemOSResponse `json:"pecas"`
}

func contemUuid[T any](itens []T, uuid string, pegar func(T) string) bool {
	for _, item := range itens {
		if pegar(item) == uuid {
			return true
		}
	}
	return false
}

func criarServico(t *testing.T, admin string, preco int64) servicoResponse {
	t.Helper()
	rsp := call(t, http.MethodPost, "/servicos", admin, map[string]any{
		"nome": "Serviço " + util.RandomString(8), "preco_centavos": preco, "tempo_estimado_minutos": 30,
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var servico servicoResponse
	rsp.decode(t, &servico)
	return servico
}

func criarPeca(t *testing.T, admin string, saldo, minimo int32) pecaResponse {
	t.Helper()
	rsp := call(t, http.MethodPost, "/pecas", admin, map[string]any{
		"codigo": "INT-" + util.RandomString(8), "nome": "Peça " + util.RandomString(6), "tipo": "PECA",
		"preco_centavos": 1500, "quantidade_estoque": saldo, "estoque_minimo": minimo,
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var peca pecaResponse
	rsp.decode(t, &peca)
	return peca
}

func TestCRUDClienteEVeiculo(t *testing.T) {
	atendente := login(t, emailAtendente, senhaAtendente)
	mecanico := login(t, emailMecanico, senhaMecanico)
	nome := "Cliente CRUD " + util.RandomString(8)

	// Criar: CNPJ também é aceito e sai formatado.
	rsp := call(t, http.MethodPost, "/clientes", atendente, map[string]any{
		"nome": nome, "documento": util.RandomCNPJ(), "email": util.RandomEmail(), "telefone": "11999990000",
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var cliente clienteResponse
	rsp.decode(t, &cliente)
	require.Equal(t, "CNPJ", cliente.TipoDocumento)
	require.Len(t, cliente.DocumentoFormatado, len("00.000.000/0000-00"))

	// Mecânico não cadastra cliente.
	rsp = call(t, http.MethodPost, "/clientes", mecanico, map[string]any{
		"nome": "X", "documento": util.RandomCPF(), "email": util.RandomEmail(),
	})
	require.Equal(t, http.StatusForbidden, rsp.Code)

	// Buscar por UUID, por documento e na listagem com busca.
	rsp = call(t, http.MethodGet, "/clientes/"+cliente.Uuid, atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)

	rsp = callDocumento(t, http.MethodGet, "/clientes/por-documento", atendente, cliente.Documento, nil)
	require.Equal(t, http.StatusOK, rsp.Code)

	rsp = call(t, http.MethodGet, "/clientes?busca="+url.QueryEscape(nome), atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var clientes []clienteResponse
	rsp.decode(t, &clientes)
	require.True(t, contemUuid(clientes, cliente.Uuid, func(c clienteResponse) string { return c.Uuid }))

	// Atualizar.
	novoEmail := util.RandomEmail()
	rsp = call(t, http.MethodPut, "/clientes/"+cliente.Uuid, atendente, map[string]any{
		"nome": nome + " Atualizado", "email": novoEmail, "telefone": "1133330000",
	})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &cliente)
	require.Equal(t, novoEmail, cliente.Email)

	// Veículo do cliente: criar, buscar, listar por cliente e atualizar.
	rsp = call(t, http.MethodPost, "/veiculos", atendente, map[string]any{
		"cliente_uuid": cliente.Uuid, "placa": util.RandomPlaca(), "marca": "Fiat", "modelo": "Uno", "ano": 2015,
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var veiculo veiculoResponse
	rsp.decode(t, &veiculo)
	require.Equal(t, cliente.Uuid, veiculo.ClienteUuid)

	rsp = call(t, http.MethodGet, "/veiculos/"+veiculo.Uuid, atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)

	rsp = call(t, http.MethodGet, "/veiculos?cliente_uuid="+cliente.Uuid, atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var veiculos []veiculoResponse
	rsp.decode(t, &veiculos)
	require.Len(t, veiculos, 1)

	novaPlaca := util.RandomPlaca()
	rsp = call(t, http.MethodPut, "/veiculos/"+veiculo.Uuid, atendente, map[string]any{
		"placa": novaPlaca, "marca": "Fiat", "modelo": "Mobi", "ano": 2022,
	})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &veiculo)
	require.Equal(t, "Mobi", veiculo.Modelo)
	require.Equal(t, int32(2022), veiculo.Ano)

	// Placa inválida na atualização.
	rsp = call(t, http.MethodPut, "/veiculos/"+veiculo.Uuid, atendente, map[string]any{
		"placa": "12-ABCD", "marca": "Fiat", "modelo": "Mobi", "ano": 2022,
	})
	require.Equal(t, http.StatusBadRequest, rsp.Code)
	require.Contains(t, string(rsp.Body), "PLACA_INVALIDA")

	// Cliente com veículo não pode ser removido.
	rsp = call(t, http.MethodDelete, "/clientes/"+cliente.Uuid, atendente, nil)
	require.Equal(t, http.StatusConflict, rsp.Code)
	require.Contains(t, string(rsp.Body), "CLIENTE_EM_USO")

	// Remover o veículo e depois o cliente.
	rsp = call(t, http.MethodDelete, "/veiculos/"+veiculo.Uuid, atendente, nil)
	require.Equal(t, http.StatusNoContent, rsp.Code)
	rsp = call(t, http.MethodGet, "/veiculos/"+veiculo.Uuid, atendente, nil)
	require.Equal(t, http.StatusNotFound, rsp.Code)

	rsp = call(t, http.MethodDelete, "/clientes/"+cliente.Uuid, atendente, nil)
	require.Equal(t, http.StatusNoContent, rsp.Code)
	rsp = call(t, http.MethodGet, "/clientes/"+cliente.Uuid, atendente, nil)
	require.Equal(t, http.StatusNotFound, rsp.Code)
}

func TestCRUDServico(t *testing.T) {
	admin := login(t, emailAdmin, senhaAdmin)
	atendente := login(t, emailAtendente, senhaAtendente)

	servico := criarServico(t, admin, 8000)

	// Só o administrador mantém o catálogo.
	rsp := call(t, http.MethodPost, "/servicos", atendente, map[string]any{"nome": "X", "preco_centavos": 1})
	require.Equal(t, http.StatusForbidden, rsp.Code)

	rsp = call(t, http.MethodGet, "/servicos/"+servico.Uuid, atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)

	rsp = call(t, http.MethodPut, "/servicos/"+servico.Uuid, admin, map[string]any{
		"nome": "Alinhamento " + util.RandomString(6), "preco_centavos": 9500, "tempo_estimado_minutos": 45, "ativo": true,
	})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &servico)
	require.Equal(t, int64(9500), servico.PrecoCentavos)

	// Inativar tira o serviço da listagem de ativos.
	rsp = call(t, http.MethodDelete, "/servicos/"+servico.Uuid, admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	rsp.decode(t, &servico)
	require.False(t, servico.Ativo)

	rsp = call(t, http.MethodGet, "/servicos?apenas_ativos=true&tamanho=100", atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var ativos []servicoResponse
	rsp.decode(t, &ativos)
	require.False(t, contemUuid(ativos, servico.Uuid, func(s servicoResponse) string { return s.Uuid }))
}

func TestCRUDPecaEControleDeEstoque(t *testing.T) {
	admin := login(t, emailAdmin, senhaAdmin)
	atendente := login(t, emailAtendente, senhaAtendente)

	peca := criarPeca(t, admin, 10, 2)
	require.False(t, peca.EstoqueBaixo)

	// Só o administrador mexe no estoque.
	rsp := call(t, http.MethodPost, "/pecas/"+peca.Uuid+"/reposicao", atendente, map[string]any{"quantidade": 1})
	require.Equal(t, http.StatusForbidden, rsp.Code)

	// Atualizar: estoque mínimo acima do saldo marca estoque baixo.
	rsp = call(t, http.MethodPut, "/pecas/"+peca.Uuid, admin, map[string]any{
		"codigo": "INT-" + util.RandomString(8), "nome": peca.Nome, "tipo": "PECA",
		"preco_centavos": 1800, "estoque_minimo": 12, "ativo": true,
	})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &peca)
	require.True(t, peca.EstoqueBaixo)

	rsp = call(t, http.MethodGet, "/pecas/estoque-baixo", admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var baixas []pecaResponse
	rsp.decode(t, &baixas)
	require.True(t, contemUuid(baixas, peca.Uuid, func(p pecaResponse) string { return p.Uuid }))

	// Reposição soma ao saldo; ajuste define o saldo contado.
	rsp = call(t, http.MethodPost, "/pecas/"+peca.Uuid+"/reposicao", admin, map[string]any{"quantidade": 5})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &peca)
	require.Equal(t, int32(15), peca.QuantidadeEstoque)
	require.False(t, peca.EstoqueBaixo)

	rsp = call(t, http.MethodPost, "/pecas/"+peca.Uuid+"/ajuste", admin, map[string]any{"quantidade": 7, "observacao": "Inventário"})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &peca)
	require.Equal(t, int32(7), peca.QuantidadeEstoque)

	// Saldo negativo é recusado.
	rsp = call(t, http.MethodPost, "/pecas/"+peca.Uuid+"/ajuste", admin, map[string]any{"quantidade": -1, "observacao": "x"})
	require.Equal(t, http.StatusBadRequest, rsp.Code)

	// Cada movimento fica registrado com saldo anterior e posterior.
	rsp = call(t, http.MethodGet, "/pecas/"+peca.Uuid+"/movimentos", admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var movimentos []struct {
		Tipo           string `json:"tipo"`
		SaldoAnterior  int32  `json:"saldo_anterior"`
		SaldoPosterior int32  `json:"saldo_posterior"`
	}
	rsp.decode(t, &movimentos)
	registrados := make([]string, 0, len(movimentos))
	for _, m := range movimentos {
		registrados = append(registrados, fmt.Sprintf("%s %d->%d", m.Tipo, m.SaldoAnterior, m.SaldoPosterior))
	}
	require.Contains(t, registrados, "ENTRADA 10->15")
	require.Contains(t, registrados, "AJUSTE 15->7")

	// Inativar tira a peça da listagem de ativas.
	rsp = call(t, http.MethodDelete, "/pecas/"+peca.Uuid, admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	rsp.decode(t, &peca)
	require.False(t, peca.Ativo)

	rsp = call(t, http.MethodGet, "/pecas?apenas_ativas=true&tamanho=100", atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var ativas []pecaResponse
	rsp.decode(t, &ativas)
	require.False(t, contemUuid(ativas, peca.Uuid, func(p pecaResponse) string { return p.Uuid }))
}

func TestDetalheEItensDaOrdemServico(t *testing.T) {
	c := prepararCenario(t, 5)
	os := abrirOS(t, c, 1)
	base := "/ordens-servico/" + os.Uuid

	// Detalhamento.
	rsp := call(t, http.MethodGet, base, c.atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var detalhe osDetalheResponse
	rsp.decode(t, &detalhe)
	require.Equal(t, "RECEBIDA", detalhe.Status)
	require.Len(t, detalhe.Servicos, 1)
	require.Len(t, detalhe.Pecas, 1)
	require.Equal(t, c.servicoPreco+c.pecaPreco, detalhe.Orcamento.TotalCentavos)

	// Incluir um serviço recalcula o orçamento; remover volta ao valor anterior.
	extra := criarServico(t, c.admin, 3000)
	rsp = call(t, http.MethodPost, base+"/servicos", c.atendente, map[string]any{"servico_uuid": extra.Uuid})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &detalhe)
	require.Len(t, detalhe.Servicos, 2)
	require.Equal(t, c.servicoPreco+c.pecaPreco+3000, detalhe.Orcamento.TotalCentavos)

	itemExtra := detalhe.Servicos[0].Uuid
	if detalhe.Servicos[1].Uuid != os.Servicos[0].Uuid {
		itemExtra = detalhe.Servicos[1].Uuid
	}
	rsp = call(t, http.MethodDelete, base+"/servicos/"+itemExtra, c.atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &detalhe)
	require.Len(t, detalhe.Servicos, 1)
	require.Equal(t, c.servicoPreco+c.pecaPreco, detalhe.Orcamento.TotalCentavos)

	rsp = call(t, http.MethodDelete, base+"/pecas/"+detalhe.Pecas[0].Uuid, c.mecanico, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &detalhe)
	require.Empty(t, detalhe.Pecas)
	require.Equal(t, c.servicoPreco, detalhe.Orcamento.TotalCentavos)

	// Serviço e peça inativos não entram na OS.
	rsp = call(t, http.MethodDelete, "/servicos/"+extra.Uuid, c.admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	rsp = call(t, http.MethodPost, base+"/servicos", c.atendente, map[string]any{"servico_uuid": extra.Uuid})
	require.Equal(t, http.StatusConflict, rsp.Code)
	require.Contains(t, string(rsp.Body), "SERVICO_INATIVO")

	inativa := criarPeca(t, c.admin, 5, 1)
	rsp = call(t, http.MethodDelete, "/pecas/"+inativa.Uuid, c.admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	rsp = call(t, http.MethodPost, base+"/pecas", c.atendente, map[string]any{"peca_uuid": inativa.Uuid, "quantidade": 1})
	require.Equal(t, http.StatusConflict, rsp.Code)
	require.Contains(t, string(rsp.Body), "PECA_INATIVA")

	// Listagem filtrada por cliente e por status.
	rsp = callDocumento(t, http.MethodGet, "/clientes/por-documento", c.atendente, c.documento, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var cliente clienteResponse
	rsp.decode(t, &cliente)

	rsp = call(t, http.MethodGet, "/ordens-servico?cliente_uuid="+cliente.Uuid, c.atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var lista []struct {
		Uuid               string `json:"uuid"`
		Status             string `json:"status"`
		ValorTotalCentavos int64  `json:"valor_total_centavos"`
		VeiculoUuid        string `json:"veiculo_uuid"`
	}
	rsp.decode(t, &lista)
	require.Len(t, lista, 1)
	require.Equal(t, os.Uuid, lista[0].Uuid)
	require.Equal(t, c.servicoPreco, lista[0].ValorTotalCentavos)

	rsp = call(t, http.MethodGet, "/ordens-servico?status=RECEBIDA&tamanho=100", c.atendente, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	var recebidas []struct {
		Uuid   string `json:"uuid"`
		Status string `json:"status"`
	}
	rsp.decode(t, &recebidas)
	for _, item := range recebidas {
		require.Equal(t, "RECEBIDA", item.Status)
	}

	rsp = call(t, http.MethodGet, "/ordens-servico?status=QUALQUER", c.atendente, nil)
	require.Equal(t, http.StatusBadRequest, rsp.Code)

	// Cliente e veículo com OS não podem ser removidos.
	rsp = call(t, http.MethodDelete, "/veiculos/"+lista[0].VeiculoUuid, c.atendente, nil)
	require.Equal(t, http.StatusConflict, rsp.Code)
	require.Contains(t, string(rsp.Body), "VEICULO_EM_USO")

	rsp = call(t, http.MethodDelete, "/clientes/"+cliente.Uuid, c.atendente, nil)
	require.Equal(t, http.StatusConflict, rsp.Code)
	require.Contains(t, string(rsp.Body), "CLIENTE_EM_USO")

	// OS inexistente.
	rsp = call(t, http.MethodGet, "/ordens-servico/"+util.RandomUuid().String(), c.atendente, nil)
	require.Equal(t, http.StatusNotFound, rsp.Code)
}

func TestGestaoDeUsuarios(t *testing.T) {
	admin := login(t, emailAdmin, senhaAdmin)
	atendente := login(t, emailAtendente, senhaAtendente)
	email := fmt.Sprintf("mecanico.%s@oficina.com", util.RandomString(8))

	// Só o administrador cria usuários.
	novo := map[string]any{"nome": "Mecânico Novo", "email": email, "senha": "segredo1", "papel": "MECANICO"}
	rsp := call(t, http.MethodPost, "/usuarios", atendente, novo)
	require.Equal(t, http.StatusForbidden, rsp.Code)

	rsp = call(t, http.MethodPost, "/usuarios", admin, novo)
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	var usuario struct {
		Uuid  string `json:"uuid"`
		Email string `json:"email"`
		Papel string `json:"papel"`
	}
	rsp.decode(t, &usuario)
	require.Equal(t, "MECANICO", usuario.Papel)

	// E-mail repetido é recusado.
	rsp = call(t, http.MethodPost, "/usuarios", admin, novo)
	require.Equal(t, http.StatusConflict, rsp.Code)

	// O novo usuário entra e vê os próprios dados.
	token := login(t, email, "segredo1")
	rsp = call(t, http.MethodGet, "/usuarios/me", token, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	require.Contains(t, string(rsp.Body), email)

	rsp = call(t, http.MethodGet, "/usuarios?tamanho=100", admin, nil)
	require.Equal(t, http.StatusOK, rsp.Code)
	require.Contains(t, string(rsp.Body), usuario.Uuid)

	// Desativado perde o acesso na hora, mesmo com token válido, e não entra de novo.
	rsp = call(t, http.MethodPatch, "/usuarios/"+usuario.Uuid+"/ativo", admin, map[string]any{"ativo": false})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	rsp = call(t, http.MethodGet, "/usuarios/me", token, nil)
	require.Equal(t, http.StatusForbidden, rsp.Code)
	require.Contains(t, string(rsp.Body), "USUARIO_INATIVO")

	rsp = call(t, http.MethodPost, "/usuarios/login", "", map[string]string{"email": email, "senha": "segredo1"})
	require.Equal(t, http.StatusForbidden, rsp.Code)

	// Reativado volta a entrar.
	rsp = call(t, http.MethodPatch, "/usuarios/"+usuario.Uuid+"/ativo", admin, map[string]any{"ativo": true})
	require.Equal(t, http.StatusOK, rsp.Code)
	login(t, email, "segredo1")
}
