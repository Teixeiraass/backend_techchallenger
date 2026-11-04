package ordemservico

import (
	"errors"
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var agora = time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

func servicoCatalogo(preco int64, ativo bool) CatalogoServico {
	return CatalogoServico{ID: 1, UUID: uuid.New(), Nome: "Troca de óleo", PrecoCentavos: preco, Ativo: ativo}
}

func pecaCatalogo(preco int64, ativa bool) CatalogoPeca {
	return CatalogoPeca{ID: 2, UUID: uuid.New(), Codigo: "FLT-01", Nome: "Filtro", Tipo: "PECA", PrecoCentavos: preco, Ativa: ativa}
}

// osComItens cria uma OS Recebida com um serviço de 8000 e duas peças de 3500.
func osComItens(t *testing.T) OrdemServico {
	t.Helper()
	os := Nova(10, 20, "Barulho ao frear")
	require.NoError(t, os.AdicionarServico(servicoCatalogo(8000, true), 1))
	require.NoError(t, os.AdicionarPeca(pecaCatalogo(3500, true), 2, 10))
	return os
}

// osNoStatus leva a OS até o status pedido pelo caminho feliz.
func osNoStatus(t *testing.T, status Status) OrdemServico {
	t.Helper()
	os := osComItens(t)
	if status == StatusRecebida {
		return os
	}
	require.NoError(t, os.IniciarDiagnostico(agora))
	if status == StatusEmDiagnostico {
		return os
	}
	require.NoError(t, os.EnviarOrcamento("diag", agora))
	if status == StatusAguardandoAprovacao {
		return os
	}
	require.NoError(t, os.AprovarOrcamento(agora))
	require.NoError(t, os.IniciarExecucao(agora))
	if status == StatusEmExecucao {
		return os
	}
	require.NoError(t, os.Finalizar(agora.Add(90*time.Minute)))
	if status == StatusFinalizada {
		return os
	}
	require.NoError(t, os.Entregar(agora.Add(2*time.Hour)))
	return os
}

func TestNova(t *testing.T) {
	os := Nova(10, 20, "Barulho")
	require.Equal(t, StatusRecebida, os.Status)
	require.Equal(t, OrcamentoNaoEnviado, os.StatusOrcamento)
	require.Equal(t, int64(10), os.ClienteID)
	require.Equal(t, int64(20), os.VeiculoID)
	require.Equal(t, "Barulho", os.DescricaoProblema)
	require.NoError(t, os.PodeAlterarItens())
	require.Empty(t, os.Itens)
	require.Zero(t, os.Orcamento().TotalCentavos)
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

// ---------------------------------------------------------------------------
// Itens
// ---------------------------------------------------------------------------

func TestAdicionarServico(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		os := Nova(1, 2, "")
		servico := servicoCatalogo(8000, true)
		require.NoError(t, os.AdicionarServico(servico, 0), "quantidade zero vale 1")
		require.NoError(t, os.AdicionarServico(servico, 2))
		require.Len(t, os.Servicos(), 2)
		require.Empty(t, os.Pecas())
		require.Equal(t, int32(1), os.Itens[0].Quantidade)
		require.Equal(t, ItemServico, os.Itens[0].Tipo)
		require.Equal(t, servico.UUID, os.Itens[0].CatalogoUUID)
		require.Equal(t, servico.Nome, os.Itens[0].Nome)
		require.Equal(t, int64(8000), os.Itens[0].ValorUnitarioCentavos, "preço congelado")
		require.Equal(t, int64(3*8000), os.ValorTotalCentavos)
	})
	t.Run("QuantidadeNegativa", func(t *testing.T) {
		os := Nova(1, 2, "")
		require.ErrorIs(t, os.AdicionarServico(servicoCatalogo(8000, true), -1), ErrQuantidadeInvalida)
		require.Empty(t, os.Itens)
	})
	t.Run("ServicoInativo", func(t *testing.T) {
		os := Nova(1, 2, "")
		require.ErrorIs(t, os.AdicionarServico(servicoCatalogo(8000, false), 1), ErrServicoInativo)
	})
	t.Run("DepoisDoEnvioDoOrcamento", func(t *testing.T) {
		for _, status := range []Status{StatusAguardandoAprovacao, StatusEmExecucao, StatusFinalizada, StatusEntregue} {
			os := osNoStatus(t, status)
			require.ErrorIs(t, os.AdicionarServico(servicoCatalogo(8000, true), 1), ErrItensNaoEditaveis, string(status))
			require.ErrorIs(t, os.PodeAlterarItens(), ErrItensNaoEditaveis)
		}
	})
}

