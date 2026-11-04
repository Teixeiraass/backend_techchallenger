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
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Os testes do serviço da OS usam os mocks das portas (unidade de trabalho,
// repositórios e catálogo): verificam que cada comando carrega o agregado
// travado, delega a regra à raiz, anota o histórico e salva. A persistência
// real é testada em internal/repository e em db/sqlc.

// ---------------------------------------------------------------------------
// Abertura
// ---------------------------------------------------------------------------

func TestAbrirOrdemServicoComItens(t *testing.T) {
	a := novaApp(t)
	servico := fxServico(true)
	peca := fxPeca(10, 2)
	salvos := a.salva()

	a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), a.cliente.Documento).Return(a.cliente, nil)
	a.catalogo.EXPECT().VeiculoPorPlaca(gomock.Any(), a.veiculo.Placa).Return(a.veiculo, nil)
	a.catalogo.EXPECT().Servico(gomock.Any(), servico.UUID).Return(servico, nil)
	a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(peca, nil)
	a.osRepo.EXPECT().Carregar(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, osUUID uuid.UUID) (*ordemservico.OrdemServico, error) {
			require.Equal(t, (*salvos)[0].UUID, osUUID)
			os := (*salvos)[0]
			return &os, nil
		})
	a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(a.cliente, nil)
	a.catalogo.EXPECT().VeiculoPorID(gomock.Any(), a.veiculo.ID).Return(a.veiculo, nil)
	a.catalogo.EXPECT().HistoricoDaOS(gomock.Any(), gomock.Any()).Return(nil, nil)

	rsp, err := a.svc.AbrirOrdemServico(context.Background(), dto.AbrirOrdemServicoRequest{
		Documento:         a.cliente.Documento,
		Placa:             a.veiculo.Placa,
		DescricaoProblema: "Barulho",
		Servicos:          []dto.ItemServicoRequest{{ServicoUuid: servico.UUID}},
		Pecas:             []dto.ItemPecaRequest{{PecaUuid: peca.UUID, Quantidade: 2}},
	}, 7)
	require.NoError(t, err)
	require.Equal(t, string(ordemservico.StatusRecebida), rsp.Status)
	require.Len(t, rsp.Servicos, 1)
	require.Len(t, rsp.Pecas, 1)
	require.Equal(t, int64(10000+2*2500), rsp.Orcamento.TotalCentavos)

	salvo := (*salvos)[0]
	require.Equal(t, a.cliente.ID, salvo.ClienteID)
	require.Equal(t, a.veiculo.ID, salvo.VeiculoID)
	require.Equal(t, "Barulho", salvo.DescricaoProblema)
	historico := requireHistorico(t, salvo, "OS aberta")
	require.Equal(t, ordemservico.Status(""), historico.StatusAnterior)
	require.Equal(t, int64(7), historico.UsuarioID)
	require.Equal(t, 1, a.transacoes)

	// "OS recebida" sai com o UUID gerado ao salvar.
	require.Equal(t, []string{ordemservico.NomeOSRecebida}, a.eventos.nomes())
	require.Equal(t, salvo.UUID, a.eventos.eventos[0].(ordemservico.OSRecebida).OSUUID)
}

func TestAbrirOrdemServicoErros(t *testing.T) {
	servico := fxServico(true)
	peca := fxPeca(10, 1)
	testCases := []struct {
		name       string
		documento  func(a *fxApp) string
		placa      func(a *fxApp) string
		buildStubs func(a *fxApp)
		pecas      bool
		servicos   bool
		wantErr    error
	}{
		{name: "DocumentoInvalido", documento: func(*fxApp) string { return "123" }, placa: func(a *fxApp) string { return a.veiculo.Placa }, buildStubs: func(*fxApp) {}, wantErr: cliente.ErrDocumentoInvalido},
		{name: "PlacaInvalida", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(*fxApp) string { return "12-ABC" }, buildStubs: func(*fxApp) {}, wantErr: veiculo.ErrPlacaInvalida},
		{
			name: "ClienteNaoEncontrado", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(a *fxApp) string { return a.veiculo.Placa },
			buildStubs: func(a *fxApp) {
				a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), gomock.Any()).Return(readmodel.Cliente{}, service.ErrClienteNaoEncontrado)
			},
			wantErr: service.ErrClienteNaoEncontrado,
		},
		{
			name: "VeiculoNaoEncontrado", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(a *fxApp) string { return a.veiculo.Placa },
			buildStubs: func(a *fxApp) {
				a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorPlaca(gomock.Any(), gomock.Any()).Return(readmodel.Veiculo{}, service.ErrVeiculoNaoEncontrado)
			},
			wantErr: service.ErrVeiculoNaoEncontrado,
		},
		{
			name: "VeiculoDeOutroCliente", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(a *fxApp) string { return a.veiculo.Placa },
			buildStubs: func(a *fxApp) {
				outro := a.veiculo
				outro.ClienteID = a.cliente.ID + 1
				a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorPlaca(gomock.Any(), gomock.Any()).Return(outro, nil)
			},
			wantErr: service.ErrVeiculoDeOutroCliente,
		},
		{
			name: "ServicoInativo", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(a *fxApp) string { return a.veiculo.Placa }, servicos: true,
			buildStubs: func(a *fxApp) {
				inativo := servico
				inativo.Ativo = false
				a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorPlaca(gomock.Any(), gomock.Any()).Return(a.veiculo, nil)
				a.catalogo.EXPECT().Servico(gomock.Any(), servico.UUID).Return(inativo, nil)
			},
			wantErr: service.ErrServicoInativo,
		},
		{
			name: "ServicoNaoEncontrado", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(a *fxApp) string { return a.veiculo.Placa }, servicos: true,
			buildStubs: func(a *fxApp) {
				a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorPlaca(gomock.Any(), gomock.Any()).Return(a.veiculo, nil)
				a.catalogo.EXPECT().Servico(gomock.Any(), servico.UUID).Return(ordemservico.CatalogoServico{}, service.ErrServicoNaoEncontrado)
			},
			wantErr: service.ErrServicoNaoEncontrado,
		},
		{
			name: "PecaSemEstoque", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(a *fxApp) string { return a.veiculo.Placa }, pecas: true,
			buildStubs: func(a *fxApp) {
				pouca := *peca
				pouca.Saldo = 1
				a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorPlaca(gomock.Any(), gomock.Any()).Return(a.veiculo, nil)
				a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(&pouca, nil)
			},
			wantErr: estoque.ErrEstoqueInsuficiente,
		},
		{
			name: "PecaNaoEncontrada", documento: func(a *fxApp) string { return a.cliente.Documento }, placa: func(a *fxApp) string { return a.veiculo.Placa }, pecas: true,
			buildStubs: func(a *fxApp) {
				a.catalogo.EXPECT().ClientePorDocumento(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorPlaca(gomock.Any(), gomock.Any()).Return(a.veiculo, nil)
				a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(nil, service.ErrPecaNaoEncontrada)
			},
			wantErr: service.ErrPecaNaoEncontrada,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			a := novaApp(t)
			tc.buildStubs(a)
			a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)

			req := dto.AbrirOrdemServicoRequest{Documento: tc.documento(a), Placa: tc.placa(a)}
			if tc.servicos {
				req.Servicos = []dto.ItemServicoRequest{{ServicoUuid: servico.UUID}}
			}
			if tc.pecas {
				req.Pecas = []dto.ItemPecaRequest{{PecaUuid: peca.UUID, Quantidade: 5}}
			}

			rsp, err := a.svc.AbrirOrdemServico(context.Background(), req, 1)
			require.True(t, errors.Is(err, tc.wantErr), "got %v", err)
			require.Nil(t, rsp)
		})
	}
}

