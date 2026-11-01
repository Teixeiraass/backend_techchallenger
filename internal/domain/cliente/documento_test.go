package cliente

import (
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

func TestNovoDocumento(t *testing.T) {
	testCases := []struct {
		name      string
		valor     string
		numero    string
		tipo      TipoDocumento
		formatado string
		valido    bool
	}{
		{"CPFSemMascara", "52998224725", "52998224725", TipoCPF, "529.982.247-25", true},
		{"CPFComMascara", "529.982.247-25", "52998224725", TipoCPF, "529.982.247-25", true},
		{"CNPJSemMascara", "11222333000181", "11222333000181", TipoCNPJ, "11.222.333/0001-81", true},
		{"CNPJComMascara", "11.222.333/0001-81", "11222333000181", TipoCNPJ, "11.222.333/0001-81", true},
		{"CPFPrimeiroDigitoErrado", "52998224735", "", "", "", false},
		{"CPFSegundoDigitoErrado", "52998224726", "", "", "", false},
		{"CPFRepetido", "11111111111", "", "", "", false},
		{"CNPJPrimeiroDigitoErrado", "11222333000191", "", "", "", false},
		{"CNPJSegundoDigitoErrado", "11222333000182", "", "", "", false},
		{"CNPJRepetido", "00000000000000", "", "", "", false},
		{"TamanhoErrado", "123456", "", "", "", false},
		{"Vazio", "", "", "", "", false},
		{"Letras", "abc.def.ghi-jk", "", "", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			documento, err := NovoDocumento(tc.valor)
			if !tc.valido {
				require.ErrorIs(t, err, ErrDocumentoInvalido)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.numero, documento.Numero())
			require.Equal(t, tc.tipo, documento.Tipo())
			require.Equal(t, tc.formatado, documento.Formatado())
		})
	}
}

func TestFormatadoSemTipo(t *testing.T) {
	require.Equal(t, "123", Documento{numero: "123"}.Formatado())
}

func TestNormalizar(t *testing.T) {
	require.Equal(t, "52998224725", Normalizar(" 529.982.247-25 "))
	require.Equal(t, "", Normalizar("abc"))
}

func TestDigitoCNPJRestoMenorQueDois(t *testing.T) {
	// soma múltipla de 11 -> resto 0 -> dígito 0
	require.Equal(t, 0, digitoCNPJ([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}))
}

func TestRandomDocumentosSaoValidos(t *testing.T) {
	for i := 0; i < 200; i++ {
		cpf, err := NovoDocumento(util.RandomCPF())
		require.NoError(t, err)
		require.Equal(t, TipoCPF, cpf.Tipo())

		cnpj, err := NovoDocumento(util.RandomCNPJ())
		require.NoError(t, err)
		require.Equal(t, TipoCNPJ, cnpj.Tipo())
	}
}
