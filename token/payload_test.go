package token

import (
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

func TestPayloadValid(t *testing.T) {
	payload, err := NewPayload(TipoAccess, util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.NoError(t, err)
	require.NoError(t, payload.Valid())

	payload.ExpiredAt = time.Now().Add(-time.Second)
	require.ErrorIs(t, payload.Valid(), ErrExpiredToken)
}

func TestNewPayloadRecusaTipoDesconhecido(t *testing.T) {
	payload, err := NewPayload("outro", util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.ErrorIs(t, err, ErrTipoInvalido)
	require.Nil(t, payload)
}

func TestPayloadEhDoTipo(t *testing.T) {
	access, err := NewPayload(TipoAccess, util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.NoError(t, err)
	require.True(t, access.EhDoTipo(TipoAccess))
	require.False(t, access.EhDoTipo(TipoRefresh))

	refresh, err := NewPayload(TipoRefresh, util.RandomEmail(), 1, util.RandomUuid(), "ATENDENTE", time.Minute)
	require.NoError(t, err)
	require.True(t, refresh.EhDoTipo(TipoRefresh))
}