// ---------------------------------------------------------------------------
// Itens
// ---------------------------------------------------------------------------

func TestAdicionarServico(t *testing.T) {
	a := novaApp(t)
	servico := fxServico(true)
	os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	os.Itens = []ordemservico.Item{fxItemServico(servico, 2)}
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()
	a.catalogo.EXPECT().Servico(gomock.Any(), servico.UUID).Return(servico, nil)

	rsp, err := a.svc.AdicionarServico(context.Background(), os.UUID, dto.ItemServicoRequest{ServicoUuid: servico.UUID, Quantidade: 2})
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.Len(t, *salvos, 1)
	require.Len(t, (*salvos)[0].Itens, 2)
	require.Equal(t, int64(40000), (*salvos)[0].ValorTotalCentavos, "2 já na OS + 2 novos, a 10000")
	require.Empty(t, (*salvos)[0].Alteracoes().Historico, "alteração de item não entra no histórico")
}

func TestAdicionarServicoErros(t *testing.T) {
	t.Run("StatusNaoEditavel", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		a.trava(os)
		a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
		_, err := a.svc.AdicionarServico(context.Background(), os.UUID, dto.ItemServicoRequest{ServicoUuid: uuid.New()})
		require.ErrorIs(t, err, ordemservico.ErrItensNaoEditaveis)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.AdicionarServico(context.Background(), uuid.New(), dto.ItemServicoRequest{ServicoUuid: uuid.New()})
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
	t.Run("ServicoNaoEncontrado", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
		a.trava(os)
		a.catalogo.EXPECT().Servico(gomock.Any(), gomock.Any()).Return(ordemservico.CatalogoServico{}, service.ErrServicoNaoEncontrado)
		_, err := a.svc.AdicionarServico(context.Background(), os.UUID, dto.ItemServicoRequest{ServicoUuid: uuid.New()})
		require.ErrorIs(t, err, service.ErrServicoNaoEncontrado)
	})
	t.Run("ErroAoSalvar", func(t *testing.T) {
		a := novaApp(t)
		servico := fxServico(true)
		os := a.os(ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
		a.trava(os)
		a.catalogo.EXPECT().Servico(gomock.Any(), gomock.Any()).Return(servico, nil)
		a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Return(sql.ErrConnDone)
		_, err := a.svc.AdicionarServico(context.Background(), os.UUID, dto.ItemServicoRequest{ServicoUuid: servico.UUID})
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestRemoverServico(t *testing.T) {
	a := novaApp(t)
	item := fxItemServico(fxServico(true), 1)
	os := a.os(ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	os.Itens = []ordemservico.Item{item}
	os.ValorTotalCentavos = 10000
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()

	_, err := a.svc.RemoverServico(context.Background(), os.UUID, item.UUID)
	require.NoError(t, err)
	require.Empty(t, (*salvos)[0].Itens)
	require.Equal(t, int64(0), (*salvos)[0].ValorTotalCentavos)
	require.Len(t, (*salvos)[0].Alteracoes().ItensRemovidos, 1)
}

func TestRemoverServicoItemNaoEncontrado(t *testing.T) {
	a := novaApp(t)
	os := a.os(ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	a.trava(os)
	a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)

	_, err := a.svc.RemoverServico(context.Background(), os.UUID, uuid.New())
	require.ErrorIs(t, err, service.ErrItemNaoEncontrado)
}

func TestAdicionarPecaConsideraQuantidadeJaNaOS(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(5, 1)
	os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 4)}
	a.trava(os)
	a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(peca, nil)
	a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)

	// 4 já na OS + 2 pedidas > saldo 5
	_, err := a.svc.AdicionarPeca(context.Background(), os.UUID, dto.ItemPecaRequest{PecaUuid: peca.UUID, Quantidade: 2})
	require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
	require.Contains(t, err.Error(), peca.Codigo)
}

func TestAdicionarPecaOK(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(5, 1)
	os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 1)}
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()
	a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(peca, nil)

	_, err := a.svc.AdicionarPeca(context.Background(), os.UUID, dto.ItemPecaRequest{PecaUuid: peca.UUID, Quantidade: 2})
	require.NoError(t, err)
	require.Equal(t, int64(7500), (*salvos)[0].ValorTotalCentavos, "1 já na OS + 2 novas, a 2500")
}

func TestAdicionarPecaErros(t *testing.T) {
	t.Run("Inativa", func(t *testing.T) {
		a := novaApp(t)
		peca := fxPeca(5, 1)
		peca.Ativa = false
		os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
		a.trava(os)
		a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(peca, nil)
		_, err := a.svc.AdicionarPeca(context.Background(), os.UUID, dto.ItemPecaRequest{PecaUuid: peca.UUID, Quantidade: 1})
		require.ErrorIs(t, err, estoque.ErrPecaInativa)
	})
	t.Run("NaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
		a.trava(os)
		a.pecas.EXPECT().Carregar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err := a.svc.AdicionarPeca(context.Background(), os.UUID, dto.ItemPecaRequest{PecaUuid: uuid.New(), Quantidade: 1})
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
	t.Run("StatusNaoEditavel", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		a.trava(os)
		_, err := a.svc.AdicionarPeca(context.Background(), os.UUID, dto.ItemPecaRequest{PecaUuid: uuid.New(), Quantidade: 1})
		require.ErrorIs(t, err, ordemservico.ErrItensNaoEditaveis)
	})
}