func TestAdicionarPeca(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		os := Nova(1, 2, "")
		peca := pecaCatalogo(3500, true)
		require.NoError(t, os.AdicionarPeca(peca, 2, 5))
		require.Len(t, os.Pecas(), 1)
		item := os.Pecas()[0]
		require.Equal(t, ItemPeca, item.Tipo)
		require.Equal(t, peca.Codigo, item.Codigo)
		require.Equal(t, peca.Tipo, item.TipoPeca)
		require.Equal(t, int64(7000), item.Subtotal())
		require.Equal(t, int64(7000), os.ValorTotalCentavos)
		require.Equal(t, int64(7000), os.Orcamento().TotalPecasCentavos)
	})
	t.Run("QuantidadeZero", func(t *testing.T) {
		os := Nova(1, 2, "")
		require.ErrorIs(t, os.AdicionarPeca(pecaCatalogo(3500, true), 0, 5), estoque.ErrQuantidadeInvalida)
	})
	t.Run("PecaInativa", func(t *testing.T) {
		os := Nova(1, 2, "")
		require.ErrorIs(t, os.AdicionarPeca(pecaCatalogo(3500, false), 1, 5), estoque.ErrPecaInativa)
	})
	t.Run("SemSaldo", func(t *testing.T) {
		os := Nova(1, 2, "")
		err := os.AdicionarPeca(pecaCatalogo(3500, true), 6, 5)
		require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
		require.Contains(t, err.Error(), "FLT-01")
		require.Empty(t, os.Itens)
	})
	t.Run("ConsideraQuantidadeJaNaOS", func(t *testing.T) {
		os := Nova(1, 2, "")
		peca := pecaCatalogo(3500, true)
		require.NoError(t, os.AdicionarPeca(peca, 4, 5))
		// 4 já pedidas + 2 > saldo 5
		require.ErrorIs(t, os.AdicionarPeca(peca, 2, 5), estoque.ErrEstoqueInsuficiente)
		require.NoError(t, os.AdicionarPeca(peca, 1, 5))
	})
	t.Run("OutraPecaNaoConta", func(t *testing.T) {
		os := Nova(1, 2, "")
		require.NoError(t, os.AdicionarPeca(pecaCatalogo(3500, true), 4, 5))
		outra := pecaCatalogo(1000, true)
		outra.ID = 99
		require.NoError(t, os.AdicionarPeca(outra, 5, 5))
	})
	t.Run("DepoisDoEnvioDoOrcamento", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.ErrorIs(t, os.AdicionarPeca(pecaCatalogo(3500, true), 1, 10), ErrItensNaoEditaveis)
	})
}

func TestRemoverItem(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		os := osComItens(t)
		os.Itens[0].UUID = uuid.New()
		os.Itens[1].UUID = uuid.New()
		alvo := os.Itens[1].UUID

		require.NoError(t, os.RemoverItem(alvo))
		require.Len(t, os.Itens, 1)
		require.Equal(t, int64(8000), os.ValorTotalCentavos)
		require.Len(t, os.Alteracoes().ItensRemovidos, 1)
		require.Equal(t, alvo, os.Alteracoes().ItensRemovidos[0].UUID)
		require.Equal(t, ItemPeca, os.Alteracoes().ItensRemovidos[0].Tipo)
	})
	t.Run("NaoEncontrado", func(t *testing.T) {
		os := osComItens(t)
		require.ErrorIs(t, os.RemoverItem(uuid.New()), ErrItemNaoEncontrado)
	})
	t.Run("ItemDeReparoNaoSaiPorAqui", func(t *testing.T) {
		os := osComItens(t)
		os.Itens[0].UUID = uuid.New()
		os.Itens[0].ReparoUUID = uuid.NullUUID{UUID: uuid.New(), Valid: true}
		require.ErrorIs(t, os.RemoverItem(os.Itens[0].UUID), ErrItemNaoEncontrado)
	})
	t.Run("DepoisDoEnvioDoOrcamento", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.ErrorIs(t, os.RemoverItem(uuid.New()), ErrItensNaoEditaveis)
	})
}

