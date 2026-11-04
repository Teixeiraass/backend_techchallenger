package domain

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestErrorConstructors(t *testing.T) {
	testCases := []struct {
		name string
		err  *Error
		kind Kind
	}{
		{"Validation", NewValidationError("A", "a"), KindValidation},
		{"Conflict", NewConflictError("B", "b"), KindConflict},
		{"NotFound", NewNotFoundError("C", "c"), KindNotFound},
		{"Forbidden", NewForbiddenError("D", "d"), KindForbidden},
		{"Unauthorized", NewUnauthorizedError("E", "e"), KindUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.kind, tc.err.Kind)
			require.Equal(t, tc.err.Message, tc.err.Error())
			require.Equal(t, tc.kind, KindOf(tc.err))
		})
	}
}

func TestErrorIs(t *testing.T) {
	base := NewConflictError("ESTOQUE", "estoque insuficiente")
	sameCode := NewConflictError("ESTOQUE", "outra mensagem")
	other := NewConflictError("OUTRO", "outro")

	wrapped := fmt.Errorf("%w: filtro", base)

	require.True(t, errors.Is(sameCode, base))
	require.True(t, errors.Is(wrapped, base))
	require.False(t, errors.Is(other, base))
	require.False(t, base.Is(errors.New("plain")))
	require.Equal(t, KindConflict, KindOf(wrapped))
	require.Equal(t, Kind(0), KindOf(errors.New("plain")))
}

type eventoDeTeste struct{ em time.Time }

func (e eventoDeTeste) Nome() string          { return "teste" }
func (e eventoDeTeste) OcorridoEm() time.Time { return e.em }

func TestRegistrador(t *testing.T) {
	var r Registrador
	require.Empty(t, r.Eventos())
	r.Registrar(eventoDeTeste{em: time.Now()})
	r.Registrar(eventoDeTeste{})
	require.Len(t, r.Eventos(), 2)
	require.Equal(t, "teste", r.Eventos()[0].Nome())
	r.LimparEventos()
	require.Empty(t, r.Eventos())
}
