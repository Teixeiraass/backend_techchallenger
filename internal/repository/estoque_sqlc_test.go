package repository_test

import (
	"context"
	"database/sql"
	"testing"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/repository"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func linhaPeca() db.Peca {
	return db.Peca{ID: 5, Uuid: uuid.New(), Codigo: "FLT-01", Nome: "Filtro", Tipo: "PECA", Unidade: "UN", PrecoCentavos: 3500, QuantidadeEstoque: 10, EstoqueMinimo: 2, Ativo: true}
}

func TestPecaRepositorioCarregar(t *testing.T) {
	ctx := context.Background()
	row := linhaPeca()

	store := novoStore(t)
	store.EXPECT().GetPeca(ctx, row.Uuid).Return(row, nil)
	store.EXPECT().GetPecaByID(ctx, row.ID).Return(row, nil)
	store.EXPECT().GetPecaForUpdate(ctx, row.Uuid).Return(row, nil)
	repo := repository.NewPecaRepositorio(store)

	p, err := repo.Carregar(ctx, row.Uuid)
	require.NoError(t, err)
	require.Equal(t, int32(10), p.Saldo)
	require.Equal(t, estoque.TipoPeca, p.Tipo)
	require.True(t, p.Ativa)
	p, err = repo.CarregarPorID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, row.Codigo, p.Codigo)
	p, err = repo.CarregarParaAtualizar(ctx, row.Uuid)
	require.NoError(t, err)
	require.Equal(t, row.ID, p.ID)
	require.Equal(t, row.Uuid, repository.ParaPeca(row).UUID)

	store.EXPECT().GetPeca(ctx, gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
	store.EXPECT().GetPecaByID(ctx, gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
	store.EXPECT().GetPecaForUpdate(ctx, gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
	_, err = repo.Carregar(ctx, uuid.New())
	require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	_, err = repo.CarregarPorID(ctx, 99)
	require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
	_, err = repo.CarregarParaAtualizar(ctx, uuid.New())
	require.ErrorIs(t, err, service.ErrPecaNaoEncontrada)
}

func TestPecaRepositorioSalvar(t *testing.T) {
	ctx := context.Background()
	row := linhaPeca()

	t.Run("Criar", func(t *testing.T) {
		store := novoStore(t)
		p, _, err := estoque.NovaPeca(estoque.Cadastro{Codigo: "flt-01", Nome: "Filtro", Tipo: "PECA", PrecoCentavos: 3500, EstoqueMinimo: 2}, 10)
		require.NoError(t, err)
		store.EXPECT().CreatePeca(ctx, db.CreatePecaParams{Codigo: "FLT-01", Nome: "Filtro", Tipo: "PECA", Unidade: "UN", PrecoCentavos: 3500, QuantidadeEstoque: 10, EstoqueMinimo: 2}).Return(row, nil)
		require.NoError(t, repository.NewPecaRepositorio(store).Salvar(ctx, p))
		require.Equal(t, row.ID, p.ID)
		require.Equal(t, row.Uuid, p.UUID)
	})
	t.Run("Atualizar", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		p.Ativa = false
		store.EXPECT().UpdatePeca(ctx, db.UpdatePecaParams{Uuid: row.Uuid, Codigo: "FLT-01", Nome: "Filtro", Tipo: "PECA", Unidade: "UN", PrecoCentavos: 3500, EstoqueMinimo: 2, Ativo: false}).Return(row, nil)
		require.NoError(t, repository.NewPecaRepositorio(store).Salvar(ctx, &p))
	})
	t.Run("CodigoDuplicado", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		store.EXPECT().UpdatePeca(ctx, gomock.Any()).Return(db.Peca{}, &pq.Error{Code: "23505"})
		require.ErrorIs(t, repository.NewPecaRepositorio(store).Salvar(ctx, &p), service.ErrCodigoPecaDuplicado)
	})
	t.Run("NaoEncontrada", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		store.EXPECT().UpdatePeca(ctx, gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
		require.ErrorIs(t, repository.NewPecaRepositorio(store).Salvar(ctx, &p), service.ErrPecaNaoEncontrada)
	})
}

