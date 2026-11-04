package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/veiculo"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/google/uuid"
)

// Comandos da OS. Padrão de cada um: abre a unidade de trabalho, carrega o
// agregado com a linha travada, chama o método da raiz, anota o histórico e
// salva pelo repositório. O estado final é lido de novo para a resposta.

// ---------------------------------------------------------------------------
// Abertura e itens
// ---------------------------------------------------------------------------

func (s *ordemServicoService) AbrirOrdemServico(ctx context.Context, req dto.AbrirOrdemServicoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	documento, err := cliente.NovoDocumento(req.Documento)
	if err != nil {
		return nil, err
	}
	placa, err := veiculo.NovaPlaca(req.Placa)
	if err != nil {
		return nil, err
	}

	var osUUID uuid.UUID
	var os ordemservico.OrdemServico
	err = s.uow.Executar(ctx, func(r Repositorios) error {
		dono, err := r.Catalogo().ClientePorDocumento(ctx, documento.Numero())
		if err != nil {
			return err
		}
		v, err := r.Catalogo().VeiculoPorPlaca(ctx, placa.String())
		if err != nil {
			return err
		}
		if v.ClienteID != dono.ID {
			return ErrVeiculoDeOutroCliente
		}

		os = ordemservico.Nova(dono.ID, v.ID, req.DescricaoProblema)
		os.AnotarHistorico(usuarioID, "OS aberta")
		for _, item := range req.Servicos {
			servico, err := r.Catalogo().Servico(ctx, item.ServicoUuid)
			if err != nil {
				return err
			}
			if err := os.AdicionarServico(servico, item.Quantidade); err != nil {
				return err
			}
		}
		for _, item := range req.Pecas {
			peca, err := r.Pecas().Carregar(ctx, item.PecaUuid)
			if err != nil {
				return err
			}
			if err := os.AdicionarPeca(catalogoPeca(peca), item.Quantidade, peca.Saldo); err != nil {
				return err
			}
		}
		if err := r.OrdemServico().Salvar(ctx, &os); err != nil {
			return err
		}
		osUUID = os.UUID
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publicar(ctx, &os, nil)

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) AdicionarServico(ctx context.Context, osUUID uuid.UUID, req dto.ItemServicoRequest) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(r Repositorios, os *ordemservico.OrdemServico) error {
		if err := os.PodeAlterarItens(); err != nil {
			return err
		}
		servico, err := r.Catalogo().Servico(ctx, req.ServicoUuid)
		if err != nil {
			return err
		}
		return os.AdicionarServico(servico, req.Quantidade)
	})
}

func (s *ordemServicoService) AdicionarPeca(ctx context.Context, osUUID uuid.UUID, req dto.ItemPecaRequest) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(r Repositorios, os *ordemservico.OrdemServico) error {
		if err := os.PodeAlterarItens(); err != nil {
			return err
		}
		peca, err := r.Pecas().Carregar(ctx, req.PecaUuid)
		if err != nil {
			return err
		}
		return os.AdicionarPeca(catalogoPeca(peca), req.Quantidade, peca.Saldo)
	})
}

func (s *ordemServicoService) RemoverServico(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(_ Repositorios, os *ordemservico.OrdemServico) error {
		return os.RemoverItem(itemUUID)
	})
}

func (s *ordemServicoService) RemoverPeca(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(_ Repositorios, os *ordemservico.OrdemServico) error {
		return os.RemoverItem(itemUUID)
	})
}

// catalogoPeca é a visão que a OS tem de uma peça do estoque.
func catalogoPeca(p *estoque.Peca) ordemservico.CatalogoPeca {
	return ordemservico.CatalogoPeca{
		ID:            p.ID,
		UUID:          p.UUID,
		Codigo:        p.Codigo,
		Nome:          p.Nome,
		Tipo:          string(p.Tipo),
		PrecoCentavos: p.PrecoCentavos,
		Ativa:         p.Ativa,
	}
}

// ---------------------------------------------------------------------------
// Ciclo de vida
// ---------------------------------------------------------------------------

func (s *ordemServicoService) IniciarDiagnostico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(_ Repositorios, os *ordemservico.OrdemServico) error {
		if err := os.IniciarDiagnostico(s.now()); err != nil {
			return err
		}
		os.AnotarHistorico(usuarioID, "Diagnóstico iniciado")
		return nil
	})
}

func (s *ordemServicoService) EnviarOrcamento(ctx context.Context, osUUID uuid.UUID, req dto.EnviarOrcamentoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var os *ordemservico.OrdemServico
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		os, err = r.OrdemServico().CarregarParaAtualizar(ctx, osUUID)
		if err != nil {
			return err
		}
		if err := os.EnviarOrcamento(req.Diagnostico, s.now()); err != nil {
			return err
		}
		os.AnotarHistorico(usuarioID, "Orçamento enviado ao cliente")
		return r.OrdemServico().Salvar(ctx, os)
	})
	if err != nil {
		return nil, err
	}
	// Depois do commit: "Orçamento enviado ao cliente" dispara a notificação.
	s.publicar(ctx, os, nil)

	return s.BuscarOrdemServico(ctx, osUUID)
}

