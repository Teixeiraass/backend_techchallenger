package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/backend_techchallenger/db/mock"
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func crudUsuario(t *testing.T, senha string, ativo bool) db.Usuario {
	hash, err := util.HashPassword(senha)
	require.NoError(t, err)
	return db.Usuario{
		ID:             util.RandomInt(1, 1000),
		Uuid:           util.RandomUuid(),
		Nome:           util.RandomName(),
		Email:          util.RandomEmail(),
		HashedPassword: hash,
		Papel:          "ATENDENTE",
		Ativo:          ativo,
	}
}

type crudStoreExpect struct {
	t     *testing.T
	store *mockdb.MockStore
}

func (s *crudStoreExpect) sessao(sessao db.Sessao, err error) {
	s.store.EXPECT().GetSessao(gomock.Any(), gomock.Any()).Return(sessao, err)
}

func TestCriarUsuario(t *testing.T) {
	store := crudNewStore(t)
	svc := service.NewUsuarioService(store, nil, crudConfig())
	req := dto.CriarUsuarioRequest{Nome: "Ana", Email: "ana@x.com", Senha: "secret1", Papel: "MECANICO"}

	store.EXPECT().CreateUsuario(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateUsuarioParams) (db.Usuario, error) {
			require.Equal(t, req.Email, arg.Email)
			require.NoError(t, util.CheckPassword(req.Senha, arg.HashedPassword))
			return db.Usuario{Email: arg.Email, Papel: arg.Papel}, nil
		})
	u, err := svc.CriarUsuario(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "MECANICO", u.Papel)

	store.EXPECT().CreateUsuario(gomock.Any(), gomock.Any()).Return(db.Usuario{}, crudUniqueErr)
	_, err = svc.CriarUsuario(context.Background(), req)
	require.ErrorIs(t, err, service.ErrEmailDuplicado)

	store.EXPECT().CreateUsuario(gomock.Any(), gomock.Any()).Return(db.Usuario{}, sql.ErrConnDone)
	_, err = svc.CriarUsuario(context.Background(), req)
	require.ErrorIs(t, err, sql.ErrConnDone)

	// bcrypt rejects passwords longer than 72 bytes.
	long := dto.CriarUsuarioRequest{Senha: string(make([]byte, 80))}
	_, err = svc.CriarUsuario(context.Background(), long)
	require.Error(t, err)
}

func TestBuscarListarAtualizarUsuario(t *testing.T) {
	store := crudNewStore(t)
	svc := service.NewUsuarioService(store, nil, crudConfig())
	ctx := context.Background()
	id := uuid.New()

	store.EXPECT().GetUsuario(gomock.Any(), id).Return(db.Usuario{Uuid: id}, nil)
	u, err := svc.BuscarUsuario(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, u.Uuid)

	store.EXPECT().GetUsuario(gomock.Any(), id).Return(db.Usuario{}, sql.ErrNoRows)
	_, err = svc.BuscarUsuario(ctx, id)
	require.ErrorIs(t, err, service.ErrUsuarioNaoEncontrado)

	store.EXPECT().ListUsuarios(gomock.Any(), db.ListUsuariosParams{Limit: 10, Offset: 10}).Return([]db.Usuario{{}, {}}, nil)
	list, err := svc.ListarUsuarios(ctx, dto.PaginacaoQuery{Pagina: 2, Tamanho: 10})
	require.NoError(t, err)
	require.Len(t, list, 2)

	store.EXPECT().UpdateUsuarioAtivo(gomock.Any(), db.UpdateUsuarioAtivoParams{Uuid: id, Ativo: false}).Return(db.Usuario{Uuid: id}, nil)
	_, err = svc.AtualizarAtivo(ctx, id, false)
	require.NoError(t, err)

	store.EXPECT().UpdateUsuarioAtivo(gomock.Any(), gomock.Any()).Return(db.Usuario{}, sql.ErrNoRows)
	_, err = svc.AtualizarAtivo(ctx, id, true)
	require.ErrorIs(t, err, service.ErrUsuarioNaoEncontrado)
}

