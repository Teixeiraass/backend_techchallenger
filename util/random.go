package util

// Os geradores deste arquivo criam dados de teste (nomes, CPFs, placas).
// Nada aqui é usado em token, senha ou outro contexto de segurança, por isso
// math/rand é suficiente (#nosec G404 nas chamadas).

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomInt generate a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1) // #nosec G404 -- dados de teste
}

// RandomString generate a random string of lenght n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)] // #nosec G404 -- dados de teste
		sb.WriteByte(c)
	}

	return sb.String()
}

func RandomName() string {
	return RandomString(8)
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}

func RandomUuid() uuid.UUID {
	return uuid.New()
}

// RandomCPF generates a CPF with valid check digits
func RandomCPF() string {
	digits := make([]int, 9, 11)
	for i := range digits {
		digits[i] = rand.Intn(10) // #nosec G404 -- dados de teste
	}
	// Evita sequências repetidas (111.111.111-11), que são CPFs inválidos.
	digits[0] = (digits[1] + 1) % 10

	digits = append(digits, checkDigit(digits, 10))
	digits = append(digits, checkDigit(digits, 11))

	return digitsToString(digits)
}

// RandomCNPJ generates a CNPJ with valid check digits
func RandomCNPJ() string {
	digits := make([]int, 12, 14)
	for i := range digits {
		digits[i] = rand.Intn(10) // #nosec G404 -- dados de teste
	}
	digits[0] = (digits[1] + 1) % 10

	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	digits = append(digits, cnpjCheckDigit(digits, weights1))
	digits = append(digits, cnpjCheckDigit(digits, weights2))

	return digitsToString(digits)
}

// RandomPlaca generates a Mercosul plate (ABC1D23)
func RandomPlaca() string {
	letters := strings.ToUpper(RandomString(4))
	numeros := rand.Perm(10) // #nosec G404 -- dados de teste
	return fmt.Sprintf("%s%d%c%d%d", letters[:3], numeros[0], letters[3], numeros[1], numeros[2])
}

func checkDigit(digits []int, weight int) int {
	sum := 0
	for _, d := range digits {
		sum += d * weight
		weight--
	}
	rest := (sum * 10) % 11
	if rest == 10 {
		return 0
	}
	return rest
}

func cnpjCheckDigit(digits []int, weights []int) int {
	sum := 0
	for i, w := range weights {
		sum += digits[i] * w
	}
	rest := sum % 11
	if rest < 2 {
		return 0
	}
	return 11 - rest
}

func digitsToString(digits []int) string {
	var sb strings.Builder
	for _, d := range digits {
		sb.WriteString(strconv.Itoa(d))
	}
	return sb.String()
}
