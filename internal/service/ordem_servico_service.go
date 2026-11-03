package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/veiculo"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

// OrdemServicoService orquestra os casos de uso da OS: cada comando abre uma
// transação, trava a OS, aplica as regras do agregado (internal/domain) e
// persiste o novo estado junto com o histórico de status.
type OrdemServicoService interface {
	AbrirOrdemServico(ctx context.Context, req dto.AbrirOrdemServicoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error)
	BuscarOrdemServico(ctx context.Context, osUUID uuid.UUID) (*dto.OrdemServicoResponse, error)
	ListarOrdensServico(ctx context.Context, query dto.ListOrdensServicoQuery) ([]dto.OrdemServicoResumoResponse, error)
	AdicionarServico(ctx context.Context, osUUID uuid.UUID, req dto.ItemServicoRequest) (*dto.OrdemServicoResponse, error)
	RemoverServico(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error)
	AdicionarPeca(ctx context.Context, osUUID uuid.UUID, req dto.ItemPecaRequest) (*dto.OrdemServicoResponse, error)
	RemoverPeca(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error)
	IniciarDiagnostico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	EnviarOrcamento(ctx context.Context, osUUID uuid.UUID, req dto.EnviarOrcamentoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error)
	AprovarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	RejeitarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	IniciarExecucao(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	FinalizarOrdemServico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	EntregarVeiculo(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	RegistrarReparoAdicional(ctx context.Context, osUUID uuid.UUID, req dto.RegistrarReparoAdicionalRequest) (*dto.OrdemServicoResponse, error)
	AutorizarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	RecusarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	ConsultarAndamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	TempoMedioExecucao(ctx context.Context, query dto.TempoMedioQuery) (*dto.TempoMedioExecucaoResponse, error)
}

type ordemServicoService struct {
	store  db.Store
	sender notification.Sender
	config util.Config
	now    func() time.Time
}

func NewOrdemServicoService(store db.Store, sender notification.Sender, config util.Config) OrdemServicoService {
	return &ordemServicoService{store: store, sender: sender, config: config, now: time.Now}
}

// ---------------------------------------------------------------------------
// Criação da OS
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

		os, err := q.CreateOrdemServico(ctx, db.CreateOrdemServicoParams{
			ClienteID:         dono.ID,
			VeiculoID:         v.ID,
			DescricaoProblema: req.DescricaoProblema,
		})
		if err != nil {
			return err
		}
		osUUID = os.Uuid

		if err := registrarHistorico(ctx, q, os.ID, "", os.Status, usuarioID, "OS aberta"); err != nil {
			return err
		}

		for _, item := range req.Servicos {
			if err := s.incluirServico(ctx, q, os, item, sql.NullInt64{}); err != nil {
				return err
			}
		}
		for _, item := range req.Pecas {
			if err := s.incluirPeca(ctx, q, os, item, sql.NullInt64{}); err != nil {
				return err
			}
		}

		_, err = recalcularOrcamento(ctx, q, os)
		return err
	})
	if err != nil {
		return nil, err
	}

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) AdicionarServico(ctx context.Context, osUUID uuid.UUID, req dto.ItemServicoRequest) (*dto.OrdemServicoResponse, error) {
	return s.alterarItens(ctx, osUUID, func(q db.Querier, os db.OrdemServico) error {
		return s.incluirServico(ctx, q, os, req, sql.NullInt64{})
	})
}

func (s *ordemServicoService) RemoverServico(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	return s.alterarItens(ctx, osUUID, func(q db.Querier, os db.OrdemServico) error {
		_, err := q.DeleteItemServico(ctx, db.DeleteItemServicoParams{Uuid: itemUUID, OrdemServicoID: os.ID})
		return notFound(err, ErrItemNaoEncontrado)
	})
}

func (s *ordemServicoService) AdicionarPeca(ctx context.Context, osUUID uuid.UUID, req dto.ItemPecaRequest) (*dto.OrdemServicoResponse, error) {
	return s.alterarItens(ctx, osUUID, func(q db.Querier, os db.OrdemServico) error {
		return s.incluirPeca(ctx, q, os, req, sql.NullInt64{})
	})
}