func TestRemoverPeca(t *testing.T) {
	a := novaApp(t)
	item := fxItemPeca(fxPeca(5, 1), 2)
	os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	os.Itens = []ordemservico.Item{item}
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()

	_, err := a.svc.RemoverPeca(context.Background(), os.UUID, item.UUID)
	require.NoError(t, err)
	require.Empty(t, (*salvos)[0].Itens)

	a.trava(os)
	_, err = a.svc.RemoverPeca(context.Background(), os.UUID, uuid.New())
	require.ErrorIs(t, err, service.ErrItemNaoEncontrado)
}

// ---------------------------------------------------------------------------
// Transições
// ---------------------------------------------------------------------------

func TestIniciarDiagnostico(t *testing.T) {
	a := novaApp(t)
	os := a.os(ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()

	_, err := a.svc.IniciarDiagnostico(context.Background(), os.UUID, 3)
	require.NoError(t, err)
	salvo := (*salvos)[0]
	require.Equal(t, ordemservico.StatusEmDiagnostico, salvo.Status)
	require.NotNil(t, salvo.DiagnosticoIniciadoEm)
	historico := requireHistorico(t, salvo, "Diagnóstico iniciado")
	require.Equal(t, ordemservico.StatusRecebida, historico.StatusAnterior)
	require.Equal(t, ordemservico.StatusEmDiagnostico, historico.StatusNovo)
	require.Equal(t, int64(3), historico.UsuarioID)
	require.Equal(t, []string{ordemservico.NomeDiagnosticoIniciado}, a.eventos.nomes())
}

func TestIniciarDiagnosticoErros(t *testing.T) {
	t.Run("TransicaoInvalida", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		a.trava(os)
		a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
		_, err := a.svc.IniciarDiagnostico(context.Background(), os.UUID, 3)
		require.ErrorIs(t, err, ordemservico.ErrTransicaoInvalida)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.IniciarDiagnostico(context.Background(), uuid.New(), 3)
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
}

func TestEnviarOrcamento(t *testing.T) {
	a := novaApp(t)
	os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	os.Itens = []ordemservico.Item{fxItemServico(fxServico(true), 1), fxItemPeca(fxPeca(10, 1), 2)}
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()

	_, err := a.svc.EnviarOrcamento(context.Background(), os.UUID, dto.EnviarOrcamentoRequest{Diagnostico: "Pastilhas gastas"}, 2)
	require.NoError(t, err)
	salvo := (*salvos)[0]
	require.Equal(t, ordemservico.StatusAguardandoAprovacao, salvo.Status)
	require.Equal(t, ordemservico.OrcamentoAguardando, salvo.StatusOrcamento)
	require.Equal(t, int64(15000), salvo.ValorTotalCentavos)
	require.Equal(t, "Pastilhas gastas", salvo.Diagnostico)
	require.NotNil(t, salvo.OrcamentoEnviadoEm)
	requireHistorico(t, salvo, "Orçamento enviado ao cliente")

	// Depois do commit: "Orçamento enviado ao cliente" é publicado com o valor.
	require.Equal(t, []string{ordemservico.NomeOrcamentoEnviadoAoCliente}, a.eventos.nomes())
	enviado := a.eventos.eventos[0].(ordemservico.OrcamentoEnviadoAoCliente)
	require.Equal(t, os.UUID, enviado.OSUUID)
	require.Equal(t, a.cliente.ID, enviado.ClienteID)
	require.Equal(t, int64(15000), enviado.ValorTotalCentavos)
	require.Equal(t, 1, a.eventos.publicacoes)
}

func TestEnviarOrcamentoErros(t *testing.T) {
	t.Run("SemItens", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
		a.trava(os)
		_, err := a.svc.EnviarOrcamento(context.Background(), os.UUID, dto.EnviarOrcamentoRequest{}, 2)
		require.ErrorIs(t, err, ordemservico.ErrOrcamentoSemItens)
		require.Zero(t, a.eventos.publicacoes, "sem commit, nada é publicado")
	})
	t.Run("TransicaoInvalida", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusRecebida, ordemservico.OrcamentoNaoEnviado)
		os.Itens = []ordemservico.Item{fxItemServico(fxServico(true), 1)}
		a.trava(os)
		_, err := a.svc.EnviarOrcamento(context.Background(), os.UUID, dto.EnviarOrcamentoRequest{}, 2)
		require.ErrorIs(t, err, ordemservico.ErrTransicaoInvalida)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.EnviarOrcamento(context.Background(), uuid.New(), dto.EnviarOrcamentoRequest{}, 2)
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
}

// Os eventos saem depois do commit, antes da leitura da resposta: mesmo que
// a resposta falhe, a OS já foi salva e o evento já foi publicado.
func TestEnviarOrcamentoPublicaDepoisDoCommit(t *testing.T) {
	a := novaApp(t)
	os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
	os.Itens = []ordemservico.Item{fxItemServico(fxServico(true), 1)}
	a.trava(os)
	a.salva()
	a.osRepo.EXPECT().Carregar(gomock.Any(), os.UUID).Return(os, nil)
	a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(readmodel.Cliente{}, sql.ErrConnDone)

	_, err := a.svc.EnviarOrcamento(context.Background(), os.UUID, dto.EnviarOrcamentoRequest{}, 2)
	require.ErrorIs(t, err, sql.ErrConnDone, "a resposta falha, mas a OS já foi salva")
	require.Equal(t, []string{ordemservico.NomeOrcamentoEnviadoAoCliente}, a.eventos.nomes())
	require.Empty(t, os.Eventos(), "eventos limpos depois de publicados")
}

// ---------------------------------------------------------------------------
// Aprovação, rejeição e execução
// ---------------------------------------------------------------------------

func TestAprovarOrcamentoIniciaExecucaoEBaixaEstoque(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(3, 2)
	reparo := fxReparo(ordemservico.ReparoPendente)
	os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 2), fxDoReparo(fxItemPeca(peca, 1), reparo)}
	os.Reparos = []ordemservico.ReparoAdicional{reparo}
	// Duas transações: aprovação e início da execução, cada uma trava a OS.
	a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), os.UUID).Times(2).Return(os, nil)
	a.leitura(os)
	salvos := a.salva()
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(peca, nil)
	a.pecas.EXPECT().AplicarMovimento(gomock.Any(), peca, gomock.Any()).
		DoAndReturn(func(_ context.Context, p *estoque.Peca, mov estoque.MovimentoEstoque) error {
			require.Equal(t, estoque.MovimentoSaida, mov.Tipo)
			require.Equal(t, int32(-2), mov.Quantidade, "só a peça do orçamento; a do reparo pendente não")
			require.Equal(t, int32(3), mov.SaldoAnterior)
			require.Equal(t, int32(1), mov.SaldoPosterior)
			require.Equal(t, os.ID, mov.OrdemServicoID)
			return nil
		})

	rsp, err := a.svc.AprovarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
	require.NoError(t, err)
	require.Empty(t, rsp.Aviso)
	require.Len(t, *salvos, 2)
	require.Equal(t, ordemservico.OrcamentoAprovado, (*salvos)[0].StatusOrcamento)
	require.Equal(t, ordemservico.StatusAguardandoAprovacao, (*salvos)[0].Status)
	requireHistorico(t, (*salvos)[0], "Orçamento aprovado pelo cliente")
	require.Equal(t, ordemservico.StatusEmExecucao, (*salvos)[1].Status)
	require.NotNil(t, (*salvos)[1].ExecucaoIniciadaEm)
	requireHistorico(t, (*salvos)[1], "Execução iniciada e estoque baixado")
	require.Equal(t, 2, a.transacoes)

	// Uma publicação por transação: aprovação; execução + baixa (saldo 1 <=
	// mínimo 2 também detecta estoque baixo, que dispara o pedido de reposição).
	require.Equal(t, 2, a.eventos.publicacoes)
	require.Equal(t, []string{
		ordemservico.NomeOrcamentoAprovado,
		ordemservico.NomeExecucaoIniciada, estoque.NomeEstoqueBaixado, estoque.NomeEstoqueBaixoDetectado,
	}, a.eventos.nomes())
	baixado := a.eventos.eventos[2].(estoque.EstoqueBaixado)
	require.Equal(t, peca.Codigo, baixado.Codigo)
	require.Equal(t, os.ID, baixado.OrdemServicoID)
}

