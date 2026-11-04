package token

import (
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

func TestPasetoMaker(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomEmail()
	userID := util.RandomInt(1, 10)
	userUUID := util.RandomUuid()
	duration := time.Minute

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	token, payload, err := maker.CreateToken(TipoAccess, username, userID, userUUID, "ATENDENTE", duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.Equal(t, userID, payload.UserID)
	require.Equal(t, userUUID, payload.UserUUID)
	require.Equal(t, "ATENDENTE", payload.Role)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, _, err := maker.CreateToken(TipoAccess, util.RandomEmail(), util.RandomInt(1, 10), util.RandomUuid(), "MECANICO", -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)
}

func TestInvalidPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	other, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, _, err := other.CreateToken(TipoAccess, util.RandomEmail(), 1, util.RandomUuid(), "ADMINISTRADOR", time.Minute)
	require.NoError(t, err)

	payload, err := maker.VerifyToken(token)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, payload)

	payload, err = maker.VerifyToken("not-a-token")
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, payload)
}

func TestPasetoInvalidKeySize(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(31))
	require.Error(t, err)
	require.Nil(t, maker)
}

func TestPasetoTokenCarregaTipo(t *testing.T) {
	key := util.RandomString(32)
	maker, err := NewPasetoMaker(key)
	require.NoError(t, err)

	token, _, err := maker.CreateToken(TipoRefresh, util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.NoError(t, err)
	payload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.Equal(t, TipoRefresh, payload.Tipo)

	// Token cifrado com a chave certa, mas sem tipo, é recusado.
	semTipo, err := NewPayload(TipoAccess, util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.NoError(t, err)
	semTipo.Tipo = ""
	cifrado, err := maker.(*PasetoMaker).paseto.Encrypt([]byte(key), semTipo, nil)
	require.NoError(t, err)
	payload, err = maker.VerifyToken(cifrado)
	require.ErrorIs(t, err, ErrTipoInvalido)
	require.Nil(t, payload)

	_, _, err = maker.CreateToken("outro", util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.ErrorIs(t, err, ErrTipoInvalido)
}