func TestPecasParaBaixaNaExecucao(t *testing.T) {
	os := osNoStatus(t, StatusEmExecucao)
	reparo, err := os.RegistrarReparo("Disco", nil, []PedidoPeca{{Peca: pecaCatalogo(100, true), Quantidade: 1, SaldoDisponivel: 10}})
	require.NoError(t, err)

	baixa := os.PecasParaBaixaNaExecucao()
	require.Len(t, baixa, 1)
	require.Equal(t, int32(2), baixa[0].Quantidade)

	doReparo := os.PecasDoReparo(reparo.UUID)
	require.Len(t, doReparo, 1)
	require.Equal(t, int32(1), doReparo[0].Quantidade)
	require.Empty(t, os.PecasDoReparo(uuid.New()))
}

// ---------------------------------------------------------------------------
// Ciclo de vida
// ---------------------------------------------------------------------------

func TestIniciarDiagnostico(t *testing.T) {
	os := osNoStatus(t, StatusRecebida)
	require.NoError(t, os.IniciarDiagnostico(agora))
	require.Equal(t, StatusEmDiagnostico, os.Status)
	require.Equal(t, agora, *os.DiagnosticoIniciadoEm)

	for _, status := range []Status{StatusEmDiagnostico, StatusAguardandoAprovacao, StatusEmExecucao, StatusFinalizada, StatusEntregue} {
		other := osNoStatus(t, status)
		err := other.IniciarDiagnostico(agora)
		require.ErrorIs(t, err, ErrTransicaoInvalida, string(status))
		require.Contains(t, err.Error(), string(status))
	}
}

func TestEnviarOrcamento(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		require.NoError(t, os.EnviarOrcamento("Pastilhas gastas", agora))
		require.Equal(t, StatusAguardandoAprovacao, os.Status)
		require.Equal(t, OrcamentoAguardando, os.StatusOrcamento)
		require.Equal(t, int64(15000), os.ValorTotalCentavos)
		require.Equal(t, "Pastilhas gastas", os.Diagnostico)
		require.Equal(t, agora, *os.OrcamentoEnviadoEm)
		require.Nil(t, os.OrcamentoDecididoEm)
	})
	t.Run("MantemDiagnosticoAnterior", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		os.Diagnostico = "anterior"
		require.NoError(t, os.EnviarOrcamento("", agora))
		require.Equal(t, "anterior", os.Diagnostico)
	})
	t.Run("SemItens", func(t *testing.T) {
		os := Nova(1, 2, "")
		require.NoError(t, os.IniciarDiagnostico(agora))
		require.ErrorIs(t, os.EnviarOrcamento("", agora), ErrOrcamentoSemItens)
		require.Equal(t, StatusEmDiagnostico, os.Status)
	})
	t.Run("TransicaoInvalida", func(t *testing.T) {
		os := osNoStatus(t, StatusRecebida)
		require.ErrorIs(t, os.EnviarOrcamento("", agora), ErrTransicaoInvalida)
	})
}

func TestAprovarOrcamento(t *testing.T) {
	os := osNoStatus(t, StatusAguardandoAprovacao)
	require.False(t, os.OrcamentoJaAprovado())
	require.NoError(t, os.AprovarOrcamento(agora))
	require.Equal(t, OrcamentoAprovado, os.StatusOrcamento)
	require.Equal(t, StatusAguardandoAprovacao, os.Status)
	require.Equal(t, agora, *os.OrcamentoDecididoEm)

	// Aprovar de novo é idempotente: nada muda e não há erro.
	require.True(t, os.OrcamentoJaAprovado())
	require.NoError(t, os.AprovarOrcamento(agora.Add(time.Hour)))
	require.Equal(t, agora, *os.OrcamentoDecididoEm)
	require.Equal(t, StatusAguardandoAprovacao, os.Status)

	// Com a execução já iniciada (ou além) a aprovação repetida também é aceita sem mudar nada.
	for _, status := range []Status{StatusEmExecucao, StatusFinalizada, StatusEntregue} {
		other := osNoStatus(t, status)
		require.NoError(t, other.AprovarOrcamento(agora))
		require.Equal(t, status, other.Status)
	}

	// Sem orçamento aguardando (não enviado ou rejeitado) continua sendo conflito.
	for _, status := range []Status{StatusRecebida, StatusEmDiagnostico} {
		other := osNoStatus(t, status)
		require.ErrorIs(t, other.AprovarOrcamento(agora), ErrOrcamentoNaoAguardando)
	}
	rejeitada := osNoStatus(t, StatusAguardandoAprovacao)
	require.NoError(t, rejeitada.RejeitarOrcamento(agora))
	require.ErrorIs(t, rejeitada.AprovarOrcamento(agora), ErrOrcamentoNaoAguardando)
}