func (s *ordemServicoService) RemoverPeca(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	return s.alterarItens(ctx, osUUID, func(q db.Querier, os db.OrdemServico) error {
		_, err := q.DeleteItemPeca(ctx, db.DeleteItemPecaParams{Uuid: itemUUID, OrdemServicoID: os.ID})
		return notFound(err, ErrItemNaoEncontrado)
	})
}

// alterarItens aplica a regra "itens só mudam antes do envio do orçamento" e
// recalcula o orçamento na mesma transação da alteração.
func (s *ordemServicoService) alterarItens(ctx context.Context, osUUID uuid.UUID, alterar func(db.Querier, db.OrdemServico) error) (*dto.OrdemServicoResponse, error) {
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, err := q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}

		os := paraDominio(row)
		if err := os.PodeAlterarItens(); err != nil {
			return err
		}

		if err := alterar(q, row); err != nil {
			return err
		}

		_, err = recalcularOrcamento(ctx, q, row)
		return err
	})
	if err != nil {
		return nil, err
	}

	return s.BuscarOrdemServico(ctx, osUUID)
}

// ---------------------------------------------------------------------------
// Acompanhamento da OS (transições de status)
// ---------------------------------------------------------------------------

func (s *ordemServicoService) IniciarDiagnostico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, err := q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}

		os := paraDominio(row)
		if err := os.IniciarDiagnostico(s.now()); err != nil {
			return err
		}
		return salvar(ctx, q, row, os, usuarioID, "Diagnóstico iniciado")
	})
	if err != nil {
		return nil, err
	}

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) EnviarOrcamento(ctx context.Context, osUUID uuid.UUID, req dto.EnviarOrcamentoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var row db.OrdemServico
	var os ordemservico.OrdemServico
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		row, err = q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}

		servicos, pecas, err := carregarItens(ctx, q, row.ID)
		if err != nil {
			return err
		}

		os = paraDominio(row)
		if err := os.EnviarOrcamento(itensServico(servicos), itensPeca(pecas), req.Diagnostico, s.now()); err != nil {
			return err
		}
		return salvar(ctx, q, row, os, usuarioID, "Orçamento enviado ao cliente")
	})
	if err != nil {
		return nil, err
	}

	s.notificarCliente(ctx, row.ClienteID, "Orçamento disponível para aprovação", fmt.Sprintf(
		"O orçamento da sua ordem de serviço é de %s. Aprove ou rejeite pelo link de acompanhamento: /api/v1/ordens-servico/%s/andamento",
		formatarDinheiro(os.ValorTotalCentavos), osUUID,
	))

	return s.BuscarOrdemServico(ctx, osUUID)
}

// AprovarOrcamento registra a decisão do cliente e, em seguida, o sistema
// inicia a execução (com baixa de estoque). Se faltar estoque, a aprovação
// continua registrada, o administrador recebe o pedido de reposição e a OS
// aguarda a reposição para iniciar a execução.
func (s *ordemServicoService) AprovarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, err := q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}
		if err := validarTitular(ctx, q, row, documento); err != nil {
			return err
		}

		os := paraDominio(row)
		if err := os.AprovarOrcamento(s.now()); err != nil {
			return err
		}
		return salvar(ctx, q, row, os, 0, "Orçamento aprovado pelo cliente")
	})
	if err != nil {
		return nil, err
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
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, err := q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}
		if err := validarTitular(ctx, q, row, documento); err != nil {
			return err
		}

		os := paraDominio(row)
		if err := os.RejeitarOrcamento(s.now()); err != nil {
			return err
		}
		return salvar(ctx, q, row, os, 0, "Orçamento rejeitado pelo cliente")
	})
	if err != nil {
		return nil, err
	}

	return s.BuscarOrdemServico(ctx, osUUID)
}

