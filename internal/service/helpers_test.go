package service_test

import (
	"context"
	"database/sql"
	"time"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// fxSender records notifications and can simulate a delivery failure.
type fxSender struct {
	err  error
	msgs []notification.Message
}

func (s *fxSender) Send(_ context.Context, msg notification.Message) error {
	s.msgs = append(s.msgs, msg)
	return s.err
}

func fxConfig() util.Config {
	return util.Config{AdminNotificationTo: "admin@oficina.com"}
}

func fxCliente() db.Cliente {
	return db.Cliente{
		ID:            util.RandomInt(1, 1000),
		Uuid:          uuid.New(),
		Nome:          util.RandomName(),
		Documento:     util.RandomCPF(),
		TipoDocumento: "CPF",
		Email:         util.RandomEmail(),
	}
}

func fxVeiculo(clienteID int64) db.Veiculo {
	return db.Veiculo{
		ID:        util.RandomInt(1, 1000),
		Uuid:      uuid.New(),
		ClienteID: clienteID,
		Placa:     util.RandomPlaca(),
		Marca:     "Fiat",
		Modelo:    "Uno",
		Ano:       2020,
	}
}

func fxServico(ativo bool) db.Servico {
	return db.Servico{
		ID:                   util.RandomInt(1, 1000),
		Uuid:                 uuid.New(),
		Nome:                 util.RandomName(),
		PrecoCentavos:        10000,
		TempoEstimadoMinutos: 30,
		Ativo:                ativo,
	}
}

func fxPeca(saldo int32, minimo int32) db.Peca {
	return db.Peca{
		ID:                util.RandomInt(1, 1000),
		Uuid:              uuid.New(),
		Codigo:            "COD-" + util.RandomString(4),
		Nome:              util.RandomName(),
		Tipo:              "PECA",
		Unidade:           "UN",
		PrecoCentavos:     2500,
		QuantidadeEstoque: saldo,
		EstoqueMinimo:     minimo,
		Ativo:             true,
	}
}

func fxOrdemServico(c db.Cliente, v db.Veiculo, status ordemservico.Status, orcamento ordemservico.StatusOrcamento) db.OrdemServico {
	return db.OrdemServico{
		ID:              util.RandomInt(1, 1000),
		Uuid:            uuid.New(),
		ClienteID:       c.ID,
		VeiculoID:       v.ID,
		Status:          string(status),
		OrcamentoStatus: string(orcamento),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func fxItemServico(osID int64, s db.Servico, quantidade int32) db.ListItensServicoByOrdemServicoRow {
	return db.ListItensServicoByOrdemServicoRow{
		ID:                    util.RandomInt(1, 1000),
		Uuid:                  uuid.New(),
		OrdemServicoID:        osID,
		Quantidade:            quantidade,
		ValorUnitarioCentavos: s.PrecoCentavos,
		ServicoUuid:           s.Uuid,
		ServicoNome:           s.Nome,
	}
}

func fxItemPeca(osID int64, p db.Peca, quantidade int32) db.ListItensPecaByOrdemServicoRow {
	return db.ListItensPecaByOrdemServicoRow{
		ID:                    util.RandomInt(1, 1000),
		Uuid:                  uuid.New(),
		OrdemServicoID:        osID,
		PecaID:                p.ID,
		Quantidade:            quantidade,
		ValorUnitarioCentavos: p.PrecoCentavos,
		PecaUuid:              p.Uuid,
		PecaCodigo:            p.Codigo,
		PecaNome:              p.Nome,
		PecaTipo:              p.Tipo,
	}
}

// fxDoReparo marks an item row as belonging to an additional repair.
func fxDoReparo(item db.ListItensPecaByOrdemServicoRow, reparo db.ReparoAdicional) db.ListItensPecaByOrdemServicoRow {
	item.ReparoAdicionalUuid = uuid.NullUUID{UUID: reparo.Uuid, Valid: true}
	item.ReparoAdicionalStatus = sql.NullString{String: reparo.Status, Valid: true}
	return item
}

func fxReparo(osID int64, status ordemservico.StatusReparo) db.ReparoAdicional {
	return db.ReparoAdicional{
		ID:             util.RandomInt(1, 1000),
		Uuid:           uuid.New(),
		OrdemServicoID: osID,
		Descricao:      "Disco empenado",
		Status:         string(status),
	}
}

// fxExecTx makes the mock itself act as the transaction Querier.
func fxExecTx(store *mockdb.MockStore) {
	store.EXPECT().ExecTx(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, fn func(db.Querier) error) error {
			return fn(store)
		})
}

// fxLeitura is the read model returned by the queries used to render an OS
// (and reused inside transactions to load items, repairs and the owner).
type fxLeitura struct {
	os       db.OrdemServico
	cliente  db.Cliente
	veiculo  db.Veiculo
	servicos []db.ListItensServicoByOrdemServicoRow
	pecas    []db.ListItensPecaByOrdemServicoRow
	reparos  []db.ReparoAdicional
}

func fxStubLeitura(store *mockdb.MockStore, l fxLeitura) {
	servicos := l.servicos
	if servicos == nil {
		servicos = []db.ListItensServicoByOrdemServicoRow{}
	}
	pecas := l.pecas
	if pecas == nil {
		pecas = []db.ListItensPecaByOrdemServicoRow{}
	}
	reparos := l.reparos
	if reparos == nil {
		reparos = []db.ReparoAdicional{}
	}

	store.EXPECT().GetOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return(l.os, nil)
	store.EXPECT().GetClienteByID(gomock.Any(), gomock.Any()).AnyTimes().Return(l.cliente, nil)
	store.EXPECT().GetVeiculoByID(gomock.Any(), gomock.Any()).AnyTimes().Return(l.veiculo, nil)
	store.EXPECT().ListItensServicoByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return(servicos, nil)
	store.EXPECT().ListItensPecaByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return(pecas, nil)
	store.EXPECT().ListReparosAdicionaisByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return(reparos, nil)
	store.EXPECT().ListHistoricoStatusOSByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return([]db.HistoricoStatusOS{}, nil)
}

// fxCapturaEstado records every persisted aggregate state.
func fxCapturaEstado(store *mockdb.MockStore) *[]db.UpdateOrdemServicoEstadoParams {
	calls := []db.UpdateOrdemServicoEstadoParams{}
	store.EXPECT().UpdateOrdemServicoEstado(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, arg db.UpdateOrdemServicoEstadoParams) (db.OrdemServico, error) {
			calls = append(calls, arg)
			return db.OrdemServico{ID: arg.ID, Status: arg.Status}, nil
		})
	return &calls
}

func fxCapturaHistorico(store *mockdb.MockStore) *[]db.CreateHistoricoStatusOSParams {
	calls := []db.CreateHistoricoStatusOSParams{}
	store.EXPECT().CreateHistoricoStatusOS(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, arg db.CreateHistoricoStatusOSParams) (db.HistoricoStatusOS, error) {
			calls = append(calls, arg)
			return db.HistoricoStatusOS{OrdemServicoID: arg.OrdemServicoID, StatusNovo: arg.StatusNovo}, nil
		})
	return &calls
}

func fxCapturaValorTotal(store *mockdb.MockStore) *[]int64 {
	calls := []int64{}
	store.EXPECT().UpdateOrdemServicoValorTotal(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, arg db.UpdateOrdemServicoValorTotalParams) (db.OrdemServico, error) {
			calls = append(calls, arg.ValorTotalCentavos)
			return db.OrdemServico{ID: arg.ID, ValorTotalCentavos: arg.ValorTotalCentavos}, nil
		})
	return &calls
}
