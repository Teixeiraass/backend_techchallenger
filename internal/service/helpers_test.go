package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	mockrepo "github.com/Teixeiraass/backend_techchallenger/internal/repository/mock"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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

// fxApp é o serviço da OS ligado aos mocks das portas: unidade de trabalho,
// repositório da OS, repositório de peças e catálogo. A unidade de trabalho
// entrega os mesmos repositórios dentro e fora da transação.
type fxApp struct {
	uow      *mockrepo.MockUnidadeDeTrabalho
	repos    *mockrepo.MockRepositorios
	osRepo   *mockrepo.MockOrdemServicoRepositorio
	pecas    *mockrepo.MockPecaRepositorio
	catalogo *mockrepo.MockCatalogo
	sender   *fxSender
	svc      service.OrdemServicoService

	cliente readmodel.Cliente
	veiculo readmodel.Veiculo
	// transacoes conta quantas unidades de trabalho foram abertas.
	transacoes int
}

func novaApp(t *testing.T) *fxApp {
	ctrl := gomock.NewController(t)
	a := &fxApp{
		uow:      mockrepo.NewMockUnidadeDeTrabalho(ctrl),
		repos:    mockrepo.NewMockRepositorios(ctrl),
		osRepo:   mockrepo.NewMockOrdemServicoRepositorio(ctrl),
		pecas:    mockrepo.NewMockPecaRepositorio(ctrl),
		catalogo: mockrepo.NewMockCatalogo(ctrl),
		sender:   &fxSender{},
	}
	a.cliente = readmodel.Cliente{ID: util.RandomInt(1, 1000), UUID: uuid.New(), Nome: util.RandomName(), Documento: util.RandomCPF(), TipoDocumento: "CPF", Email: util.RandomEmail()}
	a.veiculo = readmodel.Veiculo{ID: util.RandomInt(1, 1000), UUID: uuid.New(), ClienteID: a.cliente.ID, ClienteUUID: a.cliente.UUID, Placa: util.RandomPlaca(), Marca: "Fiat", Modelo: "Uno", Ano: 2020}

	a.repos.EXPECT().OrdemServico().AnyTimes().Return(a.osRepo)
	a.repos.EXPECT().Pecas().AnyTimes().Return(a.pecas)
	a.repos.EXPECT().Catalogo().AnyTimes().Return(a.catalogo)
	a.uow.EXPECT().Leitura().AnyTimes().Return(a.repos)
	a.uow.EXPECT().Executar(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, fn func(service.Repositorios) error) error {
			a.transacoes++
			return fn(a.repos)
		})

	// store (read model) não é usado pelos comandos: nil.
	a.svc = service.NewOrdemServicoService(a.uow, nil, a.sender, fxConfig())
	return a
}

// os cria uma OS carregada no status pedido, com um serviço (10000) e, se
// pedido, uma peça (2500 cada).
func (a *fxApp) os(status ordemservico.Status, orcamento ordemservico.StatusOrcamento) *ordemservico.OrdemServico {
	os := ordemservico.OrdemServico{
		ID:              util.RandomInt(1, 1000),
		UUID:            uuid.New(),
		ClienteID:       a.cliente.ID,
		VeiculoID:       a.veiculo.ID,
		Status:          status,
		StatusOrcamento: orcamento,
		CriadaEm:        time.Now(),
		AtualizadaEm:    time.Now(),
	}
	return ordemservico.Reconstituir(os)
}

// leitura prepara as leituras usadas na montagem da resposta.
func (a *fxApp) leitura(os *ordemservico.OrdemServico) {
	a.osRepo.EXPECT().Carregar(gomock.Any(), os.UUID).AnyTimes().Return(os, nil)
	a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).AnyTimes().Return(a.cliente, nil)
	a.catalogo.EXPECT().VeiculoPorID(gomock.Any(), a.veiculo.ID).AnyTimes().Return(a.veiculo, nil)
	a.catalogo.EXPECT().HistoricoDaOS(gomock.Any(), os.ID).AnyTimes().Return([]readmodel.HistoricoStatus{{StatusNovo: "RECEBIDA"}}, nil)
}