// IniciarExecucao muda a OS para Em execução e baixa o estoque das peças na
// mesma transação: se uma peça não tiver saldo, nada é gravado.
func (s *ordemServicoService) IniciarExecucao(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var pecasEstoqueBaixo []db.Peca
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, err := q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}

		os := paraDominio(row)
		if err := os.IniciarExecucao(s.now()); err != nil {
			return err
		}

		_, pecas, err := carregarItens(ctx, q, row.ID)
		if err != nil {
			return err
		}

		itensOrcamento := make([]db.ListItensPecaByOrdemServicoRow, 0, len(pecas))
		for _, item := range pecas {
			if !item.ReparoAdicionalUuid.Valid {
				itensOrcamento = append(itensOrcamento, item)
			}
		}

		pecasEstoqueBaixo, err = baixarEstoque(ctx, q, row.ID, itensOrcamento, usuarioID)
		if err != nil {
			return err
		}

		return salvar(ctx, q, row, os, usuarioID, "Execução iniciada e estoque baixado")
	})
	if err != nil {
		pedirReposicaoFaltante(ctx, s.sender, s.config.AdminNotificationTo, osUUID, err)
		return nil, err
	}

	notificarEstoqueBaixo(ctx, s.sender, s.config.AdminNotificationTo, pecasEstoqueBaixo)
	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) FinalizarOrdemServico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	var row db.OrdemServico
	var os ordemservico.OrdemServico
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		row, err = q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}

		reparos, err := q.ListReparosAdicionaisByOrdemServico(ctx, row.ID)
		if err != nil {
			return err
		}
		pendentes := 0
		for _, r := range reparos {
			if ordemservico.StatusReparo(r.Status) == ordemservico.ReparoPendente {
				pendentes++
			}
		}

		os = paraDominio(row)
		if err := os.Finalizar(pendentes, s.now()); err != nil {
			return err
		}

		observacao := "OS finalizada"
		if duracao, ok := os.TempoExecucao(); ok {
			observacao = fmt.Sprintf("OS finalizada. Tempo de execução: %s", formatarDuracao(int64(duracao.Seconds())))
		}
		return salvar(ctx, q, row, os, usuarioID, observacao)
	})
	if err != nil {
		return nil, err
	}

	s.notificarCliente(ctx, row.ClienteID, "Seu veículo está pronto",
		fmt.Sprintf("A ordem de serviço %s foi finalizada. O veículo já pode ser retirado.", osUUID))

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) EntregarVeiculo(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error) {
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, err := q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}

		os := paraDominio(row)
		if err := os.Entregar(s.now()); err != nil {
			return err
		}

		observacao := "Veículo entregue ao cliente"
		if os.StatusOrcamento == ordemservico.OrcamentoRejeitado {
			observacao = "Veículo entregue sem execução (orçamento rejeitado)"
		}
		return salvar(ctx, q, row, os, usuarioID, observacao)
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

	var row db.OrdemServico
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		row, err = q.GetOrdemServicoForUpdate(ctx, osUUID)
		if err != nil {
			return notFound(err, ErrOSNaoEncontrada)
		}

		os := paraDominio(row)
		if err := os.PodeRegistrarReparoAdicional(); err != nil {
			return err
		}

		reparo, err := q.CreateReparoAdicional(ctx, db.CreateReparoAdicionalParams{
			OrdemServicoID: row.ID,
			Descricao:      req.Descricao,
		})
		if err != nil {
			return err
		}

		reparoID := sql.NullInt64{Int64: reparo.ID, Valid: true}
		for _, item := range req.Servicos {
			if err := s.incluirServico(ctx, q, row, item, reparoID); err != nil {
				return err
			}
		}
		for _, item := range req.Pecas {
			if err := s.incluirPeca(ctx, q, row, item, reparoID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.notificarCliente(ctx, row.ClienteID, "Reparo adicional aguardando autorização", fmt.Sprintf(
		"Durante a execução encontramos outro problema: %s. Autorize ou recuse pelo link de acompanhamento: /api/v1/ordens-servico/%s/andamento",
		req.Descricao, osUUID,
	))

	return s.BuscarOrdemServico(ctx, osUUID)
}

// AutorizarReparoAdicional inclui os itens do reparo no orçamento e baixa o
// estoque das peças dele (a OS já está em execução).
func (s *ordemServicoService) AutorizarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	var pecasEstoqueBaixo []db.Peca
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, reparo, err := s.carregarReparoParaDecisao(ctx, q, osUUID, reparoUUID, documento)
		if err != nil {
			return err
		}

		reparo, err = q.UpdateReparoAdicionalStatus(ctx, db.UpdateReparoAdicionalStatusParams{
			ID:     reparo.ID,
			Status: string(ordemservico.ReparoAutorizado),
		})
		if err != nil {
			return err
		}

		_, pecas, err := carregarItens(ctx, q, row.ID)
		if err != nil {
			return err
		}
		itensReparo := make([]db.ListItensPecaByOrdemServicoRow, 0, len(pecas))
		for _, item := range pecas {
			if item.ReparoAdicionalUuid.Valid && item.ReparoAdicionalUuid.UUID == reparo.Uuid {
				itensReparo = append(itensReparo, item)
			}
		}

		pecasEstoqueBaixo, err = baixarEstoque(ctx, q, row.ID, itensReparo, 0)
		if err != nil {
			return err
		}

		if _, err := recalcularOrcamento(ctx, q, row); err != nil {
			return err
		}
		return registrarHistorico(ctx, q, row.ID, row.Status, row.Status, 0, "Reparo adicional autorizado pelo cliente: "+reparo.Descricao)
	})
	if err != nil {
		pedirReposicaoFaltante(ctx, s.sender, s.config.AdminNotificationTo, osUUID, err)
		return nil, err
	}

	notificarEstoqueBaixo(ctx, s.sender, s.config.AdminNotificationTo, pecasEstoqueBaixo)
	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) RecusarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		row, reparo, err := s.carregarReparoParaDecisao(ctx, q, osUUID, reparoUUID, documento)
		if err != nil {
			return err
		}

		_, err = q.UpdateReparoAdicionalStatus(ctx, db.UpdateReparoAdicionalStatusParams{
			ID:     reparo.ID,
			Status: string(ordemservico.ReparoRecusado),
		})
		if err != nil {
			return err
		}
		return registrarHistorico(ctx, q, row.ID, row.Status, row.Status, 0, "Reparo adicional recusado pelo cliente: "+reparo.Descricao)
	})
	if err != nil {
		return nil, err
	}

	return s.BuscarOrdemServico(ctx, osUUID)
}

