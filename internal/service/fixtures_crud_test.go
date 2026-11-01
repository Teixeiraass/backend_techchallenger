package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
)

var (
	crudUniqueErr = &pq.Error{Code: "23505"}
	crudFKErr     = &pq.Error{Code: "23503"}
)

func crudNewStore(t *testing.T) *mockdb.MockStore {
	ctrl := gomock.NewController(t)
	return mockdb.NewMockStore(ctrl)
}

// crudStubTx makes ExecTx run the callback against the mock itself.
func crudStubTx(store *mockdb.MockStore) {
	store.EXPECT().ExecTx(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(ctx context.Context, fn func(db.Querier) error) error {
			return fn(store)
		})
}

func crudConfig() util.Config {
	return util.Config{
		TokenSymmetricKey:    util.RandomString(32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Hour,
		AdminNotificationTo:  "admin@oficina.com",
	}
}

func crudCliente() db.Cliente {
	return db.Cliente{
		ID:            util.RandomInt(1, 1000),
		Uuid:          util.RandomUuid(),
		Nome:          util.RandomName(),
		Documento:     util.RandomCPF(),
		TipoDocumento: "CPF",
		Email:         util.RandomEmail(),
	}
}

func crudPeca(saldo, minimo int32) db.Peca {
	return db.Peca{
		ID:                util.RandomInt(1, 1000),
		Uuid:              util.RandomUuid(),
		Codigo:            "COD-" + util.RandomString(4),
		Nome:              util.RandomName(),
		Tipo:              "PECA",
		Unidade:           "UN",
		PrecoCentavos:     1000,
		QuantidadeEstoque: saldo,
		EstoqueMinimo:     minimo,
		Ativo:             true,
	}
}

func crudBool(v bool) *bool    { return &v }
func crudInt32(v int32) *int32 { return &v }

// crudSender records notifications.
type crudSender struct {
	mu   sync.Mutex
	msgs []notification.Message
	err  error
}

func (s *crudSender) Send(_ context.Context, msg notification.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
	return s.err
}