func TestRejeitarOrcamento(t *testing.T) {
	os := osNoStatus(t, StatusAguardandoAprovacao)
	require.NoError(t, os.RejeitarOrcamento(agora))
	require.Equal(t, OrcamentoRejeitado, os.StatusOrcamento)
	require.Equal(t, StatusEmDiagnostico, os.Status)
	require.NotNil(t, os.OrcamentoDecididoEm)

	require.ErrorIs(t, os.RejeitarOrcamento(agora), ErrOrcamentoNaoAguardando)
	for _, status := range []Status{StatusRecebida, StatusEmExecucao, StatusFinalizada, StatusEntregue} {
		other := osNoStatus(t, status)
		require.ErrorIs(t, other.RejeitarOrcamento(agora), ErrOrcamentoNaoAguardando)
	}

	// Itens voltam a ser editáveis e sai um novo orçamento depois da rejeição.
	require.NoError(t, os.AdicionarServico(servicoCatalogo(1000, true), 1))
	require.NoError(t, os.EnviarOrcamento("", agora))
	require.Equal(t, OrcamentoAguardando, os.StatusOrcamento)
	require.Equal(t, int64(16000), os.ValorTotalCentavos)
	require.Nil(t, os.OrcamentoDecididoEm)
}

func TestIniciarExecucao(t *testing.T) {
	t.Run("SemAprovacao", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.ErrorIs(t, os.IniciarExecucao(agora), ErrOrcamentoNaoAprovado)
		require.Equal(t, StatusAguardandoAprovacao, os.Status)
	})
	t.Run("ForaDaOrdem", func(t *testing.T) {
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

func TestFinalizar(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		require.NoError(t, os.Finalizar(agora.Add(time.Hour)))
		require.Equal(t, StatusFinalizada, os.Status)
		duracao, ok := os.TempoExecucao()
		require.True(t, ok)
		require.Equal(t, time.Hour, duracao)
	})
	t.Run("ComReparoPendente", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		_, err := os.RegistrarReparo("Disco", []PedidoServico{{Servico: servicoCatalogo(100, true), Quantidade: 1}}, nil)
		require.NoError(t, err)
		require.Equal(t, 1, os.ReparosPendentes())
		require.ErrorIs(t, os.Finalizar(agora), ErrReparoPendente)
		require.Equal(t, StatusEmExecucao, os.Status)
	})
	t.Run("ForaDaOrdem", func(t *testing.T) {
		for _, status := range []Status{StatusRecebida, StatusEmDiagnostico, StatusAguardandoAprovacao, StatusFinalizada, StatusEntregue} {
			os := osNoStatus(t, status)
			require.ErrorIs(t, os.Finalizar(agora), ErrTransicaoInvalida, string(status))
		}
	})
	t.Run("SemTempoDeExecucao", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		_, ok := os.TempoExecucao()
		require.False(t, ok)
	})
}

func TestEntregar(t *testing.T) {
	t.Run("DepoisDeFinalizada", func(t *testing.T) {
		os := osNoStatus(t, StatusFinalizada)
		require.NoError(t, os.Entregar(agora))
		require.Equal(t, StatusEntregue, os.Status)
		require.Equal(t, agora, *os.EntregueEm)
	})
	t.Run("SemExecucaoDepoisDaRejeicao", func(t *testing.T) {
		os := osNoStatus(t, StatusAguardandoAprovacao)
		require.NoError(t, os.RejeitarOrcamento(agora))
		require.NoError(t, os.Entregar(agora))
		require.Equal(t, StatusEntregue, os.Status)
	})
	t.Run("EmDiagnosticoSemRejeicao", func(t *testing.T) {
		os := osNoStatus(t, StatusEmDiagnostico)
		require.ErrorIs(t, os.Entregar(agora), ErrEntregaNaoPermitida)
	})
	t.Run("ForaDaOrdem", func(t *testing.T) {
		for _, status := range []Status{StatusRecebida, StatusAguardandoAprovacao, StatusEmExecucao, StatusEntregue} {
			os := osNoStatus(t, status)
			require.ErrorIs(t, os.Entregar(agora), ErrEntregaNaoPermitida, string(status))
		}
	})
}

