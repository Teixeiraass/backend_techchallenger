package integration

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Os eventos de domínio são publicados depois do commit: quando o handler
// roda, a OS já está gravada com o novo status. E a falha de um handler não
// desfaz a operação nem muda a resposta da API.
func TestEventosSaemDepoisDoCommitEFalhaNaoDesfaz(t *testing.T) {
	c := prepararCenario(t, 5)
	os := abrirOS(t, c, 1)
	osUUID := uuid.MustParse(os.Uuid)

	var mu sync.Mutex
	var recebidos []string
	var statusNoBanco string
	testDespachante.Assinar(ordemservico.NomeOrcamentoEnviadoAoCliente, func(ctx context.Context, e domain.Evento) error {
		ev := e.(ordemservico.OrcamentoEnviadoAoCliente)
		if ev.OSUUID != osUUID {
			return nil
		}
		// Lê o banco fora da transação do comando: o commit já aconteceu.
		row, err := testStore.GetOrdemServico(ctx, osUUID)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		recebidos = append(recebidos, e.Nome())
		statusNoBanco = row.Status
		return errors.New("handler falhou de propósito")
	})
	testDespachante.Assinar(ordemservico.NomeOSFinalizada, func(_ context.Context, e domain.Evento) error {
		if e.(ordemservico.OSFinalizada).OSUUID != osUUID {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		recebidos = append(recebidos, e.Nome())
		return nil
	})
	testDespachante.Assinar(estoque.NomeEstoqueBaixado, func(_ context.Context, e domain.Evento) error {
		if e.(estoque.EstoqueBaixado).OrdemServicoID == 0 {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		recebidos = append(recebidos, e.Nome())
		return nil
	})

	base := "/ordens-servico/" + os.Uuid
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/diagnostico", c.mecanico, nil).Code)

	rsp := call(t, http.MethodPost, base+"/orcamento/enviar", c.mecanico, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body), "a falha do handler não muda a resposta")
	rsp.decode(t, &os)
	require.Equal(t, "AGUARDANDO_APROVACAO", os.Status)

	mu.Lock()
	require.Equal(t, []string{ordemservico.NomeOrcamentoEnviadoAoCliente}, recebidos)
	require.Equal(t, "AGUARDANDO_APROVACAO", statusNoBanco, "o handler viu a OS já gravada")
	mu.Unlock()

	rsp = call(t, http.MethodPost, base+"/orcamento/aprovar", "", map[string]string{"documento": c.documento})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp = call(t, http.MethodPost, base+"/finalizar", c.mecanico, nil)
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))

	mu.Lock()
	defer mu.Unlock()
	require.Contains(t, recebidos, estoque.NomeEstoqueBaixado)
	require.Contains(t, recebidos, ordemservico.NomeOSFinalizada)
}

// Sem saldo na baixa a transação é desfeita, mas "Estoque insuficiente
// detectado" chega à política de reposição.
func TestEstoqueInsuficienteEhPublicadoMesmoComRollback(t *testing.T) {
	c := prepararCenario(t, 1)
	os := abrirOS(t, c, 1)
	base := "/ordens-servico/" + os.Uuid

	var mu sync.Mutex
	var faltas []estoque.EstoqueInsuficienteDetectado
	testDespachante.Assinar(estoque.NomeEstoqueInsuficienteDetectado, func(_ context.Context, e domain.Evento) error {
		ev := e.(estoque.EstoqueInsuficienteDetectado)
		mu.Lock()
		defer mu.Unlock()
		faltas = append(faltas, ev)
		return nil
	})

	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/diagnostico", c.mecanico, nil).Code)
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/orcamento/enviar", c.mecanico, nil).Code)
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, "/pecas/"+c.pecaUuid+"/ajuste", c.admin, map[string]any{"quantidade": 0, "observacao": "Avaria"}).Code)

	rsp := call(t, http.MethodPost, base+"/orcamento/aprovar", "", map[string]string{"documento": c.documento})
	require.Equal(t, http.StatusOK, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)
	require.Equal(t, "AGUARDANDO_APROVACAO", os.Status)
	require.NotEmpty(t, os.Aviso)
	require.Equal(t, int32(0), saldoPeca(t, c), "nada foi baixado")

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, faltas)
	require.Equal(t, int32(1), faltas[len(faltas)-1].Necessario)
}

// O reparo adicional identificado deixa uma linha no histórico com o nome
// do evento, sem mudança de status.
func TestReparoIdentificadoEntraNoHistorico(t *testing.T) {
	c := prepararCenario(t, 5)
	os := abrirOS(t, c, 1)
	base := "/ordens-servico/" + os.Uuid
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/diagnostico", c.mecanico, nil).Code)
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/orcamento/enviar", c.mecanico, nil).Code)
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, base+"/orcamento/aprovar", "", map[string]string{"documento": c.documento}).Code)

	rsp := call(t, http.MethodPost, base+"/reparos-adicionais", c.mecanico, map[string]any{
		"descricao": "Disco de freio empenado",
		"pecas":     []map[string]any{{"peca_uuid": c.pecaUuid, "quantidade": 1}},
	})
	require.Equal(t, http.StatusCreated, rsp.Code, string(rsp.Body))
	rsp.decode(t, &os)

	encontrou := false
	for _, h := range os.Historico {
		if h.Observacao == ordemservico.NomeReparoAdicionalIdentificado+": Disco de freio empenado" {
			encontrou = true
			require.Equal(t, "EM_EXECUCAO", h.StatusNovo)
		}
	}
	require.True(t, encontrou, "linha do evento no histórico")
}
