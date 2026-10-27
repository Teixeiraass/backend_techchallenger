package util

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomHelpers(t *testing.T) {
	for i := 0; i < 50; i++ {
		n := RandomInt(1, 10)
		require.GreaterOrEqual(t, n, int64(1))
		require.LessOrEqual(t, n, int64(10))
	}

	require.Len(t, RandomString(12), 12)
	require.Len(t, RandomName(), 8)
	require.True(t, strings.HasSuffix(RandomEmail(), "@email.com"))
	require.NotEqual(t, RandomUuid(), RandomUuid())

	mercosul := regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
	for i := 0; i < 50; i++ {
		require.Regexp(t, mercosul, RandomPlaca())
		require.Len(t, RandomCPF(), 11)
		require.Len(t, RandomCNPJ(), 14)
	}
}

func TestCheckDigits(t *testing.T) {
	// 529.982.247-25
	require.Equal(t, 2, checkDigit([]int{5, 2, 9, 9, 8, 2, 2, 4, 7}, 10))
	require.Equal(t, 5, checkDigit([]int{5, 2, 9, 9, 8, 2, 2, 4, 7, 2}, 11))
	// resto 10 -> 0: soma*10 % 11 == 10 quando soma % 11 == 1
	require.Equal(t, 0, checkDigit([]int{1}, 1))

	// 11.222.333/0001-81
	cnpj := []int{1, 1, 2, 2, 2, 3, 3, 3, 0, 0, 0, 1}
	require.Equal(t, 8, cnpjCheckDigit(cnpj, []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}))
	require.Equal(t, 0, cnpjCheckDigit([]int{0}, []int{1}))
}
