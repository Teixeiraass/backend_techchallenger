package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/veiculo"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fxCenario struct {
	store   *mockdb.MockStore
	sender  *fxSender
	svc     service.OrdemServicoService
	cliente db.Cliente
	veiculo db.Veiculo
	os      db.OrdemServico
}

func novoCenario(t *testing.T, status ordemservico.Status, orcamento ordemservico.StatusOrcamento) *fxCenario {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sender := &fxSender{}
	c := fxCliente()
	v := fxVeiculo(c.ID)
	return &fxCenario{
		store:   store,
		sender:  sender,
		svc:     service.NewOrdemServicoService(store, sender, fxConfig()),
		cliente: c,
		veiculo: v,
		os:      fxOrdemServico(c, v, status, orcamento),
	}
}

func (c *fxCenario) leitura(servicos []db.ListItensServicoByOrdemServicoRow, pecas []db.ListItensPecaByOrdemServicoRow, reparos []db.ReparoAdicional) {
	fxStubLeitura(c.store, fxLeitura{
		os:       c.os,
		cliente:  c.cliente,
		veiculo:  c.veiculo,
		servicos: servicos,
		pecas:    pecas,
		reparos:  reparos,
	})
}

func (c *fxCenario) travaOS() {
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), c.os.Uuid).Times(1).Return(c.os, nil)
}

// ---------------------------------------------------------------------------
// Abertura
// ---------------------------------------------------------------------------

func TestAbrirOrdemServicoComItens(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	servico := fxServico(true)
	peca := fxPeca(10, 2)
	itemServico := fxItemServico(c.os.ID, servico, 1)
	itemPeca := fxItemPeca(c.os.ID, peca, 2)
	c.leitura([]db.ListItensServicoByOrdemServicoRow{itemServico}, []db.ListItensPecaByOrdemServicoRow{itemPeca}, nil)
	historico := fxCapturaHistorico(c.store)
	totais := fxCapturaValorTotal(c.store)

	c.store.EXPECT().GetClienteByDocumento(gomock.Any(), c.cliente.Documento).Return(c.cliente, nil)
	c.store.EXPECT().GetVeiculoByPlaca(gomock.Any(), c.veiculo.Placa).Return(c.veiculo, nil)
	c.store.EXPECT().CreateOrdemServico(gomock.Any(), db.CreateOrdemServicoParams{
		ClienteID:         c.cliente.ID,
		VeiculoID:         c.veiculo.ID,
		DescricaoProblema: "Barulho",
	}).Return(c.os, nil)
	c.store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
	c.store.EXPECT().CreateItemServico(gomock.Any(), db.CreateItemServicoParams{
		OrdemServicoID:        c.os.ID,
		ServicoID:             servico.ID,
		Quantidade:            1,
		ValorUnitarioCentavos: servico.PrecoCentavos,
	}).Return(db.ItemServico{}, nil)
	c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
	c.store.EXPECT().CreateItemPeca(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateItemPecaParams) (db.ItemPeca, error) {
			require.Equal(t, int32(2), arg.Quantidade)
			require.Equal(t, peca.PrecoCentavos, arg.ValorUnitarioCentavos)
			require.False(t, arg.ReparoAdicionalID.Valid)
			return db.ItemPeca{}, nil
		})

	rsp, err := c.svc.AbrirOrdemServico(context.Background(), dto.AbrirOrdemServicoRequest{
		Documento:         c.cliente.Documento,
		Placa:             c.veiculo.Placa,
		DescricaoProblema: "Barulho",
		Servicos:          []dto.ItemServicoRequest{{ServicoUuid: servico.Uuid}},
		Pecas:             []dto.ItemPecaRequest{{PecaUuid: peca.Uuid, Quantidade: 2}},
	}, 7)
	require.NoError(t, err)
	require.Equal(t, c.os.Uuid, rsp.Uuid)
	require.Equal(t, string(ordemservico.StatusRecebida), rsp.Status)
	require.Len(t, rsp.Servicos, 1)
	require.Len(t, rsp.Pecas, 1)
	require.Equal(t, int64(10000+2*2500), (*totais)[0])
	require.Len(t, *historico, 1)
	require.Equal(t, "", (*historico)[0].StatusAnterior)
	require.Equal(t, int64(7), (*historico)[0].UsuarioID.Int64)
}

