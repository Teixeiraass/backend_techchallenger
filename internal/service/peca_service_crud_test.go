package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCriarPeca(t *testing.T) {
	ctx := context.Background()
	req := dto.CriarPecaRequest{Codigo: " flt-1 ", Nome: " Filtro ", Tipo: "PECA", PrecoCentavos: 3500, QuantidadeEstoque: 10, EstoqueMinimo: 2}

	t.Run("OKWithInitialStock", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		store.EXPECT().CreatePeca(gomock.Any(), db.CreatePecaParams{
			Codigo: "FLT-1", Nome: "Filtro", Tipo: "PECA", Unidade: "UN", PrecoCentavos: 3500, QuantidadeEstoque: 10, EstoqueMinimo: 2,
		}).Return(db.Peca{ID: 5, Codigo: "FLT-1", QuantidadeEstoque: 10}, nil)
		store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.CreateMovimentoEstoqueParams) (db.MovimentoEstoque, error) {
				require.Equal(t, int64(5), arg.PecaID)
				require.Equal(t, "ENTRADA", arg.Tipo)
				require.Equal(t, int32(10), arg.SaldoPosterior)
				require.Equal(t, sql.NullInt64{Int64: 9, Valid: true}, arg.UsuarioID)
				return db.MovimentoEstoque{}, nil
			})
		p, err := service.NewPecaService(store, &crudSender{}, crudConfig()).CriarPeca(ctx, req, 9)
		require.NoError(t, err)
		require.Equal(t, "FLT-1", p.Codigo)
	})
	t.Run("OKWithoutStockAndCustomUnit", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		r := req
		r.QuantidadeEstoque = 0
		r.Unidade = " l "
		r.Tipo = "INSUMO"
		store.EXPECT().CreatePeca(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.CreatePecaParams) (db.Peca, error) {
				require.Equal(t, "L", arg.Unidade)
				return db.Peca{}, nil
			})
		_, err := service.NewPecaService(store, &crudSender{}, crudConfig()).CriarPeca(ctx, r, 0)
		require.NoError(t, err)
	})
	t.Run("Validation", func(t *testing.T) {
		r := req
		r.Tipo = "OUTRO"
		_, err := service.NewPecaService(crudNewStore(t), &crudSender{}, crudConfig()).CriarPeca(ctx, r, 0)
		require.ErrorIs(t, err, estoque.ErrTipoInvalido)
	})
	t.Run("DuplicateCode", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		store.EXPECT().CreatePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, crudUniqueErr)
		_, err := service.NewPecaService(store, &crudSender{}, crudConfig()).CriarPeca(ctx, req, 0)
		require.ErrorIs(t, err, service.ErrCodigoPecaDuplicado)
	})
	t.Run("MovementError", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		store.EXPECT().CreatePeca(gomock.Any(), gomock.Any()).Return(db.Peca{ID: 1}, nil)
		store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, sql.ErrConnDone)
		_, err := service.NewPecaService(store, &crudSender{}, crudConfig()).CriarPeca(ctx, req, 0)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestBuscarListarAtualizarInativarPeca(t *testing.T) {
	ctx := context.Background()
	p := crudPeca(10, 2)
	store := crudNewStore(t)
	svc := service.NewPecaService(store, &crudSender{}, crudConfig())

	store.EXPECT().GetPeca(gomock.Any(), p.Uuid).Return(p, nil)
	got, err := svc.BuscarPeca(ctx, p.Uuid)
	require.NoError(t, err)
	require.Equal(t, p.Uuid, got.Uuid)

	store.EXPECT().GetPeca(gomock.Any(), p.Uuid).Return(db.Peca{}, sql.ErrNoRows)
	_, err = svc.BuscarPeca(ctx, p.Uuid)
	require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)

	store.EXPECT().ListPecas(gomock.Any(), db.ListPecasParams{ApenasAtivas: true, Tipo: "INSUMO", Busca: "ol", Limite: 20}).Return([]db.Peca{p}, nil)
	list, err := svc.ListarPecas(ctx, dto.ListPecasQuery{ApenasAtivas: true, Tipo: "INSUMO", Busca: "ol"})
	require.NoError(t, err)
	require.Len(t, list, 1)

	store.EXPECT().ListPecasEstoqueBaixo(gomock.Any()).Return([]db.Peca{p}, nil)
	baixo, err := svc.ListarEstoqueBaixo(ctx)
	require.NoError(t, err)
	require.Len(t, baixo, 1)

	upd := dto.AtualizarPecaRequest{Codigo: "abc", Nome: "N", Tipo: "PECA", PrecoCentavos: 10, EstoqueMinimo: 1, Ativo: crudBool(true)}
	store.EXPECT().UpdatePeca(gomock.Any(), db.UpdatePecaParams{
		Uuid: p.Uuid, Codigo: "ABC", Nome: "N", Tipo: "PECA", Unidade: "UN", PrecoCentavos: 10, EstoqueMinimo: 1, Ativo: true,
	}).Return(p, nil)
	_, err = svc.AtualizarPeca(ctx, p.Uuid, upd)
	require.NoError(t, err)

	updUnit := upd
	updUnit.Unidade = "kg"
	store.EXPECT().UpdatePeca(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdatePecaParams) (db.Peca, error) {
			require.Equal(t, "KG", arg.Unidade)
			return p, nil
		})
	_, err = svc.AtualizarPeca(ctx, p.Uuid, updUnit)
	require.NoError(t, err)

	bad := upd
	bad.EstoqueMinimo = -1
	_, err = svc.AtualizarPeca(ctx, p.Uuid, bad)
	require.ErrorIs(t, err, estoque.ErrEstoqueMinimoInvalido)

	store.EXPECT().UpdatePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, crudUniqueErr)
	_, err = svc.AtualizarPeca(ctx, p.Uuid, upd)
	require.ErrorIs(t, err, service.ErrCodigoPecaDuplicado)

	store.EXPECT().UpdatePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
	_, err = svc.AtualizarPeca(ctx, p.Uuid, upd)
	require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)

	store.EXPECT().InativarPeca(gomock.Any(), p.Uuid).Return(p, nil)
	_, err = svc.InativarPeca(ctx, p.Uuid)
	require.NoError(t, err)

	store.EXPECT().InativarPeca(gomock.Any(), p.Uuid).Return(db.Peca{}, sql.ErrNoRows)
	_, err = svc.InativarPeca(ctx, p.Uuid)
	require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
}

