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

func TestPecaAPI(t *testing.T) {
	peca := randomPeca()
	basePath := "/api/v1/pecas"
	itemPath := basePath + "/" + peca.Uuid.String()
	criar := gin_h{"codigo": "flt-01", "nome": "Filtro", "tipo": "PECA", "preco_centavos": 3500, "quantidade_estoque": 10, "estoque_minimo": 2}
	atualizar := gin_h{"codigo": "FLT-01", "nome": "Filtro", "tipo": "PECA", "preco_centavos": 3600, "estoque_minimo": 2, "ativo": true}
	pecaBaixa := peca
	pecaBaixa.QuantidadeEstoque = 1

	runCases(t, []apiCase{
		{
			name: "CriarOK", method: http.MethodPost, path: basePath, body: criar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				expectTx(store)
				store.EXPECT().CreatePeca(gomock.Any(), db.CreatePecaParams{
					Codigo: "FLT-01", Nome: "Filtro", Tipo: "PECA", Unidade: "UN", PrecoCentavos: 3500, QuantidadeEstoque: 10, EstoqueMinimo: 2,
				}).Return(peca, nil)
				store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)
			},
			status: http.StatusCreated,
		},
		{name: "CriarSemToken", method: http.MethodPost, path: basePath, body: criar, status: http.StatusUnauthorized},
		{name: "CriarPapelErrado", method: http.MethodPost, path: basePath, body: criar, papel: papelAtendente, status: http.StatusForbidden},
		{name: "CriarTipoInvalido", method: http.MethodPost, path: basePath, body: gin_h{"codigo": "A", "nome": "B", "tipo": "X"}, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "CriarCodigoDuplicado", method: http.MethodPost, path: basePath, body: criar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				expectTx(store)
				store.EXPECT().CreatePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, &pq.Error{Code: "23505"})
			},
			status: http.StatusConflict, check: requireErrorCode("CODIGO_PECA_DUPLICADO"),
		},
		{
			name: "ListarOK", method: http.MethodGet, path: basePath + "?tipo=INSUMO&busca=oleo", papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListPecas(gomock.Any(), db.ListPecasParams{Tipo: "INSUMO", Busca: "oleo", Limite: 20}).Return([]db.Peca{peca}, nil)
			},
			status: http.StatusOK,
		},
		{name: "ListarTipoInvalido", method: http.MethodGet, path: basePath + "?tipo=X", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "ListarErroInterno", method: http.MethodGet, path: basePath, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListPecas(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError,
		},
		{
			name: "EstoqueBaixoOK", method: http.MethodGet, path: basePath + "/estoque-baixo", papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListPecasEstoqueBaixo(gomock.Any()).Return([]db.Peca{pecaBaixa}, nil)
			},
			status: http.StatusOK,
		},
		{
			name: "EstoqueBaixoErro", method: http.MethodGet, path: basePath + "/estoque-baixo", papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListPecasEstoqueBaixo(gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError,
		},
		{
			name: "BuscarOK", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
			},
			status: http.StatusOK,
		},
		{name: "BuscarUUIDInvalido", method: http.MethodGet, path: basePath + "/1", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "BuscarNaoEncontrado", method: http.MethodGet, path: itemPath, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(db.Peca{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound, check: requireErrorCode("PECA_NAO_ENCONTRADA"),
		},
		{
			name: "AtualizarOK", method: http.MethodPut, path: itemPath, body: atualizar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdatePeca(gomock.Any(), gomock.Any()).Return(peca, nil)
			},
			status: http.StatusOK,
		},
		{name: "AtualizarUUIDInvalido", method: http.MethodPut, path: basePath + "/1", body: atualizar, papel: papelAdmin, status: http.StatusBadRequest},
		{name: "AtualizarBodyInvalido", method: http.MethodPut, path: itemPath, body: criar, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "AtualizarNaoEncontrado", method: http.MethodPut, path: itemPath, body: atualizar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdatePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "InativarOK", method: http.MethodDelete, path: itemPath, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().InativarPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
			},
			status: http.StatusOK,
		},
		{name: "InativarUUIDInvalido", method: http.MethodDelete, path: basePath + "/1", papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "InativarNaoEncontrado", method: http.MethodDelete, path: itemPath, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().InativarPeca(gomock.Any(), peca.Uuid).Return(db.Peca{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "ReporOK", method: http.MethodPost, path: itemPath + "/reposicao", body: gin_h{"quantidade": 5}, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				expectTx(store)
				store.EXPECT().GetPecaForUpdate(gomock.Any(), peca.Uuid).Return(peca, nil)
				reposta := peca
				reposta.QuantidadeEstoque = 15
				store.EXPECT().ReporEstoquePeca(gomock.Any(), db.ReporEstoquePecaParams{ID: peca.ID, Quantidade: 5}).Return(reposta, nil)
				store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)
			},
			status: http.StatusOK,
		},
		{name: "ReporPapelErrado", method: http.MethodPost, path: itemPath + "/reposicao", body: gin_h{"quantidade": 5}, papel: papelMecanico, status: http.StatusForbidden},
		{name: "ReporUUIDInvalido", method: http.MethodPost, path: basePath + "/1/reposicao", body: gin_h{"quantidade": 5}, papel: papelAdmin, status: http.StatusBadRequest},
		{name: "ReporQuantidadeZero", method: http.MethodPost, path: itemPath + "/reposicao", body: gin_h{"quantidade": 0}, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "ReporNaoEncontrada", method: http.MethodPost, path: itemPath + "/reposicao", body: gin_h{"quantidade": 5}, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				expectTx(store)
				store.EXPECT().GetPecaForUpdate(gomock.Any(), peca.Uuid).Return(db.Peca{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "AjustarOK", method: http.MethodPost, path: itemPath + "/ajuste", body: gin_h{"quantidade": 1, "observacao": "Inventário"}, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				expectTx(store)
				store.EXPECT().GetPecaForUpdate(gomock.Any(), peca.Uuid).Return(peca, nil)
				store.EXPECT().AjustarEstoquePeca(gomock.Any(), db.AjustarEstoquePecaParams{ID: peca.ID, Quantidade: 1}).Return(pecaBaixa, nil)
				store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)
			},
			status: http.StatusOK,
		},
		{name: "AjustarUUIDInvalido", method: http.MethodPost, path: basePath + "/1/ajuste", body: gin_h{"quantidade": 1, "observacao": "x"}, papel: papelAdmin, status: http.StatusBadRequest},
		{name: "AjustarSemQuantidade", method: http.MethodPost, path: itemPath + "/ajuste", body: gin_h{"observacao": "x"}, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "AjustarErroInterno", method: http.MethodPost, path: itemPath + "/ajuste", body: gin_h{"quantidade": 1, "observacao": "x"}, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ExecTx(gomock.Any(), gomock.Any()).Return(sql.ErrTxDone)
			},
			status: http.StatusInternalServerError, check: requireNoLeak,
		},
		{
			name: "MovimentosOK", method: http.MethodGet, path: itemPath + "/movimentos", papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
				store.EXPECT().ListMovimentosEstoqueByPeca(gomock.Any(), db.ListMovimentosEstoqueByPecaParams{PecaID: peca.ID, Limit: 20, Offset: 0}).
					Return([]db.MovimentoEstoque{{Tipo: "ENTRADA", Quantidade: 10, SaldoPosterior: 10}}, nil)
			},
			status: http.StatusOK,
		},
		{name: "MovimentosUUIDInvalido", method: http.MethodGet, path: basePath + "/1/movimentos", papel: papelMecanico, status: http.StatusBadRequest},
		{name: "MovimentosQueryInvalida", method: http.MethodGet, path: itemPath + "/movimentos?pagina=-1", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "MovimentosNaoEncontrada", method: http.MethodGet, path: itemPath + "/movimentos", papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(db.Peca{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
	})
}
