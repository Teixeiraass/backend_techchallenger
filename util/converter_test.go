package util

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestToNullTime(t *testing.T) {
	require.False(t, ToNullTime(nil).Valid)

	now := time.Now()
	value := ToNullTime(&now)
	require.True(t, value.Valid)
	require.Equal(t, now, value.Time)
}

func TestFromNullTime(t *testing.T) {
	require.Nil(t, FromNullTime(sql.NullTime{}))

	now := time.Now()
	value := FromNullTime(sql.NullTime{Time: now, Valid: true})
	require.NotNil(t, value)
	require.Equal(t, now, *value)
}

func TestToNullInt64(t *testing.T) {
	require.False(t, ToNullInt64(0).Valid)
	require.Equal(t, sql.NullInt64{Int64: 7, Valid: true}, ToNullInt64(7))
}
