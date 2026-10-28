package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

func TestOrdemServicoItensEHistorico(t *testing.T) {
	os, cliente, veiculo := createRandomOrdemServico(t)
	ctx := context.Background()
	servico := createRandomServico(t, 8000)
	peca := createRandomPeca(t, 3500, 10, 2)

	itemServico, err := testQueries.CreateItemServico(ctx, CreateItemServicoParams{
		OrdemServicoID:        os.ID,
		ServicoID:             servico.ID,
		Quantidade:            1,
		ValorUnitarioCentavos: servico.PrecoCentavos,
	})
	require.NoError(t, err)

	reparo, err := testQueries.CreateReparoAdicional(ctx, CreateReparoAdicionalParams{OrdemServicoID: os.ID, Descricao: "Disco empenado"})
	require.NoError(t, err)
	require.Equal(t, "PENDENTE", reparo.Status)

	_, err = testQueries.CreateItemPeca(ctx, CreateItemPecaParams{
		OrdemServicoID:        os.ID,
		PecaID:                peca.ID,
		ReparoAdicionalID:     sql.NullInt64{Int64: reparo.ID, Valid: true},
		Quantidade:            2,
		ValorUnitarioCentavos: peca.PrecoCentavos,
	})
	require.NoError(t, err)

	servicos, err := testQueries.ListItensServicoByOrdemServico(ctx, os.ID)
	require.NoError(t, err)
	require.Len(t, servicos, 1)
	require.Equal(t, servico.Nome, servicos[0].ServicoNome)
	require.False(t, servicos[0].ReparoAdicionalStatus.Valid)

	pecas, err := testQueries.ListItensPecaByOrdemServico(ctx, os.ID)
	require.NoError(t, err)
	require.Len(t, pecas, 1)
	require.Equal(t, "PENDENTE", pecas[0].ReparoAdicionalStatus.String)
	require.Equal(t, reparo.Uuid, pecas[0].ReparoAdicionalUuid.UUID)

	// Item de reparo adicional não pode ser removido pela rota de itens.
	_, err = testQueries.DeleteItemPeca(ctx, DeleteItemPecaParams{Uuid: pecas[0].Uuid, OrdemServicoID: os.ID})
	require.ErrorIs(t, err, sql.ErrNoRows)

	_, err = testQueries.DeleteItemServico(ctx, DeleteItemServicoParams{Uuid: itemServico.Uuid, OrdemServicoID: os.ID})
	require.NoError(t, err)

	locked, err := testQueries.GetReparoAdicionalForUpdate(ctx, GetReparoAdicionalForUpdateParams{Uuid: reparo.Uuid, OrdemServicoID: os.ID})
	require.NoError(t, err)
	decidido, err := testQueries.UpdateReparoAdicionalStatus(ctx, UpdateReparoAdicionalStatusParams{ID: locked.ID, Status: "AUTORIZADO"})
	require.NoError(t, err)
	require.True(t, decidido.DecididoEm.Valid)

	reparos, err := testQueries.ListReparosAdicionaisByOrdemServico(ctx, os.ID)
	require.NoError(t, err)
	require.Len(t, reparos, 1)

	_, err = testQueries.CreateHistoricoStatusOS(ctx, CreateHistoricoStatusOSParams{
		OrdemServicoID: os.ID,
		StatusAnterior: "",
		StatusNovo:     "RECEBIDA",
		Observacao:     "OS aberta",
	})
	require.NoError(t, err)
	historico, err := testQueries.ListHistoricoStatusOSByOrdemServico(ctx, os.ID)
	require.NoError(t, err)
	require.Len(t, historico, 1)

	lista, err := testQueries.ListOrdensServico(ctx, ListOrdensServicoParams{
		ClienteID: sql.NullInt64{Int64: cliente.ID, Valid: true},
		Limite:    10,
	})
	require.NoError(t, err)
	require.Len(t, lista, 1)
	require.Equal(t, veiculo.Placa, lista[0].VeiculoPlaca)

	// Cliente com OS não pode ser removido.
	_, err = testQueries.DeleteVeiculo(ctx, veiculo.Uuid)
	require.Error(t, err)
}

