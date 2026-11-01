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

func TestVeiculoAPI(t *testing.T) {
	cliente := randomCliente()
	veiculo := randomVeiculo(cliente.ID)
	basePath := "/api/v1/veiculos"
	itemPath := basePath + "/" + veiculo.Uuid.String()
	criar := gin_h{"cliente_uuid": cliente.Uuid.String(), "placa": "abc-1d23", "marca": "Volkswagen", "modelo": "Gol", "ano": 2019}
	atualizar := gin_h{"placa": "ABC1234", "marca": "VW", "modelo": "Gol", "ano": 2020}
	row := db.GetVeiculoRow{Veiculo: veiculo, ClienteUuid: cliente.Uuid}

	runCases(t, []apiCase{
		{
			name: "CriarOK", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(cliente, nil)
				store.EXPECT().CreateVeiculo(gomock.Any(), db.CreateVeiculoParams{
					ClienteID: cliente.ID, Placa: "ABC1D23", Marca: "Volkswagen", Modelo: "Gol", Ano: 2019,
				}).Return(veiculo, nil)
			},
			status: http.StatusCreated,
		},
		{name: "CriarSemToken", method: http.MethodPost, path: basePath, body: criar, status: http.StatusUnauthorized},
		{name: "CriarPapelErrado", method: http.MethodPost, path: basePath, body: criar, papel: papelMecanico, status: http.StatusForbidden},
		{name: "CriarBodyInvalido", method: http.MethodPost, path: basePath, body: gin_h{"placa": "ABC1D23"}, papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "CriarPlacaInvalida", method: http.MethodPost, path: basePath, papel: papelAtendente,
			body:   gin_h{"cliente_uuid": cliente.Uuid.String(), "placa": "12ABCDE", "marca": "VW", "modelo": "Gol", "ano": 2019},
			status: http.StatusBadRequest, check: requireErrorCode("PLACA_INVALIDA"),
		},
		{
			name: "CriarClienteNaoEncontrado", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(db.Cliente{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound, check: requireErrorCode("CLIENTE_NAO_ENCONTRADO"),
		},
		{
			name: "CriarPlacaDuplicada", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(cliente, nil)
				store.EXPECT().CreateVeiculo(gomock.Any(), gomock.Any()).Return(db.Veiculo{}, &pq.Error{Code: "23505"})
			},
			status: http.StatusConflict, check: requireErrorCode("PLACA_DUPLICADA"),
		},
		{
			name: "ListarOK", method: http.MethodGet, path: basePath + "?cliente_uuid=" + cliente.Uuid.String(), papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(cliente, nil)
				store.EXPECT().ListVeiculos(gomock.Any(), db.ListVeiculosParams{
					ClienteID: sql.NullInt64{Int64: cliente.ID, Valid: true}, Limite: 20, Deslocamento: 0,
				}).Return([]db.ListVeiculosRow{{Veiculo: veiculo, ClienteUuid: cliente.Uuid}}, nil)
			},
			status: http.StatusOK,
		},
		{name: "ListarQueryInvalida", method: http.MethodGet, path: basePath + "?pagina=0", papel: papelMecanico, status: http.StatusBadRequest},
		{name: "ListarClienteUUIDInvalido", method: http.MethodGet, path: basePath + "?cliente_uuid=xyz", papel: papelMecanico, status: http.StatusNotFound},
		{
			name: "ListarErroInterno", method: http.MethodGet, path: basePath, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListVeiculos(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError, check: requireNoLeak,
		},
		{
			name: "BuscarOK", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetVeiculo(gomock.Any(), veiculo.Uuid).Return(row, nil)
			},
			status: http.StatusOK,
		},
		{name: "BuscarUUIDInvalido", method: http.MethodGet, path: basePath + "/1", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "BuscarNaoEncontrado", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetVeiculo(gomock.Any(), veiculo.Uuid).Return(db.GetVeiculoRow{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "AtualizarOK", method: http.MethodPut, path: itemPath, body: atualizar, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetVeiculo(gomock.Any(), veiculo.Uuid).Return(row, nil)
				store.EXPECT().UpdateVeiculo(gomock.Any(), db.UpdateVeiculoParams{
					Uuid: veiculo.Uuid, Placa: "ABC1234", Marca: "VW", Modelo: "Gol", Ano: 2020,
				}).Return(veiculo, nil)
			},
			status: http.StatusOK,
		},
		{name: "AtualizarUUIDInvalido", method: http.MethodPut, path: basePath + "/1", body: atualizar, papel: papelAtendente, status: http.StatusBadRequest},
		{name: "AtualizarBodyInvalido", method: http.MethodPut, path: itemPath, body: "{", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "AtualizarAnoInvalido", method: http.MethodPut, path: itemPath, papel: papelAtendente,
			body:   gin_h{"placa": "ABC1234", "marca": "VW", "modelo": "Gol", "ano": 1800},
			status: http.StatusBadRequest, check: requireErrorCode("ANO_INVALIDO"),
		},
		{
			name: "RemoverOK", method: http.MethodDelete, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteVeiculo(gomock.Any(), veiculo.Uuid).Return(veiculo.ID, nil)
			},
			status: http.StatusNoContent,
		},
		{name: "RemoverUUIDInvalido", method: http.MethodDelete, path: basePath + "/1", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "RemoverEmUso", method: http.MethodDelete, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteVeiculo(gomock.Any(), veiculo.Uuid).Return(int64(0), &pq.Error{Code: "23503"})
			},
			status: http.StatusConflict, check: requireErrorCode("VEICULO_EM_USO"),
		},
	})
}
