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

	token, payload, err := maker.CreateToken(username, userID, userUUID, "ATENDENTE", duration)
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

	token, _, err := maker.CreateToken(util.RandomEmail(), util.RandomInt(1, 10), util.RandomUuid(), "MECANICO", -time.Minute)
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

	token, _, err := other.CreateToken(util.RandomEmail(), 1, util.RandomUuid(), "ADMINISTRADOR", time.Minute)
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
