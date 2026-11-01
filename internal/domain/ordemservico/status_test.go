package ordemservico

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransicoes(t *testing.T) {
	permitidas := map[Status][]Status{
		StatusRecebida:            {StatusEmDiagnostico},
		StatusEmDiagnostico:       {StatusAguardandoAprovacao, StatusEntregue},
		StatusAguardandoAprovacao: {StatusEmExecucao, StatusEmDiagnostico},
		StatusEmExecucao:          {StatusFinalizada},
		StatusFinalizada:          {StatusEntregue},
		StatusEntregue:            {},
	}

	for _, origem := range Todos() {
		for _, destino := range Todos() {
			esperado := false
			for _, p := range permitidas[origem] {
				if p == destino {
					esperado = true
				}
			}
			require.Equal(t, esperado, origem.PodeIrPara(destino), "%s -> %s", origem, destino)
		}
	}

	require.False(t, Status("X").PodeIrPara(StatusRecebida))
}

func TestTodosEValido(t *testing.T) {
	todos := Todos()
	require.Len(t, todos, 6)
	require.Equal(t, StatusRecebida, todos[0])
	require.Equal(t, StatusEntregue, todos[5])
	for _, s := range todos {
		require.True(t, s.Valido())
	}
	require.False(t, Status("CANCELADA").Valido())
}

func TestDescricao(t *testing.T) {
	testCases := map[Status]string{
		StatusRecebida:            "Recebida",
		StatusEmDiagnostico:       "Em diagnóstico",
		StatusAguardandoAprovacao: "Aguardando aprovação",
		StatusEmExecucao:          "Em execução",
		StatusFinalizada:          "Finalizada",
		StatusEntregue:            "Entregue",
		Status("OUTRO"):           "OUTRO",
	}
	for status, descricao := range testCases {
		require.Equal(t, descricao, status.Descricao())
	}
}
