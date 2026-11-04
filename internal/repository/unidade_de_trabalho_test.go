package repository_test

import (
	"context"
	"database/sql"
	"testing"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/repository"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func novoStore(t *testing.T) *mockdb.MockStore {
	t.Helper()
	return mockdb.NewMockStore(gomock.NewController(t))
}

// stubTx faz o ExecTx rodar a função contra o próprio mock.
func stubTx(store *mockdb.MockStore) {
	store.EXPECT().ExecTx(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, fn func(db.Querier) error) error {
			return fn(store)
		})
}

func TestUnidadeDeTrabalho(t *testing.T) {
	store := novoStore(t)
	stubTx(store)
	uow := repository.NewUnidadeDeTrabalho(store)

	var dentro service.Repositorios
	require.NoError(t, uow.Executar(context.Background(), func(r service.Repositorios) error {
		dentro = r
		return nil
	}))
	require.NotNil(t, dentro.OrdemServico())
	require.NotNil(t, dentro.Pecas())
	require.NotNil(t, dentro.Catalogo())

	leitura := uow.Leitura()
	require.NotNil(t, leitura.OrdemServico())
	require.NotNil(t, leitura.Pecas())
	require.NotNil(t, leitura.Catalogo())

	err := uow.Executar(context.Background(), func(service.Repositorios) error { return sql.ErrConnDone })
	require.ErrorIs(t, err, sql.ErrConnDone)
}