func TestAbrirOrdemServicoErros(t *testing.T) {
	testCases := []struct {
		name       string
		documento  func(c *fxCenario) string
		placa      func(c *fxCenario) string
		buildStubs func(c *fxCenario, servico db.Servico, peca db.Peca)
		pecas      bool
		servicos   bool
		wantErr    error
	}{
		{
			name:       "DocumentoInvalido",
			documento:  func(c *fxCenario) string { return "123" },
			placa:      func(c *fxCenario) string { return c.veiculo.Placa },
			buildStubs: func(c *fxCenario, _ db.Servico, _ db.Peca) {},
			wantErr:    cliente.ErrDocumentoInvalido,
		},
		{
			name:       "PlacaInvalida",
			documento:  func(c *fxCenario) string { return c.cliente.Documento },
			placa:      func(c *fxCenario) string { return "12-ABC" },
			buildStubs: func(c *fxCenario, _ db.Servico, _ db.Peca) {},
			wantErr:    veiculo.ErrPlacaInvalida,
		},
		{
			name:      "ClienteNaoEncontrado",
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			placa:     func(c *fxCenario) string { return c.veiculo.Placa },
			buildStubs: func(c *fxCenario, _ db.Servico, _ db.Peca) {
				c.store.EXPECT().GetClienteByDocumento(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrNoRows)
			},
			wantErr: service.ErrClienteNaoEncontrado,
		},
		{
			name:      "VeiculoNaoEncontrado",
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			placa:     func(c *fxCenario) string { return c.veiculo.Placa },
			buildStubs: func(c *fxCenario, _ db.Servico, _ db.Peca) {
				c.store.EXPECT().GetClienteByDocumento(gomock.Any(), gomock.Any()).Return(c.cliente, nil)
				c.store.EXPECT().GetVeiculoByPlaca(gomock.Any(), gomock.Any()).Return(db.Veiculo{}, sql.ErrNoRows)
			},
			wantErr: service.ErrVeiculoNaoEncontrado,
		},
		{
			name:      "VeiculoDeOutroCliente",
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			placa:     func(c *fxCenario) string { return c.veiculo.Placa },
			buildStubs: func(c *fxCenario, _ db.Servico, _ db.Peca) {
				outro := c.veiculo
				outro.ClienteID = c.cliente.ID + 1
				c.store.EXPECT().GetClienteByDocumento(gomock.Any(), gomock.Any()).Return(c.cliente, nil)
				c.store.EXPECT().GetVeiculoByPlaca(gomock.Any(), gomock.Any()).Return(outro, nil)
			},
			wantErr: service.ErrVeiculoDeOutroCliente,
		},
		{
			name:      "ServicoInativo",
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			placa:     func(c *fxCenario) string { return c.veiculo.Placa },
			servicos:  true,
			buildStubs: func(c *fxCenario, servico db.Servico, _ db.Peca) {
				servico.Ativo = false
				c.store.EXPECT().GetClienteByDocumento(gomock.Any(), gomock.Any()).Return(c.cliente, nil)
				c.store.EXPECT().GetVeiculoByPlaca(gomock.Any(), gomock.Any()).Return(c.veiculo, nil)
				c.store.EXPECT().CreateOrdemServico(gomock.Any(), gomock.Any()).Return(c.os, nil)
				fxCapturaHistorico(c.store)
				c.store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
			},
			wantErr: service.ErrServicoInativo,
		},
		{
			name:      "ServicoNaoEncontrado",
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			placa:     func(c *fxCenario) string { return c.veiculo.Placa },
			servicos:  true,
			buildStubs: func(c *fxCenario, servico db.Servico, _ db.Peca) {
				c.store.EXPECT().GetClienteByDocumento(gomock.Any(), gomock.Any()).Return(c.cliente, nil)
				c.store.EXPECT().GetVeiculoByPlaca(gomock.Any(), gomock.Any()).Return(c.veiculo, nil)
				c.store.EXPECT().CreateOrdemServico(gomock.Any(), gomock.Any()).Return(c.os, nil)
				fxCapturaHistorico(c.store)
				c.store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(db.Servico{}, sql.ErrNoRows)
			},
			wantErr: service.ErrServicoNaoEncontrado,
		},
		{
			name:      "PecaSemEstoque",
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			placa:     func(c *fxCenario) string { return c.veiculo.Placa },
			pecas:     true,
			buildStubs: func(c *fxCenario, _ db.Servico, peca db.Peca) {
				peca.QuantidadeEstoque = 1
				c.store.EXPECT().GetClienteByDocumento(gomock.Any(), gomock.Any()).Return(c.cliente, nil)
				c.store.EXPECT().GetVeiculoByPlaca(gomock.Any(), gomock.Any()).Return(c.veiculo, nil)
				c.store.EXPECT().CreateOrdemServico(gomock.Any(), gomock.Any()).Return(c.os, nil)
				fxCapturaHistorico(c.store)
				c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
				c.leitura(nil, nil, nil)
			},
			wantErr: estoque.ErrEstoqueInsuficiente,
		},
		{
			name:      "PecaNaoEncontrada",
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			placa:     func(c *fxCenario) string { return c.veiculo.Placa },
			pecas:     true,
			buildStubs: func(c *fxCenario, _ db.Servico, peca db.Peca) {
				c.store.EXPECT().GetClienteByDocumento(gomock.Any(), gomock.Any()).Return(c.cliente, nil)
				c.store.EXPECT().GetVeiculoByPlaca(gomock.Any(), gomock.Any()).Return(c.veiculo, nil)
				c.store.EXPECT().CreateOrdemServico(gomock.Any(), gomock.Any()).Return(c.os, nil)
				fxCapturaHistorico(c.store)
				c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(db.Peca{}, sql.ErrNoRows)
			},
			wantErr: service.ErrPecaNaoEncontrada,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
			fxExecTx(c.store)
			servico := fxServico(true)
			peca := fxPeca(10, 1)
			tc.buildStubs(c, servico, peca)

			req := dto.AbrirOrdemServicoRequest{Documento: tc.documento(c), Placa: tc.placa(c)}
			if tc.servicos {
				req.Servicos = []dto.ItemServicoRequest{{ServicoUuid: servico.Uuid}}
			}
			if tc.pecas {
				req.Pecas = []dto.ItemPecaRequest{{PecaUuid: peca.Uuid, Quantidade: 5}}
			}

			rsp, err := c.svc.AbrirOrdemServico(context.Background(), req, 1)
			require.Error(t, err)
			require.True(t, errors.Is(err, tc.wantErr), "got %v", err)
			require.Nil(t, rsp)
		})
	}
}

// ---------------------------------------------------------------------------
// Itens
// ---------------------------------------------------------------------------

func TestAdicionarServico(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	servico := fxServico(true)
	c.leitura([]db.ListItensServicoByOrdemServicoRow{fxItemServico(c.os.ID, servico, 2)}, nil, nil)
	totais := fxCapturaValorTotal(c.store)
	c.travaOS()
	c.store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
	c.store.EXPECT().CreateItemServico(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateItemServicoParams) (db.ItemServico, error) {
			require.Equal(t, int32(2), arg.Quantidade)
			return db.ItemServico{}, nil
		})

	rsp, err := c.svc.AdicionarServico(context.Background(), c.os.Uuid, dto.ItemServicoRequest{ServicoUuid: servico.Uuid, Quantidade: 2})
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.Equal(t, []int64{20000}, *totais)
}

func TestAdicionarServicoStatusNaoEditavel(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	c.travaOS()

	_, err := c.svc.AdicionarServico(context.Background(), c.os.Uuid, dto.ItemServicoRequest{ServicoUuid: uuid.New()})
	require.True(t, errors.Is(err, ordemservico.ErrItensNaoEditaveis))
}

func TestAdicionarServicoOSNaoEncontrada(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)

	_, err := c.svc.AdicionarServico(context.Background(), c.os.Uuid, dto.ItemServicoRequest{ServicoUuid: uuid.New()})
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestRemoverServico(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	totais := fxCapturaValorTotal(c.store)
	itemUUID := uuid.New()
	c.travaOS()
	c.store.EXPECT().DeleteItemServico(gomock.Any(), db.DeleteItemServicoParams{Uuid: itemUUID, OrdemServicoID: c.os.ID}).Return(int64(1), nil)

	_, err := c.svc.RemoverServico(context.Background(), c.os.Uuid, itemUUID)
	require.NoError(t, err)
	require.Equal(t, []int64{0}, *totais)
}

func TestRemoverServicoItemNaoEncontrado(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.travaOS()
	c.store.EXPECT().DeleteItemServico(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrNoRows)

	_, err := c.svc.RemoverServico(context.Background(), c.os.Uuid, uuid.New())
	require.True(t, errors.Is(err, service.ErrItemNaoEncontrado))
}