func (s *ordemServicoService) carregarReparoParaDecisao(ctx context.Context, q db.Querier, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (db.OrdemServico, db.ReparoAdicional, error) {
	row, err := q.GetOrdemServicoForUpdate(ctx, osUUID)
	if err != nil {
		return row, db.ReparoAdicional{}, notFound(err, ErrOSNaoEncontrada)
	}
	if err := validarTitular(ctx, q, row, documento); err != nil {
		return row, db.ReparoAdicional{}, err
	}

	reparo, err := q.GetReparoAdicionalForUpdate(ctx, db.GetReparoAdicionalForUpdateParams{
		Uuid:           reparoUUID,
		OrdemServicoID: row.ID,
	})
	if err != nil {
		return row, reparo, notFound(err, ErrReparoNaoEncontrado)
	}

	if err := ordemservico.DecidirReparo(ordemservico.StatusReparo(reparo.Status), ordemservico.Status(row.Status)); err != nil {
		return row, reparo, err
	}
	return row, reparo, nil
}

// ---------------------------------------------------------------------------
// Consultas
// ---------------------------------------------------------------------------

func (s *ordemServicoService) BuscarOrdemServico(ctx context.Context, osUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	os, err := s.store.GetOrdemServico(ctx, osUUID)
	if err != nil {
		return nil, notFound(err, ErrOSNaoEncontrada)
	}
	return s.montarResposta(ctx, os)
}

// ConsultarAndamento é a consulta pública do cliente. O CPF/CNPJ funciona
// como segundo fator junto com o UUID da OS; se não conferir, a resposta é a
// mesma de OS inexistente para não revelar quais OS existem.
func (s *ordemServicoService) ConsultarAndamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	os, err := s.store.GetOrdemServico(ctx, osUUID)
	if err != nil {
		return nil, notFound(err, ErrOSNaoEncontrada)
	}
	if err := validarTitular(ctx, s.store, os, documento); err != nil {
		return nil, err
	}
	return s.montarResposta(ctx, os)
}

