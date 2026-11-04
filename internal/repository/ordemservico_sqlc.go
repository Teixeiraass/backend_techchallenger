package repository

import (
	"context"
	"database/sql"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

// OrdemServicoRepositorio implementa ordemservico.Repositorio sobre as
// tabelas ordens_servico, itens_servico, itens_peca, reparos_adicionais e
// historico_status_os.
type OrdemServicoRepositorio struct {
	q db.Querier
}

func NewOrdemServicoRepositorio(q db.Querier) *OrdemServicoRepositorio {
	return &OrdemServicoRepositorio{q: q}
}

func (r *OrdemServicoRepositorio) Carregar(ctx context.Context, osUUID uuid.UUID) (*ordemservico.OrdemServico, error) {
	row, err := r.q.GetOrdemServico(ctx, osUUID)
	if err != nil {
		return nil, naoEncontrado(err, service.ErrOSNaoEncontrada)
	}
	return r.reconstituir(ctx, row)
}

func (r *OrdemServicoRepositorio) CarregarParaAtualizar(ctx context.Context, osUUID uuid.UUID) (*ordemservico.OrdemServico, error) {
	row, err := r.q.GetOrdemServicoForUpdate(ctx, osUUID)
	if err != nil {
		return nil, naoEncontrado(err, service.ErrOSNaoEncontrada)
	}
	return r.reconstituir(ctx, row)
}

func (r *OrdemServicoRepositorio) reconstituir(ctx context.Context, row db.OrdemServico) (*ordemservico.OrdemServico, error) {
	servicos, err := r.q.ListItensServicoByOrdemServico(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	pecas, err := r.q.ListItensPecaByOrdemServico(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	reparos, err := r.q.ListReparosAdicionaisByOrdemServico(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return ordemservico.Reconstituir(paraAgregado(row, servicos, pecas, reparos)), nil
}

func paraAgregado(row db.OrdemServico, servicos []db.ListItensServicoByOrdemServicoRow, pecas []db.ListItensPecaByOrdemServicoRow, reparos []db.ReparoAdicional) ordemservico.OrdemServico {
	os := ordemservico.OrdemServico{
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
	for _, rep := range reparos {
		os.Reparos = append(os.Reparos, ordemservico.ReparoAdicional{
			ID:         rep.ID,
			UUID:       rep.Uuid,
			Descricao:  rep.Descricao,
			Status:     ordemservico.StatusReparo(rep.Status),
			DecididoEm: util.FromNullTime(rep.DecididoEm),
			CriadoEm:   rep.CreatedAt,
		})
	}
	return os
}

// Salvar persiste o agregado: cria a linha da OS quando ela ainda não tem
// identidade; grava reparos novos e decididos, itens novos e removidos
// (pela diferença anotada na raiz), o estado da raiz e as entradas de
// histórico. Ao final limpa as alterações pendentes.
func (r *OrdemServicoRepositorio) Salvar(ctx context.Context, os *ordemservico.OrdemServico) error {
	if os.ID == 0 {
		row, err := r.q.CreateOrdemServico(ctx, db.CreateOrdemServicoParams{
			ClienteID:         os.ClienteID,
			VeiculoID:         os.VeiculoID,
			DescricaoProblema: os.DescricaoProblema,
		})
		if err != nil {
			return err
		}
		os.ID, os.UUID = row.ID, row.Uuid
		os.CriadaEm, os.AtualizadaEm = row.CreatedAt, row.UpdatedAt
	}

	reparoIDs := make(map[uuid.UUID]int64, len(os.Reparos))
	for i := range os.Reparos {
		reparo := &os.Reparos[i]
		if reparo.ID == 0 {
			criado, err := r.q.CreateReparoAdicional(ctx, db.CreateReparoAdicionalParams{OrdemServicoID: os.ID, Descricao: reparo.Descricao})
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
		if _, err := r.q.UpdateReparoAdicionalStatus(ctx, db.UpdateReparoAdicionalStatusParams{ID: reparo.ID, Status: string(reparo.Status)}); err != nil {
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
		if err := r.criarItem(ctx, os.ID, item, reparoID); err != nil {
			return err
		}
	}
	for _, item := range os.Alteracoes().ItensRemovidos {
		if err := r.removerItem(ctx, os.ID, item); err != nil {
			return err
		}
	}

	if _, err := r.q.UpdateOrdemServicoEstado(ctx, db.UpdateOrdemServicoEstadoParams{
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

	for _, entrada := range os.Alteracoes().Historico {
		if _, err := r.q.CreateHistoricoStatusOS(ctx, db.CreateHistoricoStatusOSParams{
			OrdemServicoID: os.ID,
			StatusAnterior: string(entrada.StatusAnterior),
			StatusNovo:     string(entrada.StatusNovo),
			UsuarioID:      util.ToNullInt64(entrada.UsuarioID),
			Observacao:     entrada.Observacao,
		}); err != nil {
			return err
		}
	}

	os.LimparAlteracoes()
	return nil
}

func (r *OrdemServicoRepositorio) criarItem(ctx context.Context, osID int64, item *ordemservico.Item, reparoID sql.NullInt64) error {
	switch item.Tipo {
	case ordemservico.ItemServico:
		criado, err := r.q.CreateItemServico(ctx, db.CreateItemServicoParams{
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
		criado, err := r.q.CreateItemPeca(ctx, db.CreateItemPecaParams{
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

func (r *OrdemServicoRepositorio) removerItem(ctx context.Context, osID int64, item ordemservico.Item) error {
	var err error
	switch item.Tipo {
	case ordemservico.ItemServico:
		_, err = r.q.DeleteItemServico(ctx, db.DeleteItemServicoParams{Uuid: item.UUID, OrdemServicoID: osID})
	case ordemservico.ItemPeca:
		_, err = r.q.DeleteItemPeca(ctx, db.DeleteItemPecaParams{Uuid: item.UUID, OrdemServicoID: osID})
	}
	return naoEncontrado(err, ordemservico.ErrItemNaoEncontrado)
}