func TestPecaRepositorioAplicarMovimento(t *testing.T) {
	ctx := context.Background()
	row := linhaPeca()

	t.Run("Saida", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		mov, err := p.Baixar(3, 77, 1)
		require.NoError(t, err)
		depois := row
		depois.QuantidadeEstoque = 7
		store.EXPECT().BaixarEstoquePeca(ctx, db.BaixarEstoquePecaParams{ID: row.ID, Quantidade: 3}).Return(depois, nil)
		store.EXPECT().CreateMovimentoEstoque(ctx, db.CreateMovimentoEstoqueParams{
			PecaID: row.ID, Tipo: "SAIDA", Quantidade: -3, SaldoAnterior: 10, SaldoPosterior: 7,
			OrdemServicoID: sql.NullInt64{Int64: 77, Valid: true}, UsuarioID: sql.NullInt64{Int64: 1, Valid: true}, Observacao: "Baixa por ordem de serviço",
		}).Return(db.MovimentoEstoque{}, nil)
		require.NoError(t, repository.NewPecaRepositorio(store).AplicarMovimento(ctx, &p, mov))
		require.Equal(t, int32(7), p.Saldo)
	})
	t.Run("SaidaSemSaldoNoBanco", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		mov, err := p.Baixar(3, 77, 1)
		require.NoError(t, err)
		store.EXPECT().BaixarEstoquePeca(ctx, gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
		err = repository.NewPecaRepositorio(store).AplicarMovimento(ctx, &p, mov)
		require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
		require.Contains(t, err.Error(), "FLT-01")
	})
	t.Run("Entrada", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		mov, err := p.Repor(5, 2, "compra")
		require.NoError(t, err)
		depois := row
		depois.QuantidadeEstoque = 15
		store.EXPECT().ReporEstoquePeca(ctx, db.ReporEstoquePecaParams{ID: row.ID, Quantidade: 5}).Return(depois, nil)
		store.EXPECT().CreateMovimentoEstoque(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.CreateMovimentoEstoqueParams) (db.MovimentoEstoque, error) {
				require.Equal(t, "ENTRADA", arg.Tipo)
				require.Equal(t, int32(5), arg.Quantidade)
				require.False(t, arg.OrdemServicoID.Valid)
				return db.MovimentoEstoque{}, nil
			})
		require.NoError(t, repository.NewPecaRepositorio(store).AplicarMovimento(ctx, &p, mov))
		require.Equal(t, int32(15), p.Saldo)
	})
	t.Run("Ajuste", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		mov, err := p.Ajustar(1, 2, "inventário")
		require.NoError(t, err)
		depois := row
		depois.QuantidadeEstoque = 1
		store.EXPECT().AjustarEstoquePeca(ctx, db.AjustarEstoquePecaParams{ID: row.ID, Quantidade: 1}).Return(depois, nil)
		store.EXPECT().CreateMovimentoEstoque(ctx, gomock.Any()).Return(db.MovimentoEstoque{}, nil)
		require.NoError(t, repository.NewPecaRepositorio(store).AplicarMovimento(ctx, &p, mov))
		require.Equal(t, int32(1), p.Saldo)
	})
	t.Run("ErroNoBanco", func(t *testing.T) {
		store := novoStore(t)
		p := repository.ParaPeca(row)
		mov, err := p.Repor(5, 2, "")
		require.NoError(t, err)
		store.EXPECT().ReporEstoquePeca(ctx, gomock.Any()).Return(db.Peca{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewPecaRepositorio(store).AplicarMovimento(ctx, &p, mov), sql.ErrConnDone)
	})
	t.Run("RegistrarMovimento", func(t *testing.T) {
		store := novoStore(t)
		store.EXPECT().CreateMovimentoEstoque(ctx, gomock.Any()).Return(db.MovimentoEstoque{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewPecaRepositorio(store).RegistrarMovimento(ctx, estoque.MovimentoEstoque{PecaID: 1}), sql.ErrConnDone)
	})
}