func TestAprovarOrcamentoEstoqueInsuficienteMantemAprovacao(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(0, 2)
	os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 2)}
	a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), os.UUID).Times(2).Return(os, nil)
	a.leitura(os)
	salvos := a.salva()
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(peca, nil)
	a.pecas.EXPECT().AplicarMovimento(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	rsp, err := a.svc.AprovarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
	require.NoError(t, err)
	require.Contains(t, rsp.Aviso, "a oficina inicia o serviço")
	require.Len(t, *salvos, 1, "só a aprovação é gravada")
	require.Equal(t, ordemservico.OrcamentoAprovado, (*salvos)[0].StatusOrcamento)

	// A execução foi desfeita, mas "Estoque insuficiente detectado" é publicado
	// mesmo assim: sem o pedido de reposição a OS ficaria parada.
	require.Equal(t, []string{ordemservico.NomeOrcamentoAprovado, estoque.NomeEstoqueInsuficienteDetectado}, a.eventos.nomes())
	insuficiente := a.eventos.eventos[1].(estoque.EstoqueInsuficienteDetectado)
	require.Equal(t, peca.Codigo, insuficiente.Codigo)
	require.Equal(t, os.ID, insuficiente.OrdemServicoID)
	require.Equal(t, int32(2), insuficiente.Necessario)
}

// O saldo acabou entre a leitura e a gravação: o UPDATE condicional do
// repositório devolve ErrEstoqueInsuficiente e a execução não começa.
func TestIniciarExecucaoConcorrenciaNaBaixa(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(5, 1)
	os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 2)}
	a.trava(os)
	a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(peca, nil)
	a.pecas.EXPECT().AplicarMovimento(gomock.Any(), peca, gomock.Any()).Return(estoque.ErrEstoqueInsuficientePara(peca.Nome, peca.Codigo))

	_, err := a.svc.IniciarExecucao(context.Background(), os.UUID, 4)
	require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
	// O agregado aceitou a baixa (saldo 5 em memória); o banco recusou. Os
	// eventos da peça (baixa registrada em memória) vão para a política mesmo
	// assim, para o administrador saber da falta.
	require.Equal(t, 1, a.eventos.publicacoes)
}

// Repetir a aprovação com a execução ainda não iniciada (aprovação gravada,
// faltou peça) não dá 409: o serviço tenta iniciar a execução de novo.
func TestAprovarOrcamentoRepetidaTentaIniciarExecucao(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(5, 1)
	os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 2)}
	a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), os.UUID).Times(2).Return(os, nil)
	a.leitura(os)
	salvos := a.salva()
	a.baixaOK(peca)

	rsp, err := a.svc.AprovarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
	require.NoError(t, err)
	require.Empty(t, rsp.Aviso)
	// Só a execução é gravada: a aprovação não entra de novo no histórico nem
	// gera evento de novo.
	require.Len(t, *salvos, 1)
	require.Equal(t, ordemservico.StatusEmExecucao, (*salvos)[0].Status)
	require.Len(t, (*salvos)[0].Alteracoes().Historico, 1)
	require.Equal(t, ordemservico.StatusEmExecucao, (*salvos)[0].Alteracoes().Historico[0].StatusNovo)
	require.Equal(t, []string{ordemservico.NomeExecucaoIniciada, estoque.NomeEstoqueBaixado}, a.eventos.nomes())
}

