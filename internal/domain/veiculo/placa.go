// Package veiculo contém os objetos de valor e invariantes do agregado Veículo.
package veiculo

import (
	"regexp"
	"strings"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
)

const AnoMinimo = 1900

var (
	ErrPlacaInvalida = domain.NewValidationError("PLACA_INVALIDA", "placa inválida: use o padrão antigo (ABC1234) ou Mercosul (ABC1D23)")
	ErrAnoInvalido   = domain.NewValidationError("ANO_INVALIDO", "ano do veículo inválido")

	// Padrão antigo: 3 letras + 4 números. Mercosul: 3 letras + número + letra + 2 números.
	placaAntiga   = regexp.MustCompile(`^[A-Z]{3}[0-9]{4}$`)
	placaMercosul = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
)

// Placa é o objeto de valor que identifica o veículo. É guardada em
// maiúsculas e sem hífen.
type Placa struct {
	valor string
}

// NovaPlaca normaliza (maiúsculas, sem hífen nem espaços) e valida a placa.
func NovaPlaca(valor string) (Placa, error) {
	normalizada := strings.ToUpper(strings.TrimSpace(valor))
	normalizada = strings.ReplaceAll(normalizada, "-", "")
	normalizada = strings.ReplaceAll(normalizada, " ", "")

	if !placaAntiga.MatchString(normalizada) && !placaMercosul.MatchString(normalizada) {
		return Placa{}, ErrPlacaInvalida
	}
	return Placa{valor: normalizada}, nil
}

func (p Placa) String() string {
	return p.valor
}

// Mercosul indica se a placa segue o padrão Mercosul.
func (p Placa) Mercosul() bool {
	return placaMercosul.MatchString(p.valor)
}

// ValidarAno aceita anos entre 1900 e o ano seguinte ao atual (modelo do próximo ano).
func ValidarAno(ano int32, agora time.Time) error {
	if ano < AnoMinimo || int(ano) > agora.Year()+1 {
		return ErrAnoInvalido
	}
	return nil
}
