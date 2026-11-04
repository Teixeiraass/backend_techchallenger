package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/eventos"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestLoginAPI(t *testing.T) {
	senha := "admin123"
	hashed, err := util.HashPassword(senha)
	require.NoError(t, err)

	usuario := db.Usuario{ID: 1, Uuid: uuid.New(), Nome: "Admin", Email: "admin@oficina.com", HashedPassword: hashed, Papel: papelAdmin, Ativo: true}
	inativo := usuario
	inativo.Ativo = false
	path := "/api/v1/usuarios/login"

	runCases(t, []apiCase{
		{
			name: "OK", method: http.MethodPost, path: path, body: gin_h{"email": usuario.Email, "senha": senha},
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUsuarioByEmail(gomock.Any(), usuario.Email).Return(usuario, nil)
				store.EXPECT().CreateSessao(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, arg db.CreateSessaoParams) (db.Sessao, error) {
						return db.Sessao{ID: arg.ID, UsuarioID: arg.UsuarioID}, nil
					})
			},
			status: http.StatusOK,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				var rsp dto.LoginResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rsp))
				require.NotEmpty(t, rsp.AccessToken)
				require.NotEmpty(t, rsp.RefreshToken)
				require.Equal(t, papelAdmin, rsp.Usuario.Papel)
			},
		},
		{name: "BodyInvalido", method: http.MethodPost, path: path, body: gin_h{"email": "x"}, status: http.StatusBadRequest},
		{
			name: "EmailInexistente", method: http.MethodPost, path: path, body: gin_h{"email": "no@oficina.com", "senha": senha},
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(db.Usuario{}, sql.ErrNoRows)
			},
			status: http.StatusUnauthorized, check: requireErrorCode("CREDENCIAIS_INVALIDAS"),
		},
		{
			name: "SenhaErrada", method: http.MethodPost, path: path, body: gin_h{"email": usuario.Email, "senha": "errada123"},
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(usuario, nil)
			},
			status: http.StatusUnauthorized, check: requireErrorCode("CREDENCIAIS_INVALIDAS"),
		},
		{
			name: "Inativo", method: http.MethodPost, path: path, body: gin_h{"email": usuario.Email, "senha": senha},
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(inativo, nil)
			},
			status: http.StatusForbidden, check: requireErrorCode("USUARIO_INATIVO"),
		},
	})
}