func TestAprovarOrcamentoRepetidaComExecucaoIniciada(t *testing.T) {
	for _, status := range []ordemservico.Status{ordemservico.StatusEmExecucao, ordemservico.StatusFinalizada} {
		t.Run(string(status), func(t *testing.T) {
			a := novaApp(t)
			os := a.os(status, ordemservico.OrcamentoAprovado)
			a.trava(os)
			a.leitura(os)
			a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
			a.pecas.EXPECT().AplicarMovimento(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			rsp, err := a.svc.AprovarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
			require.NoError(t, err)
			require.Equal(t, "Orçamento já aprovado", rsp.Aviso)
			require.Equal(t, string(status), rsp.Status)
			require.Empty(t, a.eventos.eventos, "nada aconteceu, nenhum evento")
		})
	}
}

func TestAprovarOrcamentoErros(t *testing.T) {
	t.Run("DocumentoErrado", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		a.trava(os)
		a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(a.cliente, nil)
		a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
		_, err := a.svc.AprovarOrcamento(context.Background(), os.UUID, util.RandomCPF())
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
	t.Run("NaoAguardando", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
		a.trava(os)
		a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(a.cliente, nil)
		_, err := a.svc.AprovarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
		require.ErrorIs(t, err, ordemservico.ErrOrcamentoNaoAguardando)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.AprovarOrcamento(context.Background(), uuid.New(), a.cliente.Documento)
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
	t.Run("DonoNaoCarregado", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		a.trava(os)
		a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(readmodel.Cliente{}, sql.ErrConnDone)
		_, err := a.svc.AprovarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestRejeitarOrcamento(t *testing.T) {
	a := novaApp(t)
	os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()

	_, err := a.svc.RejeitarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, ordemservico.StatusEmDiagnostico, (*salvos)[0].Status)
	require.Equal(t, ordemservico.OrcamentoRejeitado, (*salvos)[0].StatusOrcamento)
	requireHistorico(t, (*salvos)[0], "Orçamento rejeitado pelo cliente")
	require.Equal(t, []string{ordemservico.NomeOrcamentoRejeitado}, a.eventos.nomes())
}

func TestRejeitarOrcamentoErros(t *testing.T) {
	t.Run("DocumentoErrado", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		a.trava(os)
		a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(a.cliente, nil)
		_, err := a.svc.RejeitarOrcamento(context.Background(), os.UUID, "000")
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
	t.Run("NaoAguardando", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		a.trava(os)
		a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(a.cliente, nil)
		_, err := a.svc.RejeitarOrcamento(context.Background(), os.UUID, a.cliente.Documento)
		require.ErrorIs(t, err, ordemservico.ErrOrcamentoNaoAguardando)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.RejeitarOrcamento(context.Background(), uuid.New(), a.cliente.Documento)
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
}

func TestIniciarExecucaoSemPecas(t *testing.T) {
	a := novaApp(t)
	os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), gomock.Any()).Times(0)

	_, err := a.svc.IniciarExecucao(context.Background(), os.UUID, 4)
	require.NoError(t, err)
	require.Equal(t, ordemservico.StatusEmExecucao, (*salvos)[0].Status)
	require.Equal(t, []string{ordemservico.NomeExecucaoIniciada}, a.eventos.nomes())
}

func TestIniciarExecucaoErros(t *testing.T) {
	t.Run("OrcamentoNaoAprovado", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAguardando)
		a.trava(os)
		_, err := a.svc.IniciarExecucao(context.Background(), os.UUID, 4)
		require.ErrorIs(t, err, ordemservico.ErrOrcamentoNaoAprovado)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.IniciarExecucao(context.Background(), uuid.New(), 4)
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
	t.Run("PecaNaoCarregada", func(t *testing.T) {
		a := novaApp(t)
		peca := fxPeca(5, 1)
		os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
		os.Itens = []ordemservico.Item{fxItemPeca(peca, 1)}
		a.trava(os)
		a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(nil, sql.ErrConnDone)
		_, err := a.svc.IniciarExecucao(context.Background(), os.UUID, 4)
		require.ErrorIs(t, err, sql.ErrConnDone)
		require.Zero(t, a.eventos.publicacoes)
	})
	t.Run("ErroNoBancoNaBaixa", func(t *testing.T) {
		a := novaApp(t)
		peca := fxPeca(5, 1)
		os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
		os.Itens = []ordemservico.Item{fxItemPeca(peca, 1)}
		a.trava(os)
		a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(peca, nil)
		a.pecas.EXPECT().AplicarMovimento(gomock.Any(), peca, gomock.Any()).Return(sql.ErrConnDone)
		_, err := a.svc.IniciarExecucao(context.Background(), os.UUID, 4)
		require.ErrorIs(t, err, sql.ErrConnDone)
		require.Zero(t, a.eventos.publicacoes, "erro de banco não é falta de peça: nada é publicado")
	})
}

// Sem saldo em memória: o agregado recusa, a transação é desfeita e só o
// evento de estoque insuficiente é publicado.
func TestIniciarExecucaoSemEstoquePublicaFalta(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(1, 0)
	os := a.os(ordemservico.StatusAguardandoAprovacao, ordemservico.OrcamentoAprovado)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 3)}
	a.trava(os)
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(peca, nil)
	a.pecas.EXPECT().AplicarMovimento(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := a.svc.IniciarExecucao(context.Background(), os.UUID, 4)
	require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
	require.Equal(t, []string{estoque.NomeEstoqueInsuficienteDetectado}, a.eventos.nomes())
}

func TestFinalizarOrdemServico(t *testing.T) {
	a := novaApp(t)
	os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	inicio := time.Now().Add(-90 * time.Minute)
	os.ExecucaoIniciadaEm = &inicio
	os.Reparos = []ordemservico.ReparoAdicional{fxReparo(ordemservico.ReparoAutorizado)}
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()

	_, err := a.svc.FinalizarOrdemServico(context.Background(), os.UUID, 5)
	require.NoError(t, err)
	require.Equal(t, ordemservico.StatusFinalizada, (*salvos)[0].Status)
	require.NotNil(t, (*salvos)[0].FinalizadaEm)
	require.Contains(t, (*salvos)[0].Alteracoes().Historico[0].Observacao, "1h 30min")
	require.Equal(t, []string{ordemservico.NomeOSFinalizada}, a.eventos.nomes())
	require.Equal(t, 90*time.Minute, a.eventos.eventos[0].(ordemservico.OSFinalizada).TempoExecucao.Round(time.Minute))
}

func TestFinalizarOrdemServicoErros(t *testing.T) {
	t.Run("ReparoPendente", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		os.Reparos = []ordemservico.ReparoAdicional{fxReparo(ordemservico.ReparoPendente)}
		a.trava(os)
		_, err := a.svc.FinalizarOrdemServico(context.Background(), os.UUID, 5)
		require.ErrorIs(t, err, ordemservico.ErrReparoPendente)
		require.Zero(t, a.eventos.publicacoes)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.FinalizarOrdemServico(context.Background(), uuid.New(), 5)
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
}

func TestEntregarVeiculo(t *testing.T) {
	testCases := []struct {
		name       string
		status     ordemservico.Status
		orcamento  ordemservico.StatusOrcamento
		wantErr    error
		observacao string
	}{
		{name: "Finalizada", status: ordemservico.StatusFinalizada, orcamento: ordemservico.OrcamentoAprovado, observacao: "Veículo entregue ao cliente"},
		{name: "OrcamentoRejeitado", status: ordemservico.StatusEmDiagnostico, orcamento: ordemservico.OrcamentoRejeitado, observacao: "Veículo entregue sem execução (orçamento rejeitado)"},
		{name: "SemOrcamento", status: ordemservico.StatusEmDiagnostico, orcamento: ordemservico.OrcamentoNaoEnviado, wantErr: ordemservico.ErrEntregaNaoPermitida},
		{name: "EmExecucao", status: ordemservico.StatusEmExecucao, orcamento: ordemservico.OrcamentoAprovado, wantErr: ordemservico.ErrEntregaNaoPermitida},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			a := novaApp(t)
			os := a.os(tc.status, tc.orcamento)
			a.trava(os)
			a.leitura(os)
			salvos := a.salva()

			_, err := a.svc.EntregarVeiculo(context.Background(), os.UUID, 6)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Empty(t, *salvos)
				return
			}
			require.NoError(t, err)
			require.Equal(t, ordemservico.StatusEntregue, (*salvos)[0].Status)
			require.NotNil(t, (*salvos)[0].EntregueEm)
			requireHistorico(t, (*salvos)[0], tc.observacao)
			require.Equal(t, []string{ordemservico.NomeVeiculoEntregue}, a.eventos.nomes())
		})
	}

	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.EntregarVeiculo(context.Background(), uuid.New(), 6)
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
}

// ---------------------------------------------------------------------------
// Reparo adicional
// ---------------------------------------------------------------------------

func TestRegistrarReparoAdicional(t *testing.T) {
	a := novaApp(t)
	servico := fxServico(true)
	peca := fxPeca(4, 1)
	os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	// Item base já baixado (OS em execução) não conta; outro reparo pendente conta.
	outro := fxReparo(ordemservico.ReparoPendente)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 3), fxDoReparo(fxItemPeca(peca, 2), outro)}
	os.Reparos = []ordemservico.ReparoAdicional{outro}
	os.ValorTotalCentavos = 7500
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()
	a.catalogo.EXPECT().Servico(gomock.Any(), servico.UUID).Return(servico, nil)
	a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(peca, nil)

	_, err := a.svc.RegistrarReparoAdicional(context.Background(), os.UUID, dto.RegistrarReparoAdicionalRequest{
		Descricao: "Disco empenado",
		Servicos:  []dto.ItemServicoRequest{{ServicoUuid: servico.UUID}},
		Pecas:     []dto.ItemPecaRequest{{PecaUuid: peca.UUID, Quantidade: 2}},
	})
	require.NoError(t, err)
	salvo := (*salvos)[0]
	require.Len(t, salvo.Reparos, 2)
	require.Equal(t, "Disco empenado", salvo.Reparos[1].Descricao)
	require.Len(t, salvo.Itens, 4)
	require.Equal(t, int64(7500), salvo.ValorTotalCentavos, "reparo pendente fica fora do orçamento")
	// Sem mudança de status, a linha do histórico leva o nome do evento.
	historico := requireHistorico(t, salvo, ordemservico.NomeReparoAdicionalIdentificado+": Disco empenado")
	require.Equal(t, historico.StatusAnterior, historico.StatusNovo)

	// Depois do commit: "Reparo adicional identificado" chama o cliente a decidir.
	require.Equal(t, []string{ordemservico.NomeReparoAdicionalIdentificado}, a.eventos.nomes())
	require.Equal(t, "Disco empenado", a.eventos.eventos[0].(ordemservico.ReparoAdicionalIdentificado).Descricao)
}