func TestLogin(t *testing.T) {
	config := crudConfig()
	maker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	require.NoError(t, err)
	ctx := context.Background()
	usuario := crudUsuario(t, "secret1", true)
	req := dto.LoginRequest{Email: usuario.Email, Senha: "secret1"}

	t.Run("OK", func(t *testing.T) {
		store := crudNewStore(t)
		svc := service.NewUsuarioService(store, maker, config)
		store.EXPECT().GetUsuarioByEmail(gomock.Any(), usuario.Email).Return(usuario, nil)
		var sessaoGravada db.CreateSessaoParams
		store.EXPECT().CreateSessao(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.CreateSessaoParams) (db.Sessao, error) {
				require.Equal(t, usuario.ID, arg.UsuarioID)
				require.Equal(t, "agent", arg.UserAgent)
				sessaoGravada = arg
				return db.Sessao{ID: arg.ID}, nil
			})
		rsp, err := svc.Login(ctx, req, "agent", "127.0.0.1")
		require.NoError(t, err)
		require.NotEmpty(t, rsp.AccessToken)
		require.NotEmpty(t, rsp.RefreshToken)
		require.Equal(t, usuario.Email, rsp.Usuario.Email)

		payload, err := maker.VerifyToken(rsp.AccessToken)
		require.NoError(t, err)
		require.Equal(t, "ATENDENTE", payload.Role)
		require.Equal(t, token.TipoAccess, payload.Tipo)

		refreshPayload, err := maker.VerifyToken(rsp.RefreshToken)
		require.NoError(t, err)
		require.Equal(t, token.TipoRefresh, refreshPayload.Tipo)

		// A sessão guarda o hash, nunca o token em texto puro.
		require.NotEqual(t, rsp.RefreshToken, sessaoGravada.RefreshToken)
		require.Equal(t, service.HashRefreshToken(rsp.RefreshToken), sessaoGravada.RefreshToken)
	})

	t.Run("EmailNotFound", func(t *testing.T) {
		store := crudNewStore(t)
		svc := service.NewUsuarioService(store, maker, config)
		store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(db.Usuario{}, sql.ErrNoRows)
		_, err := svc.Login(ctx, req, "", "")
		require.ErrorIs(t, err, service.ErrCredenciaisInvalidas)
	})

	t.Run("WrongPassword", func(t *testing.T) {
		store := crudNewStore(t)
		svc := service.NewUsuarioService(store, maker, config)
		store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(usuario, nil)
		_, err := svc.Login(ctx, dto.LoginRequest{Email: usuario.Email, Senha: "wrong12"}, "", "")
		require.ErrorIs(t, err, service.ErrCredenciaisInvalidas)
	})

	t.Run("Inactive", func(t *testing.T) {
		store := crudNewStore(t)
		svc := service.NewUsuarioService(store, maker, config)
		inativo := crudUsuario(t, "secret1", false)
		store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(inativo, nil)
		_, err := svc.Login(ctx, req, "", "")
		require.ErrorIs(t, err, service.ErrUsuarioInativo)
	})

	t.Run("SessionError", func(t *testing.T) {
		store := crudNewStore(t)
		svc := service.NewUsuarioService(store, maker, config)
		store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(usuario, nil)
		store.EXPECT().CreateSessao(gomock.Any(), gomock.Any()).Return(db.Sessao{}, sql.ErrConnDone)
		_, err := svc.Login(ctx, req, "", "")
		require.ErrorIs(t, err, sql.ErrConnDone)
	})

	t.Run("TokenError", func(t *testing.T) {
		store := crudNewStore(t)
		svc := service.NewUsuarioService(store, crudFailingMaker{}, config)
		store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(usuario, nil)
		_, err := svc.Login(ctx, req, "", "")
		require.Error(t, err)
	})
}

type crudFailingMaker struct{}

func (crudFailingMaker) CreateToken(string, string, int64, uuid.UUID, string, time.Duration) (string, *token.Payload, error) {
	return "", nil, errors.New("token failure")
}

func (crudFailingMaker) VerifyToken(string) (*token.Payload, error) {
	return nil, token.ErrInvalidToken
}

// crudSecondCallFailingMaker succeeds on the first CreateToken and fails afterwards.
type crudSecondCallFailingMaker struct {
	token.Maker
	calls *int
}

func (m crudSecondCallFailingMaker) CreateToken(tipo string, u string, id int64, uid uuid.UUID, role string, d time.Duration) (string, *token.Payload, error) {
	*m.calls++
	if *m.calls > 1 {
		return "", nil, errors.New("token failure")
	}
	return m.Maker.CreateToken(tipo, u, id, uid, role, d)
}

func TestLoginRefreshTokenError(t *testing.T) {
	config := crudConfig()
	maker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	require.NoError(t, err)
	usuario := crudUsuario(t, "secret1", true)
	calls := 0

	store := crudNewStore(t)
	svc := service.NewUsuarioService(store, crudSecondCallFailingMaker{Maker: maker, calls: &calls}, config)
	store.EXPECT().GetUsuarioByEmail(gomock.Any(), gomock.Any()).Return(usuario, nil)
	_, err = svc.Login(context.Background(), dto.LoginRequest{Email: usuario.Email, Senha: "secret1"}, "", "")
	require.Error(t, err)
}