// AprovarOrcamento grava a aprovação do cliente e, em seguida, inicia a
// execução. Esta é a política "ao aprovar o orçamento, iniciar a execução"
// do event storming, mantida síncrona de propósito: faz parte da mesma
// requisição do cliente, que precisa receber o estado final. São duas
// transações: a aprovação fica registrada mesmo quando a execução não pode
// começar (falta de peça, falha transitória). Por isso a operação é
// idempotente: repetir a aprovação devolve o estado atual com 200, nunca 409.
//   - aprovação gravada e OS ainda Aguardando aprovação: tenta iniciar a
//     execução de novo (com aviso se ainda faltar peça);
//   - OS já Em execução ou além: devolve o estado atual com o aviso
//     "Orçamento já aprovado".
func (s *ordemServicoService) AprovarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	var jaAprovada bool
	var statusAtual ordemservico.Status
	var os *ordemservico.OrdemServico
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		os, err = r.OrdemServico().CarregarParaAtualizar(ctx, osUUID)
		if err != nil {
			return err
		}
		if err := validarTitular(ctx, r.Catalogo(), os.ClienteID, documento); err != nil {
			return err
		}

		jaAprovada = os.OrcamentoJaAprovado()
		statusAtual = os.Status
		if err := os.AprovarOrcamento(s.now()); err != nil {
			return err
		}
		if jaAprovada {
			// Nada mudou: não grava estado nem histórico de novo.
			return nil
		}
		os.AnotarHistorico(0, "Orçamento aprovado pelo cliente")
		return r.OrdemServico().Salvar(ctx, os)
	})
	if err != nil {
		return nil, err
	}
	s.publicar(ctx, os, nil)

	if jaAprovada && statusAtual != ordemservico.StatusAguardandoAprovacao {
		response, err := s.BuscarOrdemServico(ctx, osUUID)
		if err != nil {
			return nil, err
		}
		response.Aviso = "Orçamento já aprovado"
		return response, nil
	}

	response, err := s.IniciarExecucao(ctx, osUUID, 0)
	if errors.Is(err, estoque.ErrEstoqueInsuficiente) {
		response, err = s.BuscarOrdemServico(ctx, osUUID)
		if err != nil {
			return nil, err
		}
		response.Aviso = "Orçamento aprovado. Falta uma peça no estoque: a oficina inicia o serviço assim que ela chegar."
		return response, nil
	}
	return response, err
}

func (s *ordemServicoService) RejeitarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(r Repositorios, os *ordemservico.OrdemServico) error {
		if err := validarTitular(ctx, r.Catalogo(), os.ClienteID, documento); err != nil {
			return err
		}
		if err := os.RejeitarOrcamento(s.now()); err != nil {
			return err
		}
		os.AnotarHistorico(0, "Orçamento rejeitado pelo cliente")
		return nil
	})
}

// IniciarExecucao muda a OS para Em execução e baixa o estoque das peças na
// mesma transação: se uma peça não tiver saldo, nada é gravado.
func (s *ordemServicoService) IniciarExecucao(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var eventosEstoque []domain.Evento
	response, err := s.executar(ctx, osUUID, func(r Repositorios, os *ordemservico.OrdemServico) error {
		if err := os.IniciarExecucao(s.now()); err != nil {
			return err
		}
		var err error
		eventosEstoque, err = baixarEstoque(ctx, r.Pecas(), os.ID, os.PecasParaBaixaNaExecucao(), usuarioID)
		if err != nil {
			return err
		}
		os.AnotarHistorico(usuarioID, "Execução iniciada e estoque baixado")
		return nil
	}, func() []domain.Evento { return eventosEstoque })
	if err != nil {
		// A transação foi desfeita; "Estoque insuficiente detectado" ainda é
		// publicado para a política de reposição avisar o administrador.
		s.publicarFaltaDePeca(ctx, err)
		return nil, err
	}
	return response, nil
}

func (s *ordemServicoService) FinalizarOrdemServico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var os *ordemservico.OrdemServico
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		os, err = r.OrdemServico().CarregarParaAtualizar(ctx, osUUID)
		if err != nil {
			return err
		}
		if err := os.Finalizar(s.now()); err != nil {
			return err
		}

		observacao := "OS finalizada"
		if duracao, ok := os.TempoExecucao(); ok {
			observacao = fmt.Sprintf("OS finalizada. Tempo de execução: %s", formatarDuracao(int64(duracao.Seconds())))
		}
		os.AnotarHistorico(usuarioID, observacao)
		return r.OrdemServico().Salvar(ctx, os)
	})
	if err != nil {
		return nil, err
	}
	// Depois do commit: "OS finalizada" dispara o aviso de veículo pronto.
	s.publicar(ctx, os, nil)

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) EntregarVeiculo(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(_ Repositorios, os *ordemservico.OrdemServico) error {
		if err := os.Entregar(s.now()); err != nil {
			return err
		}
		observacao := "Veículo entregue ao cliente"
		if os.StatusOrcamento == ordemservico.OrcamentoRejeitado {
			observacao = "Veículo entregue sem execução (orçamento rejeitado)"
		}
		os.AnotarHistorico(usuarioID, observacao)
		return nil
	})
}

