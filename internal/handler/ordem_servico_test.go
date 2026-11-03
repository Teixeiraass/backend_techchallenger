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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func itemPeca(peca db.Peca, quantidade int32, reparo *db.ReparoAdicional) db.ListItensPecaByOrdemServicoRow {
	row := db.ListItensPecaByOrdemServicoRow{
		ID: 20, Uuid: uuid.New(), OrdemServicoID: 3, PecaID: peca.ID, Quantidade: quantidade,
		ValorUnitarioCentavos: peca.PrecoCentavos, PecaUuid: peca.Uuid, PecaCodigo: peca.Codigo, PecaNome: peca.Nome, PecaTipo: peca.Tipo,
	}
	if reparo != nil {
		row.ReparoAdicionalUuid = uuid.NullUUID{UUID: reparo.Uuid, Valid: true}
		row.ReparoAdicionalStatus = sql.NullString{String: reparo.Status, Valid: true}
	}
	return row
}

func TestAbrirOrdemServicoAPI(t *testing.T) {
	cliente := randomCliente()
	veiculo := randomVeiculo(cliente.ID)
	servico := randomServico()
	peca := randomPeca()
	os := randomOS("RECEBIDA", "NAO_ENVIADO")
	path := "/api/v1/ordens-servico"
	body := gin_h{
		"documento": "529.982.247-25",
		"placa":     "abc-1d23",
		"servicos":  []gin_h{{"servico_uuid": servico.Uuid.String()}},
		"pecas":     []gin_h{{"peca_uuid": peca.Uuid.String(), "quantidade": 2}},
	}

	runCases(t, []apiCase{
		{
			name: "OK", method: http.MethodPost, path: path, body: body, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetClienteByDocumento(gomock.Any(), cpfTeste).Return(cliente, nil)
				store.EXPECT().GetVeiculoByPlaca(gomock.Any(), "ABC1D23").Return(veiculo, nil)
				store.EXPECT().CreateOrdemServico(gomock.Any(), gomock.Any()).Return(os, nil)
				store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
				store.EXPECT().CreateItemServico(gomock.Any(), gomock.Any()).Return(db.ItemServico{}, nil)
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
				store.EXPECT().CreateItemPeca(gomock.Any(), gomock.Any()).Return(db.ItemPeca{}, nil)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusCreated,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				var rsp dto.OrdemServicoResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rsp))
				require.Equal(t, os.Uuid, rsp.Uuid)
				require.Equal(t, "RECEBIDA", rsp.Status)
				require.Equal(t, "Recebida", rsp.StatusDescricao)
			},
		},
		{name: "SemToken", method: http.MethodPost, path: path, body: body, status: http.StatusUnauthorized},
		{name: "PapelErrado", method: http.MethodPost, path: path, body: body, papel: papelMecanico, status: http.StatusForbidden},
		{name: "BodyInvalido", method: http.MethodPost, path: path, body: gin_h{"placa": "ABC1D23"}, papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "DocumentoInvalido", method: http.MethodPost, path: path, body: gin_h{"documento": "123", "placa": "ABC1D23"}, papel: papelAtendente,
			status: http.StatusBadRequest, check: requireErrorCode("DOCUMENTO_INVALIDO"),
		},
		{
			name: "PlacaInvalida", method: http.MethodPost, path: path, body: gin_h{"documento": cpfTeste, "placa": "XX"}, papel: papelAtendente,
			status: http.StatusBadRequest, check: requireErrorCode("PLACA_INVALIDA"),
		},
		{
			name: "ClienteNaoEncontrado", method: http.MethodPost, path: path, body: body, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				expectTx(store)
				store.EXPECT().GetClienteByDocumento(gomock.Any(), cpfTeste).Return(db.Cliente{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound, check: requireErrorCode("CLIENTE_NAO_ENCONTRADO"),
		},
		{
			name: "VeiculoDeOutroCliente", method: http.MethodPost, path: path, body: body, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				expectTx(store)
				store.EXPECT().GetClienteByDocumento(gomock.Any(), cpfTeste).Return(cliente, nil)
				store.EXPECT().GetVeiculoByPlaca(gomock.Any(), "ABC1D23").Return(randomVeiculo(99), nil)
			},
			status: http.StatusConflict, check: requireErrorCode("VEICULO_DE_OUTRO_CLIENTE"),
		},
		{
			name: "ErroInterno", method: http.MethodPost, path: path, body: body, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ExecTx(gomock.Any(), gomock.Any()).Return(sql.ErrConnDone)
			},
			status: http.StatusInternalServerError, check: requireNoLeak,
		},
	})
}

