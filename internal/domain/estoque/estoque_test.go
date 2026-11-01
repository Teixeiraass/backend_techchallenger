package estoque

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidarTipo(t *testing.T) {
	require.NoError(t, ValidarTipo("PECA"))
	require.NoError(t, ValidarTipo("INSUMO"))
	require.ErrorIs(t, ValidarTipo("produto"), ErrTipoInvalido)
	require.ErrorIs(t, ValidarTipo(""), ErrTipoInvalido)
}

func TestValidarCadastro(t *testing.T) {
	testCases := []struct {
		name     string
		tipo     string
		preco    int64
		saldo    int32
		minimo   int32
		expected error
	}{
		{"OK", "PECA", 100, 10, 2, nil},
		{"OKZeros", "INSUMO", 0, 0, 0, nil},
		{"TipoInvalido", "X", 100, 10, 2, ErrTipoInvalido},
		{"PrecoNegativo", "PECA", -1, 10, 2, ErrPrecoInvalido},
		{"SaldoNegativo", "PECA", 100, -1, 2, ErrSaldoNegativo},
		{"MinimoNegativo", "PECA", 100, 10, -1, ErrEstoqueMinimoInvalido},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidarCadastro(tc.tipo, tc.preco, tc.saldo, tc.minimo)
			if tc.expected == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expected)
			}
		})
	}
}

func TestValidarReposicao(t *testing.T) {
	require.NoError(t, ValidarReposicao(1))
	require.ErrorIs(t, ValidarReposicao(0), ErrQuantidadeInvalida)
	require.ErrorIs(t, ValidarReposicao(-5), ErrQuantidadeInvalida)
}

func TestValidarAjuste(t *testing.T) {
	require.NoError(t, ValidarAjuste(0))
	require.NoError(t, ValidarAjuste(10))
	require.ErrorIs(t, ValidarAjuste(-1), ErrSaldoNegativo)
}

func TestValidarDisponibilidade(t *testing.T) {
	testCases := []struct {
		name       string
		saldo      int32
		jaNaOS     int32
		quantidade int32
		ativa      bool
		expected   error
	}{
		{"OK", 10, 0, 10, true, nil},
		{"OKConsiderandoOS", 10, 4, 6, true, nil},
		{"Insuficiente", 10, 0, 11, true, ErrEstoqueInsuficiente},
		{"InsuficienteComOS", 10, 5, 6, true, ErrEstoqueInsuficiente},
		{"QuantidadeZero", 10, 0, 0, true, ErrQuantidadeInvalida},
		{"Inativa", 10, 0, 1, false, ErrPecaInativa},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidarDisponibilidade(tc.saldo, tc.jaNaOS, tc.quantidade, tc.ativa)
			if tc.expected == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expected)
			}
		})
	}
}

func TestEstoqueBaixo(t *testing.T) {
	require.True(t, EstoqueBaixo(2, 2))
	require.True(t, EstoqueBaixo(1, 2))
	require.False(t, EstoqueBaixo(3, 2))
	require.True(t, EstoqueBaixo(0, 0))
}
