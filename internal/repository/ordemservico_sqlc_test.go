package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/repository"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func linhaOS(status string) db.OrdemServico {
	return db.OrdemServico{
		ID: 7, Uuid: uuid.New(), ClienteID: 1, VeiculoID: 2, Status: status, OrcamentoStatus: "NAO_ENVIADO",
		DescricaoProblema: "Barulho", ValorTotalCentavos: 100, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		ExecucaoIniciadaEm: sql.NullTime{Time: time.Now(), Valid: true},
	}
}

func TestOrdemServicoRepositorioCarregar(t *testing.T) {
	ctx := context.Background()
	row := linhaOS("EM_EXECUCAO")
	reparo := db.ReparoAdicional{ID: 3, Uuid: uuid.New(), OrdemServicoID: row.ID, Descricao: "Disco", Status: "PENDENTE", DecididoEm: sql.NullTime{}}
	servico := db.ListItensServicoByOrdemServicoRow{ID: 10, Uuid: uuid.New(), OrdemServicoID: row.ID, Quantidade: 1, ValorUnitarioCentavos: 8000, ServicoUuid: uuid.New(), ServicoNome: "Troca de óleo"}
	peca := db.ListItensPecaByOrdemServicoRow{
		ID: 11, Uuid: uuid.New(), OrdemServicoID: row.ID, PecaID: 5, Quantidade: 2, ValorUnitarioCentavos: 2500,
		PecaUuid: uuid.New(), PecaCodigo: "FLT-01", PecaNome: "Filtro", PecaTipo: "PECA",
		ReparoAdicionalUuid: uuid.NullUUID{UUID: reparo.Uuid, Valid: true}, ReparoAdicionalStatus: sql.NullString{String: "PENDENTE", Valid: true},
	}

	t.Run("SemTrava", func(t *testing.T) {
		store := novoStore(t)
		store.EXPECT().GetOrdemServico(ctx, row.Uuid).Return(row, nil)
		store.EXPECT().ListItensServicoByOrdemServico(ctx, row.ID).Return([]db.ListItensServicoByOrdemServicoRow{servico}, nil)
		store.EXPECT().ListItensPecaByOrdemServico(ctx, row.ID).Return([]db.ListItensPecaByOrdemServicoRow{peca}, nil)
		store.EXPECT().ListReparosAdicionaisByOrdemServico(ctx, row.ID).Return([]db.ReparoAdicional{reparo}, nil)

		os, err := repository.NewOrdemServicoRepositorio(store).Carregar(ctx, row.Uuid)
		require.NoError(t, err)
		require.Equal(t, row.ID, os.ID)
		require.Equal(t, ordemservico.StatusEmExecucao, os.Status)
		require.Equal(t, "Barulho", os.DescricaoProblema)
		require.NotNil(t, os.ExecucaoIniciadaEm)
		require.Len(t, os.Itens, 2)
		require.Equal(t, ordemservico.ItemServico, os.Itens[0].Tipo)
		require.Equal(t, "Troca de óleo", os.Itens[0].Nome)
		require.Equal(t, ordemservico.ItemPeca, os.Itens[1].Tipo)
		require.Equal(t, int64(5), os.Itens[1].CatalogoID)
		require.Equal(t, "FLT-01", os.Itens[1].Codigo)
		require.True(t, os.Itens[1].DoReparo(reparo.Uuid))
		require.Equal(t, ordemservico.ReparoPendente, os.Itens[1].StatusReparo)
		require.Len(t, os.Reparos, 1)
		require.Equal(t, "Disco", os.Reparos[0].Descricao)
		require.Nil(t, os.Reparos[0].DecididoEm)
		require.Empty(t, os.Alteracoes().Historico)
	})
	t.Run("ComTrava", func(t *testing.T) {
		store := novoStore(t)
		store.EXPECT().GetOrdemServicoForUpdate(ctx, row.Uuid).Return(row, nil)
		store.EXPECT().ListItensServicoByOrdemServico(ctx, row.ID).Return([]db.ListItensServicoByOrdemServicoRow{}, nil)
		store.EXPECT().ListItensPecaByOrdemServico(ctx, row.ID).Return([]db.ListItensPecaByOrdemServicoRow{}, nil)
		store.EXPECT().ListReparosAdicionaisByOrdemServico(ctx, row.ID).Return([]db.ReparoAdicional{}, nil)

		os, err := repository.NewOrdemServicoRepositorio(store).CarregarParaAtualizar(ctx, row.Uuid)
		require.NoError(t, err)
		require.Empty(t, os.Itens)
	})
	t.Run("NaoEncontrada", func(t *testing.T) {
		store := novoStore(t)
		store.EXPECT().GetOrdemServico(ctx, gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)
		store.EXPECT().GetOrdemServicoForUpdate(ctx, gomock.Any()).Return(db.OrdemServico{}, sql.ErrNoRows)
		repo := repository.NewOrdemServicoRepositorio(store)
		_, err := repo.Carregar(ctx, uuid.New())
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
		_, err = repo.CarregarParaAtualizar(ctx, uuid.New())
		require.ErrorIs(t, err, service.ErrOSNaoEncontrada)
	})
	t.Run("ErroNasLeituras", func(t *testing.T) {
		for _, falha := range []string{"servicos", "pecas", "reparos"} {
			store := novoStore(t)
			store.EXPECT().GetOrdemServico(ctx, row.Uuid).Return(row, nil)
			if falha == "servicos" {
				store.EXPECT().ListItensServicoByOrdemServico(ctx, row.ID).Return(nil, sql.ErrConnDone)
			} else {
				store.EXPECT().ListItensServicoByOrdemServico(ctx, row.ID).Return(nil, nil)
			}
			if falha == "pecas" {
				store.EXPECT().ListItensPecaByOrdemServico(ctx, row.ID).Return(nil, sql.ErrConnDone)
			} else if falha == "reparos" {
				store.EXPECT().ListItensPecaByOrdemServico(ctx, row.ID).Return(nil, nil)
				store.EXPECT().ListReparosAdicionaisByOrdemServico(ctx, row.ID).Return(nil, sql.ErrConnDone)
			}
			_, err := repository.NewOrdemServicoRepositorio(store).Carregar(ctx, row.Uuid)
			require.ErrorIs(t, err, sql.ErrConnDone, falha)
		}
	})
}