func TestRenewAccessTokenAPI(t *testing.T) {
	config := util.Config{
		TokenSymmetricKey:    util.RandomString(32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Hour,
	}

	testCases := []struct {
		name   string
		body   func(token string) any
		stubs  func(store *mockdb.MockStore, token string, sessionID uuid.UUID)
		status int
	}{
		{
			name: "OK",
			body: func(token string) any { return gin_h{"refresh_token": token} },
			stubs: func(store *mockdb.MockStore, token string, sessionID uuid.UUID) {
				store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{
					ID: sessionID, UsuarioID: 1, RefreshToken: token, ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			status: http.StatusOK,
		},
		{
			name:   "BodyInvalido",
			body:   func(string) any { return gin_h{} },
			stubs:  func(*mockdb.MockStore, string, uuid.UUID) {},
			status: http.StatusBadRequest,
		},
		{
			name:   "TokenInvalido",
			body:   func(string) any { return gin_h{"refresh_token": "abc"} },
			stubs:  func(*mockdb.MockStore, string, uuid.UUID) {},
			status: http.StatusUnauthorized,
		},
		{
			name: "SessaoBloqueada",
			body: func(token string) any { return gin_h{"refresh_token": token} },
			stubs: func(store *mockdb.MockStore, token string, sessionID uuid.UUID) {
				store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{
					ID: sessionID, UsuarioID: 1, RefreshToken: token, IsBlocked: true, ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)

			server, err := NewServer(config, store, eventos.NovoDespachante())
			require.NoError(t, err)

			refreshToken, payload, err := server.tokenMaker.CreateToken("admin@oficina.com", 1, uuid.New(), papelAdmin, time.Hour)
			require.NoError(t, err)
			tc.stubs(store, refreshToken, payload.ID)

			request, err := http.NewRequest(http.MethodPost, "/api/v1/tokens/refresh", requestBody(t, tc.body(refreshToken)))
			require.NoError(t, err)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)
			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
		})
	}
}

func TestUsuarioAPI(t *testing.T) {
	usuario := db.Usuario{ID: 9, Uuid: uuid.New(), Nome: "Ana", Email: "ana@oficina.com", Papel: papelAtendente, Ativo: true}
	sessionID := uuid.New()
	criar := gin_h{"nome": "Ana", "email": "ana@oficina.com", "senha": "segredo1", "papel": papelAtendente}

	runCases(t, []apiCase{
		{name: "MeOK", method: http.MethodGet, path: "/api/v1/usuarios/me", papel: papelMecanico, status: http.StatusOK},
		{name: "MeSemToken", method: http.MethodGet, path: "/api/v1/usuarios/me", status: http.StatusUnauthorized},
		{
			name: "LogoutOK", method: http.MethodPost, path: "/api/v1/usuarios/logout", body: gin_h{"session_id": sessionID.String()}, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{ID: sessionID, UsuarioID: 7}, nil)
				store.EXPECT().BlockSessao(gomock.Any(), sessionID).Return(nil)
			},
			status: http.StatusNoContent,
		},
		{name: "LogoutBodyInvalido", method: http.MethodPost, path: "/api/v1/usuarios/logout", body: gin_h{}, papel: papelMecanico, status: http.StatusBadRequest},
		{
			name: "LogoutSessaoDeOutroUsuario", method: http.MethodPost, path: "/api/v1/usuarios/logout", body: gin_h{"session_id": sessionID.String()}, papel: papelMecanico,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{ID: sessionID, UsuarioID: 99}, nil)
			},
			status: http.StatusUnauthorized,
		},
		{
			name: "CriarOK", method: http.MethodPost, path: "/api/v1/usuarios", body: criar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUsuario(gomock.Any(), gomock.Any()).Return(usuario, nil)
			},
			status: http.StatusCreated,
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.NotContains(t, recorder.Body.String(), "hashed_password")
			},
		},
		{name: "CriarPapelErrado", method: http.MethodPost, path: "/api/v1/usuarios", body: criar, papel: papelAtendente, status: http.StatusForbidden},
		{name: "CriarPapelInvalido", method: http.MethodPost, path: "/api/v1/usuarios", body: gin_h{"nome": "A", "email": "a@a.com", "senha": "123456", "papel": "CHEFE"}, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "CriarEmailDuplicado", method: http.MethodPost, path: "/api/v1/usuarios", body: criar, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUsuario(gomock.Any(), gomock.Any()).Return(db.Usuario{}, &pq.Error{Code: "23505"})
			},
			status: http.StatusConflict, check: requireErrorCode("EMAIL_DUPLICADO"),
		},
		{
			name: "ListarOK", method: http.MethodGet, path: "/api/v1/usuarios", papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsuarios(gomock.Any(), db.ListUsuariosParams{Limit: 20, Offset: 0}).Return([]db.Usuario{usuario}, nil)
			},
			status: http.StatusOK,
		},
		{name: "ListarQueryInvalida", method: http.MethodGet, path: "/api/v1/usuarios?tamanho=1000", papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "ListarErroInterno", method: http.MethodGet, path: "/api/v1/usuarios", papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsuarios(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			status: http.StatusInternalServerError,
		},
		{
			name: "AtivoOK", method: http.MethodPatch, path: "/api/v1/usuarios/" + usuario.Uuid.String() + "/ativo", body: gin_h{"ativo": false}, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUsuarioAtivo(gomock.Any(), db.UpdateUsuarioAtivoParams{Uuid: usuario.Uuid, Ativo: false}).Return(usuario, nil)
			},
			status: http.StatusOK,
		},
		{name: "AtivoUUIDInvalido", method: http.MethodPatch, path: "/api/v1/usuarios/x/ativo", body: gin_h{"ativo": false}, papel: papelAdmin, status: http.StatusBadRequest},
		{name: "AtivoBodyInvalido", method: http.MethodPatch, path: "/api/v1/usuarios/" + usuario.Uuid.String() + "/ativo", body: gin_h{}, papel: papelAdmin, status: http.StatusBadRequest},
		{
			name: "AtivoNaoEncontrado", method: http.MethodPatch, path: "/api/v1/usuarios/" + usuario.Uuid.String() + "/ativo", body: gin_h{"ativo": true}, papel: papelAdmin,
			stubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUsuarioAtivo(gomock.Any(), gomock.Any()).Return(db.Usuario{}, sql.ErrNoRows)
			},
			status: http.StatusNotFound,
		},
	})
}

func TestMeUsuarioNaoEncontradoAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	// O middleware encontra o usuário; a segunda leitura (handler /me) falha.
	gomock.InOrder(
		store.EXPECT().GetUsuario(gomock.Any(), gomock.Any()).Return(db.Usuario{Papel: papelAdmin, Ativo: true}, nil),
		store.EXPECT().GetUsuario(gomock.Any(), gomock.Any()).Return(db.Usuario{}, sql.ErrNoRows),
	)

	request, err := http.NewRequest(http.MethodGet, "/api/v1/usuarios/me", nil)
	require.NoError(t, err)
	addAuth(t, request, server, papelAdmin)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestNewServerTokenInvalido(t *testing.T) {
	_, err := NewServer(util.Config{TokenSymmetricKey: "curta"}, nil, eventos.NovoDespachante())
	require.Error(t, err)
}

func TestRouter(t *testing.T) {
	server := newTestServer(t, mockdb.NewMockStore(nil))
	require.NotNil(t, server.Router())
}