// ---------------------------------------------------------------------------
// Reparo adicional
// ---------------------------------------------------------------------------

func TestRegistrarReparo(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		totalAntes := os.ValorTotalCentavos
		reparo, err := os.RegistrarReparo("Disco empenado",
			[]PedidoServico{{Servico: servicoCatalogo(5000, true), Quantidade: 0}},
			[]PedidoPeca{{Peca: pecaCatalogo(1000, true), Quantidade: 2, SaldoDisponivel: 10}})
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, reparo.UUID)
		require.Equal(t, ReparoPendente, reparo.Status)
		require.True(t, reparo.Pendente())
		require.Equal(t, "Disco empenado", reparo.Descricao)
		require.Len(t, os.Reparos, 1)
		require.Len(t, os.Itens, 4)
		for _, item := range os.Itens[2:] {
			require.True(t, item.DoReparo(reparo.UUID))
			require.Equal(t, ReparoPendente, item.StatusReparo)
			require.False(t, item.EntraNoOrcamento())
		}
		require.Equal(t, totalAntes, os.ValorTotalCentavos, "reparo pendente fica fora do orçamento")
		require.Equal(t, 1, os.ReparosPendentes())
		require.Empty(t, os.Alteracoes().ReparosDecididos)
	})
	t.Run("SemItens", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		_, err := os.RegistrarReparo("x", nil, nil)
		require.ErrorIs(t, err, ErrReparoSemItens)
	})
	t.Run("ForaDeExecucao", func(t *testing.T) {
		for _, status := range []Status{StatusRecebida, StatusEmDiagnostico, StatusAguardandoAprovacao, StatusFinalizada, StatusEntregue} {
			os := osNoStatus(t, status)
			_, err := os.RegistrarReparo("x", []PedidoServico{{Servico: servicoCatalogo(1, true), Quantidade: 1}}, nil)
			require.ErrorIs(t, err, ErrReparoForaDeExecucao, string(status))
			require.ErrorIs(t, os.PodeRegistrarReparoAdicional(), ErrReparoForaDeExecucao)
		}
	})
	t.Run("ServicoInativo", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		_, err := os.RegistrarReparo("x", []PedidoServico{{Servico: servicoCatalogo(1, false), Quantidade: 1}}, nil)
		require.ErrorIs(t, err, ErrServicoInativo)
		require.Empty(t, os.Reparos)
		require.Len(t, os.Itens, 2, "nada foi incluído")
	})
	t.Run("PecaSemSaldo", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		_, err := os.RegistrarReparo("x",
			[]PedidoServico{{Servico: servicoCatalogo(1, true), Quantidade: 1}},
			[]PedidoPeca{{Peca: pecaCatalogo(1, true), Quantidade: 3, SaldoDisponivel: 2}})
		require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
		require.Empty(t, os.Reparos)
		require.Len(t, os.Itens, 2, "nem o serviço do reparo ficou")
	})
	t.Run("PecasDoMesmoReparoContamNoSaldo", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		peca := pecaCatalogo(1, true)
		// As 2 peças do orçamento já foram baixadas (OS em execução): não contam.
		_, err := os.RegistrarReparo("x", nil, []PedidoPeca{
			{Peca: peca, Quantidade: 3, SaldoDisponivel: 5},
			{Peca: peca, Quantidade: 3, SaldoDisponivel: 5},
		})
		require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)

		_, err = os.RegistrarReparo("x", nil, []PedidoPeca{
			{Peca: peca, Quantidade: 3, SaldoDisponivel: 5},
			{Peca: peca, Quantidade: 2, SaldoDisponivel: 5},
		})
		require.NoError(t, err)
	})
	t.Run("ReparoPendenteAnteriorContaNoSaldo", func(t *testing.T) {
		os := osNoStatus(t, StatusEmExecucao)
		peca := pecaCatalogo(1, true)
		_, err := os.RegistrarReparo("primeiro", nil, []PedidoPeca{{Peca: peca, Quantidade: 4, SaldoDisponivel: 5}})
		require.NoError(t, err)
		_, err = os.RegistrarReparo("segundo", nil, []PedidoPeca{{Peca: peca, Quantidade: 2, SaldoDisponivel: 5}})
		require.ErrorIs(t, err, estoque.ErrEstoqueInsuficiente)
	})
}