// Salvar de uma OS nova: cria a linha, os itens e o histórico; os itens
// ganham identidade do banco.
func TestOrdemServicoRepositorioSalvarNova(t *testing.T) {
	ctx := context.Background()
	store := novoStore(t)
	row := linhaOS("RECEBIDA")
	servico := ordemservico.CatalogoServico{ID: 4, UUID: uuid.New(), Nome: "Troca", PrecoCentavos: 8000, Ativo: true}
	peca := ordemservico.CatalogoPeca{ID: 5, UUID: uuid.New(), Codigo: "FLT", Nome: "Filtro", Tipo: "PECA", PrecoCentavos: 2500, Ativa: true}

	os := ordemservico.Nova(1, 2, "Barulho")
	require.NoError(t, os.AdicionarServico(servico, 1))
	require.NoError(t, os.AdicionarPeca(peca, 2, 10))
	os.AnotarHistorico(9, "OS aberta")

	store.EXPECT().CreateOrdemServico(ctx, db.CreateOrdemServicoParams{ClienteID: 1, VeiculoID: 2, DescricaoProblema: "Barulho"}).Return(row, nil)
	store.EXPECT().CreateItemServico(ctx, db.CreateItemServicoParams{OrdemServicoID: row.ID, ServicoID: 4, Quantidade: 1, ValorUnitarioCentavos: 8000}).
		Return(db.ItemServico{ID: 10, Uuid: uuid.New()}, nil)
	store.EXPECT().CreateItemPeca(ctx, db.CreateItemPecaParams{OrdemServicoID: row.ID, PecaID: 5, Quantidade: 2, ValorUnitarioCentavos: 2500}).
		Return(db.ItemPeca{ID: 11, Uuid: uuid.New()}, nil)
	store.EXPECT().UpdateOrdemServicoEstado(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdateOrdemServicoEstadoParams) (db.OrdemServico, error) {
			require.Equal(t, row.ID, arg.ID)
			require.Equal(t, "RECEBIDA", arg.Status)
			require.Equal(t, int64(13000), arg.ValorTotalCentavos)
			return row, nil
		})
	store.EXPECT().CreateHistoricoStatusOS(ctx, db.CreateHistoricoStatusOSParams{
		OrdemServicoID: row.ID, StatusAnterior: "", StatusNovo: "RECEBIDA", UsuarioID: sql.NullInt64{Int64: 9, Valid: true}, Observacao: "OS aberta",
	}).Return(db.HistoricoStatusOS{}, nil)

	require.NoError(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, &os))
	require.Equal(t, row.ID, os.ID)
	require.Equal(t, row.Uuid, os.UUID)
	require.Equal(t, int64(10), os.Itens[0].ID)
	require.Equal(t, int64(11), os.Itens[1].ID)
	require.Empty(t, os.Alteracoes().Historico, "alterações limpas depois de salvar")
}

