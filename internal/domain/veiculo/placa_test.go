package veiculo

import (
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

func TestNovaPlaca(t *testing.T) {
	testCases := []struct {
		name     string
		valor    string
		esperado string
		mercosul bool
		valida   bool
	}{
		{"Antiga", "ABC1234", "ABC1234", false, true},
		{"AntigaComHifen", "abc-1234", "ABC1234", false, true},
		{"Mercosul", "ABC1D23", "ABC1D23", true, true},
		{"MercosulMinusculaEspacos", " abc 1d23 ", "ABC1D23", true, true},
		{"Curta", "AB1234", "", false, false},
		{"Longa", "ABCD1234", "", false, false},
		{"NumerosNoInicio", "1BC1234", "", false, false},
		{"LetraNoFinal", "ABC123D", "", false, false},
		{"Vazia", "", "", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			placa, err := NovaPlaca(tc.valor)
			if !tc.valida {
				require.ErrorIs(t, err, ErrPlacaInvalida)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.esperado, placa.String())
			require.Equal(t, tc.mercosul, placa.Mercosul())
		})
	}
}

func TestRandomPlacaEValida(t *testing.T) {
	for i := 0; i < 100; i++ {
		_, err := NovaPlaca(util.RandomPlaca())
		require.NoError(t, err)
	}
}

func TestValidarAno(t *testing.T) {
	agora := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name   string
		ano    int32
		valido bool
	}{
		{"Minimo", AnoMinimo, true},
		{"Atual", 2026, true},
		{"ProximoAno", 2027, true},
		{"AntesDoMinimo", 1899, false},
		{"DoisAnosAFrente", 2028, false},
		{"Zero", 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidarAno(tc.ano, agora)
			if tc.valido {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrAnoInvalido)
			}
		})
	}
}