func (s *ordemServicoService) ListarOrdensServico(ctx context.Context, query dto.ListOrdensServicoQuery) ([]dto.OrdemServicoResumoResponse, error) {
	var clienteID sql.NullInt64
	if query.ClienteUuid != "" {
		clienteUUID, err := uuid.Parse(query.ClienteUuid)
		if err != nil {
			return nil, ErrClienteNaoEncontrado
		}
		dono, err := s.store.GetCliente(ctx, clienteUUID)
		if err != nil {
			return nil, notFound(err, ErrClienteNaoEncontrado)
		}
		clienteID = sql.NullInt64{Int64: dono.ID, Valid: true}
	}

	rows, err := s.store.ListOrdensServico(ctx, db.ListOrdensServicoParams{
		Status:       query.Status,
		ClienteID:    clienteID,
		Limite:       query.Limite(),
		Deslocamento: query.Deslocamento(),
	})
	if err != nil {
		return nil, err
	}

	response := make([]dto.OrdemServicoResumoResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, dto.NewOrdemServicoResumoResponse(row))
	}
	return response, nil
}

// TempoMedioExecucao calcula a média entre o início da execução e a
// finalização das OS finalizadas, no geral e por serviço.
func (s *ordemServicoService) TempoMedioExecucao(ctx context.Context, query dto.TempoMedioQuery) (*dto.TempoMedioExecucaoResponse, error) {
	dataInicio, err := parseData(query.DataInicio)
	if err != nil {
		return nil, err
	}
	dataFim, err := parseData(query.DataFim)
	if err != nil {
		return nil, err
	}
	if dataFim.Valid {
		// data_fim é inclusiva: soma um dia para usar "<" na consulta.
		dataFim.Time = dataFim.Time.AddDate(0, 0, 1)
	}
	if dataInicio.Valid && dataFim.Valid && !dataInicio.Time.Before(dataFim.Time) {
		return nil, ErrIntervaloDatasInvalido
	}

	geral, err := s.store.GetTempoMedioExecucao(ctx, db.GetTempoMedioExecucaoParams{DataInicio: dataInicio, DataFim: dataFim})
	if err != nil {
		return nil, err
	}

	porServico, err := s.store.ListTempoMedioExecucaoPorServico(ctx, db.ListTempoMedioExecucaoPorServicoParams{DataInicio: dataInicio, DataFim: dataFim})
	if err != nil {
		return nil, err
	}

	servicos := make([]dto.TempoMedioServicoResponse, 0, len(porServico))
	for _, row := range porServico {
		media := int64(row.MediaSegundos)
		servicos = append(servicos, dto.TempoMedioServicoResponse{
			ServicoUuid:          row.ServicoUuid,
			ServicoNome:          row.ServicoNome,
			TempoEstimadoMinutos: row.TempoEstimadoMinutos,
			TotalOrdens:          row.TotalOrdens,
			MediaSegundos:        media,
			MediaFormatada:       formatarDuracao(media),
		})
	}

	media := int64(geral.MediaSegundos)
	return &dto.TempoMedioExecucaoResponse{
		TotalOrdens:    geral.TotalOrdens,
		MediaSegundos:  media,
		MediaFormatada: formatarDuracao(media),
		MinimoSegundos: int64(geral.MinimoSegundos),
		MaximoSegundos: int64(geral.MaximoSegundos),
		PorServico:     servicos,
	}, nil
}

