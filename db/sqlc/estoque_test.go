package db

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBaixarEstoqueNuncaFicaNegativo(t *testing.T) {
	peca := createRandomPeca(t, 1000, 3, 1)
	ctx := context.Background()

	atualizada, err := testQueries.BaixarEstoquePeca(ctx, BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 2})
	require.NoError(t, err)
	require.Equal(t, int32(1), atualizada.QuantidadeEstoque)

	// Sem saldo suficiente a query não atualiza nenhuma linha.
	_, err = testQueries.BaixarEstoquePeca(ctx, BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 2})
	require.ErrorIs(t, err, sql.ErrNoRows)

	atual, err := testQueries.GetPeca(ctx, peca.Uuid)
	require.NoError(t, err)
	require.Equal(t, int32(1), atual.QuantidadeEstoque)
}

func TestBaixarEstoqueConcorrente(t *testing.T) {
	peca := createRandomPeca(t, 1000, 5, 0)

	// 10 baixas de 1 unidade ao mesmo tempo sobre saldo 5: exatamente 5 passam.
	var wg sync.WaitGroup
	var mu sync.Mutex
	sucessos := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := testQueries.BaixarEstoquePeca(context.Background(), BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 1})
			if err == nil {
				mu.Lock()
				sucessos++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	require.Equal(t, 5, sucessos)
	atual, err := testQueries.GetPeca(context.Background(), peca.Uuid)
	require.NoError(t, err)
	require.Equal(t, int32(0), atual.QuantidadeEstoque)
}

func TestReporEAjustarEstoque(t *testing.T) {
	peca := createRandomPeca(t, 1000, 2, 5)
	ctx := context.Background()

	baixo, err := testQueries.ListPecasEstoqueBaixo(ctx)
	require.NoError(t, err)
	require.True(t, containsPeca(baixo, peca.ID))

	reposta, err := testQueries.ReporEstoquePeca(ctx, ReporEstoquePecaParams{ID: peca.ID, Quantidade: 10})
	require.NoError(t, err)
	require.Equal(t, int32(12), reposta.QuantidadeEstoque)

	ajustada, err := testQueries.AjustarEstoquePeca(ctx, AjustarEstoquePecaParams{ID: peca.ID, Quantidade: 7})
	require.NoError(t, err)
	require.Equal(t, int32(7), ajustada.QuantidadeEstoque)

	movimento, err := testQueries.CreateMovimentoEstoque(ctx, CreateMovimentoEstoqueParams{
		PecaID:         peca.ID,
		Tipo:           "AJUSTE",
		Quantidade:     -5,
		SaldoAnterior:  12,
		SaldoPosterior: 7,
		Observacao:     "inventário",
	})
	require.NoError(t, err)

	movimentos, err := testQueries.ListMovimentosEstoqueByPeca(ctx, ListMovimentosEstoqueByPecaParams{PecaID: peca.ID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, movimentos, 1)
	require.Equal(t, movimento.ID, movimentos[0].ID)

	// O CHECK do banco também protege o saldo.
	_, err = testQueries.AjustarEstoquePeca(ctx, AjustarEstoquePecaParams{ID: peca.ID, Quantidade: -1})
	require.Error(t, err)
}

func TestExecTxRollback(t *testing.T) {
	peca := createRandomPeca(t, 1000, 4, 0)
	ctx := context.Background()
	errProposital := errors.New("falha proposital")

	err := testStore.ExecTx(ctx, func(q Querier) error {
		if _, err := q.BaixarEstoquePeca(ctx, BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 4}); err != nil {
			return err
		}
		return errProposital
	})
	require.ErrorIs(t, err, errProposital)

	atual, err := testQueries.GetPeca(ctx, peca.Uuid)
	require.NoError(t, err)
	require.Equal(t, int32(4), atual.QuantidadeEstoque, "a baixa deve ser desfeita pelo rollback")
}

func containsPeca(pecas []Peca, id int64) bool {
	for _, p := range pecas {
		if p.ID == id {
			return true
		}
	}
	return false
}
