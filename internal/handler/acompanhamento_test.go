package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Rotas públicas: nenhum caso usa papel (sem token).
func TestAcompanhamentoAPI(t *testing.T) {
	peca := randomPeca()
	aguardando := randomOS("AGUARDANDO_APROVACAO", "AGUARDANDO")
	aprovada := aguardando
	aprovada.OrcamentoStatus = "APROVADO"
	recebida := randomOS("RECEBIDA", "NAO_ENVIADO")
	emExecucao := randomOS("EM_EXECUCAO", "APROVADO")

	osPath := func(os db.OrdemServico, sufixo string) string {
		return "/api/v1/ordens-servico/" + os.Uuid.String() + sufixo
	}
	titular := gin_h{"documento": "529.982.247-25"}
	outroCPF := gin_h{"documento": "11144477735"}

	// O fluxo de aprovação trava a OS duas vezes: na aprovação e no início da execução.
	expectAprovacao := func(store *mockdb.MockStore) {
		store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), aguardando.Uuid).Times(1).Return(aguardando, nil)
		store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), aguardando.Uuid).Times(1).Return(aprovada, nil)
	}

	runCases(t, []apiCase{
		{
			name: "AndamentoOK", method: http.MethodGet, path: osPath(emExecucao, "/andamento?documento="+cpfTeste),
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emExecucao, nil, nil) },
			status: http.StatusOK,
		},
		{
			name: "AndamentoDocumentoErrado", method: http.MethodGet, path: osPath(emExecucao, "/andamento?documento=11144477735"),
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emExecucao, nil, nil) },
			status: http.StatusNotFound, check: requireErrorCode("OS_NAO_ENCONTRADA"),
		},
		{name: "AndamentoSemDocumento", method: http.MethodGet, path: osPath(emExecucao, "/andamento"), status: http.StatusBadRequest},
		{name: "AndamentoUUIDInvalido", method: http.MethodGet, path: "/api/v1/ordens-servico/x/andamento?documento=" + cpfTeste, status: http.StatusBadRequest},
		{
			name: "AndamentoNaoEncontrada", method: http.MethodGet, path: osPath(emExecucao, "/andamento?documento="+cpfTeste),
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrdemServico(gomock.Any(), emExecucao.Uuid).Return(db.OrdemServico{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "AprovarOK", method: http.MethodPost, path: osPath(aguardando, "/orcamento/aprovar"), body: titular,
			stubs: func(store *mockdb.MockStore) {
				expectAprovacao(store)
				stubOS(store, aguardando, nil, nil)
			},
			status: http.StatusOK,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.NotContains(t, recorder.Body.String(), `"aviso"`)
			},
		},
		{
			name: "AprovarSemEstoqueRegistraAviso", method: http.MethodPost, path: osPath(aguardando, "/orcamento/aprovar"), body: titular,
			stubs: func(store *mockdb.MockStore) {
				expectAprovacao(store)
				store.EXPECT().GetPecaByID(gomock.Any(), peca.ID).Return(peca, nil)
				// O agregado recusa antes do UPDATE (saldo 10 < 50): nada chega ao banco.
				store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Times(0)
				stubOS(store, aguardando, []db.ListItensPecaByOrdemServicoRow{itemPeca(peca, 50, nil)}, nil)
			},
			status: http.StatusOK,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Contains(t, recorder.Body.String(), `"aviso"`)
			},
		},
		{
			name: "AprovarDocumentoErrado", method: http.MethodPost, path: osPath(aguardando, "/orcamento/aprovar"), body: outroCPF,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, aguardando, nil, nil) },
			status: http.StatusNotFound, check: requireErrorCode("OS_NAO_ENCONTRADA"),
		},
		{name: "AprovarSemDocumento", method: http.MethodPost, path: osPath(aguardando, "/orcamento/aprovar"), body: gin_h{}, status: http.StatusBadRequest},
		{name: "AprovarUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/orcamento/aprovar", body: titular, status: http.StatusBadRequest},
		{
			name: "AprovarSemOrcamentoAguardando", method: http.MethodPost, path: osPath(recebida, "/orcamento/aprovar"), body: titular,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, recebida, nil, nil) },
			status: http.StatusConflict, check: requireErrorCode("ORCAMENTO_NAO_AGUARDANDO"),
		},
		{
			name: "RejeitarOK", method: http.MethodPost, path: osPath(aguardando, "/orcamento/rejeitar"), body: titular,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, aguardando, nil, nil) },
			status: http.StatusOK,
		},
		{name: "RejeitarUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/orcamento/rejeitar", body: titular, status: http.StatusBadRequest},
		{name: "RejeitarBodyInvalido", method: http.MethodPost, path: osPath(aguardando, "/orcamento/rejeitar"), body: "{", status: http.StatusBadRequest},
		{
			name: "RejeitarStatusErrado", method: http.MethodPost, path: osPath(recebida, "/orcamento/rejeitar"), body: titular,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, recebida, nil, nil) },
			status: http.StatusConflict,
		},
	})
}