func TestRegistrarReparoAdicionalErros(t *testing.T) {
	t.Run("SemItens", func(t *testing.T) {
		a := novaApp(t)
		_, err := a.svc.RegistrarReparoAdicional(context.Background(), uuid.New(), dto.RegistrarReparoAdicionalRequest{Descricao: "x"})
		require.ErrorIs(t, err, ordemservico.ErrReparoSemItens)
		require.Zero(t, a.transacoes)
	})
	t.Run("ForaDeExecucao", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmDiagnostico, ordemservico.OrcamentoNaoEnviado)
		a.trava(os)
		a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
		_, err := a.svc.RegistrarReparoAdicional(context.Background(), os.UUID, dto.RegistrarReparoAdicionalRequest{
			Descricao: "x", Servicos: []dto.ItemServicoRequest{{ServicoUuid: uuid.New()}},
		})
		require.ErrorIs(t, err, ordemservico.ErrReparoForaDeExecucao)
	})
	t.Run("OSNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
		_, err := a.svc.RegistrarReparoAdicional(context.Background(), uuid.New(), dto.RegistrarReparoAdicionalRequest{
			Descricao: "x", Pecas: []dto.ItemPecaRequest{{PecaUuid: uuid.New(), Quantidade: 1}},
		})
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
	t.Run("ServicoNaoEncontrado", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		a.trava(os)
		a.catalogo.EXPECT().Servico(gomock.Any(), gomock.Any()).Return(ordemservico.CatalogoServico{}, service.ErrServicoNaoEncontrado)
		_, err := a.svc.RegistrarReparoAdicional(context.Background(), os.UUID, dto.RegistrarReparoAdicionalRequest{
			Descricao: "x", Servicos: []dto.ItemServicoRequest{{ServicoUuid: uuid.New()}},
		})
		require.ErrorIs(t, err, service.ErrServicoNaoEncontrado)
	})
	t.Run("PecaNaoEncontrada", func(t *testing.T) {
		a := novaApp(t)
		os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		a.trava(os)
		a.pecas.EXPECT().Carregar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err := a.svc.RegistrarReparoAdicional(context.Background(), os.UUID, dto.RegistrarReparoAdicionalRequest{
			Descricao: "x", Pecas: []dto.ItemPecaRequest{{PecaUuid: uuid.New(), Quantidade: 1}},
		})
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
	t.Run("PecaSemEstoque", func(t *testing.T) {
		a := novaApp(t)
		peca := fxPeca(4, 1)
		os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
		outro := fxReparo(ordemservico.ReparoPendente)
		os.Itens = []ordemservico.Item{fxDoReparo(fxItemPeca(peca, 3), outro)}
		os.Reparos = []ordemservico.ReparoAdicional{outro}
		a.trava(os)
		a.pecas.EXPECT().Carregar(gomock.Any(), peca.UUID).Return(peca, nil)
		a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
		_, err := a.svc.RegistrarReparoAdicional(context.Background(), os.UUID, dto.RegistrarReparoAdicionalRequest{
			Descricao: "x", Pecas: []dto.ItemPecaRequest{{PecaUuid: peca.UUID, Quantidade: 2}},
		})
		require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
		require.Zero(t, a.eventos.publicacoes, "a raiz recusou antes do commit: nada é publicado")
	})
}

func TestAutorizarReparoAdicional(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(5, 0)
	reparo := fxReparo(ordemservico.ReparoPendente)
	os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	os.Itens = []ordemservico.Item{fxItemPeca(peca, 1), fxDoReparo(fxItemPeca(peca, 2), reparo)}
	os.Reparos = []ordemservico.ReparoAdicional{reparo}
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(peca, nil)
	a.pecas.EXPECT().AplicarMovimento(gomock.Any(), peca, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ *estoque.Peca, mov estoque.MovimentoEstoque) error {
			require.Equal(t, int32(-2), mov.Quantidade, "só as peças do reparo")
			return nil
		})

	_, err := a.svc.AutorizarReparoAdicional(context.Background(), os.UUID, reparo.UUID, a.cliente.Documento)
	require.NoError(t, err)
	salvo := (*salvos)[0]
	require.Equal(t, ordemservico.ReparoAutorizado, salvo.Reparos[0].Status)
	require.Equal(t, int64(3*2500), salvo.ValorTotalCentavos)
	historico := requireHistorico(t, salvo, "Reparo adicional autorizado pelo cliente: Disco empenado")
	require.Equal(t, historico.StatusAnterior, historico.StatusNovo)
	require.Equal(t, []string{ordemservico.NomeReparoAdicionalAutorizado, estoque.NomeEstoqueBaixado}, a.eventos.nomes(), "saldo 3 > mínimo 0: sem estoque baixo")
}

