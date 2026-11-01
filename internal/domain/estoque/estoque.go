// Package estoque contém as regras do agregado Peça (peças e insumos) e do
// controle de saldo.
package estoque

import (
	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
)

// Tipo diferencia peça (item trocado) de insumo (material consumido).
type Tipo string

const (
	TipoPeca   Tipo = "PECA"
	TipoInsumo Tipo = "INSUMO"
)

type TipoMovimento string

const (
	MovimentoEntrada TipoMovimento = "ENTRADA"
	MovimentoSaida   TipoMovimento = "SAIDA"
	MovimentoAjuste  TipoMovimento = "AJUSTE"
)

var (
	ErrTipoInvalido          = domain.NewValidationError("TIPO_PECA_INVALIDO", "tipo deve ser PECA ou INSUMO")
	ErrQuantidadeInvalida    = domain.NewValidationError("QUANTIDADE_INVALIDA", "a quantidade deve ser maior que zero")
	ErrSaldoNegativo         = domain.NewValidationError("SALDO_NEGATIVO", "o saldo de estoque não pode ser negativo")
	ErrEstoqueMinimoInvalido = domain.NewValidationError("ESTOQUE_MINIMO_INVALIDO", "o estoque mínimo não pode ser negativo")
	ErrPrecoInvalido         = domain.NewValidationError("PRECO_INVALIDO", "o preço não pode ser negativo")
	ErrEstoqueInsuficiente   = domain.NewConflictError("ESTOQUE_INSUFICIENTE", "estoque insuficiente para a peça")
	ErrPecaInativa           = domain.NewConflictError("PECA_INATIVA", "a peça está inativa")
)

// ValidarTipo garante que o tipo é PECA ou INSUMO.
func ValidarTipo(tipo string) error {
	switch Tipo(tipo) {
	case TipoPeca, TipoInsumo:
		return nil
	default:
		return ErrTipoInvalido
	}
}

// ValidarCadastro valida os valores numéricos informados no cadastro da peça.
func ValidarCadastro(tipo string, precoCentavos int64, saldoInicial int32, estoqueMinimo int32) error {
	if err := ValidarTipo(tipo); err != nil {
		return err
	}
	if precoCentavos < 0 {
		return ErrPrecoInvalido
	}
	if saldoInicial < 0 {
		return ErrSaldoNegativo
	}
	if estoqueMinimo < 0 {
		return ErrEstoqueMinimoInvalido
	}
	return nil
}

// ValidarReposicao: reposição só com quantidade maior que zero.
func ValidarReposicao(quantidade int32) error {
	if quantidade <= 0 {
		return ErrQuantidadeInvalida
	}
	return nil
}

// ValidarAjuste: o ajuste de inventário define o novo saldo, que não pode ser negativo.
func ValidarAjuste(novoSaldo int32) error {
	if novoSaldo < 0 {
		return ErrSaldoNegativo
	}
	return nil
}

// ValidarDisponibilidade confere se há saldo para a quantidade pedida,
// considerando o que a mesma OS já reservou em itens anteriores da peça.
func ValidarDisponibilidade(saldo int32, jaNaOS int32, quantidade int32, ativa bool) error {
	if quantidade <= 0 {
		return ErrQuantidadeInvalida
	}
	if !ativa {
		return ErrPecaInativa
	}
	if jaNaOS+quantidade > saldo {
		return ErrEstoqueInsuficiente
	}
	return nil
}

// EstoqueBaixo indica se o saldo chegou ao estoque mínimo (dispara pedido de reposição).
func EstoqueBaixo(saldo int32, estoqueMinimo int32) bool {
	return saldo <= estoqueMinimo
}
