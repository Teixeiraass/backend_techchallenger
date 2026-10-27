package token

import (
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

func TestPayloadValid(t *testing.T) {
	payload, err := NewPayload(util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.NoError(t, err)
	require.NoError(t, payload.Valid())

	payload.ExpiredAt = time.Now().Add(-time.Second)
	require.ErrorIs(t, payload.Valid(), ErrExpiredToken)
}
