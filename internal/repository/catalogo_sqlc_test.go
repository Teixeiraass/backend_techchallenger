package repository_test

import (
	"context"
	"database/sql"
	"testing"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/repository"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCatalogo(t *testing.T) {
	ctx := context.Background()
	store := novoStore(t)
	stubTx(store)
	catalogo := repository.NewUnidadeDeTrabalho(store).Leitura().Catalogo()

	servico := db.Servico{ID: 4, Uuid: uuid.New(), Nome: "Troca", PrecoCentavos: 8000, Ativo: true}
	cliente := db.Cliente{ID: 1, Uuid: uuid.New(), Nome: "Maria", Documento: "52998224725", Email: "m@x.com"}
	veiculo := db.Veiculo{ID: 2, Uuid: uuid.New(), ClienteID: 1, Placa: "ABC1D23", Marca: "VW", Modelo: "Gol", Ano: 2019}

	store.EXPECT().GetServico(ctx, servico.Uuid).Return(servico, nil)
	s, err := catalogo.Servico(ctx, servico.Uuid)
	require.NoError(t, err)
	require.Equal(t, servico.ID, s.ID)
	require.Equal(t, int64(8000), s.PrecoCentavos)
	require.True(t, s.Ativo)

	store.EXPECT().GetClienteByDocumento(ctx, cliente.Documento).Return(cliente, nil)
	c, err := catalogo.ClientePorDocumento(ctx, cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, cliente.ID, c.ID)
	require.Equal(t, "m@x.com", c.Email)

	store.EXPECT().GetClienteByID(ctx, cliente.ID).Return(cliente, nil)
	c, err = catalogo.ClientePorID(ctx, cliente.ID)
	require.NoError(t, err)
	require.Equal(t, cliente.Uuid, c.UUID)

	store.EXPECT().GetVeiculoByPlaca(ctx, veiculo.Placa).Return(veiculo, nil)
	v, err := catalogo.VeiculoPorPlaca(ctx, veiculo.Placa)
	require.NoError(t, err)
	require.Equal(t, int64(1), v.ClienteID)

	store.EXPECT().GetVeiculoByID(ctx, veiculo.ID).Return(veiculo, nil)
	v, err = catalogo.VeiculoPorID(ctx, veiculo.ID)
	require.NoError(t, err)
	require.Equal(t, "Gol", v.Modelo)

	store.EXPECT().ListHistoricoStatusOSByOrdemServico(ctx, int64(7)).Return([]db.HistoricoStatusOS{{StatusNovo: "RECEBIDA", Observacao: "OS aberta"}}, nil)
	h, err := catalogo.HistoricoDaOS(ctx, 7)
	require.NoError(t, err)
	require.Len(t, h, 1)
	require.Equal(t, "OS aberta", h[0].Observacao)

	// Não encontrados viram erros de domínio; outros erros sobem como estão.
	store.EXPECT().GetServico(ctx, gomock.Any()).Return(db.Servico{}, sql.ErrNoRows)
	_, err = catalogo.Servico(ctx, uuid.New())
	require.ErrorIs(t, err, service.ErrServicoNaoEncontrado)
	store.EXPECT().GetClienteByDocumento(ctx, gomock.Any()).Return(db.Cliente{}, sql.ErrNoRows)
	_, err = catalogo.ClientePorDocumento(ctx, "x")
	require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)
	store.EXPECT().GetClienteByID(ctx, gomock.Any()).Return(db.Cliente{}, sql.ErrNoRows)
	_, err = catalogo.ClientePorID(ctx, 9)
	require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)
	store.EXPECT().GetVeiculoByPlaca(ctx, gomock.Any()).Return(db.Veiculo{}, sql.ErrNoRows)
	_, err = catalogo.VeiculoPorPlaca(ctx, "x")
	require.ErrorIs(t, err, service.ErrVeiculoNaoEncontrado)
	store.EXPECT().GetVeiculoByID(ctx, gomock.Any()).Return(db.Veiculo{}, sql.ErrNoRows)
	_, err = catalogo.VeiculoPorID(ctx, 9)
	require.ErrorIs(t, err, service.ErrVeiculoNaoEncontrado)
	store.EXPECT().ListHistoricoStatusOSByOrdemServico(ctx, gomock.Any()).Return(nil, sql.ErrConnDone)
	_, err = catalogo.HistoricoDaOS(ctx, 9)
	require.ErrorIs(t, err, sql.ErrConnDone)
}
