package handler

import (
	"database/sql"
	"net/http"
	"testing"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
)

func TestServicoAPI(t *testing.T) {
	servico := randomServico()
	basePath := "/api/v1/servicos"
	itemPath := basePath + "/" + servico.Uuid.String()
	criar := gin_h{"nome": "Troca de óleo", "preco_centavos": 8000, "tempo_estimado_minutos": 30}
	atualizar := gin_h{"nome": "Troca de óleo", "preco_centavos": 9000, "tempo_estimado_minutos": 30, "ativo": true}

	runCases(t, []apiCase{
		{
			name: "CriarOK", method: http.MethodPost, path: basePath, body: criar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateServico(gomock.Any(), db.CreateServicoParams{
					Nome: "Troca de óleo", PrecoCentavos: 8000, TempoEstimadoMinutos: 30,
				}).Return(servico, nil)
			},
			status: http.StatusCreated,
		},
		{name: "CriarSemToken", method: http.MethodPost, path: basePath, body: criar, status: http.StatusUnauthorized},
		{name: "CriarPapelErrado", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente, status: http.StatusForbidden},
		{name: "CriarBodyInvalido", method: http.MethodPost, path: basePath, body: gin_h{"preco_centavos": -1}, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "CriarDuplicado", method: http.MethodPost, path: basePath, body: criar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateServico(gomock.Any(), gomock.Any()).Return(db.Servico{}, &pq.Error{Code: "23505"})
			},
			status: http.StatusConflict, check: requireErrorCode("SERVICO_DUPLICADO"),
		},
		{
			name: "ListarOK", method: http.MethodGet, path: basePath + "?apenas_ativos=true", papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListServicos(gomock.Any(), db.ListServicosParams{ApenasAtivos: true, Limite: 20}).
					Return([]db.Servico{servico}, nil)
			},
			status: http.StatusOK,
		},
		{name: "ListarQueryInvalida", method: http.MethodGet, path: basePath + "?tamanho=0", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "ListarErroInterno", method: http.MethodGet, path: basePath, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListServicos(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError,
		},
		{
			name: "BuscarOK", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
			},
			status: http.StatusOK,
		},
		{name: "BuscarUUIDInvalido", method: http.MethodGet, path: basePath + "/1", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "BuscarNaoEncontrado", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(db.Servico{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "AtualizarOK", method: http.MethodPut, path: itemPath, body: atualizar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateServico(gomock.Any(), db.UpdateServicoParams{
					Uuid: servico.Uuid, Nome: "Troca de óleo", PrecoCentavos: 9000, TempoEstimadoMinutos: 30, Ativo: true,
				}).Return(servico, nil)
			},
			status: http.StatusOK,
		},
		{name: "AtualizarUUIDInvalido", method: http.MethodPut, path: basePath + "/1", body: atualizar, papel: papelAdmin, status: http.StatusBadRequest},
		{name: "AtualizarSemAtivo", method: http.MethodPut, path: itemPath, body: criar, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "AtualizarNaoEncontrado", method: http.MethodPut, path: itemPath, body: atualizar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateServico(gomock.Any(), gomock.Any()).Return(db.Servico{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "InativarOK", method: http.MethodDelete, path: itemPath, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().InativarServico(gomock.Any(), servico.Uuid).Return(servico, nil)
			},
			status: http.StatusOK,
		},
		{name: "InativarUUIDInvalido", method: http.MethodDelete, path: basePath + "/1", papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "InativarNaoEncontrado", method: http.MethodDelete, path: itemPath, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().InativarServico(gomock.Any(), servico.Uuid).Return(db.Servico{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
	})
}