func TestAdicionarPecaConsideraQuantidadeJaNaOS(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	peca := fxPeca(5, 1)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, peca, 4)}, nil)
	c.travaOS()
	c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
	c.store.EXPECT().CreateItemPeca(gomock.Any(), gomock.Any()).Times(0)

	// 4 já na OS + 2 pedidas > saldo 5
	_, err := c.svc.AdicionarPeca(context.Background(), c.os.Uuid, dto.ItemPecaRequest{PecaUuid: peca.Uuid, Quantidade: 2})
	require.True(t, errors.Is(err, estoque.ErrEstoqueInsuficiente))
	require.Contains(t, err.Error(), peca.Codigo)
}

func TestAdicionarPecaOK(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	peca := fxPeca(5, 1)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, peca, 1)}, nil)
	totais := fxCapturaValorTotal(c.store)
	c.travaOS()
	c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
	c.store.EXPECT().CreateItemPeca(gomock.Any(), gomock.Any()).Return(db.ItemPeca{}, nil)

	_, err := c.svc.AdicionarPeca(context.Background(), c.os.Uuid, dto.ItemPecaRequest{PecaUuid: peca.Uuid, Quantidade: 2})
	require.NoError(t, err)
	require.Equal(t, []int64{2500}, *totais)
}

func TestAdicionarPecaInativa(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	peca := fxPeca(5, 1)
	peca.Ativo = false
	c.leitura(nil, nil, nil)
	c.travaOS()
	c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)

	_, err := c.svc.AdicionarPeca(context.Background(), c.os.Uuid, dto.ItemPecaRequest{PecaUuid: peca.Uuid, Quantidade: 1})
	require.True(t, errors.Is(err, estoque.ErrPecaInativa))
}

func TestRemoverPeca(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	fxCapturaValorTotal(c.store)
	c.travaOS()
	c.store.EXPECT().DeleteItemPeca(gomock.Any(), gomock.Any()).Return(int64(1), nil)

	_, err := c.svc.RemoverPeca(context.Background(), c.os.Uuid, uuid.New())
	require.NoError(t, err)
}

func TestRemoverPecaItemNaoEncontrado(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.travaOS()
	c.store.EXPECT().DeleteItemPeca(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrNoRows)

	_, err := c.svc.RemoverPeca(context.Background(), c.os.Uuid, uuid.New())
	require.True(t, errors.Is(err, service.ErrItemNaoEncontrado))
}

// ---------------------------------------------------------------------------
// Transições
// ---------------------------------------------------------------------------

func TestIniciarDiagnostico(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	estados := fxCapturaEstado(c.store)
	historico := fxCapturaHistorico(c.store)
	c.travaOS()

	_, err := c.svc.IniciarDiagnostico(context.Background(), c.os.Uuid, 3)
	require.NoError(t, err)
	require.Len(t, *estados, 1)
	require.Equal(t, string(ordemservico.StatusEmDiagnostico), (*estados)[0].Status)
	require.True(t, (*estados)[0].DiagnosticoIniciadoEm.Valid)
	require.Equal(t, string(ordemservico.StatusRecebida), (*historico)[0].StatusAnterior)
	require.Equal(t, string(ordemservico.StatusEmDiagnostico), (*historico)[0].StatusNovo)
}

func TestIniciarDiagnosticoTransicaoInvalida(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	c.travaOS()
	c.store.EXPECT().UpdateOrdemServicoEstado(gomock.Any(), gomock.Any()).Times(0)

	_, err := c.svc.IniciarDiagnostico(context.Background(), c.os.Uuid, 3)
	require.True(t, errors.Is(err, ordemservico.ErrTransicaoInvalida))
}

func TestIniciarDiagnosticoOSNaoEncontrada(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)

	_, err := c.svc.IniciarDiagnostico(context.Background(), c.os.Uuid, 3)
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestEnviarOrcamento(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	servico := fxServico(true)
	peca := fxPeca(10, 1)
	c.leitura([]db.ListItensServicoByOrdemServicoRow{fxItemServico(c.os.ID, servico, 1)},
		[]db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, peca, 2)}, nil)
	estados := fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)
	c.travaOS()

	_, err := c.svc.EnviarOrcamento(context.Background(), c.os.Uuid, dto.EnviarOrcamentoRequest{Diagnostico: "Pastilhas gastas"}, 2)
	require.NoError(t, err)
	estado := (*estados)[0]
	require.Equal(t, string(ordemservico.StatusAguardandoAprovacao), estado.Status)
	require.Equal(t, string(ordemservico.OrcamentoAguardando), estado.OrcamentoStatus)
	require.Equal(t, int64(15000), estado.ValorTotalCentavos)
	require.Equal(t, "Pastilhas gastas", estado.Diagnostico)
	require.True(t, estado.OrcamentoEnviadoEm.Valid)

	require.Len(t, c.sender.msgs, 1)
	require.Equal(t, c.cliente.Email, c.sender.msgs[0].To)
	require.Contains(t, c.sender.msgs[0].Body, "R$ 150,00")
}

func TestEnviarOrcamentoSemItens(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	c.travaOS()

	_, err := c.svc.EnviarOrcamento(context.Background(), c.os.Uuid, dto.EnviarOrcamentoRequest{}, 2)
	require.True(t, errors.Is(err, ordemservico.ErrOrcamentoSemItens))
	require.Empty(t, c.sender.msgs)
}

func TestEnviarOrcamentoTransicaoInvalida(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.leitura([]db.ListItensServicoByOrdemServicoRow{fxItemServico(c.os.ID, fxServico(true), 1)}, nil, nil)
	c.travaOS()

	_, err := c.svc.EnviarOrcamento(context.Background(), c.os.Uuid, dto.EnviarOrcamentoRequest{}, 2)
	require.True(t, errors.Is(err, ordemservico.ErrTransicaoInvalida))
}

func TestEnviarOrcamentoNotificacaoFalhaNaoDesfaz(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	c.sender.err = errors.New("smtp fora do ar")
	fxExecTx(c.store)
	c.leitura([]db.ListItensServicoByOrdemServicoRow{fxItemServico(c.os.ID, fxServico(true), 1)}, nil, nil)
	fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)
	c.travaOS()

	rsp, err := c.svc.EnviarOrcamento(context.Background(), c.os.Uuid, dto.EnviarOrcamentoRequest{}, 2)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.Equal(t, c.cliente.Email, c.sender.msgs[0].To)
}

// A notificação é por e-mail (linguagem ubíqua §1.2): cliente sem e-mail não
// recebe aviso, e o envio do orçamento continua válido.
func TestEnviarOrcamentoClienteSemEmailNaoNotifica(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.cliente.Email = ""
	c.cliente.Telefone = "11999990000"
	c.leitura([]db.ListItensServicoByOrdemServicoRow{fxItemServico(c.os.ID, fxServico(true), 1)}, nil, nil)
	fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)
	c.travaOS()

	rsp, err := c.svc.EnviarOrcamento(context.Background(), c.os.Uuid, dto.EnviarOrcamentoRequest{}, 2)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.Empty(t, c.sender.msgs)
}

