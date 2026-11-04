package estoque

import (
	"errors"
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

func TestErrEstoqueInsuficientePara(t *testing.T) {
	err := ErrEstoqueInsuficientePara("Filtro", "FLT-01")
	require.True(t, errors.Is(err, ErrEstoqueInsuficiente))
	require.Equal(t, "estoque insuficiente para a peça: Filtro (FLT-01)", err.Error())
}

func cadastroPeca() Cadastro {
	return Cadastro{Codigo: " flt-01 ", Nome: " Filtro ", Descricao: "d", Tipo: "PECA", Unidade: "", PrecoCentavos: 3500, EstoqueMinimo: 2}
}

func TestNovaPeca(t *testing.T) {
	p, mov, err := NovaPeca(cadastroPeca(), 10)
	require.NoError(t, err)
	require.Equal(t, "FLT-01", p.Codigo)
	require.Equal(t, "Filtro", p.Nome)
	require.Equal(t, "UN", p.Unidade, "unidade vazia vira UN")
	require.Equal(t, TipoPeca, p.Tipo)
	require.True(t, p.Ativa)
	require.Equal(t, int32(10), p.Saldo)
	require.NotNil(t, mov)
	require.Equal(t, MovimentoEntrada, mov.Tipo)
	require.Equal(t, int32(10), mov.Quantidade)
	require.Equal(t, int32(0), mov.SaldoAnterior)
	require.Equal(t, int32(10), mov.SaldoPosterior)
	require.Equal(t, "Saldo inicial", mov.Observacao)

	semSaldo, mov, err := NovaPeca(cadastroPeca(), 0)
	require.NoError(t, err)
	require.Nil(t, mov, "sem saldo inicial não há movimento")
	require.Equal(t, int32(0), semSaldo.Saldo)

	_, _, err = NovaPeca(Cadastro{Tipo: "X"}, 0)
	require.ErrorIs(t, err, ErrTipoInvalido)
	_, _, err = NovaPeca(cadastroPeca(), -1)
	require.ErrorIs(t, err, ErrSaldoNegativo)
}

func TestPecaAtualizarEInativar(t *testing.T) {
	p, _, err := NovaPeca(cadastroPeca(), 5)
	require.NoError(t, err)

	novo := cadastroPeca()
	novo.Nome = "Filtro premium"
	novo.Unidade = "cx"
	novo.EstoqueMinimo = 4
	require.NoError(t, p.Atualizar(novo, false))
	require.Equal(t, "Filtro premium", p.Nome)
	require.Equal(t, "CX", p.Unidade)
	require.Equal(t, int32(4), p.EstoqueMinimo)
	require.False(t, p.Ativa)
	require.Equal(t, int32(5), p.Saldo, "o saldo não muda pelo cadastro")

	novo.PrecoCentavos = -1
	require.ErrorIs(t, p.Atualizar(novo, true), ErrPrecoInvalido)

	p.Ativa = true
	p.Inativar()
	require.False(t, p.Ativa)
}

func TestPecaBaixar(t *testing.T) {
	p, _, err := NovaPeca(cadastroPeca(), 5)
	require.NoError(t, err)
	p.ID = 9

	mov, err := p.Baixar(2, 77, 3)
	require.NoError(t, err)
	require.Equal(t, int32(3), p.Saldo)
	require.Equal(t, MovimentoSaida, mov.Tipo)
	require.Equal(t, int32(-2), mov.Quantidade)
	require.Equal(t, int32(5), mov.SaldoAnterior)
	require.Equal(t, int32(3), mov.SaldoPosterior)
	require.Equal(t, int64(77), mov.OrdemServicoID)
	require.Equal(t, int64(3), mov.UsuarioID)
	require.Equal(t, int64(9), mov.PecaID)
	require.False(t, p.EstoqueBaixo())

	_, err = p.Baixar(4, 77, 3)
	require.ErrorIs(t, err, ErrEstoqueInsuficiente)
	require.Contains(t, err.Error(), "FLT-01")
	require.Equal(t, int32(3), p.Saldo, "sem saldo nada muda")

	_, err = p.Baixar(0, 77, 3)
	require.ErrorIs(t, err, ErrQuantidadeInvalida)

	_, err = p.Baixar(1, 77, 3)
	require.NoError(t, err)
	require.True(t, p.EstoqueBaixo(), "saldo 2 <= mínimo 2")
}

func TestPecaReporEAjustar(t *testing.T) {
	p, _, err := NovaPeca(cadastroPeca(), 1)
	require.NoError(t, err)

	mov, err := p.Repor(4, 2, "compra")
	require.NoError(t, err)
	require.Equal(t, int32(5), p.Saldo)
	require.Equal(t, MovimentoEntrada, mov.Tipo)
	require.Equal(t, int32(4), mov.Quantidade)
	require.Equal(t, int32(1), mov.SaldoAnterior)
	require.Equal(t, "compra", mov.Observacao)
	_, err = p.Repor(0, 2, "")
	require.ErrorIs(t, err, ErrQuantidadeInvalida)

	mov, err = p.Ajustar(2, 2, "inventário")
	require.NoError(t, err)
	require.Equal(t, int32(2), p.Saldo)
	require.Equal(t, MovimentoAjuste, mov.Tipo)
	require.Equal(t, int32(-3), mov.Quantidade)
	require.Equal(t, int32(5), mov.SaldoAnterior)
	require.Equal(t, int32(2), mov.SaldoPosterior)
	_, err = p.Ajustar(-1, 2, "")
	require.ErrorIs(t, err, ErrSaldoNegativo)
}
