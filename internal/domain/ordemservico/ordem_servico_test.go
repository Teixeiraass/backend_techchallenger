package ordemservico

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	agora    = time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	servicos = []Item{{Quantidade: 1, ValorUnitarioCentavos: 8000}}
	pecas    = []Item{{Quantidade: 2, ValorUnitarioCentavos: 3500}}
)

func osNoStatus(t *testing.T, status Status) OrdemServico {
	os := Nova()
	if status == StatusRecebida {
		return os
	}
	require.NoError(t, os.IniciarDiagnostico(agora))
	if status == StatusEmDiagnostico {
		return os
	}
	require.NoError(t, os.EnviarOrcamento(servicos, pecas, "diag", agora))
	if status == StatusAguardandoAprovacao {
		return os
	}
	require.NoError(t, os.AprovarOrcamento(agora))
	require.NoError(t, os.IniciarExecucao(agora))
	if status == StatusEmExecucao {
		return os
	}
	require.NoError(t, os.Finalizar(0, agora.Add(90*time.Minute)))
	if status == StatusFinalizada {
		return os
	}
	require.NoError(t, os.Entregar(agora.Add(2*time.Hour)))
	return os
}

func TestNova(t *testing.T) {
	os := Nova()
	require.Equal(t, StatusRecebida, os.Status)
	require.Equal(t, OrcamentoNaoEnviado, os.StatusOrcamento)
	require.NoError(t, os.PodeAlterarItens())
}

func TestFluxoCompleto(t *testing.T) {
	os := osNoStatus(t, StatusEntregue)
	require.Equal(t, StatusEntregue, os.Status)
	require.Equal(t, OrcamentoAprovado, os.StatusOrcamento)
	require.Equal(t, int64(8000+7000), os.ValorTotalCentavos)
	require.Equal(t, "diag", os.Diagnostico)
	require.NotNil(t, os.DiagnosticoIniciadoEm)
	require.NotNil(t, os.OrcamentoEnviadoEm)
	require.NotNil(t, os.OrcamentoDecididoEm)
	require.NotNil(t, os.ExecucaoIniciadaEm)
	require.NotNil(t, os.FinalizadaEm)
	require.NotNil(t, os.EntregueEm)

	duracao, ok := os.TempoExecucao()
	require.True(t, ok)
	require.Equal(t, 90*time.Minute, duracao)
}

func TestIniciarDiagnostico(t *testing.T) {
	for _, status := range Todos() {
		t.Run(string(status), func(t *testing.T) {
			os := osNoStatus(t, status)
			err := os.IniciarDiagnostico(agora)
			if status == StatusRecebida {
				require.NoError(t, err)
				require.Equal(t, StatusEmDiagnostico, os.Status)
				return
			}
			require.ErrorIs(t, err, ErrTransicaoInvalida)
			require.Equal(t, status, os.Status)
		})
	}
}

func TestPodeAlterarItens(t *testing.T) {
	for _, status := range Todos() {
		os := osNoStatus(t, status)
		err := os.PodeAlterarItens()
		if status == StatusRecebida || status == StatusEmDiagnostico {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, ErrItensNaoEditaveis)
		}
	}
}

func TestAtualizarOrcamento(t *testing.T) {
	os := Nova()
	orcamento := os.AtualizarOrcamento(servicos, pecas)
	require.Equal(t, int64(15000), orcamento.TotalCentavos)
	require.Equal(t, int64(15000), os.ValorTotalCentavos)
}

func TestEnviarOrcamento(t *testing.T) {
	t.Run("SemItens", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		err := os.EnviarOrcamento(nil, nil, "", agora)
		require.ErrorIs(t, err, ErrOrcamentoSemItens)
		require.Equal(t, StatusEmDiagnostico, os.Status)
	})

	t.Run("SoItensDeReparoPendente", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		err := os.EnviarOrcamento([]Item{{Quantidade: 1, ValorUnitarioCentavos: 1, StatusReparo: ReparoPendente}}, nil, "", agora)
		require.ErrorIs(t, err, ErrOrcamentoSemItens)
	})

	t.Run("AindaRecebida", func(t *testing.T) {
		os := Nova()
		err := os.EnviarOrcamento(servicos, nil, "", agora)
		require.ErrorIs(t, err, ErrTransicaoInvalida)
	})

	t.Run("DiagnosticoVazioMantemAnterior", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		os.Diagnostico = "anterior"
		require.NoError(t, os.EnviarOrcamento(servicos, nil, "", agora))
		require.Equal(t, "anterior", os.Diagnostico)
		require.Equal(t, OrcamentoAguardando, os.StatusOrcamento)
		require.Nil(t, os.OrcamentoDecididoEm)
		require.Equal(t, int64(8000), os.ValorTotalCentavos)
	})
}

func TestAprovarOrcamento(t *testing.T) {
	os := osNoStatus(t, StatusAguardandoAprovacao)
	require.NoError(t, os.AprovarOrcamento(agora))
	require.Equal(t, OrcamentoAprovado, os.StatusOrcamento)
	require.Equal(t, StatusAguardandoAprovacao, os.Status)

	// já decidido
	require.ErrorIs(t, os.AprovarOrcamento(agora), ErrOrcamentoNaoAguardando)

	for _, status := range []Status{StatusRecebida, StatusEmDiagnostico, StatusEmExecucao, StatusFinalizada, StatusEntregue} {
		other := osNoStatus(t, status)
		require.ErrorIs(t, other.AprovarOrcamento(agora), ErrOrcamentoNaoAguardando)
	}
}