func TestAprovarOrcamentoIniciaExecucaoEBaixaEstoque(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	fxExecTx(c.store)
	peca := fxPeca(3, 2)
	reparo := fxReparo(c.os.ID, ordemservico.ReparoPendente)
	itemBase := fxItemPeca(c.os.ID, peca, 2)
	itemReparo := fxDoReparo(fxItemPeca(c.os.ID, peca, 1), reparo)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{itemBase, itemReparo}, nil)
	estados := fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)

	aprovada := c.os
	aprovada.OrcamentoStatus = string(ordemservico.OrcamentoAprovado)
	c.travaOS()
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), c.os.Uuid).Times(1).Return(aprovada, nil)

	depois := peca
	depois.QuantidadeEstoque = 1
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), db.BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 2}).Times(1).Return(depois, nil)
	c.store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateMovimentoEstoqueParams) (db.MovimentoEstoque, error) {
			require.Equal(t, string(estoque.MovimentoSaida), arg.Tipo)
			require.Equal(t, int32(-2), arg.Quantidade)
			require.Equal(t, int32(3), arg.SaldoAnterior)
			require.Equal(t, int32(1), arg.SaldoPosterior)
			require.Equal(t, c.os.ID, arg.OrdemServicoID.Int64)
			return db.MovimentoEstoque{}, nil
		})

	rsp, err := c.svc.AprovarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.NoError(t, err)
	require.Empty(t, rsp.Aviso)
	require.Len(t, *estados, 2)
	require.Equal(t, string(ordemservico.OrcamentoAprovado), (*estados)[0].OrcamentoStatus)
	require.Equal(t, string(ordemservico.StatusAguardandoAprovacao), (*estados)[0].Status)
	require.Equal(t, string(ordemservico.StatusEmExecucao), (*estados)[1].Status)
	require.True(t, (*estados)[1].ExecucaoIniciadaEm.Valid)

	// saldo 1 <= mínimo 2: pedido de reposição para o administrador
	require.Len(t, c.sender.msgs, 1)
	require.Equal(t, "admin@oficina.com", c.sender.msgs[0].To)
}

func TestAprovarOrcamentoEstoqueInsuficienteMantemAprovacao(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	fxExecTx(c.store)
	peca := fxPeca(0, 2)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, peca, 2)}, nil)
	estados := fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)

	aprovada := c.os
	aprovada.OrcamentoStatus = string(ordemservico.OrcamentoAprovado)
	c.travaOS()
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), c.os.Uuid).Times(1).Return(aprovada, nil)
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)

	rsp, err := c.svc.AprovarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.NoError(t, err)
	require.Contains(t, rsp.Aviso, "a oficina inicia o serviço")
	require.Len(t, *estados, 1)
	require.Equal(t, string(ordemservico.OrcamentoAprovado), (*estados)[0].OrcamentoStatus)

	// Falta de saldo gera pedido de reposição, senão a OS fica parada.
	require.Len(t, c.sender.msgs, 1)
	require.Equal(t, "admin@oficina.com", c.sender.msgs[0].To)
	require.Contains(t, c.sender.msgs[0].Body, peca.Codigo)
	require.Contains(t, c.sender.msgs[0].Body, c.os.Uuid.String())
}

// Repetir a aprovação com a execução ainda não iniciada (aprovação gravada,
// faltou peça) não dá 409: o serviço tenta iniciar a execução de novo.
func TestAprovarOrcamentoRepetidaTentaIniciarExecucao(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	peca := fxPeca(5, 1)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, peca, 2)}, nil)
	estados := fxCapturaEstado(c.store)
	historico := fxCapturaHistorico(c.store)
	c.travaOS()
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), c.os.Uuid).Times(1).Return(c.os, nil)
	depois := peca
	depois.QuantidadeEstoque = 3
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), db.BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 2}).Times(1).Return(depois, nil)
	c.store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)

	rsp, err := c.svc.AprovarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.NoError(t, err)
	require.Empty(t, rsp.Aviso)
	// Só a execução é gravada: a aprovação não entra de novo no histórico.
	require.Len(t, *estados, 1)
	require.Equal(t, string(ordemservico.StatusEmExecucao), (*estados)[0].Status)
	require.Len(t, *historico, 1)
	require.Equal(t, string(ordemservico.StatusEmExecucao), (*historico)[0].StatusNovo)
}

func TestAprovarOrcamentoRepetidaComExecucaoIniciada(t *testing.T) {
	for _, status := range []ordemservico.Status{ordemservico.StatusEmExecucao, ordemservico.StatusFinalizada} {
		t.Run(string(status), func(t *testing.T) {
			c := novoCenario(t, status, ordemservico.OrcamentoAprovado)
			fxExecTx(c.store)
			c.leitura(nil, nil, nil)
			c.travaOS()
			c.store.EXPECT().UpdateOrdemServicoEstado(gomock.Any(), gomock.Any()).Times(0)
			c.store.EXPECT().CreateHistoricoStatusOS(gomock.Any(), gomock.Any()).Times(0)
			c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Times(0)

			rsp, err := c.svc.AprovarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
			require.NoError(t, err)
			require.Equal(t, "Orçamento já aprovado", rsp.Aviso)
			require.Equal(t, string(status), rsp.Status)
		})
	}
}

func TestAprovarOrcamentoDocumentoErrado(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	c.travaOS()
	c.store.EXPECT().UpdateOrdemServicoEstado(gomock.Any(), gomock.Any()).Times(0)

	_, err := c.svc.AprovarOrcamento(context.Background(), c.os.Uuid, util.RandomCPF())
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestAprovarOrcamentoNaoAguardando(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	c.travaOS()

	_, err := c.svc.AprovarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.True(t, errors.Is(err, ordemservico.ErrOrcamentoNaoAguardando))
}

func TestAprovarOrcamentoOSNaoEncontrada(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	fxExecTx(c.store)
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)

	_, err := c.svc.AprovarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestRejeitarOrcamento(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	estados := fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)
	c.travaOS()

	_, err := c.svc.RejeitarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, string(ordemservico.StatusEmDiagnostico), (*estados)[0].Status)
	require.Equal(t, string(ordemservico.OrcamentoRejeitado), (*estados)[0].OrcamentoStatus)
}

