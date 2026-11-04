package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	papelAdmin     = "ADMINISTRADOR"
	papelAtendente = "ATENDENTE"
	papelMecanico  = "MECANICO"
)

func TestClienteAPI(t *testing.T) {
	cliente := randomCliente()
	basePath := "/api/v1/clientes"
	itemPath := basePath + "/" + cliente.Uuid.String()
	criar := gin_h{"nome": "Maria Silva", "documento": "529.982.247-25", "email": "maria@email.com"}
	atualizar := gin_h{"nome": "Maria S.", "email": "m@email.com", "telefone": "11"}

	runCases(t, []apiCase{
		{
			name: "CriarOK", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateCliente(gomock.Any(), db.CreateClienteParams{
					Nome: "Maria Silva", Documento: cpfTeste, TipoDocumento: "CPF", Email: "maria@email.com",
				}).Times(1).Return(cliente, nil)
			},
			status: http.StatusCreated,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				var rsp dto.ClienteResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rsp))
				require.Equal(t, cliente.Uuid, rsp.Uuid)
				require.Equal(t, "529.982.247-25", rsp.DocumentoFormatado)
			},
		},
		{name: "CriarSemToken", method: http.MethodPost, path: basePath, body: criar, status: http.StatusUnauthorized},
		{name: "CriarPapelErrado", method: http.MethodPost, path: basePath, body: criar, papel: papelMecanico, status: http.StatusForbidden, check: requireErrorCode("PAPEL_SEM_PERMISSAO")},
		{name: "CriarBodyInvalido", method: http.MethodPost, path: basePath, body: "{", papel: papelAtendente, status: http.StatusBadRequest, check: requireErrorCode("REQUISICAO_INVALIDA")},
		{
			name: "CriarDocumentoInvalido", method: http.MethodPost, path: basePath, papel: papelAdmin,
			body:   gin_h{"nome": "X", "documento": "111.111.111-11", "email": "x@email.com"},
			status: http.StatusBadRequest, check: requireErrorCode("DOCUMENTO_INVALIDO"),
		},
		// A notificação ao cliente é por e-mail (linguagem ubíqua §1.2), então o e-mail é obrigatório.
		{
			name: "CriarSemEmail", method: http.MethodPost, path: basePath, papel: papelAtendente,
			body:   gin_h{"nome": "Maria Silva", "documento": "529.982.247-25", "telefone": "11999990000"},
			status: http.StatusBadRequest, check: requireErrorCode("REQUISICAO_INVALIDA"),
		},
		{
			name: "CriarDuplicado", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, &pq.Error{Code: "23505"})
			},
			status: http.StatusConflict, check: requireErrorCode("DOCUMENTO_DUPLICADO"),
		},
		{
			name: "CriarErroInterno", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError, check: requireNoLeak,
		},
		{
			name: "ListarOK", method: http.MethodGet, path: basePath + "?busca=maria&pagina=2&tamanho=5", papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListClientes(gomock.Any(), db.ListClientesParams{Busca: "maria", Limite: 5, Deslocamento: 5}).
					Return([]db.Cliente{cliente}, nil)
			},
			status: http.StatusOK,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				var rsp []dto.ClienteResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rsp))
				require.Len(t, rsp, 1)
			},
		},
		{name: "ListarPaginacaoInvalida", method: http.MethodGet, path: basePath + "?tamanho=500", papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "ListarErroInterno", method: http.MethodGet, path: basePath, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListClientes(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError,
		},
		{
			name: "BuscarOK", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(cliente, nil)
			},
			status: http.StatusOK,
		},
		{name: "BuscarUUIDInvalido", method: http.MethodGet, path: basePath + "/abc", papel: papelAtendente, status: http.StatusBadRequest, check: requireErrorCode("UUID_INVALIDO")},
		{
			name: "BuscarNaoEncontrado", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(db.Cliente{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound, check: requireErrorCode("CLIENTE_NAO_ENCONTRADO"),
		},
		{
			name: "BuscarPorDocumentoOK", method: http.MethodGet, path: basePath + "/por-documento", headers: map[string]string{HeaderDocumento: cpfTeste}, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetClienteByDocumento(gomock.Any(), cpfTeste).Return(cliente, nil)
			},
			status: http.StatusOK,
		},
		{name: "BuscarPorDocumentoInvalido", method: http.MethodGet, path: basePath + "/por-documento", headers: map[string]string{HeaderDocumento: "123"}, papel: papelAtendente, status: http.StatusBadRequest},
		{name: "BuscarPorDocumentoSemHeader", method: http.MethodGet, path: basePath + "/por-documento", papel: papelAtendente, status: http.StatusBadRequest, check: requireErrorCode("DOCUMENTO_AUSENTE")},
		{
			name: "AtualizarOK", method: http.MethodPut, path: itemPath, body: atualizar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateCliente(gomock.Any(), db.UpdateClienteParams{Uuid: cliente.Uuid, Nome: "Maria S.", Email: "m@email.com", Telefone: "11"}).
					Return(cliente, nil)
			},
			status: http.StatusOK,
		},
		{name: "AtualizarUUIDInvalido", method: http.MethodPut, path: basePath + "/x", body: atualizar, papel: papelAtendente, status: http.StatusBadRequest},
		{name: "AtualizarBodyInvalido", method: http.MethodPut, path: itemPath, body: gin_h{"email": "nao-email"}, papel: papelAtendente, status: http.StatusBadRequest},
		{name: "AtualizarSemEmail", method: http.MethodPut, path: itemPath, body: gin_h{"nome": "Maria S.", "telefone": "11"}, papel: papelAtendente, status: http.StatusBadRequest, check: requireErrorCode("REQUISICAO_INVALIDA")},
		{
			name: "AtualizarNaoEncontrado", method: http.MethodPut, path: itemPath, body: atualizar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "RemoverOK", method: http.MethodDelete, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteCliente(gomock.Any(), cliente.Uuid).Return(cliente.ID, nil)
			},
			status: http.StatusNoContent,
		},
		{name: "RemoverUUIDInvalido", method: http.MethodDelete, path: basePath + "/x", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "RemoverEmUso", method: http.MethodDelete, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteCliente(gomock.Any(), cliente.Uuid).Return(int64(0), &pq.Error{Code: "23503"})
			},
			status: http.StatusConflict, check: requireErrorCode("CLIENTE_EM_USO"),
		},
		{name: "RemoverPapelErrado", method: http.MethodDelete, path: itemPath, papel: papelMecanico, status: http.StatusForbidden},
	})
}

// gin_h keeps request bodies short in the tables.
type gin_h map[string]any
