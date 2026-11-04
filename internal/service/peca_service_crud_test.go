package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fxEstoque é o serviço de peças ligado aos mocks das portas.
type fxEstoque struct {
	app   *fxApp
	store *mockdb.MockStore
	svc   service.PecaService
}

func novoEstoque(t *testing.T) *fxEstoque {
	app := novaApp(t)
	store := mockdb.NewMockStore(gomock.NewController(t))
	return &fxEstoque{app: app, store: store, svc: service.NewPecaService(app.uow, store, app.sender, fxConfig())}
}

func TestCriarPeca(t *testing.T) {
	ctx := context.Background()
	req := dto.CriarPecaRequest{Codigo: " flt-1 ", Nome: " Filtro ", Tipo: "PECA", PrecoCentavos: 3500, QuantidadeEstoque: 10, EstoqueMinimo: 2}

	t.Run("OKComSaldoInicial", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().Salvar(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p *estoque.Peca) error {
				require.Equal(t, "FLT-1", p.Codigo)
				require.Equal(t, "Filtro", p.Nome)
				require.Equal(t, "UN", p.Unidade)
				require.Equal(t, int32(10), p.Saldo)
				p.ID = 5
				return nil
			})
		e.app.pecas.EXPECT().RegistrarMovimento(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, mov estoque.MovimentoEstoque) error {
				require.Equal(t, int64(5), mov.PecaID)
				require.Equal(t, estoque.MovimentoEntrada, mov.Tipo)
				require.Equal(t, int32(10), mov.Quantidade)
				require.Equal(t, int32(10), mov.SaldoPosterior)
				require.Equal(t, int64(9), mov.UsuarioID)
				require.Equal(t, "Saldo inicial", mov.Observacao)
				return nil
			})
		p, err := e.svc.CriarPeca(ctx, req, 9)
		require.NoError(t, err)
		require.Equal(t, "FLT-1", p.Codigo)
		require.Equal(t, 1, e.app.transacoes)
	})
	t.Run("OKSemSaldoEUnidadePropria", func(t *testing.T) {
		e := novoEstoque(t)
		r := req
		r.QuantidadeEstoque = 0
		r.Unidade = " l "
		r.Tipo = "INSUMO"
		e.app.pecas.EXPECT().Salvar(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p *estoque.Peca) error {
				require.Equal(t, "L", p.Unidade)
				require.Equal(t, estoque.TipoInsumo, p.Tipo)
				return nil
			})
		e.app.pecas.EXPECT().RegistrarMovimento(gomock.Any(), gomock.Any()).Times(0)
		_, err := e.svc.CriarPeca(ctx, r, 0)
		require.NoError(t, err)
	})
	t.Run("Validacao", func(t *testing.T) {
		e := novoEstoque(t)
		r := req
		r.Tipo = "OUTRO"
		_, err := e.svc.CriarPeca(ctx, r, 0)
		require.ErrorIs(t, err, estoque.ErrTipoInvalido)
		require.Zero(t, e.app.transacoes)
	})
	t.Run("CodigoDuplicado", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().Salvar(gomock.Any(), gomock.Any()).Return(service.ErrCodigoPecaDuplicado)
		_, err := e.svc.CriarPeca(ctx, req, 0)
		require.ErrorIs(t, err, service.ErrCodigoPecaDuplicado)
	})
	t.Run("ErroNoMovimento", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().Salvar(gomock.Any(), gomock.Any()).Return(nil)
		e.app.pecas.EXPECT().RegistrarMovimento(gomock.Any(), gomock.Any()).Return(sql.ErrConnDone)
		_, err := e.svc.CriarPeca(ctx, req, 0)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestBuscarEListarPecas(t *testing.T) {
	ctx := context.Background()
	p := fxPeca(10, 2)

	t.Run("Buscar", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().Carregar(gomock.Any(), p.UUID).Return(p, nil)
		got, err := e.svc.BuscarPeca(ctx, p.UUID)
		require.NoError(t, err)
		require.Equal(t, p.Codigo, got.Codigo)

		e.app.pecas.EXPECT().Carregar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err = e.svc.BuscarPeca(ctx, uuid.New())
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
	t.Run("Listar", func(t *testing.T) {
		e := novoEstoque(t)
		row := db.Peca{ID: 1, Uuid: uuid.New(), Codigo: "A", Tipo: "PECA", QuantidadeEstoque: 1, EstoqueMinimo: 2, Ativo: true}
		e.store.EXPECT().ListPecas(gomock.Any(), db.ListPecasParams{ApenasAtivas: true, Tipo: "PECA", Busca: "a", Limite: 20}).Return([]db.Peca{row}, nil)
		pecas, err := e.svc.ListarPecas(ctx, dto.ListPecasQuery{ApenasAtivas: true, Tipo: "PECA", Busca: "a"})
		require.NoError(t, err)
		require.Len(t, pecas, 1)
		require.True(t, pecas[0].EstoqueBaixo())

		e.store.EXPECT().ListPecas(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
		_, err = e.svc.ListarPecas(ctx, dto.ListPecasQuery{})
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
	t.Run("EstoqueBaixo", func(t *testing.T) {
		e := novoEstoque(t)
		e.store.EXPECT().ListPecasEstoqueBaixo(gomock.Any()).Return([]db.Peca{{Codigo: "B"}}, nil)
		pecas, err := e.svc.ListarEstoqueBaixo(ctx)
		require.NoError(t, err)
		require.Equal(t, "B", pecas[0].Codigo)

		e.store.EXPECT().ListPecasEstoqueBaixo(gomock.Any()).Return(nil, sql.ErrConnDone)
		_, err = e.svc.ListarEstoqueBaixo(ctx)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestAtualizarEInativarPeca(t *testing.T) {
	ctx := context.Background()
	ativo := true
	req := dto.AtualizarPecaRequest{Codigo: "flt-2", Nome: "Filtro 2", Tipo: "PECA", Unidade: "cx", PrecoCentavos: 4000, EstoqueMinimo: 3, Ativo: &ativo}

	t.Run("AtualizarOK", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(10, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().Salvar(gomock.Any(), p).Return(nil)
		got, err := e.svc.AtualizarPeca(ctx, p.UUID, req)
		require.NoError(t, err)
		require.Equal(t, "FLT-2", got.Codigo)
		require.Equal(t, "CX", got.Unidade)
		require.Equal(t, int32(3), got.EstoqueMinimo)
		require.Equal(t, int32(10), got.Saldo, "o saldo não muda pelo cadastro")
	})
	t.Run("AtualizarValidacao", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(10, 2)
		invalida := req
		invalida.Tipo = "X"
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().Salvar(gomock.Any(), gomock.Any()).Times(0)
		_, err := e.svc.AtualizarPeca(ctx, p.UUID, invalida)
		require.ErrorIs(t, err, estoque.ErrTipoInvalido)
	})
	t.Run("AtualizarNaoEncontrada", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err := e.svc.AtualizarPeca(ctx, uuid.New(), req)
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
	t.Run("AtualizarCodigoDuplicado", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(10, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().Salvar(gomock.Any(), gomock.Any()).Return(service.ErrCodigoPecaDuplicado)
		_, err := e.svc.AtualizarPeca(ctx, p.UUID, req)
		require.ErrorIs(t, err, service.ErrCodigoPecaDuplicado)
	})
	t.Run("InativarOK", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(10, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().Salvar(gomock.Any(), p).Return(nil)
		got, err := e.svc.InativarPeca(ctx, p.UUID)
		require.NoError(t, err)
		require.False(t, got.Ativa)
	})
	t.Run("InativarNaoEncontrada", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err := e.svc.InativarPeca(ctx, uuid.New())
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
}

func TestReporEstoque(t *testing.T) {
	ctx := context.Background()

	t.Run("OK", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(1, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().AplicarMovimento(gomock.Any(), p, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ *estoque.Peca, mov estoque.MovimentoEstoque) error {
				require.Equal(t, estoque.MovimentoEntrada, mov.Tipo)
				require.Equal(t, int32(5), mov.Quantidade)
				require.Equal(t, int32(1), mov.SaldoAnterior)
				require.Equal(t, int32(6), mov.SaldoPosterior)
				require.Equal(t, int64(3), mov.UsuarioID)
				require.Equal(t, "compra", mov.Observacao)
				return nil
			})
		got, err := e.svc.ReporEstoque(ctx, p.UUID, dto.ReporEstoqueRequest{Quantidade: 5, Observacao: "compra"}, 3)
		require.NoError(t, err)
		require.Equal(t, int32(6), got.Saldo)
		require.Empty(t, e.app.sender.msgs)
	})
	t.Run("QuantidadeInvalida", func(t *testing.T) {
		e := novoEstoque(t)
		_, err := e.svc.ReporEstoque(ctx, uuid.New(), dto.ReporEstoqueRequest{Quantidade: 0}, 3)
		require.ErrorIs(t, err, estoque.ErrQuantidadeInvalida)
		require.Zero(t, e.app.transacoes)
	})
	t.Run("NaoEncontrada", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err := e.svc.ReporEstoque(ctx, uuid.New(), dto.ReporEstoqueRequest{Quantidade: 5}, 3)
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
	t.Run("ErroNoBanco", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(1, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().AplicarMovimento(gomock.Any(), gomock.Any(), gomock.Any()).Return(sql.ErrConnDone)
		_, err := e.svc.ReporEstoque(ctx, p.UUID, dto.ReporEstoqueRequest{Quantidade: 5}, 3)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestAjustarEstoque(t *testing.T) {
	ctx := context.Background()
	quantidade := func(v int32) *int32 { return &v }

	t.Run("OKSemPedidoDeReposicao", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(10, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().AplicarMovimento(gomock.Any(), p, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ *estoque.Peca, mov estoque.MovimentoEstoque) error {
				require.Equal(t, estoque.MovimentoAjuste, mov.Tipo)
				require.Equal(t, int32(-3), mov.Quantidade)
				require.Equal(t, int32(7), mov.SaldoPosterior)
				return nil
			})
		got, err := e.svc.AjustarEstoque(ctx, p.UUID, dto.AjustarEstoqueRequest{Quantidade: quantidade(7), Observacao: "inventário"}, 3)
		require.NoError(t, err)
		require.Equal(t, int32(7), got.Saldo)
		require.Empty(t, e.app.sender.msgs)
	})
	t.Run("OKComPedidoDeReposicao", func(t *testing.T) {
		e := novoEstoque(t)
		p := fxPeca(10, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().AplicarMovimento(gomock.Any(), p, gomock.Any()).Return(nil)
		got, err := e.svc.AjustarEstoque(ctx, p.UUID, dto.AjustarEstoqueRequest{Quantidade: quantidade(1)}, 3)
		require.NoError(t, err)
		require.True(t, got.EstoqueBaixo())
		require.Len(t, e.app.sender.msgs, 1)
		require.Equal(t, "admin@oficina.com", e.app.sender.msgs[0].To)
		require.Contains(t, e.app.sender.msgs[0].Body, p.Codigo)
	})
	t.Run("NotificacaoFalhaNaoDesfaz", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.sender.err = errors.New("smtp fora do ar")
		p := fxPeca(10, 2)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), p.UUID).Return(p, nil)
		e.app.pecas.EXPECT().AplicarMovimento(gomock.Any(), p, gomock.Any()).Return(nil)
		_, err := e.svc.AjustarEstoque(ctx, p.UUID, dto.AjustarEstoqueRequest{Quantidade: quantidade(0)}, 3)
		require.NoError(t, err)
	})
	t.Run("SaldoNegativo", func(t *testing.T) {
		e := novoEstoque(t)
		_, err := e.svc.AjustarEstoque(ctx, uuid.New(), dto.AjustarEstoqueRequest{Quantidade: quantidade(-1)}, 3)
		require.ErrorIs(t, err, estoque.ErrSaldoNegativo)
	})
	t.Run("NaoEncontrada", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().CarregarParaAtualizar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err := e.svc.AjustarEstoque(ctx, uuid.New(), dto.AjustarEstoqueRequest{Quantidade: quantidade(1)}, 3)
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
}

func TestListarMovimentos(t *testing.T) {
	ctx := context.Background()
	p := fxPeca(10, 2)

	t.Run("OK", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().Carregar(gomock.Any(), p.UUID).Return(p, nil)
		e.store.EXPECT().ListMovimentosEstoqueByPeca(gomock.Any(), db.ListMovimentosEstoqueByPecaParams{PecaID: p.ID, Limit: 20}).Return([]db.MovimentoEstoque{{PecaID: p.ID}}, nil)
		movs, err := e.svc.ListarMovimentos(ctx, p.UUID, dto.PaginacaoQuery{})
		require.NoError(t, err)
		require.Len(t, movs, 1)
	})
	t.Run("NaoEncontrada", func(t *testing.T) {
		e := novoEstoque(t)
		e.app.pecas.EXPECT().Carregar(gomock.Any(), gomock.Any()).Return(nil, service.ErrPecaNaoEncontrada)
		_, err := e.svc.ListarMovimentos(ctx, uuid.New(), dto.PaginacaoQuery{})
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
}