func TestRejeitarOrcamento(t *testing.T) {
	os := osNoStatus(t, StatusAguardandoAprovacao)
	require.NoError(t, os.RejeitarOrcamento(agora))
	require.Equal(t, OrcamentoRejeitado, os.StatusOrcamento)
	require.Equal(t, StatusEmDiagnostico, os.Status)
	require.NotNil(t, os.OrcamentoDecididoEm)

	require.ErrorIs(t, os.RejeitarOrcamento(agora), ErrOrcamentoNaoAguardando)

	// novo orçamento depois da rejeição
	require.NoError(t, os.EnviarOrcamento(servicos, nil, "", agora))
	require.Equal(t, OrcamentoAguardando, os.StatusOrcamento)
	require.Nil(t, os.OrcamentoDecididoEm)
}

func TestIniciarExecucao(t *testing.T) {
	t.Run("SemAprovacao", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.ErrorIs(t, os.IniciarExecucao(agora), ErrOrcamentoNaoAprovado)
		require.Equal(t, StatusAguardandoAprovacao, os.Status)
	})

	t.Run("Rejeitado", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.NoError(t, os.RejeitarOrcamento(agora))
		require.ErrorIs(t, os.IniciarExecucao(agora), ErrOrcamentoNaoAprovado)
	})

	t.Run("JaEmExecucao", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		require.ErrorIs(t, os.IniciarExecucao(agora), ErrTransicaoInvalida)
	})

	t.Run("OK", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.NoError(t, os.AprovarOrcamento(agora))
		require.NoError(t, os.IniciarExecucao(agora))
		require.Equal(t, StatusEmExecucao, os.Status)
		require.Equal(t, agora, *os.ExecucaoIniciadaEm)
	})
}

func TestPodeRegistrarReparoAdicional(t *testing.T) {
	for _, status := range Todos() {
		os := osNoStatus(t, status)
		err := os.PodeRegistrarReparoAdicional()
		if status == StatusEmExecucao {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, ErrReparoForaDeExecucao)
		}
	}
}

func TestFinalizar(t *testing.T) {
	t.Run("ReparoPendente", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		require.ErrorIs(t, os.Finalizar(1, agora), ErrReparoPendente)
		require.Equal(t, StatusEmExecucao, os.Status)
		require.Nil(t, os.FinalizadaEm)
	})

	t.Run("ForaDeExecucao", func(t *testing.T) {
		for _, status := range []Status{StatusRecebida, StatusEmDiagnostico, StatusAguardandoAprovacao, StatusFinalizada, StatusEntregue} {
			os := osNoStatus(t, status)
			require.ErrorIs(t, os.Finalizar(0, agora), ErrTransicaoInvalida)
		}
	})

	t.Run("OK", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		require.NoError(t, os.Finalizar(0, agora))
		require.Equal(t, StatusFinalizada, os.Status)
	})
}

func TestEntregar(t *testing.T) {
	t.Run("Finalizada", func(t *testing.T) {
		os := osNoStatus(t, StatusFinalizada)
		require.NoError(t, os.Entregar(agora))
		require.Equal(t, StatusEntregue, os.Status)
		require.NotNil(t, os.EntregueEm)
	})

	t.Run("DepoisDeRejeicao", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.NoError(t, os.RejeitarOrcamento(agora))
		require.NoError(t, os.Entregar(agora))
		require.Equal(t, StatusEntregue, os.Status)
		_, ok := os.TempoExecucao()
		require.False(t, ok)
	})

	t.Run("EmDiagnosticoSemRejeicao", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		require.ErrorIs(t, os.Entregar(agora), ErrEntregaNaoPermitida)
	})

	t.Run("OutrosStatus", func(t *testing.T) {
		for _, status := range []Status{StatusRecebida, StatusAguardandoAprovacao, StatusEmExecucao, StatusEntregue} {
			os := osNoStatus(t, status)
			require.ErrorIs(t, os.Entregar(agora), ErrEntregaNaoPermitida, status)
		}
	})
}

func TestTransicaoInvalidaMensagem(t *testing.T) {
	os := Nova()
	err := os.Finalizar(0, agora)
	require.ErrorIs(t, err, ErrTransicaoInvalida)
	require.Contains(t, err.Error(), "RECEBIDA -> FINALIZADA")
}

func TestTempoExecucaoIncompleto(t *testing.T) {
	os := osNoStatus(t, StatusEmExecucao)
	_, ok := os.TempoExecucao()
	require.False(t, ok)
}

func TestValidarQuantidade(t *testing.T) {
	require.NoError(t, ValidarQuantidade(1))
	require.ErrorIs(t, ValidarQuantidade(0), ErrQuantidadeInvalida)
	require.ErrorIs(t, ValidarQuantidade(-1), ErrQuantidadeInvalida)
}

func TestDecidirReparo(t *testing.T) {
	testCases := []struct {
		name     string
		reparo   StatusReparo
		statusOS Status
		expected error
	}{
		{"OK", ReparoPendente, StatusEmExecucao, nil},
		{"JaAutorizado", ReparoAutorizado, StatusEmExecucao, ErrReparoJaDecidido},
		{"JaRecusado", ReparoRecusado, StatusEmExecucao, ErrReparoJaDecidido},
		{"OSFinalizada", ReparoPendente, StatusFinalizada, ErrReparoForaDeExecucao},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := DecidirReparo(tc.reparo, tc.statusOS)
			if tc.expected == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expected)
			}
		})
	}
}