func TestConsultasOrdemServicoAPI(t *testing.T) {
	cliente := randomCliente()
	os := randomOS("EM_EXECUCAO", "APROVADO")
	path := "/api/v1/ordens-servico"
	row := db.ListOrdensServicoRow{OrdemServico: os, ClienteUuid: cliente.Uuid, ClienteNome: cliente.Nome, VeiculoPlaca: "ABC1D23"}

	runCases(t, []apiCase{
		{
			name: "ListarOK", method: http.MethodGet, path: path + "?status=EM_EXECUCAO&cliente_uuid=" + cliente.Uuid.String(), papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(cliente, nil)
				store.EXPECT().ListOrdensServico(gomock.Any(), db.ListOrdensServicoParams{
					Status: "EM_EXECUCAO", ClienteID: sql.NullInt64{Int64: cliente.ID, Valid: true}, Limite: 20,
				}).Return([]db.ListOrdensServicoRow{row}, nil)
			},
			status: http.StatusOK,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				var rsp []dto.OrdemServicoResumoResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rsp))
				require.Len(t, rsp, 1)
				require.Equal(t, "Em execução", rsp[0].StatusDescricao)
			},
		},
		{name: "ListarSemToken", method: http.MethodGet, path: path, status: http.StatusUnauthorized},
		{name: "ListarStatusInvalido", method: http.MethodGet, path: path + "?status=CANCELADA", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "ListarClienteNaoEncontrado", method: http.MethodGet, path: path + "?cliente_uuid=" + cliente.Uuid.String(), papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetCliente(gomock.Any(), cliente.Uuid).Return(db.Cliente{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
		{
			name: "ListarErroInterno", method: http.MethodGet, path: path, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListOrdensServico(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError, check: requireNoLeak,
		},
		{
			name: "BuscarOK", method: http.MethodGet, path: path + "/" + os.Uuid.String(), papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				stubOS(store, os, nil, nil)
			},
			status: http.StatusOK,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				var rsp dto.OrdemServicoResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rsp))
				require.Len(t, rsp.Servicos, 1)
				require.Len(t, rsp.Historico, 1)
				require.Equal(t, int64(8000), rsp.Orcamento.TotalServicosCentavos)
			},
		},
		{name: "BuscarUUIDInvalido", method: http.MethodGet, path: path + "/123", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "BuscarNaoEncontrada", method: http.MethodGet, path: path + "/" + os.Uuid.String(), papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrdemServico(gomock.Any(), os.Uuid).Return(db.OrdemServico{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound, check: requireErrorCode("OS_NAO_ENCONTRADA"),
		},
		{
			name: "TempoMedioOK", method: http.MethodGet, path: path + "/tempo-medio?data_inicio=2026-01-01&data_fim=2026-12-31", papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetTempoMedioExecucao(gomock.Any(), gomock.Any()).Return(db.GetTempoMedioExecucaoRow{
					TotalOrdens: 2, MediaSegundos: 5400, MinimoSegundos: 3600, MaximoSegundos: 7200,
				}, nil)
				store.EXPECT().ListTempoMedioExecucaoPorServico(gomock.Any(), gomock.Any()).Return([]db.ListTempoMedioExecucaoPorServicoRow{
					{ServicoUuid: uuid.New(), ServicoNome: "Troca de óleo", TotalOrdens: 2, MediaSegundos: 1800},
				}, nil)
			},
			status: http.StatusOK,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				var rsp dto.TempoMedioExecucaoResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rsp))
				require.Equal(t, int64(5400), rsp.MediaSegundos)
				require.Equal(t, "1h 30min", rsp.MediaFormatada)
				require.Len(t, rsp.PorServico, 1)
			},
		},
		{name: "TempoMedioPapelErrado", method: http.MethodGet, path: path + "/tempo-medio", papel: papelAtendente, status: http.StatusForbidden},
		{
			name: "TempoMedioDataInvalida", method: http.MethodGet, path: path + "/tempo-medio?data_inicio=01/01/2026", papel: papelAdmin,
			status: http.StatusBadRequest, check: requireErrorCode("DATA_INVALIDA"),
		},
		{
			name: "TempoMedioIntervaloInvalido", method: http.MethodGet, path: path + "/tempo-medio?data_inicio=2026-12-31&data_fim=2026-01-01", papel: papelAdmin,
			status: http.StatusBadRequest, check: requireErrorCode("INTERVALO_INVALIDO"),
		},
		{
			name: "TempoMedioErroInterno", method: http.MethodGet, path: path + "/tempo-medio", papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetTempoMedioExecucao(gomock.Any(), gomock.Any()).Return(db.GetTempoMedioExecucaoRow{}, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError,
		},
	})
}

