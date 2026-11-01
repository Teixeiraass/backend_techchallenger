package ordemservico

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalcularOrcamento(t *testing.T) {
	servicos := []Item{
		{Quantidade: 1, ValorUnitarioCentavos: 8000},
		{Quantidade: 2, ValorUnitarioCentavos: 5000},
		{Quantidade: 1, ValorUnitarioCentavos: 99999, StatusReparo: ReparoPendente},
		{Quantidade: 1, ValorUnitarioCentavos: 3000, StatusReparo: ReparoAutorizado},
	}
	pecas := []Item{
		{Quantidade: 4, ValorUnitarioCentavos: 4500},
		{Quantidade: 1, ValorUnitarioCentavos: 77777, StatusReparo: ReparoRecusado},
		{Quantidade: 2, ValorUnitarioCentavos: 1000, StatusReparo: ReparoAutorizado},
	}

	orcamento := CalcularOrcamento(servicos, pecas)
	require.Equal(t, int64(8000+10000+3000), orcamento.TotalServicosCentavos)
	require.Equal(t, int64(18000+2000), orcamento.TotalPecasCentavos)
	require.Equal(t, orcamento.TotalServicosCentavos+orcamento.TotalPecasCentavos, orcamento.TotalCentavos)

	require.Equal(t, 5, QuantidadeItens(servicos, pecas))
}

func TestCalcularOrcamentoVazio(t *testing.T) {
	orcamento := CalcularOrcamento(nil, nil)
	require.Zero(t, orcamento.TotalCentavos)
	require.Zero(t, QuantidadeItens(nil, nil))
}

func TestItem(t *testing.T) {
	item := Item{Quantidade: 3, ValorUnitarioCentavos: 250}
	require.Equal(t, int64(750), item.Subtotal())
	require.True(t, item.EntraNoOrcamento())
	require.True(t, Item{StatusReparo: ReparoAutorizado}.EntraNoOrcamento())
	require.False(t, Item{StatusReparo: ReparoPendente}.EntraNoOrcamento())
	require.False(t, Item{StatusReparo: ReparoRecusado}.EntraNoOrcamento())
}