func TestRejeitarOrcamentoErros(t *testing.T) {
	t.Run("DocumentoErrado", func(t *testing.T) {
		c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		fxExecTx(c.store)
		c.leitura(nil, nil, nil)
		c.travaOS()
		_, err := c.svc.RejeitarOrcamento(context.Background(), c.os.Uuid, "000")
		require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
	})
	t.Run("NaoAguardando", func(t *testing.T) {
		c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		fxExecTx(c.store)
		c.leitura(nil, nil, nil)
		c.travaOS()
		_, err := c.svc.RejeitarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
		require.True(t, errors.Is(err, ordemservico.ErrOrcamentoNaoAguardando))
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		fxExecTx(c.store)
		c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)
		_, err := c.svc.RejeitarOrcamento(context.Background(), c.os.Uuid, c.cliente.Documento)
		require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
	})
}

func TestIniciarExecucaoSemPecas(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	estados := fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)
	c.travaOS()
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Times(0)

	_, err := c.svc.IniciarExecucao(context.Background(), c.os.Uuid, 4)
	require.NoError(t, err)
	require.Equal(t, string(ordemservico.StatusEmExecucao), (*estados)[0].Status)
	require.Empty(t, c.sender.msgs)
}

func TestIniciarExecucaoErros(t *testing.T) {
	t.Run("OrcamentoNaoAprovado", func(t *testing.T) {
		c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		fxExecTx(c.store)
		c.travaOS()
		_, err := c.svc.IniciarExecucao(context.Background(), c.os.Uuid, 4)
		require.True(t, errors.Is(err, ordemservico.ErrOrcamentoNaoAprovado))
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
		fxExecTx(c.store)
		c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)
		_, err := c.svc.IniciarExecucao(context.Background(), c.os.Uuid, 4)
		require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
	})
	t.Run("ErroNoBanco", func(t *testing.T) {
		c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
		fxExecTx(c.store)
		c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, fxPeca(5, 1), 1)}, nil)
		c.travaOS()
		c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrConnDone)
		_, err := c.svc.IniciarExecucao(context.Background(), c.os.Uuid, 4)
		require.ErrorIs(t, err, sql.ErrConnDone)
		require.Empty(t, c.sender.msgs)
	})
}

func TestFinalizarOrdemServico(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	inicio := time.Now().Add(-90 * time.Minute)
	c.os.ExecucaoIniciadaEm = sql.NullTime{Time: inicio, Valid: true}
	fxExecTx(c.store)
	c.leitura(nil, nil, []db.ReparoAdicional{fxReparo(c.os.ID, ordemservico.ReparoAutorizado)})
	estados := fxCapturaEstado(c.store)
	historico := fxCapturaHistorico(c.store)
	c.travaOS()

	_, err := c.svc.FinalizarOrdemServico(context.Background(), c.os.Uuid, 5)
	require.NoError(t, err)
	require.Equal(t, string(ordemservico.StatusFinalizada), (*estados)[0].Status)
	require.True(t, (*estados)[0].FinalizadaEm.Valid)
	require.Contains(t, (*historico)[0].Observacao, "1h 30min")
	require.Len(t, c.sender.msgs, 1)
}

// Cliente antigo sem e-mail: a OS é finalizada e o aviso não sai.
func TestFinalizarOrdemServicoClienteSemEmailNaoNotifica(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	c.os.ExecucaoIniciadaEm = sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true}
	c.cliente.Email = ""
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	estados := fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)
	c.travaOS()

	_, err := c.svc.FinalizarOrdemServico(context.Background(), c.os.Uuid, 5)
	require.NoError(t, err)
	require.Equal(t, string(ordemservico.StatusFinalizada), (*estados)[0].Status)
	require.Empty(t, c.sender.msgs)
}

func TestFinalizarOrdemServicoReparoPendente(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	c.leitura(nil, nil, []db.ReparoAdicional{fxReparo(c.os.ID, ordemservico.ReparoPendente)})
	c.travaOS()

	_, err := c.svc.FinalizarOrdemServico(context.Background(), c.os.Uuid, 5)
	require.True(t, errors.Is(err, ordemservico.ErrReparoPendente))
	require.Empty(t, c.sender.msgs)
}

func TestFinalizarOrdemServicoOSNaoEncontrada(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)

	_, err := c.svc.FinalizarOrdemServico(context.Background(), c.os.Uuid, 5)
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestFinalizarOrdemServicoNotificacaoFalha(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	c.sender.err = errors.New("push indisponível")
	fxExecTx(c.store)
	c.leitura(nil, nil, nil)
	fxCapturaEstado(c.store)
	fxCapturaHistorico(c.store)
	c.travaOS()

	_, err := c.svc.FinalizarOrdemServico(context.Background(), c.os.Uuid, 5)
	require.NoError(t, err)
}

func TestEntregarVeiculo(t *testing.T) {
	testCases := []struct {
		name       string
		status     ordemservico.Status
		orcamento  ordemservico.StatusOrcamento
		wantErr    error
		observacao string
	}{
		{name: "Finalizada", status: ordemservico.StatusFinalizada, orcamento: ordemservico.OrcamentoAprovado, observacao: "Veículo entregue"},
		{name: "OrcamentoRejeitado", status: ordemservico.StatusEmDiagnostico, orcamento: ordemservico.OrcamentoRejeitado, observacao: "sem execução"},
		{name: "SemOrcamento", status: ordemservico.StatusEmDiagnostico, orcamento: ordemservico.OrcamentoNaoEnviado, wantErr: ordemservico.ErrEntregaNaoPermitida},
		{name: "EmExecucao", status: ordemservico.StatusEmExecucao, orcamento: ordemservico.OrcamentoAprovado, wantErr: ordemservico.ErrEntregaNaoPermitida},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			c := novoCenario(t, tc.status, tc.orcamento)
			fxExecTx(c.store)
			c.leitura(nil, nil, nil)
			estados := fxCapturaEstado(c.store)
			historico := fxCapturaHistorico(c.store)
			c.travaOS()

			_, err := c.svc.EntregarVeiculo(context.Background(), c.os.Uuid, 6)
			if tc.wantErr != nil {
				require.True(t, errors.Is(err, tc.wantErr))
				require.Empty(t, *estados)
				return
			}
			require.NoError(t, err)
			require.Equal(t, string(ordemservico.StatusEntregue), (*estados)[0].Status)
			require.True(t, (*estados)[0].EntregueEm.Valid)
			require.Contains(t, (*historico)[0].Observacao, tc.observacao)
		})
	}
}

func TestEntregarVeiculoOSNaoEncontrada(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusFinalizada, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)

	_, err := c.svc.EntregarVeiculo(context.Background(), c.os.Uuid, 6)
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

// ---------------------------------------------------------------------------
// Reparo adicional
// ---------------------------------------------------------------------------

func TestRegistrarReparoAdicionalSemItens(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)

	_, err := c.svc.RegistrarReparoAdicional(context.Background(), c.os.Uuid, dto.RegistrarReparoAdicionalRequest{Descricao: "x"})
	require.True(t, errors.Is(err, ordemservico.ErrReparoSemItens))
}