func TestAutorizarReparoAdicionalSemEstoque(t *testing.T) {
	a := novaApp(t)
	peca := fxPeca(0, 0)
	reparo := fxReparo(ordemservico.ReparoPendente)
	os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	os.Itens = []ordemservico.Item{fxDoReparo(fxItemPeca(peca, 2), reparo)}
	os.Reparos = []ordemservico.ReparoAdicional{reparo}
	a.trava(os)
	a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(a.cliente, nil)
	a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
	a.pecas.EXPECT().CarregarPorID(gomock.Any(), peca.ID).Return(peca, nil)

	_, err := a.svc.AutorizarReparoAdicional(context.Background(), os.UUID, reparo.UUID, a.cliente.Documento)
	require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
	require.Contains(t, err.Error(), peca.Codigo)
	require.Equal(t, []string{estoque.NomeEstoqueInsuficienteDetectado}, a.eventos.nomes())
}

func TestRecusarReparoAdicional(t *testing.T) {
	a := novaApp(t)
	reparo := fxReparo(ordemservico.ReparoPendente)
	os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	os.Reparos = []ordemservico.ReparoAdicional{reparo}
	a.trava(os)
	a.leitura(os)
	salvos := a.salva()
	a.pecas.EXPECT().AplicarMovimento(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := a.svc.RecusarReparoAdicional(context.Background(), os.UUID, reparo.UUID, a.cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, ordemservico.ReparoRecusado, (*salvos)[0].Reparos[0].Status)
	requireHistorico(t, (*salvos)[0], "Reparo adicional recusado pelo cliente: Disco empenado")
	require.Equal(t, []string{ordemservico.NomeReparoAdicionalRecusado}, a.eventos.nomes())
}

func TestDecidirReparoAdicionalErros(t *testing.T) {
	testCases := []struct {
		name      string
		status    ordemservico.Status
		documento func(a *fxApp) string
		reparos   []ordemservico.ReparoAdicional
		osSumiu   bool
		wantErr   error
	}{
		{name: "JaDecidido", status: ordemservico.StatusEmExecucao, documento: func(a *fxApp) string { return a.cliente.Documento }, reparos: []ordemservico.ReparoAdicional{fxReparo(ordemservico.ReparoAutorizado)}, wantErr: ordemservico.ErrReparoJaDecidido},
		{name: "OSForaDeExecucao", status: ordemservico.StatusFinalizada, documento: func(a *fxApp) string { return a.cliente.Documento }, reparos: []ordemservico.ReparoAdicional{fxReparo(ordemservico.ReparoPendente)}, wantErr: ordemservico.ErrReparoForaDeExecucao},
		{name: "ReparoNaoEncontrado", status: ordemservico.StatusEmExecucao, documento: func(a *fxApp) string { return a.cliente.Documento }, wantErr: service.ErrReparoNaoEncontrado},
		{name: "DocumentoErrado", status: ordemservico.StatusEmExecucao, documento: func(*fxApp) string { return util.RandomCPF() }, reparos: []ordemservico.ReparoAdicional{fxReparo(ordemservico.ReparoPendente)}, wantErr: service.ErrOSNaoEncontrada},
		{name: "OSNaoEncontrada", status: ordemservico.StatusEmExecucao, documento: func(a *fxApp) string { return a.cliente.Documento }, osSumiu: true, wantErr: service.ErrOSNaoEncontrada},
	}

	for i := range testCases {
		tc := testCases[i]
		for _, autorizar := range []bool{true, false} {
			t.Run(tc.name, func(t *testing.T) {
				a := novaApp(t)
				os := a.os(tc.status, ordemservico.OrcamentoAprovado)
				os.Reparos = tc.reparos
				if tc.osSumiu {
					a.osRepo.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
				} else {
					a.trava(os)
					a.catalogo.EXPECT().ClientePorID(gomock.Any(), a.cliente.ID).Return(a.cliente, nil)
				}
				a.osRepo.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)

				reparoUUID := uuid.New()
				if len(tc.reparos) > 0 {
					reparoUUID = tc.reparos[0].UUID
				}
				var err error
				if autorizar {
					_, err = a.svc.AutorizarReparoAdicional(context.Background(), os.UUID, reparoUUID, tc.documento(a))
				} else {
					_, err = a.svc.RecusarReparoAdicional(context.Background(), os.UUID, reparoUUID, tc.documento(a))
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
	a := novaApp(t)
	os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
	os.ValorTotalCentavos = 12345
	a.leitura(os)

	rsp, err := a.svc.ConsultarAndamento(context.Background(), os.UUID, a.cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, "Em execução", rsp.StatusDescricao)
	require.Equal(t, a.veiculo.Placa, rsp.Veiculo.Placa)
	require.Equal(t, int64(12345), rsp.Orcamento.TotalCentavos)
	require.Len(t, rsp.Historico, 1)

	_, err = a.svc.ConsultarAndamento(context.Background(), os.UUID, util.RandomCNPJ())
	require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
}

func TestConsultarAndamentoOSNaoEncontrada(t *testing.T) {
	a := novaApp(t)
	a.osRepo.EXPECT().Carregar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)

	_, err := a.svc.ConsultarAndamento(context.Background(), uuid.New(), a.cliente.Documento)
	require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
}

func TestBuscarOrdemServicoErros(t *testing.T) {
	a := novaApp(t)
	a.osRepo.EXPECT().Carregar(gomock.Any(), gomock.Any()).Return(nil, service.ErrOSNaoEncontrada)
	_, err := a.svc.BuscarOrdemServico(context.Background(), uuid.New())
	require.ErrorIs(t, err, service.ErrOSNaoEncontrada)

	for _, falha := range []string{"cliente", "veiculo", "historico"} {
		t.Run(falha, func(t *testing.T) {
			a := novaApp(t)
			os := a.os(ordemservico.StatusEmExecucao, ordemservico.OrcamentoAprovado)
			a.osRepo.EXPECT().Carregar(gomock.Any(), os.UUID).Return(os, nil)
			switch falha {
			case "cliente":
				a.catalogo.EXPECT().ClientePorID(gomock.Any(), gomock.Any()).Return(readmodel.Cliente{}, sql.ErrConnDone)
			case "veiculo":
				a.catalogo.EXPECT().ClientePorID(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorID(gomock.Any(), gomock.Any()).Return(readmodel.Veiculo{}, sql.ErrConnDone)
			case "historico":
				a.catalogo.EXPECT().ClientePorID(gomock.Any(), gomock.Any()).Return(a.cliente, nil)
				a.catalogo.EXPECT().VeiculoPorID(gomock.Any(), gomock.Any()).Return(a.veiculo, nil)
				a.catalogo.EXPECT().HistoricoDaOS(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			}
			_, err := a.svc.BuscarOrdemServico(context.Background(), os.UUID)
			require.ErrorIs(t, err, sql.ErrConnDone)
		})
	}
}

// As listagens e o tempo médio leem o read model (SQL direto pelo Store).
func novaAppComStore(t *testing.T) (*fxApp, *mockdb.MockStore) {
	a := novaApp(t)
	store := mockdb.NewMockStore(gomock.NewController(t))
	a.svc = service.NewOrdemServicoService(a.uow, store, a.eventos)
	return a, store
}

func TestListarOrdensServico(t *testing.T) {
	a, store := novaAppComStore(t)
	cliente := db.Cliente{ID: a.cliente.ID, Uuid: a.cliente.UUID, Nome: a.cliente.Nome}
	row := db.ListOrdensServicoRow{
		OrdemServico: db.OrdemServico{ID: 1, Uuid: uuid.New(), Status: "RECEBIDA", OrcamentoStatus: "NAO_ENVIADO"},
		ClienteUuid:  a.cliente.UUID,
		ClienteNome:  a.cliente.Nome,
		VeiculoUuid:  a.veiculo.UUID,
		VeiculoPlaca: a.veiculo.Placa,
	}

	t.Run("SemFiltroDeCliente", func(t *testing.T) {
		store.EXPECT().ListOrdensServico(gomock.Any(), db.ListOrdensServicoParams{Status: "RECEBIDA", Limite: 10, Deslocamento: 10}).Return([]db.ListOrdensServicoRow{row}, nil)
		rsp, err := a.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{
			PaginacaoQuery: dto.PaginacaoQuery{Pagina: 2, Tamanho: 10},
			Status:         "RECEBIDA",
		})
		require.NoError(t, err)
		require.Len(t, rsp, 1)
		require.Equal(t, "Recebida", rsp[0].StatusDescricao)
		require.Equal(t, a.veiculo.Placa, rsp[0].VeiculoPlaca)
	})
	t.Run("ComCliente", func(t *testing.T) {
		store.EXPECT().GetCliente(gomock.Any(), a.cliente.UUID).Return(cliente, nil)
		store.EXPECT().ListOrdensServico(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.ListOrdensServicoParams) ([]db.ListOrdensServicoRow, error) {
				require.True(t, arg.ClienteID.Valid)
				require.Equal(t, a.cliente.ID, arg.ClienteID.Int64)
				return []db.ListOrdensServicoRow{}, nil
			})
		rsp, err := a.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{ClienteUuid: a.cliente.UUID.String()})
		require.NoError(t, err)
		require.Empty(t, rsp)
	})
	t.Run("ClienteUUIDInvalido", func(t *testing.T) {
		_, err := a.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{ClienteUuid: "abc"})
		require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)
	})
	t.Run("ClienteNaoEncontrado", func(t *testing.T) {
		store.EXPECT().GetCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrNoRows)
		_, err := a.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{ClienteUuid: uuid.NewString()})
		require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)
	})
	t.Run("ErroNoBanco", func(t *testing.T) {
		store.EXPECT().ListOrdensServico(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
		_, err := a.svc.ListarOrdensServico(context.Background(), dto.ListOrdensServicoQuery{})
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestTempoMedioExecucao(t *testing.T) {
	a, store := novaAppComStore(t)
	servicoUUID := uuid.New()
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	fimExclusivo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local)

	store.EXPECT().GetTempoMedioExecucao(gomock.Any(), db.GetTempoMedioExecucaoParams{
		DataInicio: sql.NullTime{Time: inicio, Valid: true},
		DataFim:    sql.NullTime{Time: fimExclusivo, Valid: true},
	}).Return(db.GetTempoMedioExecucaoRow{TotalOrdens: 2, MediaSegundos: 5400, MinimoSegundos: 1800, MaximoSegundos: 9000}, nil)
	store.EXPECT().ListTempoMedioExecucaoPorServico(gomock.Any(), gomock.Any()).
		Return([]db.ListTempoMedioExecucaoPorServicoRow{
			{ServicoUuid: servicoUUID, ServicoNome: "Troca de óleo", TotalOrdens: 1, MediaSegundos: 600},
			{ServicoUuid: uuid.New(), ServicoNome: "Rápido", TotalOrdens: 1, MediaSegundos: 45},
		}, nil)

	rsp, err := a.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataInicio: "2026-01-01", DataFim: "2026-01-31"})
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
	a, store := novaAppComStore(t)
	store.EXPECT().GetTempoMedioExecucao(gomock.Any(), db.GetTempoMedioExecucaoParams{}).Return(db.GetTempoMedioExecucaoRow{}, nil)
	store.EXPECT().ListTempoMedioExecucaoPorServico(gomock.Any(), db.ListTempoMedioExecucaoPorServicoParams{}).Return([]db.ListTempoMedioExecucaoPorServicoRow{}, nil)

	rsp, err := a.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(0), rsp.TotalOrdens)
	require.Equal(t, "0s", rsp.MediaFormatada)
	require.Empty(t, rsp.PorServico)
}

func TestTempoMedioExecucaoErros(t *testing.T) {
	a, store := novaAppComStore(t)

	_, err := a.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataInicio: "01/01/2026"})
	require.ErrorIs(t, err, service.ErrDataInvalida)
	_, err = a.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataFim: "x"})
	require.ErrorIs(t, err, service.ErrDataInvalida)
	_, err = a.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{DataInicio: "2026-02-01", DataFim: "2026-01-01"})
	require.ErrorIs(t, err, service.ErrIntervaloDatasInvalido)

	store.EXPECT().GetTempoMedioExecucao(gomock.Any(), gomock.Any()).Return(db.GetTempoMedioExecucaoRow{}, sql.ErrConnDone)
	_, err = a.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{})
	require.ErrorIs(t, err, sql.ErrConnDone)

	store.EXPECT().GetTempoMedioExecucao(gomock.Any(), gomock.Any()).Return(db.GetTempoMedioExecucaoRow{}, nil)
	store.EXPECT().ListTempoMedioExecucaoPorServico(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
	_, err = a.svc.TempoMedioExecucao(context.Background(), dto.TempoMedioQuery{})
	require.ErrorIs(t, err, sql.ErrConnDone)
}
