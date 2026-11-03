package service

import (
	"context"
	"errors"
	"fmt"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/veiculo"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/google/uuid"
)

// Comandos da OS. Padrão de cada um: abre a transação, carrega o agregado com
// a linha travada, chama o método da raiz e persiste com salvarOS. O estado
// final é lido de novo fora da transação para a resposta.

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
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		dono, err := q.GetClienteByDocumento(ctx, documento.Numero())
		if err != nil {
			return notFound(err, ErrClienteNaoEncontrado)
		}
		v, err := q.GetVeiculoByPlaca(ctx, placa.String())
		if err != nil {
			return notFound(err, ErrVeiculoNaoEncontrado)
		}
		if v.ClienteID != dono.ID {
			return ErrVeiculoDeOutroCliente
		}

		os := ordemservico.Nova(dono.ID, v.ID, req.DescricaoProblema)
		if err := criarOS(ctx, q, &os); err != nil {
			return err
		}
		osUUID = os.UUID

		if err := registrarHistorico(ctx, q, os.ID, "", string(os.Status), usuarioID, "OS aberta"); err != nil {
			return err
		}
		for _, item := range req.Servicos {
			servico, err := servicoDoCatalogo(ctx, q, item.ServicoUuid)
			if err != nil {
				return err
			}
			if err := os.AdicionarServico(servico, item.Quantidade); err != nil {
				return err
			}
		}
		for _, item := range req.Pecas {
			peca, saldo, err := pecaDoCatalogo(ctx, q, item.PecaUuid)
			if err != nil {
				return err
			}
			if err := os.AdicionarPeca(peca, item.Quantidade, saldo); err != nil {
				return err
			}
		}
		return salvarOS(ctx, q, &os, os.Status, usuarioID, "")
	})
	if err != nil {
		return nil, err
	}

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) AdicionarServico(ctx context.Context, osUUID uuid.UUID, req dto.ItemServicoRequest) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, 0, "", func(q db.Querier, os *ordemservico.OrdemServico) error {
		if err := os.PodeAlterarItens(); err != nil {
			return err
		}
		servico, err := servicoDoCatalogo(ctx, q, req.ServicoUuid)
		if err != nil {
			return err
		}
		return os.AdicionarServico(servico, req.Quantidade)
	})
}

func (s *ordemServicoService) AdicionarPeca(ctx context.Context, osUUID uuid.UUID, req dto.ItemPecaRequest) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, 0, "", func(q db.Querier, os *ordemservico.OrdemServico) error {
		if err := os.PodeAlterarItens(); err != nil {
			return err
		}
		peca, saldo, err := pecaDoCatalogo(ctx, q, req.PecaUuid)
		if err != nil {
			return err
		}
		return os.AdicionarPeca(peca, req.Quantidade, saldo)
	})
}

func (s *ordemServicoService) RemoverServico(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, 0, "", func(_ db.Querier, os *ordemservico.OrdemServico) error {
		return os.RemoverItem(itemUUID)
	})
}

func (s *ordemServicoService) RemoverPeca(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, 0, "", func(_ db.Querier, os *ordemservico.OrdemServico) error {
		return os.RemoverItem(itemUUID)
	})
}

// ---------------------------------------------------------------------------
// Ciclo de vida
// ---------------------------------------------------------------------------

func (s *ordemServicoService) IniciarDiagnostico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	return s.executar(ctx, osUUID, usuarioID, "Diagnóstico iniciado", func(_ db.Querier, os *ordemservico.OrdemServico) error {
		return os.IniciarDiagnostico(s.now())
	})
}

func (s *ordemServicoService) EnviarOrcamento(ctx context.Context, osUUID uuid.UUID, req dto.EnviarOrcamentoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var os *ordemservico.OrdemServico
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		os, err = carregarOS(ctx, q, osUUID, true)
		if err != nil {
			return err
		}
		anterior := os.Status
		if err := os.EnviarOrcamento(req.Diagnostico, s.now()); err != nil {
			return err
		}
		return salvarOS(ctx, q, os, anterior, usuarioID, "Orçamento enviado ao cliente")
	})
	if err != nil {
		return nil, err
	}

	// Política: ao enviar o orçamento, o cliente é notificado.
	s.notificarCliente(ctx, os.ClienteID, "Orçamento disponível para aprovação", fmt.Sprintf(
		"O orçamento da sua ordem de serviço é de %s. Aprove ou rejeite pelo link de acompanhamento: /api/v1/ordens-servico/%s/andamento",
		formatarDinheiro(os.ValorTotalCentavos), osUUID,
	))

	return s.BuscarOrdemServico(ctx, osUUID)
}