func TestReporEstoque(t *testing.T) {
	ctx := context.Background()
	p := crudPeca(3, 2)
	req := dto.ReporEstoqueRequest{Quantidade: 7, Observacao: "NF 1"}

	t.Run("OK", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		depois := p
		depois.QuantidadeEstoque = 10
		store.EXPECT().GetPecaForUpdate(gomock.Any(), p.Uuid).Return(p, nil)
		store.EXPECT().ReporEstoquePeca(gomock.Any(), db.ReporEstoquePecaParams{ID: p.ID, Quantidade: 7}).Return(depois, nil)
		store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.CreateMovimentoEstoqueParams) (db.MovimentoEstoque, error) {
				require.Equal(t, "ENTRADA", arg.Tipo)
				require.Equal(t, int32(3), arg.SaldoAnterior)
				require.Equal(t, int32(10), arg.SaldoPosterior)
				require.Equal(t, "NF 1", arg.Observacao)
				return db.MovimentoEstoque{}, nil
			})
		got, err := service.NewPecaService(store, &crudSender{}, crudConfig()).ReporEstoque(ctx, p.Uuid, req, 1)
		require.NoError(t, err)
		require.Equal(t, int32(10), got.QuantidadeEstoque)
	})
	t.Run("InvalidQuantity", func(t *testing.T) {
		_, err := service.NewPecaService(crudNewStore(t), &crudSender{}, crudConfig()).ReporEstoque(ctx, p.Uuid, dto.ReporEstoqueRequest{Quantidade: 0}, 1)
		require.ErrorIs(t, err, estoque.ErrQuantidadeInvalida)
	})
	t.Run("NotFound", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		store.EXPECT().GetPecaForUpdate(gomock.Any(), p.Uuid).Return(db.Peca{}, sql.ErrNoRows)
		_, err := service.NewPecaService(store, &crudSender{}, crudConfig()).ReporEstoque(ctx, p.Uuid, req, 1)
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
	t.Run("UpdateError", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		store.EXPECT().GetPecaForUpdate(gomock.Any(), p.Uuid).Return(p, nil)
		store.EXPECT().ReporEstoquePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrConnDone)
		_, err := service.NewPecaService(store, &crudSender{}, crudConfig()).ReporEstoque(ctx, p.Uuid, req, 1)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestAjustarEstoque(t *testing.T) {
	ctx := context.Background()
	p := crudPeca(10, 4)

	t.Run("OKAboveMinimum", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		sender := &crudSender{}
		depois := p
		depois.QuantidadeEstoque = 8
		store.EXPECT().GetPecaForUpdate(gomock.Any(), p.Uuid).Return(p, nil)
		store.EXPECT().AjustarEstoquePeca(gomock.Any(), db.AjustarEstoquePecaParams{ID: p.ID, Quantidade: 8}).Return(depois, nil)
		store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.CreateMovimentoEstoqueParams) (db.MovimentoEstoque, error) {
				require.Equal(t, "AJUSTE", arg.Tipo)
				require.Equal(t, int32(-2), arg.Quantidade)
				return db.MovimentoEstoque{}, nil
			})
		_, err := service.NewPecaService(store, sender, crudConfig()).AjustarEstoque(ctx, p.Uuid, dto.AjustarEstoqueRequest{Quantidade: crudInt32(8), Observacao: "inv"}, 1)
		require.NoError(t, err)
		require.Empty(t, sender.msgs)
	})
	t.Run("LowStockNotifiesAdmin", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		sender := &crudSender{err: errors.New("smtp down")} // falha de envio não quebra o ajuste
		depois := p
		depois.QuantidadeEstoque = 1
		store.EXPECT().GetPecaForUpdate(gomock.Any(), p.Uuid).Return(p, nil)
		store.EXPECT().AjustarEstoquePeca(gomock.Any(), gomock.Any()).Return(depois, nil)
		store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)
		got, err := service.NewPecaService(store, sender, crudConfig()).AjustarEstoque(ctx, p.Uuid, dto.AjustarEstoqueRequest{Quantidade: crudInt32(1)}, 1)
		require.NoError(t, err)
		require.Equal(t, int32(1), got.QuantidadeEstoque)
		require.Len(t, sender.msgs, 1)
		require.Equal(t, "admin@oficina.com", sender.msgs[0].To)
		require.Contains(t, sender.msgs[0].Body, p.Codigo)
	})
	t.Run("Negative", func(t *testing.T) {
		_, err := service.NewPecaService(crudNewStore(t), &crudSender{}, crudConfig()).AjustarEstoque(ctx, p.Uuid, dto.AjustarEstoqueRequest{Quantidade: crudInt32(-1)}, 1)
		require.ErrorIs(t, err, estoque.ErrSaldoNegativo)
	})
	t.Run("NotFound", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		store.EXPECT().GetPecaForUpdate(gomock.Any(), p.Uuid).Return(db.Peca{}, sql.ErrNoRows)
		_, err := service.NewPecaService(store, &crudSender{}, crudConfig()).AjustarEstoque(ctx, p.Uuid, dto.AjustarEstoqueRequest{Quantidade: crudInt32(1)}, 1)
		require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	})
	t.Run("UpdateError", func(t *testing.T) {
		store := crudNewStore(t)
		crudStubTx(store)
		store.EXPECT().GetPecaForUpdate(gomock.Any(), p.Uuid).Return(p, nil)
		store.EXPECT().AjustarEstoquePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrConnDone)
		_, err := service.NewPecaService(store, &crudSender{}, crudConfig()).AjustarEstoque(ctx, p.Uuid, dto.AjustarEstoqueRequest{Quantidade: crudInt32(1)}, 1)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestListarMovimentos(t *testing.T) {
	ctx := context.Background()
	p := crudPeca(1, 0)
	store := crudNewStore(t)
	svc := service.NewPecaService(store, &crudSender{}, crudConfig())

	store.EXPECT().GetPeca(gomock.Any(), p.Uuid).Return(p, nil)
	store.EXPECT().ListMovimentosEstoqueByPeca(gomock.Any(), db.ListMovimentosEstoqueByPecaParams{PecaID: p.ID, Limit: 5, Offset: 5}).
		Return([]db.MovimentoEstoque{{}, {}}, nil)
	movs, err := svc.ListarMovimentos(ctx, p.Uuid, dto.PaginacaoQuery{Pagina: 2, Tamanho: 5})
	require.NoError(t, err)
	require.Len(t, movs, 2)

	id := uuid.New()
	store.EXPECT().GetPeca(gomock.Any(), id).Return(db.Peca{}, sql.ErrNoRows)
	_, err = svc.ListarMovimentos(ctx, id, dto.PaginacaoQuery{})
	require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
}