func TestUpdateOrdemServicoEstadoETempoMedio(t *testing.T) {
	os, _, _ := createRandomOrdemServico(t)
	ctx := context.Background()
	servico := createRandomServico(t, 5000)
	_, err := testQueries.CreateItemServico(ctx, CreateItemServicoParams{
		OrdemServicoID:        os.ID,
		ServicoID:             servico.ID,
		Quantidade:            1,
		ValorUnitarioCentavos: servico.PrecoCentavos,
	})
	require.NoError(t, err)

	// Um ano aleatório no futuro distante isola este cenário das outras OS
	// do banco, inclusive das criadas em execuções anteriores do teste.
	ano := int(util.RandomInt(2100, 9000))
	inicio := time.Date(ano, 1, 10, 8, 0, 0, 0, time.UTC)
	fim := inicio.Add(90 * time.Minute)

	locked, err := testQueries.GetOrdemServicoForUpdate(ctx, os.Uuid)
	require.NoError(t, err)

	updated, err := testQueries.UpdateOrdemServicoEstado(ctx, UpdateOrdemServicoEstadoParams{
		ID:                 locked.ID,
		Status:             "FINALIZADA",
		OrcamentoStatus:    "APROVADO",
		ValorTotalCentavos: 5000,
		ExecucaoIniciadaEm: sql.NullTime{Time: inicio, Valid: true},
		FinalizadaEm:       sql.NullTime{Time: fim, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, "FINALIZADA", updated.Status)

	total, err := testQueries.UpdateOrdemServicoValorTotal(ctx, UpdateOrdemServicoValorTotalParams{ID: os.ID, ValorTotalCentavos: 5000})
	require.NoError(t, err)
	require.Equal(t, int64(5000), total.ValorTotalCentavos)

	periodo := GetTempoMedioExecucaoParams{
		DataInicio: sql.NullTime{Time: time.Date(ano, 1, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		DataFim:    sql.NullTime{Time: time.Date(ano, 1, 11, 0, 0, 0, 0, time.UTC), Valid: true},
	}
	media, err := testQueries.GetTempoMedioExecucao(ctx, periodo)
	require.NoError(t, err)
	require.Equal(t, int64(1), media.TotalOrdens)
	require.Equal(t, float64(90*60), media.MediaSegundos)

	porServico, err := testQueries.ListTempoMedioExecucaoPorServico(ctx, ListTempoMedioExecucaoPorServicoParams(periodo))
	require.NoError(t, err)
	require.Len(t, porServico, 1)
	require.Equal(t, servico.Uuid, porServico[0].ServicoUuid)

	// Sem OS no período, a média vem zerada (COALESCE) em vez de NULL.
	vazio, err := testQueries.GetTempoMedioExecucao(ctx, GetTempoMedioExecucaoParams{
		DataInicio: sql.NullTime{Time: time.Date(9001, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), vazio.TotalOrdens)
	require.Equal(t, float64(0), vazio.MediaSegundos)

	// O CHECK de status recusa valores fora dos seis do enunciado.
	_, err = testQueries.UpdateOrdemServicoEstado(ctx, UpdateOrdemServicoEstadoParams{
		ID:              os.ID,
		Status:          "CANCELADA",
		OrcamentoStatus: "APROVADO",
	})
	require.Error(t, err)
}

func TestUsuarioESessao(t *testing.T) {
	ctx := context.Background()
	usuario, err := testQueries.GetUsuarioByEmail(ctx, "admin@oficina.com")
	require.NoError(t, err, "o seed da migration 000005 cria o administrador")
	require.Equal(t, "ADMINISTRADOR", usuario.Papel)

	byUUID, err := testQueries.GetUsuario(ctx, usuario.Uuid)
	require.NoError(t, err)
	require.Equal(t, usuario.ID, byUUID.ID)

	usuarios, err := testQueries.ListUsuarios(ctx, ListUsuariosParams{Limit: 10})
	require.NoError(t, err)
	require.NotEmpty(t, usuarios)

	_, err = testQueries.CreateUsuario(ctx, CreateUsuarioParams{
		Nome:           "Inválido",
		Email:          "papel-invalido@oficina.com",
		HashedPassword: "x",
		Papel:          "GERENTE",
	})
	require.Error(t, err, "papel fora do CHECK deve ser recusado")
}