func (s *ordemServicoService) montarResposta(ctx context.Context, os db.OrdemServico) (*dto.OrdemServicoResponse, error) {
	dono, err := s.store.GetClienteByID(ctx, os.ClienteID)
	if err != nil {
		return nil, err
	}
	v, err := s.store.GetVeiculoByID(ctx, os.VeiculoID)
	if err != nil {
		return nil, err
	}
	servicos, pecas, err := carregarItens(ctx, s.store, os.ID)
	if err != nil {
		return nil, err
	}
	reparos, err := s.store.ListReparosAdicionaisByOrdemServico(ctx, os.ID)
	if err != nil {
		return nil, err
	}
	historico, err := s.store.ListHistoricoStatusOSByOrdemServico(ctx, os.ID)
	if err != nil {
		return nil, err
	}

	response := dto.NewOrdemServicoResponse(dto.OrdemServicoDetalhe{
		OrdemServico: os,
		Cliente:      dono,
		Veiculo:      v,
		Servicos:     servicos,
		Pecas:        pecas,
		Reparos:      reparos,
		Historico:    historico,
	})
	return &response, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// incluirServico adiciona um serviço ativo do catálogo com o preço atual.
func (s *ordemServicoService) incluirServico(ctx context.Context, q db.Querier, os db.OrdemServico, req dto.ItemServicoRequest, reparoID sql.NullInt64) error {
	quantidade := req.Quantidade
	if quantidade == 0 {
		quantidade = 1
	}
	if err := ordemservico.ValidarQuantidade(quantidade); err != nil {
		return err
	}

	servico, err := q.GetServico(ctx, req.ServicoUuid)
	if err != nil {
		return notFound(err, ErrServicoNaoEncontrado)
	}
	if !servico.Ativo {
		return ErrServicoInativo
	}

	_, err = q.CreateItemServico(ctx, db.CreateItemServicoParams{
		OrdemServicoID:        os.ID,
		ServicoID:             servico.ID,
		ReparoAdicionalID:     reparoID,
		Quantidade:            quantidade,
		ValorUnitarioCentavos: servico.PrecoCentavos,
	})
	return err
}

// incluirPeca adiciona a peça se houver saldo para ela, contando o que a OS
// já pediu da mesma peça e ainda não foi baixado. Não há reserva: o saldo é
// conferido de novo na baixa, ao iniciar a execução.
func (s *ordemServicoService) incluirPeca(ctx context.Context, q db.Querier, os db.OrdemServico, req dto.ItemPecaRequest, reparoID sql.NullInt64) error {
	peca, err := q.GetPeca(ctx, req.PecaUuid)
	if err != nil {
		return notFound(err, ErrPecaNaoEncontrada)
	}

	_, itens, err := carregarItens(ctx, q, os.ID)
	if err != nil {
		return err
	}

	var jaNaOS int32
	for _, item := range itens {
		if item.PecaID != peca.ID {
			continue
		}
		aindaNaoBaixado := !item.ReparoAdicionalUuid.Valid && os.Status != string(ordemservico.StatusEmExecucao)
		reparoPendente := item.ReparoAdicionalStatus.Valid && ordemservico.StatusReparo(item.ReparoAdicionalStatus.String) == ordemservico.ReparoPendente
		if aindaNaoBaixado || reparoPendente {
			jaNaOS += item.Quantidade
		}
	}

	if err := estoque.ValidarDisponibilidade(peca.QuantidadeEstoque, jaNaOS, req.Quantidade, peca.Ativo); err != nil {
		if errors.Is(err, estoque.ErrEstoqueInsuficiente) {
			return erroEstoqueInsuficiente(peca)
		}
		return err
	}

	_, err = q.CreateItemPeca(ctx, db.CreateItemPecaParams{
		OrdemServicoID:        os.ID,
		PecaID:                peca.ID,
		ReparoAdicionalID:     reparoID,
		Quantidade:            req.Quantidade,
		ValorUnitarioCentavos: peca.PrecoCentavos,
	})
	return err
}

// baixarEstoque dá saída das peças e registra o movimento. Devolve as peças
// que ficaram no estoque mínimo para o pedido de reposição.
func baixarEstoque(ctx context.Context, q db.Querier, osID int64, itens []db.ListItensPecaByOrdemServicoRow, usuarioID int64) ([]db.Peca, error) {
	estoqueBaixo := make([]db.Peca, 0)
	for _, item := range itens {
		peca, err := q.BaixarEstoquePeca(ctx, db.BaixarEstoquePecaParams{ID: item.PecaID, Quantidade: item.Quantidade})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, &faltaPeca{
					err:  erroEstoqueInsuficiente(db.Peca{Nome: item.PecaNome, Codigo: item.PecaCodigo}),
					item: item,
				}
			}
			return nil, err
		}

		_, err = q.CreateMovimentoEstoque(ctx, db.CreateMovimentoEstoqueParams{
			PecaID:         peca.ID,
			Tipo:           string(estoque.MovimentoSaida),
			Quantidade:     -item.Quantidade,
			SaldoAnterior:  peca.QuantidadeEstoque + item.Quantidade,
			SaldoPosterior: peca.QuantidadeEstoque,
			OrdemServicoID: sql.NullInt64{Int64: osID, Valid: true},
			UsuarioID:      util.ToNullInt64(usuarioID),
			Observacao:     "Baixa por ordem de serviço",
		})
		if err != nil {
			return nil, err
		}

		if estoque.EstoqueBaixo(peca.QuantidadeEstoque, peca.EstoqueMinimo) {
			estoqueBaixo = append(estoqueBaixo, peca)
		}
	}
	return estoqueBaixo, nil
}