func TestRegistrarReparoAdicionalForaDeExecucao(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	fxExecTx(c.store)
	c.travaOS()
	c.store.EXPECT().CreateReparoAdicional(gomock.Any(), gomock.Any()).Times(0)

	_, err := c.svc.RegistrarReparoAdicional(context.Background(), c.os.Uuid, dto.RegistrarReparoAdicionalRequest{
		Descricao: "x",
		Servicos:  []dto.ItemServicoRequest{{ServicoUuid: uuid.New()}},
	})
	require.True(t, errors.Is(err, ordemservico.ErrReparoForaDeExecucao))
}

func TestRegistrarReparoAdicionalOSNaoEncontrada(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)

	_, err := c.svc.RegistrarReparoAdicional(context.Background(), c.os.Uuid, dto.RegistrarReparoAdicionalRequest{
		Descricao: "x",
		Pecas:     []dto.ItemPecaRequest{{PecaUuid: uuid.New(), Quantidade: 1}},
	})
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestRegistrarReparoAdicional(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	servico := fxServico(true)
	peca := fxPeca(4, 1)
	reparo := fxReparo(c.os.ID, ordemservico.ReparoPendente)
	// Item base já baixado (OS em execução) não conta; outro reparo pendente conta.
	outroPendente := fxDoReparo(fxItemPeca(c.os.ID, peca, 2), fxReparo(c.os.ID, ordemservico.ReparoPendente))
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, peca, 3), outroPendente}, nil)
	c.travaOS()
	c.store.EXPECT().CreateReparoAdicional(gomock.Any(), db.CreateReparoAdicionalParams{
		OrdemServicoID: c.os.ID,
		Descricao:      "Disco empenado",
	}).Return(reparo, nil)
	c.store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
	c.store.EXPECT().CreateItemServico(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateItemServicoParams) (db.ItemServico, error) {
			require.Equal(t, reparo.ID, arg.ReparoAdicionalID.Int64)
			return db.ItemServico{}, nil
		})
	c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
	c.store.EXPECT().CreateItemPeca(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateItemPecaParams) (db.ItemPeca, error) {
			require.True(t, arg.ReparoAdicionalID.Valid)
			return db.ItemPeca{}, nil
		})

	_, err := c.svc.RegistrarReparoAdicional(context.Background(), c.os.Uuid, dto.RegistrarReparoAdicionalRequest{
		Descricao: "Disco empenado",
		Servicos:  []dto.ItemServicoRequest{{ServicoUuid: servico.Uuid}},
		Pecas:     []dto.ItemPecaRequest{{PecaUuid: peca.Uuid, Quantidade: 2}},
	})
	require.NoError(t, err)
	require.Len(t, c.sender.msgs, 1)
	require.Contains(t, c.sender.msgs[0].Body, "Disco empenado")
}

// Cliente antigo sem e-mail: o reparo é registrado e o aviso não sai.
func TestRegistrarReparoAdicionalClienteSemEmailNaoNotifica(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	c.cliente.Email = ""
	fxExecTx(c.store)
	servico := fxServico(true)
	c.leitura(nil, nil, nil)
	c.travaOS()
	c.store.EXPECT().CreateReparoAdicional(gomock.Any(), gomock.Any()).Return(fxReparo(c.os.ID, ordemservico.ReparoPendente), nil)
	c.store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
	c.store.EXPECT().CreateItemServico(gomock.Any(), gomock.Any()).Return(db.ItemServico{}, nil)

	_, err := c.svc.RegistrarReparoAdicional(context.Background(), c.os.Uuid, dto.RegistrarReparoAdicionalRequest{
		Descricao: "Disco empenado",
		Servicos:  []dto.ItemServicoRequest{{ServicoUuid: servico.Uuid}},
	})
	require.NoError(t, err)
	require.Empty(t, c.sender.msgs)
}

func TestRegistrarReparoAdicionalPecaSemEstoque(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	peca := fxPeca(4, 1)
	outroPendente := fxDoReparo(fxItemPeca(c.os.ID, peca, 3), fxReparo(c.os.ID, ordemservico.ReparoPendente))
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{outroPendente}, nil)
	c.travaOS()
	c.store.EXPECT().CreateReparoAdicional(gomock.Any(), gomock.Any()).Return(fxReparo(c.os.ID, ordemservico.ReparoPendente), nil)
	c.store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)

	_, err := c.svc.RegistrarReparoAdicional(context.Background(), c.os.Uuid, dto.RegistrarReparoAdicionalRequest{
		Descricao: "x",
		Pecas:     []dto.ItemPecaRequest{{PecaUuid: peca.Uuid, Quantidade: 2}},
	})
	require.True(t, errors.Is(err, estoque.ErrEstoqueInsuficiente))
	require.Empty(t, c.sender.msgs)
}

func TestAutorizarReparoAdicional(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	peca := fxPeca(5, 0)
	reparo := fxReparo(c.os.ID, ordemservico.ReparoPendente)
	autorizado := reparo
	autorizado.Status = string(ordemservico.ReparoAutorizado)
	itemBase := fxItemPeca(c.os.ID, peca, 1)
	itemReparo := fxDoReparo(fxItemPeca(c.os.ID, peca, 2), autorizado)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{itemBase, itemReparo}, nil)
	totais := fxCapturaValorTotal(c.store)
	historico := fxCapturaHistorico(c.store)
	c.travaOS()
	c.store.EXPECT().GetReparoAdicionalForUpdate(gomock.Any(), db.GetReparoAdicionalForUpdateParams{
		Uuid:           reparo.Uuid,
		OrdemServicoID: c.os.ID,
	}).Return(reparo, nil)
	c.store.EXPECT().UpdateReparoAdicionalStatus(gomock.Any(), db.UpdateReparoAdicionalStatusParams{
		ID:     reparo.ID,
		Status: string(ordemservico.ReparoAutorizado),
	}).Return(autorizado, nil)
	depois := peca
	depois.QuantidadeEstoque = 3
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), db.BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 2}).Times(1).Return(depois, nil)
	c.store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)

	_, err := c.svc.AutorizarReparoAdicional(context.Background(), c.os.Uuid, reparo.Uuid, c.cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, []int64{3 * 2500}, *totais)
	require.Contains(t, (*historico)[0].Observacao, "autorizado")
	require.Empty(t, c.sender.msgs)
}