func TestRenewAccessToken(t *testing.T) {
	config := crudConfig()
	maker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	require.NoError(t, err)
	ctx := context.Background()
	userID := int64(7)

	refresh, payload, err := maker.CreateToken(token.TipoRefresh, "a@b.com", userID, uuid.New(), "MECANICO", time.Hour)
	require.NoError(t, err)
	// A sessão guarda só o hash do refresh token.
	sessaoOK := db.Sessao{ID: payload.ID, UsuarioID: userID, RefreshToken: service.HashRefreshToken(refresh), ExpiresAt: time.Now().Add(time.Hour)}
	access, _, err := maker.CreateToken(token.TipoAccess, "a@b.com", userID, uuid.New(), "MECANICO", time.Hour)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		token   string
		stub    func(store *crudStoreExpect)
		wantErr error
	}{
		{name: "OK", token: refresh, stub: func(s *crudStoreExpect) { s.sessao(sessaoOK, nil) }},
		{name: "InvalidToken", token: "invalid", stub: func(s *crudStoreExpect) {}, wantErr: service.ErrSessaoInvalida},
		{name: "AccessTokenNaoRenova", token: access, stub: func(s *crudStoreExpect) {}, wantErr: service.ErrSessaoInvalida},
		{name: "SessionNotFound", token: refresh, stub: func(s *crudStoreExpect) { s.sessao(db.Sessao{}, sql.ErrNoRows) }, wantErr: service.ErrSessaoInvalida},
		{name: "DBError", token: refresh, stub: func(s *crudStoreExpect) { s.sessao(db.Sessao{}, sql.ErrConnDone) }, wantErr: sql.ErrConnDone},
		{name: "Blocked", token: refresh, stub: func(s *crudStoreExpect) {
			blocked := sessaoOK
			blocked.IsBlocked = true
			s.sessao(blocked, nil)
		}, wantErr: service.ErrSessaoInvalida},
		{name: "WrongUser", token: refresh, stub: func(s *crudStoreExpect) {
			other := sessaoOK
			other.UsuarioID = 99
			s.sessao(other, nil)
		}, wantErr: service.ErrSessaoInvalida},
		{name: "MismatchedToken", token: refresh, stub: func(s *crudStoreExpect) {
			other := sessaoOK
			other.RefreshToken = service.HashRefreshToken("other")
			s.sessao(other, nil)
		}, wantErr: service.ErrSessaoInvalida},
		{name: "Expired", token: refresh, stub: func(s *crudStoreExpect) {
			expired := sessaoOK
			expired.ExpiresAt = time.Now().Add(-time.Minute)
			s.sessao(expired, nil)
		}, wantErr: service.ErrSessaoInvalida},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := crudNewStore(t)
			tc.stub(&crudStoreExpect{t: t, store: store})
			svc := service.NewUsuarioService(store, maker, config)
			rsp, err := svc.RenewAccessToken(ctx, dto.RenewAccessTokenRequest{RefreshToken: tc.token})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			newPayload, err := maker.VerifyToken(rsp.AccessToken)
			require.NoError(t, err)
			require.Equal(t, "MECANICO", newPayload.Role)
		})
	}
}

func TestRenewAccessTokenCreateError(t *testing.T) {
	config := crudConfig()
	maker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	require.NoError(t, err)
	refresh, payload, err := maker.CreateToken(token.TipoRefresh, "a@b.com", 1, uuid.New(), "MECANICO", time.Hour)
	require.NoError(t, err)

	calls := 1 // already "used" one call, next CreateToken fails
	store := crudNewStore(t)
	store.EXPECT().GetSessao(gomock.Any(), payload.ID).
		Return(db.Sessao{ID: payload.ID, UsuarioID: 1, RefreshToken: service.HashRefreshToken(refresh), ExpiresAt: time.Now().Add(time.Hour)}, nil)
	svc := service.NewUsuarioService(store, crudSecondCallFailingMaker{Maker: maker, calls: &calls}, config)
	_, err = svc.RenewAccessToken(context.Background(), dto.RenewAccessTokenRequest{RefreshToken: refresh})
	require.Error(t, err)
}

func TestLogout(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New()

	t.Run("OK", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{ID: sessionID, UsuarioID: 3}, nil)
		store.EXPECT().BlockSessao(gomock.Any(), sessionID).Return(nil)
		require.NoError(t, service.NewUsuarioService(store, nil, crudConfig()).Logout(ctx, sessionID, 3))
	})
	t.Run("NotFound", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{}, sql.ErrNoRows)
		err := service.NewUsuarioService(store, nil, crudConfig()).Logout(ctx, sessionID, 3)
		require.ErrorIs(t, err, service.ErrSessaoInvalida)
	})
	t.Run("DBError", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{}, sql.ErrConnDone)
		err := service.NewUsuarioService(store, nil, crudConfig()).Logout(ctx, sessionID, 3)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
	t.Run("OtherUser", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetSessao(gomock.Any(), sessionID).Return(db.Sessao{ID: sessionID, UsuarioID: 4}, nil)
		err := service.NewUsuarioService(store, nil, crudConfig()).Logout(ctx, sessionID, 3)
		require.ErrorIs(t, err, service.ErrSessaoInvalida)
	})
}