// ---------------------------------------------------------------------------
// Reparo adicional
// ---------------------------------------------------------------------------

func (s *ordemServicoService) RegistrarReparoAdicional(ctx context.Context, osUUID uuid.UUID, req dto.RegistrarReparoAdicionalRequest) (*dto.OrdemServicoResponse, error) {
	if len(req.Servicos)+len(req.Pecas) == 0 {
		return nil, ordemservico.ErrReparoSemItens
	}

	var os *ordemservico.OrdemServico
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		os, err = r.OrdemServico().CarregarParaAtualizar(ctx, osUUID)
		if err != nil {
			return err
		}
		if err := os.PodeRegistrarReparoAdicional(); err != nil {
			return err
		}

		servicos := make([]ordemservico.PedidoServico, 0, len(req.Servicos))
		for _, item := range req.Servicos {
			servico, err := r.Catalogo().Servico(ctx, item.ServicoUuid)
			if err != nil {
				return err
			}
			servicos = append(servicos, ordemservico.PedidoServico{Servico: servico, Quantidade: item.Quantidade})
		}
		pecas := make([]ordemservico.PedidoPeca, 0, len(req.Pecas))
		for _, item := range req.Pecas {
			peca, err := r.Pecas().Carregar(ctx, item.PecaUuid)
			if err != nil {
				return err
			}
			pecas = append(pecas, ordemservico.PedidoPeca{Peca: catalogoPeca(peca), Quantidade: item.Quantidade, SaldoDisponivel: peca.Saldo})
		}

		if _, err := os.RegistrarReparo(req.Descricao, servicos, pecas); err != nil {
			return err
		}
		// Sem mudança de status: a linha do histórico leva o nome do evento.
		os.AnotarHistorico(0, ordemservico.NomeReparoAdicionalIdentificado+": "+req.Descricao)
		return r.OrdemServico().Salvar(ctx, os)
	})
	if err != nil {
		return nil, err
	}
	// Depois do commit: "Reparo adicional identificado" chama o cliente a decidir.
	s.publicar(ctx, os, nil)

	return s.BuscarOrdemServico(ctx, osUUID)
}

// AutorizarReparoAdicional inclui os itens do reparo no orçamento e baixa o
// estoque das peças dele (a OS já está em execução).
func (s *ordemServicoService) AutorizarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	var eventosEstoque []domain.Evento
	response, err := s.executar(ctx, osUUID, func(r Repositorios, os *ordemservico.OrdemServico) error {
		if err := validarTitular(ctx, r.Catalogo(), os.ClienteID, documento); err != nil {
			return err
		}
		if err := os.AutorizarReparo(reparoUUID, s.now()); err != nil {
			return err
		}
		reparo, err := os.Reparo(reparoUUID)
		if err != nil {
			return err
		}
		eventosEstoque, err = baixarEstoque(ctx, r.Pecas(), os.ID, os.PecasDoReparo(reparoUUID), 0)
		if err != nil {
			return err
		}
		os.AnotarHistorico(0, "Reparo adicional autorizado pelo cliente: "+reparo.Descricao)
		return nil
	}, func() []domain.Evento { return eventosEstoque })
	if err != nil {
		s.publicarFaltaDePeca(ctx, err)
		return nil, err
	}
	return response, nil
}

func (s *ordemServicoService) RecusarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, func(r Repositorios, os *ordemservico.OrdemServico) error {
		if err := validarTitular(ctx, r.Catalogo(), os.ClienteID, documento); err != nil {
			return err
		}
		if err := os.RecusarReparo(reparoUUID, s.now()); err != nil {
			return err
		}
		reparo, err := os.Reparo(reparoUUID)
		if err != nil {
			return err
		}
		os.AnotarHistorico(0, "Reparo adicional recusado pelo cliente: "+reparo.Descricao)
		return nil
	})
}

// ---------------------------------------------------------------------------
// Esqueleto comum dos comandos
// ---------------------------------------------------------------------------

// executar roda um comando sobre o agregado travado, salva o resultado pelo
// repositório e, depois do commit, publica os eventos da raiz (mais os
// extras, como os de estoque). O comando anota o histórico quando há
// mudança de status ou decisão do cliente; alterações de itens não deixam
// linha no histórico.
func (s *ordemServicoService) executar(ctx context.Context, osUUID uuid.UUID, comando func(Repositorios, *ordemservico.OrdemServico) error, extras ...func() []domain.Evento) (*dto.OrdemServicoResponse, error) {
	var os *ordemservico.OrdemServico
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		os, err = r.OrdemServico().CarregarParaAtualizar(ctx, osUUID)
		if err != nil {
			return err
		}
		if err := comando(r, os); err != nil {
			return err
		}
		return r.OrdemServico().Salvar(ctx, os)
	})
	if err != nil {
		return nil, err
	}
	var deEstoque []domain.Evento
	for _, lista := range extras {
		deEstoque = append(deEstoque, lista()...)
	}
	s.publicar(ctx, os, deEstoque)
	return s.BuscarOrdemServico(ctx, osUUID)
}