// AprovarOrcamento grava a aprovação do cliente e, em seguida, inicia a
// execução (política "ao aprovar, iniciar execução"). São duas transações de
// propósito: a aprovação fica registrada mesmo quando a execução não pode
// começar (falta de peça, falha transitória). Por isso a operação é
// idempotente: repetir a aprovação devolve o estado atual com 200, nunca 409.
//   - aprovação gravada e OS ainda Aguardando aprovação: tenta iniciar a
//     execução de novo (com aviso se ainda faltar peça);
//   - OS já Em execução ou além: devolve o estado atual com o aviso
//     "Orçamento já aprovado".
func (s *ordemServicoService) AprovarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	var jaAprovada bool
	var statusAtual ordemservico.Status
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		os, err := carregarOS(ctx, q, osUUID, true)
		if err != nil {
			return err
		}
		if err := validarTitular(ctx, q, os.ClienteID, documento); err != nil {
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
		return salvarOS(ctx, q, os, statusAtual, 0, "Orçamento aprovado pelo cliente")
	})
	if err != nil {
		return nil, err
	}

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
	return s.executar(ctx, osUUID, 0, "Orçamento rejeitado pelo cliente", func(q db.Querier, os *ordemservico.OrdemServico) error {
		if err := validarTitular(ctx, q, os.ClienteID, documento); err != nil {
			return err
		}
		return os.RejeitarOrcamento(s.now())
	})
}

// IniciarExecucao muda a OS para Em execução e baixa o estoque das peças na
// mesma transação: se uma peça não tiver saldo, nada é gravado.
func (s *ordemServicoService) IniciarExecucao(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var pecasEstoqueBaixo []db.Peca
	response, err := s.executar(ctx, osUUID, usuarioID, "Execução iniciada e estoque baixado", func(q db.Querier, os *ordemservico.OrdemServico) error {
		if err := os.IniciarExecucao(s.now()); err != nil {
			return err
		}
		var err error
		pecasEstoqueBaixo, err = baixarEstoque(ctx, q, os.ID, os.PecasParaBaixaNaExecucao(), usuarioID)
		return err
	})
	if err != nil {
		// Política: sem saldo, o administrador recebe o pedido de reposição.
		pedirReposicaoFaltante(ctx, s.sender, s.config.AdminNotificationTo, osUUID, err)
		return nil, err
	}

	notificarEstoqueBaixo(ctx, s.sender, s.config.AdminNotificationTo, pecasEstoqueBaixo)
	return response, nil
}

func (s *ordemServicoService) FinalizarOrdemServico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var os *ordemservico.OrdemServico
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		os, err = carregarOS(ctx, q, osUUID, true)
		if err != nil {
			return err
		}
		anterior := os.Status
		if err := os.Finalizar(s.now()); err != nil {
			return err
		}

		observacao := "OS finalizada"
		if duracao, ok := os.TempoExecucao(); ok {
			observacao = fmt.Sprintf("OS finalizada. Tempo de execução: %s", formatarDuracao(int64(duracao.Seconds())))
		}
		return salvarOS(ctx, q, os, anterior, usuarioID, observacao)
	})
	if err != nil {
		return nil, err
	}

	// Política: ao finalizar, o cliente é avisado de que o veículo está pronto.
	s.notificarCliente(ctx, os.ClienteID, "Seu veículo está pronto",
		fmt.Sprintf("A ordem de serviço %s foi finalizada. O veículo já pode ser retirado.", osUUID))

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) EntregarVeiculo(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var os *ordemservico.OrdemServico
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		os, err = carregarOS(ctx, q, osUUID, true)
		if err != nil {
			return err
		}
		anterior := os.Status
		if err := os.Entregar(s.now()); err != nil {
			return err
		}

		observacao := "Veículo entregue ao cliente"
		if os.StatusOrcamento == ordemservico.OrcamentoRejeitado {
			observacao = "Veículo entregue sem execução (orçamento rejeitado)"
		}
		return salvarOS(ctx, q, os, anterior, usuarioID, observacao)
	})
	if err != nil {
		return nil, err
	}

	return s.BuscarOrdemServico(ctx, osUUID)
}

// ---------------------------------------------------------------------------
// Reparo adicional
// ---------------------------------------------------------------------------