// Salvar de uma OS existente: reparo novo com itens, reparo decidido, item
// removido e histórico sem mudança de status.
func TestOrdemServicoRepositorioSalvarAlteracoes(t *testing.T) {
	ctx := context.Background()
	store := novoStore(t)
	row := linhaOS("EM_EXECUCAO")
	decidido := ordemservico.ReparoAdicional{ID: 30, UUID: uuid.New(), Descricao: "Antigo", Status: ordemservico.ReparoPendente}
	removido := ordemservico.Item{ID: 12, UUID: uuid.New(), Tipo: ordemservico.ItemServico, Quantidade: 1}
	os := ordemservico.Reconstituir(ordemservico.OrdemServico{
		ID: row.ID, UUID: row.Uuid, ClienteID: 1, VeiculoID: 2, Status: ordemservico.StatusEmExecucao, StatusOrcamento: ordemservico.OrcamentoAprovado,
		Reparos: []ordemservico.ReparoAdicional{decidido},
		Itens:   []ordemservico.Item{removido, {ID: 13, UUID: uuid.New(), Tipo: ordemservico.ItemPeca, CatalogoID: 5, Quantidade: 1, ValorUnitarioCentavos: 100, ReparoUUID: uuid.NullUUID{UUID: decidido.UUID, Valid: true}, StatusReparo: ordemservico.ReparoPendente}},
	})
	require.NoError(t, os.AutorizarReparo(decidido.UUID, time.Now()))
	os.Status = ordemservico.StatusRecebida // só para permitir a remoção no teste
	require.NoError(t, os.RemoverItem(removido.UUID))
	os.Status = ordemservico.StatusEmExecucao
	novo, err := os.RegistrarReparo("Disco", []ordemservico.PedidoServico{{Servico: ordemservico.CatalogoServico{ID: 4, Ativo: true, PrecoCentavos: 500}, Quantidade: 1}}, nil)
	require.NoError(t, err)
	os.AnotarHistorico(0, "Reparo adicional autorizado pelo cliente: Antigo")

	criadoUUID := uuid.New()
	store.EXPECT().CreateReparoAdicional(ctx, db.CreateReparoAdicionalParams{OrdemServicoID: row.ID, Descricao: "Disco"}).
		Return(db.ReparoAdicional{ID: 31, Uuid: criadoUUID, CreatedAt: time.Now()}, nil)
	store.EXPECT().UpdateReparoAdicionalStatus(ctx, db.UpdateReparoAdicionalStatusParams{ID: 30, Status: "AUTORIZADO"}).Return(db.ReparoAdicional{}, nil)
	store.EXPECT().CreateItemServico(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateItemServicoParams) (db.ItemServico, error) {
			require.Equal(t, int64(31), arg.ReparoAdicionalID.Int64, "o item do reparo novo aponta para o id gerado")
			return db.ItemServico{ID: 14, Uuid: uuid.New()}, nil
		})
	store.EXPECT().DeleteItemServico(ctx, db.DeleteItemServicoParams{Uuid: removido.UUID, OrdemServicoID: row.ID}).Return(int64(12), nil)
	store.EXPECT().UpdateOrdemServicoEstado(ctx, gomock.Any()).Return(row, nil)
	store.EXPECT().CreateHistoricoStatusOS(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateHistoricoStatusOSParams) (db.HistoricoStatusOS, error) {
			require.Equal(t, "EM_EXECUCAO", arg.StatusAnterior)
			require.Equal(t, "EM_EXECUCAO", arg.StatusNovo)
			require.False(t, arg.UsuarioID.Valid)
			return db.HistoricoStatusOS{}, nil
		})

	require.NoError(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os))
	require.Equal(t, criadoUUID, novo.UUID, "o UUID do reparo passa a ser o do banco")
	require.True(t, os.PecasDoReparo(criadoUUID) != nil)
	for _, item := range os.Itens {
		require.NotZero(t, item.ID)
	}
}