func TestItensOrdemServicoAPI(t *testing.T) {
	servico := randomServico()
	peca := randomPeca()
	os := randomOS("EM_DIAGNOSTICO", "NAO_ENVIADO")
	emExecucao := randomOS("EM_EXECUCAO", "APROVADO")
	osPath := "/api/v1/ordens-servico/" + os.Uuid.String()
	itemUUID := uuid.New()

	runCases(t, []apiCase{
		{
			name: "AdicionarServicoOK", method: http.MethodPost, path: osPath + "/servicos", body: gin_h{"servico_uuid": servico.Uuid.String(), "quantidade": 1}, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
				store.EXPECT().CreateItemServico(gomock.Any(), gomock.Any()).Return(db.ItemServico{}, nil)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusOK,
		},
		{name: "AdicionarServicoSemToken", method: http.MethodPost, path: osPath + "/servicos", body: gin_h{"servico_uuid": servico.Uuid.String()}, status: http.StatusUnauthorized},
		{name: "AdicionarServicoUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/servicos", body: gin_h{"servico_uuid": servico.Uuid.String()}, papel: papelMecanico, status: http.StatusBadRequest},
		{name: "AdicionarServicoBodyInvalido", method: http.MethodPost, path: osPath + "/servicos", body: gin_h{"servico_uuid": "nao-uuid"}, papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "AdicionarServicoInativo", method: http.MethodPost, path: osPath + "/servicos", body: gin_h{"servico_uuid": servico.Uuid.String()}, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				inativo := servico
				inativo.Ativo = false
				store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(inativo, nil)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusConflict, check: requireErrorCode("SERVICO_INATIVO"),
		},
		{
			name: "AdicionarServicoOSEmExecucao", method: http.MethodPost, path: osPath + "/servicos", body: gin_h{"servico_uuid": servico.Uuid.String()}, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				stubOS(store, emExecucao, nil, nil)
			},
			status: http.StatusConflict, check: requireErrorCode("ITENS_NAO_EDITAVEIS"),
		},
		{
			name: "AdicionarServicoOSNaoEncontrada", method: http.MethodPost, path: osPath + "/servicos", body: gin_h{"servico_uuid": servico.Uuid.String()}, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrdemServicoForUpdate(gomock.Any(), os.Uuid).Return(db.OrdemServico{}, sql.ErrNoRows)
				expectTx(store)
			},
			status: http.StatusNotFound, check: requireErrorCode("OS_NAO_ENCONTRADA"),
		},
		{
			name: "RemoverServicoOK", method: http.MethodDelete, path: osPath + "/servicos/" + itemUUID.String(), papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteItemServico(gomock.Any(), db.DeleteItemServicoParams{Uuid: itemUUID, OrdemServicoID: os.ID}).Return(int64(10), nil)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusOK,
		},
		{name: "RemoverServicoOSUUIDInvalido", method: http.MethodDelete, path: "/api/v1/ordens-servico/x/servicos/" + itemUUID.String(), papel: papelAtendente, status: http.StatusBadRequest},
		{name: "RemoverServicoItemUUIDInvalido", method: http.MethodDelete, path: osPath + "/servicos/x", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "RemoverServicoItemNaoEncontrado", method: http.MethodDelete, path: osPath + "/servicos/" + itemUUID.String(), papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteItemServico(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrNoRows)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusNotFound, check: requireErrorCode("ITEM_NAO_ENCONTRADO"),
		},
		{
			name: "AdicionarPecaOK", method: http.MethodPost, path: osPath + "/pecas", body: gin_h{"peca_uuid": peca.Uuid.String(), "quantidade": 2}, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
				store.EXPECT().CreateItemPeca(gomock.Any(), gomock.Any()).Return(db.ItemPeca{}, nil)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusOK,
		},
		{name: "AdicionarPecaUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/pecas", body: gin_h{"peca_uuid": peca.Uuid.String(), "quantidade": 1}, papel: papelAtendente, status: http.StatusBadRequest},
		{name: "AdicionarPecaSemQuantidade", method: http.MethodPost, path: osPath + "/pecas", body: gin_h{"peca_uuid": peca.Uuid.String()}, papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "AdicionarPecaSemEstoque", method: http.MethodPost, path: osPath + "/pecas", body: gin_h{"peca_uuid": peca.Uuid.String(), "quantidade": 9}, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(peca, nil)
				// A OS já pediu 2 unidades: 2 + 9 > 10 em estoque.
				stubOS(store, os, []db.ListItensPecaByOrdemServicoRow{itemPeca(peca, 2, nil)}, nil)
			},
			status: http.StatusConflict, check: requireErrorCode("ESTOQUE_INSUFICIENTE"),
		},
		{
			name: "AdicionarPecaNaoEncontrada", method: http.MethodPost, path: osPath + "/pecas", body: gin_h{"peca_uuid": peca.Uuid.String(), "quantidade": 1}, papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPeca(gomock.Any(), peca.Uuid).Return(db.Peca{}, sql.ErrNoRows)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusNotFound, check: requireErrorCode("PECA_NAO_ENCONTRADA"),
		},
		{
			name: "RemoverPecaOK", method: http.MethodDelete, path: osPath + "/pecas/" + itemUUID.String(), papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteItemPeca(gomock.Any(), db.DeleteItemPecaParams{Uuid: itemUUID, OrdemServicoID: os.ID}).Return(int64(20), nil)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusOK,
		},
		{name: "RemoverPecaOSUUIDInvalido", method: http.MethodDelete, path: "/api/v1/ordens-servico/x/pecas/" + itemUUID.String(), papel: papelMecanico, status: http.StatusBadRequest},
		{name: "RemoverPecaItemUUIDInvalido", method: http.MethodDelete, path: osPath + "/pecas/x", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "RemoverPecaItemNaoEncontrado", method: http.MethodDelete, path: osPath + "/pecas/" + itemUUID.String(), papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteItemPeca(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrNoRows)
				stubOS(store, os, nil, nil)
			},
			status: http.StatusNotFound,
		},
	})
}