func TestAutorizarReparoAdicionalSemEstoque(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	peca := fxPeca(0, 0)
	reparo := fxReparo(c.os.ID, ordemservico.ReparoPendente)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxDoReparo(fxItemPeca(c.os.ID, peca, 2), reparo)}, nil)
	c.travaOS()
	c.store.EXPECT().GetReparoAdicionalForUpdate(gomock.Any(), gomock.Any()).Return(reparo, nil)
	c.store.EXPECT().UpdateReparoAdicionalStatus(gomock.Any(), gomock.Any()).Return(reparo, nil)
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)

	_, err := c.svc.AutorizarReparoAdicional(context.Background(), c.os.Uuid, reparo.Uuid, c.cliente.Documento)
	require.True(t, errors.Is(err, estoque.ErrEstoqueInsuficiente))
	require.Contains(t, err.Error(), peca.Codigo)
	require.Len(t, c.sender.msgs, 1)
	require.Equal(t, "admin@oficina.com", c.sender.msgs[0].To)
	require.Contains(t, c.sender.msgs[0].Body, peca.Codigo)
}

// Falha no envio do pedido de reposição não troca o erro devolvido.
func TestIniciarExecucaoSemEstoquePedidoDeReposicaoFalha(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
	c.sender.err = errors.New("smtp fora do ar")
	fxExecTx(c.store)
	c.leitura(nil, []db.ListItensPecaByOrdemServicoRow{fxItemPeca(c.os.ID, fxPeca(1, 0), 3)}, nil)
	c.travaOS()
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)

	_, err := c.svc.IniciarExecucao(context.Background(), c.os.Uuid, 4)
	require.True(t, errors.Is(err, estoque.ErrEstoqueInsuficiente))
	require.Len(t, c.sender.msgs, 1)
}

func TestRecusarReparoAdicional(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	fxExecTx(c.store)
	reparo := fxReparo(c.os.ID, ordemservico.ReparoPendente)
	c.leitura(nil, nil, nil)
	historico := fxCapturaHistorico(c.store)
	c.travaOS()
	c.store.EXPECT().GetReparoAdicionalForUpdate(gomock.Any(), gomock.Any()).Return(reparo, nil)
	c.store.EXPECT().UpdateReparoAdicionalStatus(gomock.Any(), db.UpdateReparoAdicionalStatusParams{
		ID:     reparo.ID,
		Status: string(ordemservico.ReparoRecusado),
	}).Return(reparo, nil)
	c.store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Times(0)
	c.store.EXPECT().UpdateOrdemServicoValorTotal(gomock.Any(), gomock.Any()).Times(0)

	_, err := c.svc.RecusarReparoAdicional(context.Background(), c.os.Uuid, reparo.Uuid, c.cliente.Documento)
	require.NoError(t, err)
	require.Contains(t, (*historico)[0].Observacao, "recusado")
}

