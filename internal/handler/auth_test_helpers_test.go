package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/middleware"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const cpfTeste = "52998224725"

// apiCase is one HTTP scenario: request, auth role, store stubs and expected status.
type apiCase struct {
	name   string
	method string
	path   string
	body   any
	papel  string
	stubs  func(store *mockdb.MockStore)
	status int
	check  func(t *testing.T, recorder *httptest.ResponseRecorder)
}

func runCases(t *testing.T, cases []apiCase) {
	t.Helper()
	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)

			request, err := http.NewRequest(tc.method, tc.path, requestBody(t, tc.body))
			require.NoError(t, err)
			if tc.body != nil {
				request.Header.Set("Content-Type", "application/json")
			}

			if tc.papel != "" {
				addAuth(t, request, server, tc.papel)
				expectUsuario(store, tc.papel)
			}
			if tc.stubs != nil {
				tc.stubs(store)
			}

			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)

			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			if tc.check != nil {
				tc.check(t, recorder)
			}
		})
	}
}

func requestBody(t *testing.T, body any) io.Reader {
	switch v := body.(type) {
	case nil:
		return nil
	case string:
		return bytes.NewBufferString(v)
	default:
		data, err := json.Marshal(v)
		require.NoError(t, err)
		return bytes.NewReader(data)
	}
}

func addAuth(t *testing.T, request *http.Request, server *Server, papel string) {
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer,
		"user@oficina.com", 7, uuid.New(), papel, time.Minute)
}

// expectUsuario stubs the lookup done by the auth middleware (and /usuarios/me).
func expectUsuario(store *mockdb.MockStore, papel string) {
	store.EXPECT().GetUsuario(gomock.Any(), gomock.Any()).AnyTimes().Return(db.Usuario{
		ID:    7,
		Uuid:  uuid.New(),
		Nome:  "Usuário Teste",
		Email: "user@oficina.com",
		Papel: papel,
		Ativo: true,
	}, nil)
}

func expectTx(store *mockdb.MockStore) {
	store.EXPECT().ExecTx(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(ctx context.Context, fn func(db.Querier) error) error {
			return fn(store)
		})
}

func requireErrorCode(code string) func(t *testing.T, recorder *httptest.ResponseRecorder) {
	return func(t *testing.T, recorder *httptest.ResponseRecorder) {
		var body ErrorResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		require.Equal(t, code, body.Code)
	}
}

// requireNoLeak checks the generic 500 body.
func requireNoLeak(t *testing.T, recorder *httptest.ResponseRecorder) {
	require.JSONEq(t, `{"error":"erro interno do servidor","code":"ERRO_INTERNO"}`, recorder.Body.String())
}

func randomCliente() db.Cliente {
	return db.Cliente{
		ID:            1,
		Uuid:          uuid.New(),
		Nome:          "Maria Silva",
		Documento:     cpfTeste,
		TipoDocumento: "CPF",
		Email:         "maria@email.com",
		Telefone:      "11999990000",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func randomVeiculo(clienteID int64) db.Veiculo {
	return db.Veiculo{
		ID:        2,
		Uuid:      uuid.New(),
		ClienteID: clienteID,
		Placa:     "ABC1D23",
		Marca:     "Volkswagen",
		Modelo:    "Gol",
		Ano:       2019,
	}
}

func randomServico() db.Servico {
	return db.Servico{ID: 4, Uuid: uuid.New(), Nome: "Troca de óleo", PrecoCentavos: 8000, TempoEstimadoMinutos: 30, Ativo: true}
}

func randomPeca() db.Peca {
	return db.Peca{
		ID: 5, Uuid: uuid.New(), Codigo: "FLT-01", Nome: "Filtro", Tipo: "PECA", Unidade: "UN",
		PrecoCentavos: 3500, QuantidadeEstoque: 10, EstoqueMinimo: 2, Ativo: true,
	}
}

func randomOS(status, orcamentoStatus string) db.OrdemServico {
	return db.OrdemServico{
		ID: 3, Uuid: uuid.New(), ClienteID: 1, VeiculoID: 2,
		Status: status, OrcamentoStatus: orcamentoStatus, ValorTotalCentavos: 8000,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// stubOS stubs every read/write the OS service uses with permissive
// expectations. Per-test overrides must be declared before calling it.
func stubOS(store *mockdb.MockStore, os db.OrdemServico, pecas []db.ListItensPecaByOrdemServicoRow, reparos []db.ReparoAdicional) {
	cliente := randomCliente()
	veiculo := randomVeiculo(cliente.ID)
	if pecas == nil {
		pecas = []db.ListItensPecaByOrdemServicoRow{}
	}
	if reparos == nil {
		reparos = []db.ReparoAdicional{}
	}

	expectTx(store)
	store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), gomock.Any()).AnyTimes().Return(os, nil)
	store.EXPECT().GetOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return(os, nil)
	store.EXPECT().GetClienteByID(gomock.Any(), gomock.Any()).AnyTimes().Return(cliente, nil)
	store.EXPECT().GetVeiculoByID(gomock.Any(), gomock.Any()).AnyTimes().Return(veiculo, nil)
	store.EXPECT().ListItensServicoByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return([]db.ListItensServicoByOrdemServicoRow{
		{ID: 10, Uuid: uuid.New(), OrdemServicoID: os.ID, Quantidade: 1, ValorUnitarioCentavos: 8000, ServicoUuid: uuid.New(), ServicoNome: "Troca de óleo"},
	}, nil)
	store.EXPECT().ListItensPecaByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return(pecas, nil)
	store.EXPECT().ListReparosAdicionaisByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return(reparos, nil)
	store.EXPECT().ListHistoricoStatusOSByOrdemServico(gomock.Any(), gomock.Any()).AnyTimes().Return([]db.HistoricoStatusOS{
		{ID: 1, OrdemServicoID: os.ID, StatusNovo: "RECEBIDA", CreatedAt: time.Now()},
	}, nil)
	store.EXPECT().UpdateOrdemServicoEstado(gomock.Any(), gomock.Any()).AnyTimes().Return(os, nil)
	store.EXPECT().UpdateOrdemServicoValorTotal(gomock.Any(), gomock.Any()).AnyTimes().Return(os, nil)
	store.EXPECT().CreateHistoricoStatusOS(gomock.Any(), gomock.Any()).AnyTimes().Return(db.HistoricoStatusOS{}, nil)
}