// trava prepara o carregamento travado da OS pelo comando.
func (a *fxApp) trava(os *ordemservico.OrdemServico) {
	a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), os.UUID).Times(1).Return(os, nil)
}

// salva captura o agregado salvo pelo comando.
func (a *fxApp) salva() *[]ordemservico.OrdemServico {
	salvos := []ordemservico.OrdemServico{}
	a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, os *ordemservico.OrdemServico) error {
			for i := range os.Itens {
				if os.Itens[i].ID == 0 {
					os.Itens[i].ID = util.RandomInt(1, 1000)
					os.Itens[i].UUID = uuid.New()
				}
			}
			for i := range os.Reparos {
				if os.Reparos[i].ID == 0 {
					os.Reparos[i].ID = util.RandomInt(1, 1000)
				}
			}
			if os.ID == 0 {
				os.ID, os.UUID = util.RandomInt(1, 1000), uuid.New()
			}
			// A cópia guarda as alterações pendentes: o que o repositório gravaria.
			salvos = append(salvos, *os)
			os.LimparAlteracoes()
			return nil
		})
	return &salvos
}

func fxServico(ativo bool) ordemservico.CatalogoServico {
	return ordemservico.CatalogoServico{ID: util.RandomInt(1, 1000), UUID: uuid.New(), Nome: util.RandomName(), PrecoCentavos: 10000, Ativo: ativo}
}

func fxPeca(saldo int32, minimo int32) *estoque.Peca {
	return &estoque.Peca{
		ID: util.RandomInt(1, 1000), UUID: uuid.New(), Codigo: "COD-" + util.RandomString(4), Nome: util.RandomName(),
		Tipo: estoque.TipoPeca, Unidade: "UN", PrecoCentavos: 2500, Saldo: saldo, EstoqueMinimo: minimo, Ativa: true,
	}
}

func fxItemServico(servico ordemservico.CatalogoServico, quantidade int32) ordemservico.Item {
	return ordemservico.Item{ID: util.RandomInt(1, 1000), UUID: uuid.New(), Tipo: ordemservico.ItemServico, CatalogoID: servico.ID, CatalogoUUID: servico.UUID, Nome: servico.Nome, Quantidade: quantidade, ValorUnitarioCentavos: servico.PrecoCentavos}
}

func fxItemPeca(peca *estoque.Peca, quantidade int32) ordemservico.Item {
	return ordemservico.Item{ID: util.RandomInt(1, 1000), UUID: uuid.New(), Tipo: ordemservico.ItemPeca, CatalogoID: peca.ID, CatalogoUUID: peca.UUID, Nome: peca.Nome, Codigo: peca.Codigo, TipoPeca: string(peca.Tipo), Quantidade: quantidade, ValorUnitarioCentavos: peca.PrecoCentavos}
}

// fxDoReparo liga o item a um reparo adicional.
func fxDoReparo(item ordemservico.Item, reparo ordemservico.ReparoAdicional) ordemservico.Item {
	item.ReparoUUID = uuid.NullUUID{UUID: reparo.UUID, Valid: true}
	item.StatusReparo = reparo.Status
	return item
}

func fxReparo(status ordemservico.StatusReparo) ordemservico.ReparoAdicional {
	return ordemservico.ReparoAdicional{ID: util.RandomInt(1, 1000), UUID: uuid.New(), Descricao: "Disco empenado", Status: status}
}

// baixaOK prepara a baixa de uma peça pelo repositório de estoque.
func (a *fxApp) baixaOK(peca *estoque.Peca) {
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Times(1).Return(peca, nil)
	a.pecas.EXPECT().AplicarMovimento(gomock.Any(), peca, gomock.Any()).Times(1).Return(nil)
}

func requireHistorico(t *testing.T, os ordemservico.OrdemServico, observacao string) ordemservico.EntradaHistorico {
	t.Helper()
	for _, h := range os.Alteracoes().Historico {
		if h.Observacao == observacao || (observacao == "" && len(os.Alteracoes().Historico) == 1) {
			return h
		}
	}
	require.Failf(t, "histórico não encontrado", "observação %q em %+v", observacao, os.Alteracoes().Historico)
	return ordemservico.EntradaHistorico{}
}