func TestTransicoesOrdemServicoAPI(t *testing.T) {
	peca := randomPeca()
	recebida := randomOS("RECEBIDA", "NAO_ENVIADO")
	emDiagnostico := randomOS("EM_DIAGNOSTICO", "NAO_ENVIADO")
	aprovada := randomOS("AGUARDANDO_APROVACAO", "APROVADO")
	emExecucao := randomOS("EM_EXECUCAO", "APROVADO")
	finalizada := randomOS("FINALIZADA", "APROVADO")
	entregue := randomOS("ENTREGUE", "APROVADO")
	path := func(os db.OrdemServico, acao string) string {
		return "/api/v1/ordens-servico/" + os.Uuid.String() + "/" + acao
	}
	reparoPendente := db.ReparoAdicional{ID: 30, Uuid: uuid.New(), OrdemServicoID: emExecucao.ID, Descricao: "Disco", Status: "PENDENTE"}

	runCases(t, []apiCase{
		{
			name: "DiagnosticoOK", method: http.MethodPost, path: path(recebida, "diagnostico"), papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, recebida, nil, nil) },
			status: http.StatusOK,
		},
		{name: "DiagnosticoSemToken", method: http.MethodPost, path: path(recebida, "diagnostico"), status: http.StatusUnauthorized},
		{name: "DiagnosticoPapelErrado", method: http.MethodPost, path: path(recebida, "diagnostico"), papel: papelAtendente, status: http.StatusForbidden},
		{name: "DiagnosticoUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/diagnostico", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "DiagnosticoTransicaoInvalida", method: http.MethodPost, path: path(entregue, "diagnostico"), papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, entregue, nil, nil) },
			status: http.StatusConflict, check: requireErrorCode("TRANSICAO_INVALIDA"),
		},
		{
			name: "EnviarOrcamentoOK", method: http.MethodPost, path: path(emDiagnostico, "orcamento/enviar"), body: gin_h{"diagnostico": "Pastilhas gastas"}, papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emDiagnostico, nil, nil) },
			status: http.StatusOK,
		},
		{
			name: "EnviarOrcamentoSemBody", method: http.MethodPost, path: path(emDiagnostico, "orcamento/enviar"), papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emDiagnostico, nil, nil) },
			status: http.StatusOK,
		},
		{
			name: "EnviarOrcamentoAdmin", method: http.MethodPost, path: path(emDiagnostico, "orcamento/enviar"), papel: papelAdmin,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emDiagnostico, nil, nil) },
			status: http.StatusOK,
		},
		{name: "EnviarOrcamentoPapelErrado", method: http.MethodPost, path: path(emDiagnostico, "orcamento/enviar"), papel: papelAtendente, status: http.StatusForbidden},
		{name: "EnviarOrcamentoUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/orcamento/enviar", papel: papelMecanico, status: http.StatusBadRequest},
		{name: "EnviarOrcamentoBodyInvalido", method: http.MethodPost, path: path(emDiagnostico, "orcamento/enviar"), body: "{", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "EnviarOrcamentoStatusErrado", method: http.MethodPost, path: path(recebida, "orcamento/enviar"), papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, recebida, nil, nil) },
			status: http.StatusConflict, check: requireErrorCode("TRANSICAO_INVALIDA"),
		},
		{
			name: "IniciarExecucaoOK", method: http.MethodPost, path: path(aprovada, "execucao"), papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				baixada := peca
				baixada.QuantidadeEstoque = 1 // fica no estoque mínimo: pedido de reposição
				store.EXPECT().BaixarEstoquePeca(gomock.Any(), db.BaixarEstoquePecaParams{ID: peca.ID, Quantidade: 2}).Return(baixada, nil)
				store.EXPECT().CreateMovimentoEstoque(gomock.Any(), gomock.Any()).Return(db.MovimentoEstoque{}, nil)
				stubOS(store, aprovada, []db.ListItensPecaByOrdemServicoRow{itemPeca(peca, 2, nil)}, nil)
			},
			status: http.StatusOK,
		},
		{name: "IniciarExecucaoUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/execucao", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "IniciarExecucaoSemEstoque", method: http.MethodPost, path: path(aprovada, "execucao"), papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().BaixarEstoquePeca(gomock.Any(), gomock.Any()).Return(db.Peca{}, sql.ErrNoRows)
				stubOS(store, aprovada, []db.ListItensPecaByOrdemServicoRow{itemPeca(peca, 2, nil)}, nil)
			},
			status: http.StatusConflict, check: requireErrorCode("ESTOQUE_INSUFICIENTE"),
		},
		{
			name: "IniciarExecucaoSemAprovacao", method: http.MethodPost, path: path(emDiagnostico, "execucao"), papel: papelAtendente,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emDiagnostico, nil, nil) },
			status: http.StatusConflict, check: requireErrorCode("ORCAMENTO_NAO_APROVADO"),
		},
		{
			name: "FinalizarOK", method: http.MethodPost, path: path(emExecucao, "finalizar"), papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emExecucao, nil, nil) },
			status: http.StatusOK,
		},
		{name: "FinalizarPapelErrado", method: http.MethodPost, path: path(emExecucao, "finalizar"), papel: papelAtendente, status: http.StatusForbidden},
		{name: "FinalizarUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/finalizar", papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "FinalizarComReparoPendente", method: http.MethodPost, path: path(emExecucao, "finalizar"), papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, emExecucao, nil, []db.ReparoAdicional{reparoPendente}) },
			status: http.StatusConflict, check: requireErrorCode("REPARO_PENDENTE"),
		},
		{
			name: "EntregarOK", method: http.MethodPost, path: path(finalizada, "entregar"), papel: papelAtendente,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, finalizada, nil, nil) },
			status: http.StatusOK,
		},
		{name: "EntregarPapelErrado", method: http.MethodPost, path: path(finalizada, "entregar"), papel: papelMecanico, status: http.StatusForbidden},
		{name: "EntregarUUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/entregar", papel: papelAtendente, status: http.StatusBadRequest},
		{
			name: "EntregarAntesDeFinalizar", method: http.MethodPost, path: path(recebida, "entregar"), papel: papelAtendente,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, recebida, nil, nil) },
			status: http.StatusConflict, check: requireErrorCode("ENTREGA_NAO_PERMITIDA"),
		},
		{
			name: "EntregarErroInterno", method: http.MethodPost, path: path(finalizada, "entregar"), papel: papelAtendente,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ExecTx(gomock.Any(), gomock.Any()).Return(sql.ErrConnDone)
			},
			status: http.StatusInternalServerError, check: requireNoLeak,
		},
	})
}