func erroEstoqueInsuficiente(peca db.Peca) error {
	return fmt.Errorf("%w: %s (%s)", estoque.ErrEstoqueInsuficiente, peca.Nome, peca.Codigo)
}

// faltaPeca é o erro de saldo insuficiente na baixa. Guarda o item da OS para
// o pedido de reposição; errors.Is continua achando ErrEstoqueInsuficiente.
type faltaPeca struct {
	err  error
	item db.ListItensPecaByOrdemServicoRow
}

func (f *faltaPeca) Error() string { return f.err.Error() }
func (f *faltaPeca) Unwrap() error { return f.err }

// pedirReposicaoFaltante avisa o administrador quando a baixa falhou por falta
// de saldo. Sem a reposição, a OS aprovada não entra em execução.
func pedirReposicaoFaltante(ctx context.Context, sender notification.Sender, to string, osUUID uuid.UUID, err error) {
	var falta *faltaPeca
	if !errors.As(err, &falta) {
		return
	}
	if err := sender.Send(ctx, notification.Message{
		To:      to,
		Subject: "Pedido de reposição de estoque",
		Body: fmt.Sprintf("%s (%s) não tem saldo para a OS %s, que precisa de %d. A OS aguarda a reposição para entrar em execução.",
			falta.item.PecaNome, falta.item.PecaCodigo, osUUID, falta.item.Quantidade),
	}); err != nil {
		log.Printf("não foi possível enviar o pedido de reposição: %v", err)
	}
}

func carregarItens(ctx context.Context, q db.Querier, osID int64) ([]db.ListItensServicoByOrdemServicoRow, []db.ListItensPecaByOrdemServicoRow, error) {
	servicos, err := q.ListItensServicoByOrdemServico(ctx, osID)
	if err != nil {
		return nil, nil, err
	}
	pecas, err := q.ListItensPecaByOrdemServico(ctx, osID)
	if err != nil {
		return nil, nil, err
	}
	return servicos, pecas, nil
}

// recalcularOrcamento grava o novo valor total a partir dos itens da OS.
func recalcularOrcamento(ctx context.Context, q db.Querier, row db.OrdemServico) (db.OrdemServico, error) {
	servicos, pecas, err := carregarItens(ctx, q, row.ID)
	if err != nil {
		return row, err
	}

	os := paraDominio(row)
	orcamento := os.AtualizarOrcamento(itensServico(servicos), itensPeca(pecas))

	return q.UpdateOrdemServicoValorTotal(ctx, db.UpdateOrdemServicoValorTotalParams{
		ID:                 row.ID,
		ValorTotalCentavos: orcamento.TotalCentavos,
	})
}

func itensServico(rows []db.ListItensServicoByOrdemServicoRow) []ordemservico.Item {
	itens := make([]ordemservico.Item, 0, len(rows))
	for _, row := range rows {
		itens = append(itens, dto.ItemServicoParaDominio(row))
	}
	return itens
}

func itensPeca(rows []db.ListItensPecaByOrdemServicoRow) []ordemservico.Item {
	itens := make([]ordemservico.Item, 0, len(rows))
	for _, row := range rows {
		itens = append(itens, dto.ItemPecaParaDominio(row))
	}
	return itens
}

// validarTitular confere se o CPF/CNPJ informado é o do dono da OS.
func validarTitular(ctx context.Context, q db.Querier, os db.OrdemServico, documento string) error {
	dono, err := q.GetClienteByID(ctx, os.ClienteID)
	if err != nil {
		return err
	}
	if cliente.Normalizar(documento) != dono.Documento {
		return ErrOSNaoEncontrada
	}
	return nil
}