func TestDecidirReparo(t *testing.T) {
	registrar := func(t *testing.T) (OrdemServico, uuid.UUID) {
		os := osNoStatus(t, StatusEmExecucao)
		reparo, err := os.RegistrarReparo("Disco",
			[]PedidoServico{{Servico: servicoCatalogo(5000, true), Quantidade: 1}},
			[]PedidoPeca{{Peca: pecaCatalogo(1000, true), Quantidade: 2, SaldoDisponivel: 10}})
		require.NoError(t, err)
		return os, reparo.UUID
	}

	t.Run("Autorizar", func(t *testing.T) {
		os, reparoUUID := registrar(t)
		require.NoError(t, os.AutorizarReparo(reparoUUID, agora))
		reparo, err := os.Reparo(reparoUUID)
		require.NoError(t, err)
		require.Equal(t, ReparoAutorizado, reparo.Status)
		require.Equal(t, agora, *reparo.DecididoEm)
		require.Equal(t, int64(15000+5000+2000), os.ValorTotalCentavos, "itens do reparo entram no orçamento")
		require.Zero(t, os.ReparosPendentes())
		require.Equal(t, []uuid.UUID{reparoUUID}, os.Alteracoes().ReparosDecididos)
		for _, item := range os.PecasDoReparo(reparoUUID) {
			require.Equal(t, ReparoAutorizado, item.StatusReparo)
		}
		require.NoError(t, os.Finalizar(agora))
	})
	t.Run("Recusar", func(t *testing.T) {
		os, reparoUUID := registrar(t)
		require.NoError(t, os.RecusarReparo(reparoUUID, agora))
		reparo, err := os.Reparo(reparoUUID)
		require.NoError(t, err)
		require.Equal(t, ReparoRecusado, reparo.Status)
		require.Equal(t, int64(15000), os.ValorTotalCentavos, "reparo recusado fica fora do orçamento")
		require.Zero(t, os.ReparosPendentes())
	})
	t.Run("JaDecidido", func(t *testing.T) {
		os, reparoUUID := registrar(t)
		require.NoError(t, os.AutorizarReparo(reparoUUID, agora))
		require.ErrorIs(t, os.RecusarReparo(reparoUUID, agora), ErrReparoJaDecidido)
		require.ErrorIs(t, os.AutorizarReparo(reparoUUID, agora), ErrReparoJaDecidido)
	})
	t.Run("NaoEncontrado", func(t *testing.T) {
		os, _ := registrar(t)
		require.ErrorIs(t, os.AutorizarReparo(uuid.New(), agora), ErrReparoNaoEncontrado)
		_, err := os.Reparo(uuid.New())
		require.ErrorIs(t, err, ErrReparoNaoEncontrado)
	})
	t.Run("ForaDeExecucao", func(t *testing.T) {
		os := osNoStatus(t, StatusFinalizada)
		os.Reparos = append(os.Reparos, ReparoAdicional{UUID: uuid.New(), Status: ReparoPendente})
		require.ErrorIs(t, os.AutorizarReparo(os.Reparos[0].UUID, agora), ErrReparoForaDeExecucao)
	})
}

func TestValidarQuantidade(t *testing.T) {
	require.NoError(t, ValidarQuantidade(1))
	require.ErrorIs(t, ValidarQuantidade(0), ErrQuantidadeInvalida)
	require.ErrorIs(t, ValidarQuantidade(-3), ErrQuantidadeInvalida)
}

func TestErrosDoAgregadoSaoDeDominio(t *testing.T) {
	// errors.Is compara pelo código: o serviço pode embrulhar sem perder o tipo.
	err := ErrTransicaoInvalida
	require.True(t, errors.Is(err, ErrTransicaoInvalida))
	require.False(t, errors.Is(err, ErrReparoPendente))
}

