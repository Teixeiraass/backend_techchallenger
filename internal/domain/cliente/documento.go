// Package cliente contém os objetos de valor do agregado Cliente.
package cliente

import (
	"strings"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
)

type TipoDocumento string

const (
	TipoCPF  TipoDocumento = "CPF"
	TipoCNPJ TipoDocumento = "CNPJ"
)

var ErrDocumentoInvalido = domain.NewValidationError("DOCUMENTO_INVALIDO", "CPF/CNPJ inválido")

// Documento é o objeto de valor CpfCnpj: identifica o cliente e só existe
// com dígitos verificadores válidos. É guardado sem pontuação.
type Documento struct {
	numero string
	tipo   TipoDocumento
}

// NovoDocumento normaliza (remove pontos, traços e barras) e valida um CPF ou CNPJ.
func NovoDocumento(valor string) (Documento, error) {
	numero := Normalizar(valor)

	switch len(numero) {
	case 11:
		if !cpfValido(numero) {
			return Documento{}, ErrDocumentoInvalido
		}
		return Documento{numero: numero, tipo: TipoCPF}, nil
	case 14:
		if !cnpjValido(numero) {
			return Documento{}, ErrDocumentoInvalido
		}
		return Documento{numero: numero, tipo: TipoCNPJ}, nil
	default:
		return Documento{}, ErrDocumentoInvalido
	}
}

// Normalizar mantém apenas os dígitos do documento.
func Normalizar(valor string) string {
	var sb strings.Builder
	for _, r := range valor {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func (d Documento) Numero() string {
	return d.numero
}

func (d Documento) Tipo() TipoDocumento {
	return d.tipo
}

// Formatado devolve o documento com a máscara usual (000.000.000-00 ou 00.000.000/0000-00).
func (d Documento) Formatado() string {
	n := d.numero
	switch d.tipo {
	case TipoCPF:
		return n[0:3] + "." + n[3:6] + "." + n[6:9] + "-" + n[9:11]
	case TipoCNPJ:
		return n[0:2] + "." + n[2:5] + "." + n[5:8] + "/" + n[8:12] + "-" + n[12:14]
	default:
		return n
	}
}

func cpfValido(cpf string) bool {
	if todosIguais(cpf) {
		return false
	}
	digitos := paraDigitos(cpf)

	return digitos[9] == digitoCPF(digitos[:9], 10) && digitos[10] == digitoCPF(digitos[:10], 11)
}

func digitoCPF(digitos []int, peso int) int {
	soma := 0
	for _, d := range digitos {
		soma += d * peso
		peso--
	}
	resto := (soma * 10) % 11
	if resto == 10 {
		return 0
	}
	return resto
}

func cnpjValido(cnpj string) bool {
	if todosIguais(cnpj) {
		return false
	}
	digitos := paraDigitos(cnpj)

	pesos1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	pesos2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	return digitos[12] == digitoCNPJ(digitos, pesos1) && digitos[13] == digitoCNPJ(digitos, pesos2)
}

func digitoCNPJ(digitos []int, pesos []int) int {
	soma := 0
	for i, p := range pesos {
		soma += digitos[i] * p
	}
	resto := soma % 11
	if resto < 2 {
		return 0
	}
	return 11 - resto
}

func todosIguais(valor string) bool {
	for i := 1; i < len(valor); i++ {
		if valor[i] != valor[0] {
			return false
		}
	}
	return true
}

func paraDigitos(valor string) []int {
	digitos := make([]int, len(valor))
	for i, r := range valor {
		digitos[i] = int(r - '0')
	}
	return digitos
}
