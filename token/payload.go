package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Tipos de token emitidos. O tipo vai no token (claim "typ") para que o
// middleware recuse um refresh token no lugar de um access token e vice-versa.
const (
	TipoAccess  = "access"
	TipoRefresh = "refresh"
)

// Erros devolvidos por VerifyToken.
var (
	ErrExpiredToken = errors.New("token expirado")
	ErrInvalidToken = errors.New("token inválido")
	ErrTipoInvalido = errors.New("tipo de token inválido")
)

// Payload contains the payload data of the token
type Payload struct {
	ID        uuid.UUID `json:"id"`
	Tipo      string    `json:"typ"`
	UserID    int64     `json:"user_id"`
	UserUUID  uuid.UUID `json:"user_uuid"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// NewPayload creates a new token payload with a specific type, user and duration
func NewPayload(tipo string, username string, userID int64, userUUID uuid.UUID, role string, duration time.Duration) (*Payload, error) {
	if tipo != TipoAccess && tipo != TipoRefresh {
		return nil, ErrTipoInvalido
	}

	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &Payload{
		ID:        tokenID,
		Tipo:      tipo,
		UserID:    userID,
		UserUUID:  userUUID,
		Username:  username,
		Role:      role,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}

	return payload, nil
}

// Valid checks if the token payload is valid or not
func (payload *Payload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}

// EhDoTipo informa se o token foi emitido com o tipo esperado.
func (payload *Payload) EhDoTipo(tipo string) bool {
	return payload.Tipo == tipo
}
