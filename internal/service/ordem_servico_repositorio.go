package service

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

// Este arquivo faz a ponte entre o agregado OrdemServico e as tabelas:
// carrega a raiz com itens e reparos, grava o que mudou e dá baixa no
// estoque. Os comandos (ordem_servico_comandos.go) não tocam nas tabelas
// de itens e reparos diretamente.

// carregarOS monta o agregado a partir das linhas da OS, dos itens e dos
// reparos. Com paraAtualizar, a linha da OS fica travada na transação.
func carregarOS(ctx context.Context, q db.Querier, osUUID uuid.UUID, paraAtualizar bool) (*ordemservico.OrdemServico, error) {
	var row db.OrdemServico
	var err error
	if paraAtualizar {
		row, err = q.GetOrdemServicoForUpdate(ctx, osUUID)
	} else {
		row, err = q.GetOrdemServico(ctx, osUUID)
	}
	if err != nil {
		return nil, notFound(err, ErrOSNaoEncontrada)
	}

	servicos, err := q.ListItensServicoByOrdemServico(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	pecas, err := q.ListItensPecaByOrdemServico(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	reparos, err := q.ListReparosAdicionaisByOrdemServico(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return paraAgregado(row, servicos, pecas, reparos), nil
}

func paraAgregado(row db.OrdemServico, servicos []db.ListItensServicoByOrdemServicoRow, pecas []db.ListItensPecaByOrdemServicoRow, reparos []db.ReparoAdicional) *ordemservico.OrdemServico {
	os := &ordemservico.OrdemServico{
		ID:                    row.ID,
		UUID:                  row.Uuid,
		ClienteID:             row.ClienteID,
		VeiculoID:             row.VeiculoID,
		DescricaoProblema:     row.DescricaoProblema,
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
		CriadaEm:              row.CreatedAt,
		AtualizadaEm:          row.UpdatedAt,
		Itens:                 make([]ordemservico.Item, 0, len(servicos)+len(pecas)),
		Reparos:               make([]ordemservico.ReparoAdicional, 0, len(reparos)),
	}
	for _, item := range servicos {
		os.Itens = append(os.Itens, ordemservico.Item{
			ID:                    item.ID,
			UUID:                  item.Uuid,
			Tipo:                  ordemservico.ItemServico,
			CatalogoUUID:          item.ServicoUuid,
			Nome:                  item.ServicoNome,
			Quantidade:            item.Quantidade,
			ValorUnitarioCentavos: item.ValorUnitarioCentavos,
			ReparoUUID:            item.ReparoAdicionalUuid,
			StatusReparo:          ordemservico.StatusReparo(item.ReparoAdicionalStatus.String),
			CriadoEm:              item.CreatedAt,
		})
	}
	for _, item := range pecas {
		os.Itens = append(os.Itens, ordemservico.Item{
			ID:                    item.ID,
			UUID:                  item.Uuid,
			Tipo:                  ordemservico.ItemPeca,
			CatalogoID:            item.PecaID,
			CatalogoUUID:          item.PecaUuid,
			Nome:                  item.PecaNome,
			Codigo:                item.PecaCodigo,
			TipoPeca:              item.PecaTipo,
			Quantidade:            item.Quantidade,
			ValorUnitarioCentavos: item.ValorUnitarioCentavos,
			ReparoUUID:            item.ReparoAdicionalUuid,
			StatusReparo:          ordemservico.StatusReparo(item.ReparoAdicionalStatus.String),
			CriadoEm:              item.CreatedAt,
		})
	}
	for _, r := range reparos {
		os.Reparos = append(os.Reparos, ordemservico.ReparoAdicional{
			ID:         r.ID,
			UUID:       r.Uuid,
			Descricao:  r.Descricao,
			Status:     ordemservico.StatusReparo(r.Status),
			DecididoEm: util.FromNullTime(r.DecididoEm),
			CriadoEm:   r.CreatedAt,
		})
	}
	return os
}

// criarOS grava a linha da OS e devolve a raiz já com identidade.
func criarOS(ctx context.Context, q db.Querier, os *ordemservico.OrdemServico) error {
	row, err := q.CreateOrdemServico(ctx, db.CreateOrdemServicoParams{
		ClienteID:         os.ClienteID,
		VeiculoID:         os.VeiculoID,
		DescricaoProblema: os.DescricaoProblema,
	})
	if err != nil {
		return err
	}
	os.ID, os.UUID = row.ID, row.Uuid
	os.CriadaEm, os.AtualizadaEm = row.CreatedAt, row.UpdatedAt
	return nil
}

// salvarOS persiste o agregado: reparos novos e decididos, itens novos e
// removidos, o estado da raiz e, quando há observação, a linha do histórico
// (statusAnterior é o status com que a OS foi carregada).
func salvarOS(ctx context.Context, q db.Querier, os *ordemservico.OrdemServico, statusAnterior ordemservico.Status, usuarioID int64, observacao string) error {
	reparoIDs := make(map[uuid.UUID]int64, len(os.Reparos))
	for i := range os.Reparos {
		reparo := &os.Reparos[i]
		if reparo.ID == 0 {
			criado, err := q.CreateReparoAdicional(ctx, db.CreateReparoAdicionalParams{OrdemServicoID: os.ID, Descricao: reparo.Descricao})
			if err != nil {
				return err
			}
			reparo.ID, reparo.CriadoEm = criado.ID, criado.CreatedAt
			if criado.Uuid != uuid.Nil {
				// O banco gera o UUID: os itens do reparo passam a apontar para ele.
				for j := range os.Itens {
					if os.Itens[j].DoReparo(reparo.UUID) {
						os.Itens[j].ReparoUUID.UUID = criado.Uuid
					}
				}
				reparo.UUID = criado.Uuid
			}
		}
		reparoIDs[reparo.UUID] = reparo.ID
	}
	for _, reparoUUID := range os.Alteracoes().ReparosDecididos {
		reparo, err := os.Reparo(reparoUUID)
		if err != nil {
			return err
		}
		if _, err := q.UpdateReparoAdicionalStatus(ctx, db.UpdateReparoAdicionalStatusParams{ID: reparo.ID, Status: string(reparo.Status)}); err != nil {
			return err
		}
	}

	for i := range os.Itens {
		item := &os.Itens[i]
		if item.ID != 0 {
			continue
		}
		reparoID := sql.NullInt64{}
		if item.ReparoUUID.Valid {
			reparoID = sql.NullInt64{Int64: reparoIDs[item.ReparoUUID.UUID], Valid: true}
		}
		if err := criarItem(ctx, q, os.ID, item, reparoID); err != nil {
			return err
		}
	}
	for _, item := range os.Alteracoes().ItensRemovidos {
		if err := removerItem(ctx, q, os.ID, item); err != nil {
			return err
		}
	}

	if _, err := q.UpdateOrdemServicoEstado(ctx, db.UpdateOrdemServicoEstadoParams{
		ID:                    os.ID,
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
	}); err != nil {
		return err
	}

	if observacao == "" {
		return nil
	}
	return registrarHistorico(ctx, q, os.ID, string(statusAnterior), string(os.Status), usuarioID, observacao)
}

func criarItem(ctx context.Context, q db.Querier, osID int64, item *ordemservico.Item, reparoID sql.NullInt64) error {
	switch item.Tipo {
	case ordemservico.ItemServico:
		criado, err := q.CreateItemServico(ctx, db.CreateItemServicoParams{
			OrdemServicoID:        osID,
			ServicoID:             item.CatalogoID,
			ReparoAdicionalID:     reparoID,
			Quantidade:            item.Quantidade,
			ValorUnitarioCentavos: item.ValorUnitarioCentavos,
		})
		if err != nil {
			return err
		}
		item.ID, item.UUID = criado.ID, criado.Uuid
	case ordemservico.ItemPeca:
		criado, err := q.CreateItemPeca(ctx, db.CreateItemPecaParams{
			OrdemServicoID:        osID,
			PecaID:                item.CatalogoID,
			ReparoAdicionalID:     reparoID,
			Quantidade:            item.Quantidade,
			ValorUnitarioCentavos: item.ValorUnitarioCentavos,
		})
		if err != nil {
			return err
		}
		item.ID, item.UUID = criado.ID, criado.Uuid
	}
	return nil
}

// removerItem apaga a linha do item que a raiz tirou da OS.
func removerItem(ctx context.Context, q db.Querier, osID int64, item ordemservico.Item) error {
	var err error
	switch item.Tipo {
	case ordemservico.ItemServico:
		_, err = q.DeleteItemServico(ctx, db.DeleteItemServicoParams{Uuid: item.UUID, OrdemServicoID: osID})
	case ordemservico.ItemPeca:
		_, err = q.DeleteItemPeca(ctx, db.DeleteItemPecaParams{Uuid: item.UUID, OrdemServicoID: osID})
	}
	return notFound(err, ErrItemNaoEncontrado)
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

// baixarEstoque dá saída das peças e registra o movimento. A condição
// "quantidade_estoque >= qtd" no UPDATE garante que o saldo nunca fica
// negativo mesmo com OS concorrentes. Devolve as peças que ficaram no
// estoque mínimo para o pedido de reposição.
func baixarEstoque(ctx context.Context, q db.Querier, osID int64, itens []ordemservico.Item, usuarioID int64) ([]db.Peca, error) {
	estoqueBaixo := make([]db.Peca, 0)
	for _, item := range itens {
		peca, err := q.BaixarEstoquePeca(ctx, db.BaixarEstoquePecaParams{ID: item.CatalogoID, Quantidade: item.Quantidade})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, &faltaPeca{err: estoque.ErrEstoqueInsuficientePara(item.Nome, item.Codigo), item: item}
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

// catálogo: leituras usadas pelos comandos para montar os pedidos do agregado.

func servicoDoCatalogo(ctx context.Context, q db.Querier, servicoUUID uuid.UUID) (ordemservico.CatalogoServico, error) {
	servico, err := q.GetServico(ctx, servicoUUID)
	if err != nil {
		return ordemservico.CatalogoServico{}, notFound(err, ErrServicoNaoEncontrado)
	}
	return ordemservico.CatalogoServico{
		ID:            servico.ID,
		UUID:          servico.Uuid,
		Nome:          servico.Nome,
		PrecoCentavos: servico.PrecoCentavos,
		Ativo:         servico.Ativo,
	}, nil
}

func pecaDoCatalogo(ctx context.Context, q db.Querier, pecaUUID uuid.UUID) (ordemservico.CatalogoPeca, int32, error) {
	peca, err := q.GetPeca(ctx, pecaUUID)
	if err != nil {
		return ordemservico.CatalogoPeca{}, 0, notFound(err, ErrPecaNaoEncontrada)
	}
	return ordemservico.CatalogoPeca{
		ID:            peca.ID,
		UUID:          peca.Uuid,
		Codigo:        peca.Codigo,
		Nome:          peca.Nome,
		Tipo:          peca.Tipo,
		PrecoCentavos: peca.PrecoCentavos,
		Ativa:         peca.Ativo,
	}, peca.QuantidadeEstoque, nil
}
