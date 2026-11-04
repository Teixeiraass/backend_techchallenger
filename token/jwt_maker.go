package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minSecretKeySize = 32

// JWTMaker is a JSON Web Token maker (HS256)
type JWTMaker struct {
	secretKey string
}

type jwtClaims struct {
	Payload
	jwt.RegisteredClaims
}

// NewJWTMaker creates a new JWTMaker
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}
	return &JWTMaker{secretKey: secretKey}, nil
}

func (maker *JWTMaker) CreateToken(tipo string, username string, userID int64, userUUID uuid.UUID, role string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(tipo, username, userID, userUUID, role, duration)
	if err != nil {
		return "", payload, err
	}

	claims := jwtClaims{
		Payload: *payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        payload.ID.String(),
			Subject:   payload.UserUUID.String(),
			IssuedAt:  jwt.NewNumericDate(payload.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	claims := &jwtClaims{}
	_, err := jwt.ParseWithClaims(token, claims, keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims.Tipo != TipoAccess && claims.Tipo != TipoRefresh {
		return nil, ErrTipoInvalido
	}

	return &claims.Payload, nil
}

// NewMaker cria o emissor de token escolhido em TOKEN_TYPE (jwt ou paseto).
// JWT é o padrão, como pede o enunciado; PASETO fica como opção.
func NewMaker(tokenType string, symmetricKey string) (Maker, error) {
	switch tokenType {
	case "", "jwt":
		return NewJWTMaker(symmetricKey)
	case "paseto":
		return NewPasetoMaker(symmetricKey)
	default:
		return nil, fmt.Errorf("unsupported token type %q", tokenType)
	}
}