func TestOrdemServicoRepositorioSalvarErros(t *testing.T) {
	ctx := context.Background()
	row := linhaOS("RECEBIDA")
	servico := ordemservico.CatalogoServico{ID: 4, Ativo: true}
	peca := ordemservico.CatalogoPeca{ID: 5, Ativa: true}

	t.Run("CriarOS", func(t *testing.T) {
		store := novoStore(t)
		os := ordemservico.Nova(1, 2, "")
		store.EXPECT().CreateOrdemServico(ctx, gomock.Any()).Return(db.OrdemServico{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, &os), sql.ErrConnDone)
	})
	t.Run("CriarItemServico", func(t *testing.T) {
		store := novoStore(t)
		os := ordemservico.Reconstituir(ordemservico.OrdemServico{ID: row.ID, Status: ordemservico.StatusRecebida})
		require.NoError(t, os.AdicionarServico(servico, 1))
		store.EXPECT().CreateItemServico(ctx, gomock.Any()).Return(db.ItemServico{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os), sql.ErrConnDone)
	})
	t.Run("CriarItemPeca", func(t *testing.T) {
		store := novoStore(t)
		os := ordemservico.Reconstituir(ordemservico.OrdemServico{ID: row.ID, Status: ordemservico.StatusRecebida})
		require.NoError(t, os.AdicionarPeca(peca, 1, 5))
		store.EXPECT().CreateItemPeca(ctx, gomock.Any()).Return(db.ItemPeca{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os), sql.ErrConnDone)
	})
	t.Run("RemoverItemPeca", func(t *testing.T) {
		store := novoStore(t)
		item := ordemservico.Item{ID: 3, UUID: uuid.New(), Tipo: ordemservico.ItemPeca}
		os := ordemservico.Reconstituir(ordemservico.OrdemServico{ID: row.ID, Status: ordemservico.StatusRecebida, Itens: []ordemservico.Item{item}})
		require.NoError(t, os.RemoverItem(item.UUID))
		store.EXPECT().DeleteItemPeca(ctx, db.DeleteItemPecaParams{Uuid: item.UUID, OrdemServicoID: row.ID}).Return(int64(0), sql.ErrNoRows)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os), ordemservico.ErrItemNaoEncontrado)
	})
	t.Run("CriarReparo", func(t *testing.T) {
		store := novoStore(t)
		os := ordemservico.Reconstituir(ordemservico.OrdemServico{ID: row.ID, Status: ordemservico.StatusEmExecucao, StatusOrcamento: ordemservico.OrcamentoAprovado})
		_, err := os.RegistrarReparo("x", []ordemservico.PedidoServico{{Servico: servico, Quantidade: 1}}, nil)
		require.NoError(t, err)
		store.EXPECT().CreateReparoAdicional(ctx, gomock.Any()).Return(db.ReparoAdicional{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os), sql.ErrConnDone)
	})
	t.Run("DecidirReparo", func(t *testing.T) {
		store := novoStore(t)
		reparo := ordemservico.ReparoAdicional{ID: 30, UUID: uuid.New(), Status: ordemservico.ReparoPendente}
		os := ordemservico.Reconstituir(ordemservico.OrdemServico{ID: row.ID, Status: ordemservico.StatusEmExecucao, Reparos: []ordemservico.ReparoAdicional{reparo}})
		require.NoError(t, os.RecusarReparo(reparo.UUID, time.Now()))
		store.EXPECT().UpdateReparoAdicionalStatus(ctx, gomock.Any()).Return(db.ReparoAdicional{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os), sql.ErrConnDone)
	})
	t.Run("EstadoEHistorico", func(t *testing.T) {
		store := novoStore(t)
		os := ordemservico.Reconstituir(ordemservico.OrdemServico{ID: row.ID, Status: ordemservico.StatusRecebida})
		store.EXPECT().UpdateOrdemServicoEstado(ctx, gomock.Any()).Return(db.OrdemServico{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os), sql.ErrConnDone)

		os.AnotarHistorico(1, "x")
		store.EXPECT().UpdateOrdemServicoEstado(ctx, gomock.Any()).Return(db.OrdemServico{}, nil)
		store.EXPECT().CreateHistoricoStatusOS(ctx, gomock.Any()).Return(db.HistoricoStatusOS{}, sql.ErrConnDone)
		require.ErrorIs(t, repository.NewOrdemServicoRepositorio(store).Salvar(ctx, os), sql.ErrConnDone)
	})
}
