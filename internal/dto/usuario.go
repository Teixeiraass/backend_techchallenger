package dto

import (
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/google/uuid"
)

type CriarUsuarioRequest struct {
	Nome  string `json:"nome" binding:"required,max=150"`
	Email string `json:"email" binding:"required,email,max=150"`
	Senha string `json:"senha" binding:"required,min=6"`
	Papel string `json:"papel" binding:"required,oneof=ADMINISTRADOR ATENDENTE MECANICO"`
}

type AtualizarUsuarioAtivoRequest struct {
	Ativo *bool `json:"ativo" binding:"required"`
}

type UsuarioResponse struct {
	Uuid      uuid.UUID `json:"uuid"`
	Nome      string    `json:"nome"`
	Email     string    `json:"email"`
	Papel     string    `json:"papel"`
	Ativo     bool      `json:"ativo"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUsuarioResponse(usuario db.Usuario) UsuarioResponse {
	return UsuarioResponse{
		Uuid:      usuario.Uuid,
		Nome:      usuario.Nome,
		Email:     usuario.Email,
		Papel:     usuario.Papel,
		Ativo:     usuario.Ativo,
		CreatedAt: usuario.CreatedAt,
	}
}

type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required,min=6"`
}

type LoginResponse struct {
	SessionID             uuid.UUID       `json:"session_id"`
	AccessToken           string          `json:"access_token"`
	AccessTokenExpiresAt  time.Time       `json:"access_token_expires_at"`
	RefreshToken          string          `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time       `json:"refresh_token_expires_at"`
	Usuario               UsuarioResponse `json:"usuario"`
}

type RenewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RenewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

type LogoutRequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
}
