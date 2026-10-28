package token

import (
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomEmail()
	userID := util.RandomInt(1, 10)
	userUUID := util.RandomUuid()
	duration := time.Minute

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	token, payload, err := maker.CreateToken(username, userID, userUUID, "ADMINISTRADOR", duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.Equal(t, userID, payload.UserID)
	require.Equal(t, userUUID, payload.UserUUID)
	require.Equal(t, "ADMINISTRADOR", payload.Role)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	token, _, err := maker.CreateToken(util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", -time.Minute)
	require.NoError(t, err)

	payload, err := maker.VerifyToken(token)
	require.ErrorIs(t, err, ErrExpiredToken)
	require.Nil(t, payload)
}

func TestInvalidJWTTokenAlgNone(t *testing.T) {
	payload, err := NewPayload(util.RandomEmail(), 1, util.RandomUuid(), "ADMINISTRADOR", time.Minute)
	require.NoError(t, err)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwtClaims{
		Payload: *payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	})
	token, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	payload, err = maker.VerifyToken(token)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, payload)
}

func TestInvalidJWTTokenOtherAlg(t *testing.T) {
	key := util.RandomString(32)
	payload, err := NewPayload(util.RandomEmail(), 1, util.RandomUuid(), "ADMINISTRADOR", time.Minute)
	require.NoError(t, err)

	// HS512 com a mesma chave: assinatura válida, mas algoritmo não aceito.
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS512, jwtClaims{
		Payload: *payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	})
	token, err := jwtToken.SignedString([]byte(key))
	require.NoError(t, err)

	maker, err := NewJWTMaker(key)
	require.NoError(t, err)

	payload, err = maker.VerifyToken(token)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, payload)
}

func TestInvalidJWTTokenWrongKey(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)
	other, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	token, _, err := other.CreateToken(util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.NoError(t, err)

	payload, err := maker.VerifyToken(token)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, payload)

	payload, err = maker.VerifyToken("garbage")
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, payload)
}

func TestJWTInvalidKeySize(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(31))
	require.Error(t, err)
	require.Nil(t, maker)
}

func TestNewMaker(t *testing.T) {
	key := util.RandomString(32)

	testCases := []struct {
		name      string
		tokenType string
		wantType  interface{}
		wantErr   bool
	}{
		{"Default", "", &PasetoMaker{}, false},
		{"Paseto", "paseto", &PasetoMaker{}, false},
		{"JWT", "jwt", &JWTMaker{}, false},
		{"Unsupported", "saml", nil, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker, err := NewMaker(tc.tokenType, key)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, maker)
				return
			}
			require.NoError(t, err)
			require.IsType(t, tc.wantType, maker)
		})
	}
}