func TestReconstituirEHistorico(t *testing.T) {
	carregada := Reconstituir(osNoStatus(t, StatusEmDiagnostico))
	require.NoError(t, carregada.EnviarOrcamento("", agora))
	carregada.AnotarHistorico(7, "Orçamento enviado ao cliente")
	carregada.AnotarHistorico(0, "sem mudança")

	historico := carregada.Alteracoes().Historico
	require.Len(t, historico, 2)
	require.Equal(t, StatusEmDiagnostico, historico[0].StatusAnterior)
	require.Equal(t, StatusAguardandoAprovacao, historico[0].StatusNovo)
	require.Equal(t, int64(7), historico[0].UsuarioID)
	require.Equal(t, "Orçamento enviado ao cliente", historico[0].Observacao)

	carregada.LimparAlteracoes()
	require.Empty(t, carregada.Alteracoes().Historico)
	carregada.AnotarHistorico(0, "depois de salvar")
	require.Equal(t, StatusAguardandoAprovacao, carregada.Alteracoes().Historico[0].StatusAnterior, "o status de referência avança ao salvar")

	// OS nova: o status anterior da primeira entrada é vazio.
	nova := Nova(1, 2, "")
	nova.AnotarHistorico(1, "OS aberta")
	require.Equal(t, Status(""), nova.Alteracoes().Historico[0].StatusAnterior)
	require.Equal(t, StatusRecebida, nova.Alteracoes().Historico[0].StatusNovo)
}

func nomesDosEventos(os *OrdemServico) []string {
	nomes := make([]string, 0)
	for _, e := range os.Eventos() {
		nomes = append(nomes, e.Nome())
	}
	return nomes
}

// Cada comando registra o evento pivotal da linguagem ubíqua; os eventos
// levam o UUID da OS mesmo quando ela só ganhou identidade ao ser salva.
func TestEventosDoCicloDeVida(t *testing.T) {
	os := osNoStatus(t, StatusEntregue)
	require.Equal(t, []string{
		NomeOSRecebida, NomeDiagnosticoIniciado, NomeOrcamentoEnviadoAoCliente, NomeOrcamentoAprovado,
		NomeExecucaoIniciada, NomeOSFinalizada, NomeVeiculoEntregue,
	}, nomesDosEventos(&os))

	os.UUID = uuid.New()
	for _, e := range os.Eventos() {
		require.NotZero(t, e.OcorridoEm())
		switch ev := e.(type) {
		case OSRecebida:
			require.Equal(t, os.UUID, ev.OSUUID)
			require.Equal(t, int64(10), ev.ClienteID)
		case OrcamentoEnviadoAoCliente:
			require.Equal(t, os.UUID, ev.OSUUID)
			require.Equal(t, int64(15000), ev.ValorTotalCentavos)
		case OSFinalizada:
			require.Equal(t, 90*time.Minute, ev.TempoExecucao)
		}
	}

	os.LimparEventos()
	require.Empty(t, os.Eventos())

	// Aprovação repetida (idempotente) não registra evento de novo.
	require.NoError(t, os.AprovarOrcamento(agora))
	require.Empty(t, os.Eventos())
}

func TestEventosDeRejeicaoEReparo(t *testing.T) {
	os := osNoStatus(t, StatusAguardandoAprovacao)
	os.LimparEventos()
	require.NoError(t, os.RejeitarOrcamento(agora))
	require.Equal(t, []string{NomeOrcamentoRejeitado}, nomesDosEventos(&os))
	require.IsType(t, OrcamentoRejeitadoPeloCliente{}, os.Eventos()[0])

	os = osNoStatus(t, StatusEmExecucao)
	os.LimparEventos()
	reparo, err := os.RegistrarReparo("Disco", []PedidoServico{{Servico: servicoCatalogo(100, true), Quantidade: 1}}, nil)
	require.NoError(t, err)
	require.NoError(t, os.AutorizarReparo(reparo.UUID, agora))
	eventos := os.Eventos()
	require.Equal(t, []string{NomeReparoAdicionalIdentificado, NomeReparoAdicionalAutorizado}, nomesDosEventos(&os))
	require.Equal(t, "Disco", eventos[0].(ReparoAdicionalIdentificado).Descricao)
	require.Equal(t, reparo.UUID, eventos[1].(ReparoAdicionalAutorizado).ReparoUUID)

	os = osNoStatus(t, StatusEmExecucao)
	os.LimparEventos()
	reparo, err = os.RegistrarReparo("Disco", []PedidoServico{{Servico: servicoCatalogo(100, true), Quantidade: 1}}, nil)
	require.NoError(t, err)
	require.NoError(t, os.RecusarReparo(reparo.UUID, agora))
	require.Equal(t, NomeReparoAdicionalRecusado, os.Eventos()[1].Nome())
	require.Equal(t, "Disco", os.Eventos()[1].(ReparoAdicionalRecusado).Descricao)
}