func TestReparoAdicionalAPI(t *testing.T) {
	servico := randomServico()
	emExecucao := randomOS("EM_EXECUCAO", "APROVADO")
	recebida := randomOS("RECEBIDA", "NAO_ENVIADO")
	path := "/api/v1/ordens-servico/" + emExecucao.Uuid.String() + "/reparos-adicionais"
	body := gin_h{"descricao": "Disco empenado", "servicos": []gin_h{{"servico_uuid": servico.Uuid.String()}}}

	runCases(t, []apiCase{
		{
			name: "OK", method: http.MethodPost, path: path, body: body, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateReparoAdicional(gomock.Any(), db.CreateReparoAdicionalParams{OrdemServicoID: emExecucao.ID, Descricao: "Disco empenado"}).
					Return(db.ReparoAdicional{ID: 30, Uuid: uuid.New(), Status: "PENDENTE"}, nil)
				store.EXPECT().GetServico(gomock.Any(), servico.Uuid).Return(servico, nil)
				store.EXPECT().CreateItemServico(gomock.Any(), gomock.Any()).Return(db.ItemServico{}, nil)
				stubOS(store, emExecucao, nil, nil)
			},
			status: http.StatusCreated,
		},
		{name: "PapelErrado", method: http.MethodPost, path: path, body: body, papel: papelAtendente, status: http.StatusForbidden},
		{name: "UUIDInvalido", method: http.MethodPost, path: "/api/v1/ordens-servico/x/reparos-adicionais", body: body, papel: papelMecanico, status: http.StatusBadRequest},
		{name: "BodyInvalido", method: http.MethodPost, path: path, body: gin_h{}, papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "SemItens", method: http.MethodPost, path: path, body: gin_h{"descricao": "x"}, papel: papelMecanico,
			status: http.StatusBadRequest, check: requireErrorCode("REPARO_SEM_ITENS"),
		},
		{
			name: "ForaDeExecucao", method: http.MethodPost, path: "/api/v1/ordens-servico/" + recebida.Uuid.String() + "/reparos-adicionais", body: body, papel: papelMecanico,
			stubs:  func(store *mockdb.MockStore) { stubOS(store, recebida, nil, nil) },
			status: http.StatusConflict, check: requireErrorCode("REPARO_FORA_DE_EXECUCAO"),
		},
	})
}