func TestDecisaoReparoAdicionalAPI(t *testing.T) {
	peca := randomPeca()
	emExecucao := randomOS("EM_EXECUCAO", "APROVADO")
	reparo := db.ReparoAdicional{ID: 30, Uuid: uuid.New(), OrdemServicoID: emExecucao.ID, Descricao: "Disco", Status: "PENDENTE"}
	autorizado := reparo
	autorizado.Status = "AUTORIZADO"
	path := func(acao string) string {
		return "/api/v1/ordens-servico/" + emExecucao.Uuid.String() + "/reparos-adicionais/" + reparo.Uuid.String() + "/" + acao
	}
	titular := gin_h{"documento": cpfTeste}

	runCases(t, []apiCase{
		{
			name: "AutorizarOK", method: http.MethodPost, path: path("autorizar"), body: titular,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateReparoAdicionalStatus(gomock.Any(), db.UpdateReparoAdicionalStatusParams{ID: reparo.ID, Status: "AUTORIZADO"}).Return(autorizado, nil)
				store.EXPECT().GetPecaByID(gomock.Any(), peca.ID).Return(peca, nil)
				store.EXPECT().BaixarEstoquePeca(gomock.Any(), db.BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 1}).Return(peca, nil)
				store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)
				stubOS(store, emExecucao, []db.ListItensPecaByOrdemServicoRow{itemPeca(peca, 1, &reparo)}, []db.ReparoAdicional{reparo})
			},
			status: http.StatusOK,
		},
		{
			name: "RecusarOK", method: http.MethodPost, path: path("recusar"), body: titular,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateReparoAdicionalStatus(gomock.Any(), db.UpdateReparoAdicionalStatusParams{ID: reparo.ID, Status: "RECUSADO"}).Return(reparo, nil)
				stubOS(store, emExecucao, nil, []db.ReparoAdicional{reparo})
			},
			status: http.StatusOK,
		},
		{
			name: "AutorizarNaoEncontrado", method: http.MethodPost, path: path("autorizar"), body: titular,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emExecucao, nil, nil) },
			status: http.StatusNotFound, check: requireErrorCode("REPARO_NAO_ENCONTRADO"),
		},
		{
			name: "RecusarJaDecidido", method: http.MethodPost, path: path("recusar"), body: titular,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emExecucao, nil, []db.ReparoAdicional{autorizado}) },
			status: http.StatusConflict, check: requireErrorCode("REPARO_JA_DECIDIDO"),
		},
		{
			name: "AutorizarDocumentoErrado", method: http.MethodPost, path: path("autorizar"), body: gin_h{"documento": "11144477735"},
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emExecucao, nil, nil) },
			status: http.StatusNotFound,
		},
		{name: "AutorizarOSUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/reparos-adicionais/" + reparo.Uuid.String() + "/autorizar", body: titular, status: http.StatusBadRequest},
		{name: "AutorizarReparoUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/" + emExecucao.Uuid.String() + "/reparos-adicionais/x/autorizar", body: titular, status: http.StatusBadRequest},
		{name: "RecusarSemDocumento", method: http.MethodPost, path: path("recusar"), body: gin_h{}, status: http.StatusBadRequest},
	})
}