func (s *ordemServicoService) RegistrarReparoAdicional(ctx context.Context, osUUID uuid.UUID, req dto.RegistrarReparoAdicionalRequest) (*dto.OrdemServicoResponse, error) {
	if len(req.Servicos)+len(req.Pecas) == 0 {
		return nil, ordemservico.ErrReparoSemItens
	}

	var os *ordemservico.OrdemServico
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		os, err = carregarOS(ctx, q, osUUID, true)
		if err != nil {
			return err
		}
		if err := os.PodeRegistrarReparoAdicional(); err != nil {
			return err
		}

		servicos := make([]ordemservico.PedidoServico, 0, len(req.Servicos))
		for _, item := range req.Servicos {
			servico, err := servicoDoCatalogo(ctx, q, item.ServicoUuid)
			if err != nil {
				return err
			}
			servicos = append(servicos, ordemservico.PedidoServico{Servico: servico, Quantidade: item.Quantidade})
		}
		pecas := make([]ordemservico.PedidoPeca, 0, len(req.Pecas))
		for _, item := range req.Pecas {
			peca, saldo, err := pecaDoCatalogo(ctx, q, item.PecaUuid)
			if err != nil {
				return err
			}
			pecas = append(pecas, ordemservico.PedidoPeca{Peca: peca, Quantidade: item.Quantidade, SaldoDisponivel: saldo})
		}

		if _, err := os.RegistrarReparo(req.Descricao, servicos, pecas); err != nil {
			return err
		}
		return salvarOS(ctx, q, os, os.Status, 0, "")
	})
	if err != nil {
		return nil, err
	}

	// Política: ao identificar o reparo, o cliente é chamado a decidir.
	s.notificarCliente(ctx, os.ClienteID, "Reparo adicional aguardando autorização", fmt.Sprintf(
		"Durante a execução encontramos outro problema: %s. Autorize ou recuse pelo link de acompanhamento: /api/v1/ordens-servico/%s/andamento",
		req.Descricao, osUUID,
	))

	return s.BuscarOrdemServico(ctx, osUUID)
}

// AutorizarReparoAdicional inclui os itens do reparo no orçamento e baixa o
// estoque das peças dele (a OS já está em execução).
func (s *ordemServicoService) AutorizarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	var pecasEstoqueBaixo []db.Peca
	var descricao string
	response, err := s.executarComObservacao(ctx, osUUID, 0, func(q db.Querier, os *ordemservico.OrdemServico) (string, error) {
		if err := validarTitular(ctx, q, os.ClienteID, documento); err != nil {
			return "", err
		}
		if err := os.AutorizarReparo(reparoUUID, s.now()); err != nil {
			return "", err
		}
		reparo, err := os.Reparo(reparoUUID)
		if err != nil {
			return "", err
		}
		descricao = reparo.Descricao
		pecasEstoqueBaixo, err = baixarEstoque(ctx, q, os.ID, os.PecasDoReparo(reparoUUID), 0)
		if err != nil {
			return "", err
		}
		return "Reparo adicional autorizado pelo cliente: " + descricao, nil
	})
	if err != nil {
		pedirReposicaoFaltante(ctx, s.sender, s.config.AdminNotificationTo, osUUID, err)
		return nil, err
	}

	notificarEstoqueBaixo(ctx, s.sender, s.config.AdminNotificationTo, pecasEstoqueBaixo)
	return response, nil
}

func (s *ordemServicoService) RecusarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	return s.executarComObservacao(ctx, osUUID, 0, func(q db.Querier, os *ordemservico.OrdemServico) (string, error) {
		if err := validarTitular(ctx, q, os.ClienteID, documento); err != nil {
			return "", err
		}
		if err := os.RecusarReparo(reparoUUID, s.now()); err != nil {
			return "", err
		}
		reparo, err := os.Reparo(reparoUUID)
		if err != nil {
			return "", err
		}
		return "Reparo adicional recusado pelo cliente: " + reparo.Descricao, nil
	})
}

// ---------------------------------------------------------------------------
// Esqueleto comum dos comandos
// ---------------------------------------------------------------------------

// executar roda um comando sobre o agregado travado e persiste o resultado.
// Com observação vazia nada entra no histórico (alterações de itens).
func (s *ordemServicoService) executar(ctx context.Context, osUUID uuid.UUID, usuarioID int64, observacao string, comando func(db.Querier, *ordemservico.OrdemServico) error) (*dto.OrdemServicoResponse, error) {
	return s.executarComObservacao(ctx, osUUID, usuarioID, func(q db.Querier, os *ordemservico.OrdemServico) (string, error) {
		return observacao, comando(q, os)
	})
}

// executarComObservacao é executar com a observação do histórico decidida
// pelo próprio comando (ex.: descrição do reparo decidido).
func (s *ordemServicoService) executarComObservacao(ctx context.Context, osUUID uuid.UUID, usuarioID int64, comando func(db.Querier, *ordemservico.OrdemServico) (string, error)) (*dto.OrdemServicoResponse, error) {
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		os, err := carregarOS(ctx, q, osUUID, true)
		if err != nil {
			return err
		}
		anterior := os.Status
		observacao, err := comando(q, os)
		if err != nil {
			return err
		}
		return salvarOS(ctx, q, os, anterior, usuarioID, observacao)
	})
	if err != nil {
		return nil, err
	}
	return s.BuscarOrdemServico(ctx, osUUID)
}