func TestDecidirReparoAdicionalErros(t *testing.T) {
	testCases := []struct {
		name       string
		status     ordemservico.Status
		documento  func(c *fxCenario) string
		buildStubs func(c *fxCenario)
		wantErr    error
	}{
		{
			name:      "JaDecidido",
			status:    ordemservico.StatusEmExecucao,
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			buildStubs: func(c *fxCenario) {
				c.travaOS()
				c.store.EXPECT().GetReparoAdicionalForUpdate(gomock.Any(), gomock.Any()).
					Return(fxReparo(c.os.ID, ordemservico.ReparoAutorizado), nil)
			},
			wantErr: ordemservico.ErrReparoJaDecidido,
		},
		{
			name:      "OSForaDeExecucao",
			status:    ordemservico.StatusFinalizada,
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			buildStubs: func(c *fxCenario) {
				c.travaOS()
				c.store.EXPECT().GetReparoAdicionalForUpdate(gomock.Any(), gomock.Any()).
					Return(fxReparo(c.os.ID, ordemservico.ReparoPendente), nil)
			},
			wantErr: ordemservico.ErrReparoForaDeExecucao,
		},
		{
			name:      "ReparoNaoEncontrado",
			status:    ordemservico.StatusEmExecucao,
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			buildStubs: func(c *fxCenario) {
				c.travaOS()
				c.store.EXPECT().GetReparoAdicionalForUpdate(gomock.Any(), gomock.Any()).
					Return(db.ReparoAdicional{}, sql.ErrNoRows)
			},
			wantErr: service.ErrReparoNaoEncontrado,
		},
		{
			name:      "DocumentoErrado",
			status:    ordemservico.StatusEmExecucao,
			documento: func(c *fxCenario) string { return util.RandomCPF() },
			buildStubs: func(c *fxCenario) {
				c.travaOS()
			},
			wantErr: service.ErrOSNaoEncontrada,
		},
		{
			name:      "OSNaoEncontrada",
			status:    ordemservico.StatusEmExecucao,
			documento: func(c *fxCenario) string { return c.cliente.Documento },
			buildStubs: func(c *fxCenario) {
				c.store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)
			},
			wantErr: service.ErrOSNaoEncontrada,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		for _, autorizar := range []bool{true, false} {
			t.Run(tc.name, func(t *testing.T) {
				c := novoCenario(t, tc.status, ordemservico.OrcamentoAprovado)
				fxExecTx(c.store)
				c.leitura(nil, nil, nil)
				tc.buildStubs(c)
				c.store.EXPECT().UpdateReparoAdicionalStatus(gomock.Any(), gomock.Any()).Times(0)

				var err error
				if autorizar {
					_, err = c.svc.AutorizarReparoAdicional(context.Background(), c.os.Uuid, uuid.New(), tc.documento(c))
				} else {
					_, err = c.svc.RecusarReparoAdicional(context.Background(), c.os.Uuid, uuid.New(), tc.documento(c))
				}
				require.True(t, errors.Is(err, tc.wantErr), "got %v", err)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Consultas
// ---------------------------------------------------------------------------

func TestConsultarAndamento(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	c.os.ValorTotalCentavos = 12345
	c.leitura(nil, nil, nil)

	rsp, err := c.svc.ConsultarAndamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, "Em execução", rsp.StatusDescricao)
	require.Equal(t, c.veiculo.Placa, rsp.Veiculo.Placa)
	require.Equal(t, int64(12345), rsp.Orcamento.TotalCentavos)

	_, err = c.svc.ConsultarAndamento(context.Background(), c.os.Uuid, util.RandomCNPJ())
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestConsultarAndamentoOSNaoEncontrada(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	c.store.EXPECT().GetOrdemServico(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)

	_, err := c.svc.ConsultarAndamento(context.Background(), c.os.Uuid, c.cliente.Documento)
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))
}

func TestBuscarOrdemServicoErros(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	c.store.EXPECT().GetOrdemServico(gomock.Any(), gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)
	_, err := c.svc.BuscarOrdemServico(context.Background(), c.os.Uuid)
	require.True(t, errors.Is(err, service.ErrOSNaoEncontrada))

	c = novoCenario(t, ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	c.store.EXPECT().GetOrdemServico(gomock.Any(), gomock.Any()).Return(c.os, nil)
	c.store.EXPECT().GetClienteByID(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrConnDone)
	_, err = c.svc.BuscarOrdemServico(context.Background(), c.os.Uuid)
	require.ErrorIs(t, err, sql.ErrConnDone)
}

func TestListarOrdensServico(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	row := db.ListOrdensServicoRow{
		OrdemServico: c.os,
		ClienteUuid:  c.cliente.Uuid,
		ClienteNome:  c.cliente.Nome,
		VeiculoUuid:  c.veiculo.Uuid,
		VeiculoPlaca: c.veiculo.Placa,
	}

	t.Run("SemFiltroDeCliente", func(t *testing.T) {
		c.store.EXPECT().ListOrdensServico(gomock.Any(), db.ListOrdensServicoParams{
			Status:       "RECEBIDA",
			Limite:       10,
			Deslocamento: 10,
		}).Return([]db.ListOrdensServicoRow{row}, nil)

		rsp, err := c.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{
			PaginacaoQuery: dto.PaginacaoQuery{Pagina: 2, Tamanho: 10},
			Status:         "RECEBIDA",
		})
		require.NoError(t, err)
		require.Len(t, rsp, 1)
		require.Equal(t, "Recebida", rsp[0].StatusDescricao)
		require.Equal(t, c.veiculo.Placa, rsp[0].VeiculoPlaca)
	})

	t.Run("ComCliente", func(t *testing.T) {
		c.store.EXPECT().GetCliente(gomock.Any(), c.cliente.Uuid).Return(c.cliente, nil)
		c.store.EXPECT().ListOrdensServico(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.ListOrdensServicoParams) ([]db.ListOrdensServicoRow, error) {
				require.True(t, arg.ClienteID.Valid)
				require.Equal(t, c.cliente.ID, arg.ClienteID.Int64)
				return []db.ListOrdensServicoRow{}, nil
			})

		rsp, err := c.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{ClienteUuid: c.cliente.Uuid.String()})
		require.NoError(t, err)
		require.Empty(t, rsp)
	})

	t.Run("ClienteUUIDInvalido", func(t *testing.T) {
		_, err := c.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{ClienteUuid: "abc"})
		require.True(t, errors.Is(err, service.ErrClienteNaoEncontrado))
	})

	t.Run("ClienteNaoEncontrado", func(t *testing.T) {
		c.store.EXPECT().GetCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrNoRows)
		_, err := c.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{ClienteUuid: uuid.NewString()})
		require.True(t, errors.Is(err, service.ErrClienteNaoEncontrado))
	})

	t.Run("ErroNoBanco", func(t *testing.T) {
		c.store.EXPECT().ListOrdensServico(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
		_, err := c.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{})
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestTempoMedioExecucao(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusFinalizada, ordemservico.OrcamentoAprovado)
	servicoUUID := uuid.New()
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	fimExclusivo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local)

	c.store.EXPECT().GetTempoMedioExecucao(gomock.Any(), db.GetTempoMedioExecucaoParams{
		DataInicio: sql.NullTime{Time: inicio, Valid: true},
		DataFim:    sql.NullTime{Time: fimExclusivo, Valid: true},
	}).Return(db.GetTempoMedioExecucaoRow{TotalOrdens: 2, MediaSegundos: 5400, MinimoSegundos: 1800, MaximoSegundos: 9000}, nil)
	c.store.EXPECT().ListTempoMedioExecucaoPorServico(gomock.Any(), gomock.Any()).
		Return([]db.ListTempoMedioExecucaoPorServicoRow{
			{ServicoUuid: servicoUUID, ServicoNome: "Troca de óleo", TotalOrdens: 1, MediaSegundos: 600},
			{ServicoUuid: uuid.New(), ServicoNome: "Rápido", TotalOrdens: 1, MediaSegundos: 45},
		}, nil)

	rsp, err := c.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataInicio: "2026-01-01", DataFim: "2026-01-31"})
	require.NoError(t, err)
	require.Equal(t, int64(2), rsp.TotalOrdens)
	require.Equal(t, int64(5400), rsp.MediaSegundos)
	require.Equal(t, "1h 30min", rsp.MediaFormatada)
	require.Equal(t, int64(1800), rsp.MinimoSegundos)
	require.Equal(t, int64(9000), rsp.MaximoSegundos)
	require.Len(t, rsp.PorServico, 2)
	require.Equal(t, servicoUUID, rsp.PorServico[0].ServicoUuid)
	require.Equal(t, "10min", rsp.PorServico[0].MediaFormatada)
	require.Equal(t, "45s", rsp.PorServico[1].MediaFormatada)
}

func TestTempoMedioExecucaoSemFiltro(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusFinalizada, ordemservico.OrcamentoAprovado)
	c.store.EXPECT().GetTempoMedioExecucao(gomock.Any(), db.GetTempoMedioExecucaoParams{}).Return(db.GetTempoMedioExecucaoRow{}, nil)
	c.store.EXPECT().ListTempoMedioExecucaoPorServico(gomock.Any(), db.ListTempoMedioExecucaoPorServicoParams{}).
		Return([]db.ListTempoMedioExecucaoPorServicoRow{}, nil)

	rsp, err := c.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(0), rsp.TotalOrdens)
	require.Equal(t, "0s", rsp.MediaFormatada)
	require.Empty(t, rsp.PorServico)
}

func TestTempoMedioExecucaoErros(t *testing.T) {
	c := novoCenario(t, ordemservico.StatusFinalizada, ordemservico.OrcamentoAprovado)

	_, err := c.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataInicio: "01/01/2026"})
	require.True(t, errors.Is(err, service.ErrDataInvalida))

	_, err = c.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataFim: "2026-13-01"})
	require.True(t, errors.Is(err, service.ErrDataInvalida))

	_, err = c.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataInicio: "2026-03-01", DataFim: "2026-01-31"})
	require.True(t, errors.Is(err, service.ErrIntervaloDatasInvalido))

	c.store.EXPECT().GetTempoMedioExecucao(gomock.Any(), gomock.Any()).Return(db.GetTempoMedioExecucaoRow{}, sql.ErrConnDone)
	_, err = c.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{})
	require.ErrorIs(t, err, sql.ErrConnDone)

	c.store.EXPECT().GetTempoMedioExecucao(gomock.Any(), gomock.Any()).Return(db.GetTempoMedioExecucaoRow{}, nil)
	c.store.EXPECT().ListTempoMedioExecucaoPorServico(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
	_, err = c.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{})
	require.ErrorIs(t, err, sql.ErrConnDone)
}