func paraDominio(row db.OrdemServico) ordemservico.OrdemServico {
	return ordemservico.OrdemServico{
		Status:                ordemservico.Status(row.Status),
		StatusOrcamento:       ordemservico.StatusOrcamento(row.OrcamentoStatus),
		ValorTotalCentavos:    row.ValorTotalCentavos,
		Diagnostico:           row.Diagnostico,
		OrcamentoEnviadoEm:    util.FromNullTime(row.OrcamentoEnviadoEm),
		OrcamentoDecididoEm:   util.FromNullTime(row.OrcamentoDecididoEm),
		DiagnosticoIniciadoEm: util.FromNullTime(row.DiagnosticoIniciadoEm),
		ExecucaoIniciadaEm:    util.FromNullTime(row.ExecucaoIniciadaEm),
		FinalizadaEm:          util.FromNullTime(row.FinalizadaEm),
		EntregueEm:            util.FromNullTime(row.EntregueEm),
	}
}

// salvar persiste o estado do agregado e registra a entrada no histórico.
func salvar(ctx context.Context, q db.Querier, row db.OrdemServico, os ordemservico.OrdemServico, usuarioID int64, observacao string) error {
	_, err := q.UpdateOrdemServicoEstado(ctx, db.UpdateOrdemServicoEstadoParams{
		ID:                    row.ID,
		Status:                string(os.Status),
		Diagnostico:           os.Diagnostico,
		OrcamentoStatus:       string(os.StatusOrcamento),
		ValorTotalCentavos:    os.ValorTotalCentavos,
		OrcamentoEnviadoEm:    util.ToNullTime(os.OrcamentoEnviadoEm),
		OrcamentoDecididoEm:   util.ToNullTime(os.OrcamentoDecididoEm),
		DiagnosticoIniciadoEm: util.ToNullTime(os.DiagnosticoIniciadoEm),
		ExecucaoIniciadaEm:    util.ToNullTime(os.ExecucaoIniciadaEm),
		FinalizadaEm:          util.ToNullTime(os.FinalizadaEm),
		EntregueEm:            util.ToNullTime(os.EntregueEm),
	})
	if err != nil {
		return err
	}

	return registrarHistorico(ctx, q, row.ID, row.Status, string(os.Status), usuarioID, observacao)
}

func registrarHistorico(ctx context.Context, q db.Querier, osID int64, anterior string, novo string, usuarioID int64, observacao string) error {
	_, err := q.CreateHistoricoStatusOS(ctx, db.CreateHistoricoStatusOSParams{
		OrdemServicoID: osID,
		StatusAnterior: anterior,
		StatusNovo:     novo,
		UsuarioID:      util.ToNullInt64(usuarioID),
		Observacao:     observacao,
	})
	return err
}

// notificarCliente envia o aviso por e-mail depois do commit; falha de envio
// não desfaz a operação. O e-mail é obrigatório no cadastro; um cliente antigo
// sem e-mail não recebe aviso e o fato fica no log.
func (s *ordemServicoService) notificarCliente(ctx context.Context, clienteID int64, assunto string, mensagem string) {
	dono, err := s.store.GetClienteByID(ctx, clienteID)
	if err != nil {
		log.Printf("não foi possível carregar o cliente %d para o aviso: %v", clienteID, err)
		return
	}

	if dono.Email == "" {
		log.Printf("cliente %d sem e-mail; aviso não enviado", clienteID)
		return
	}

	if err := s.sender.Send(ctx, notification.Message{To: dono.Email, Subject: assunto, Body: mensagem}); err != nil {
		log.Printf("não foi possível avisar o cliente %d: %v", clienteID, err)
	}
}

func parseData(valor string) (sql.NullTime, error) {
	if valor == "" {
		return sql.NullTime{}, nil
	}
	data, err := time.ParseInLocation("2006-01-02", valor, time.Local)
	if err != nil {
		return sql.NullTime{}, ErrDataInvalida
	}
	return sql.NullTime{Time: data, Valid: true}, nil
}

func formatarDuracao(segundos int64) string {
	horas := segundos / 3600
	minutos := (segundos % 3600) / 60
	if horas > 0 {
		return fmt.Sprintf("%dh %02dmin", horas, minutos)
	}
	if minutos > 0 {
		return fmt.Sprintf("%dmin", minutos)
	}
	return fmt.Sprintf("%ds", segundos)
}

func formatarDinheiro(centavos int64) string {
	return fmt.Sprintf("R$ %d,%02d", centavos/100, centavos%100)
}
