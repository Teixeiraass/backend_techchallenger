package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

type UsuarioService interface {
	CriarUsuario(ctx context.Context, req dto.CriarUsuarioRequest) (*db.Usuario, error)
	BuscarUsuario(ctx context.Context, usuarioUUID uuid.UUID) (*db.Usuario, error)
	ListarUsuarios(ctx context.Context, paginacao dto.PaginacaoQuery) ([]db.Usuario, error)
	AtualizarAtivo(ctx context.Context, usuarioUUID uuid.UUID, ativo bool) (*db.Usuario, error)
	Login(ctx context.Context, req dto.LoginRequest, userAgent string, clientIp string) (*dto.LoginResponse, error)
	RenewAccessToken(ctx context.Context, req dto.RenewAccessTokenRequest) (*dto.RenewAccessTokenResponse, error)
	Logout(ctx context.Context, sessionID uuid.UUID, usuarioID int64) error
}

type usuarioService struct {
	store      db.Store
	tokenMaker token.Maker
	config     util.Config
}

func NewUsuarioService(store db.Store, tokenMaker token.Maker, config util.Config) UsuarioService {
	return &usuarioService{
		store:      store,
		tokenMaker: tokenMaker,
		config:     config,
	}
}

func (s *usuarioService) CriarUsuario(ctx context.Context, req dto.CriarUsuarioRequest) (*db.Usuario, error) {
	hashedPassword, err := util.HashPassword(req.Senha)
	if err != nil {
		return nil, err
	}

	usuario, err := s.store.CreateUsuario(ctx, db.CreateUsuarioParams{
		Nome:           req.Nome,
		Email:          req.Email,
		HashedPassword: hashedPassword,
		Papel:          req.Papel,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailDuplicado
		}
		return nil, err
	}
	return &usuario, nil
}

func (s *usuarioService) BuscarUsuario(ctx context.Context, usuarioUUID uuid.UUID) (*db.Usuario, error) {
	usuario, err := s.store.GetUsuario(ctx, usuarioUUID)
	if err != nil {
		return nil, notFound(err, ErrUsuarioNaoEncontrado)
	}
	return &usuario, nil
}

func (s *usuarioService) ListarUsuarios(ctx context.Context, paginacao dto.PaginacaoQuery) ([]db.Usuario, error) {
	return s.store.ListUsuarios(ctx, db.ListUsuariosParams{
		Limit:  paginacao.Limite(),
		Offset: paginacao.Deslocamento(),
	})
}

func (s *usuarioService) AtualizarAtivo(ctx context.Context, usuarioUUID uuid.UUID, ativo bool) (*db.Usuario, error) {
	usuario, err := s.store.UpdateUsuarioAtivo(ctx, db.UpdateUsuarioAtivoParams{Uuid: usuarioUUID, Ativo: ativo})
	if err != nil {
		return nil, notFound(err, ErrUsuarioNaoEncontrado)
	}
	return &usuario, nil
}

func (s *usuarioService) Login(ctx context.Context, req dto.LoginRequest, userAgent string, clientIp string) (*dto.LoginResponse, error) {
	usuario, err := s.store.GetUsuarioByEmail(ctx, req.Email)
	if err != nil {
		// Mesmo erro para e-mail inexistente e senha errada: não revela quais e-mails existem.
		return nil, notFound(err, ErrCredenciaisInvalidas)
	}

	if err := util.CheckPassword(req.Senha, usuario.HashedPassword); err != nil {
		return nil, ErrCredenciaisInvalidas
	}

	if !usuario.Ativo {
		return nil, ErrUsuarioInativo
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(
		token.TipoAccess,
		usuario.Email,
		usuario.ID,
		usuario.Uuid,
		usuario.Papel,
		s.config.AccessTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshPayload, err := s.tokenMaker.CreateToken(
		token.TipoRefresh,
		usuario.Email,
		usuario.ID,
		usuario.Uuid,
		usuario.Papel,
		s.config.RefreshTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	sessao, err := s.store.CreateSessao(ctx, db.CreateSessaoParams{
		ID:           refreshPayload.ID,
		UsuarioID:    usuario.ID,
		RefreshToken: HashRefreshToken(refreshToken),
		UserAgent:    userAgent,
		ClientIp:     clientIp,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		SessionID:             sessao.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		Usuario:               dto.NewUsuarioResponse(usuario),
	}, nil
}

func (s *usuarioService) RenewAccessToken(ctx context.Context, req dto.RenewAccessTokenRequest) (*dto.RenewAccessTokenResponse, error) {
	refreshPayload, err := s.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil || !refreshPayload.EhDoTipo(token.TipoRefresh) {
		// Um access token (ou qualquer outro tipo) não renova sessão.
		return nil, ErrSessaoInvalida
	}

	sessao, err := s.store.GetSessao(ctx, refreshPayload.ID)
	if err != nil {
		return nil, notFound(err, ErrSessaoInvalida)
	}

	if sessao.IsBlocked ||
		sessao.UsuarioID != refreshPayload.UserID ||
		sessao.RefreshToken != HashRefreshToken(req.RefreshToken) ||
		time.Now().After(sessao.ExpiresAt) {
		return nil, ErrSessaoInvalida
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(
		token.TipoAccess,
		refreshPayload.Username,
		refreshPayload.UserID,
		refreshPayload.UserUUID,
		refreshPayload.Role,
		s.config.AccessTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	return &dto.RenewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}, nil
}

func (s *usuarioService) Logout(ctx context.Context, sessionID uuid.UUID, usuarioID int64) error {
	sessao, err := s.store.GetSessao(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSessaoInvalida
		}
		return err
	}

	if sessao.UsuarioID != usuarioID {
		return ErrSessaoInvalida
	}

	return s.store.BlockSessao(ctx, sessionID)
}

// HashRefreshToken devolve o SHA-256 (hex) do refresh token. A sessão guarda
// só o hash: quem lê o banco não consegue reutilizar o token.
func HashRefreshToken(refreshToken string) string {
	sum := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(sum[:])
}
