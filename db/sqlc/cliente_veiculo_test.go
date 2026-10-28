package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestClienteCRUD(t *testing.T) {
	cliente := createRandomCliente(t)
	ctx := context.Background()

	byUUID, err := testQueries.GetCliente(ctx, cliente.Uuid)
	require.NoError(t, err)
	require.Equal(t, cliente.ID, byUUID.ID)

	byDocumento, err := testQueries.GetClienteByDocumento(ctx, cliente.Documento)
	require.NoError(t, err)
	require.Equal(t, cliente.ID, byDocumento.ID)

	updated, err := testQueries.UpdateCliente(ctx, UpdateClienteParams{
		Uuid:     cliente.Uuid,
		Nome:     "Nome Atualizado",
		Email:    cliente.Email,
		Telefone: "11888880000",
	})
	require.NoError(t, err)
	require.Equal(t, "Nome Atualizado", updated.Nome)

	list, err := testQueries.ListClientes(ctx, ListClientesParams{Busca: cliente.Documento, Limite: 10})
	require.NoError(t, err)
	require.Len(t, list, 1)

	_, err = testQueries.DeleteCliente(ctx, cliente.Uuid)
	require.NoError(t, err)

	_, err = testQueries.GetCliente(ctx, cliente.Uuid)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestClienteDocumentoUnico(t *testing.T) {
	cliente := createRandomCliente(t)

	_, err := testQueries.CreateCliente(context.Background(), CreateClienteParams{
		Nome:          "Outro",
		Documento:     cliente.Documento,
		TipoDocumento: "CPF",
	})
	var pqErr *pq.Error
	require.ErrorAs(t, err, &pqErr)
	require.Equal(t, "unique_violation", pqErr.Code.Name())
}

func TestClienteComVeiculoNaoPodeSerRemovido(t *testing.T) {
	cliente := createRandomCliente(t)
	createRandomVeiculo(t, cliente.ID)

	_, err := testQueries.DeleteCliente(context.Background(), cliente.Uuid)
	var pqErr *pq.Error
	require.ErrorAs(t, err, &pqErr)
	require.Equal(t, "foreign_key_violation", pqErr.Code.Name())
}

func TestVeiculoCRUD(t *testing.T) {
	cliente := createRandomCliente(t)
	veiculo := createRandomVeiculo(t, cliente.ID)
	ctx := context.Background()

	row, err := testQueries.GetVeiculo(ctx, veiculo.Uuid)
	require.NoError(t, err)
	require.Equal(t, veiculo.ID, row.Veiculo.ID)
	require.Equal(t, cliente.Uuid, row.ClienteUuid)

	byPlaca, err := testQueries.GetVeiculoByPlaca(ctx, veiculo.Placa)
	require.NoError(t, err)
	require.Equal(t, veiculo.ID, byPlaca.ID)

	list, err := testQueries.ListVeiculos(ctx, ListVeiculosParams{
		ClienteID: sql.NullInt64{Int64: cliente.ID, Valid: true},
		Limite:    10,
	})
	require.NoError(t, err)
	require.Len(t, list, 1)

	updated, err := testQueries.UpdateVeiculo(ctx, UpdateVeiculoParams{
		Uuid:   veiculo.Uuid,
		Placa:  veiculo.Placa,
		Marca:  "Toyota",
		Modelo: "Corolla",
		Ano:    2022,
	})
	require.NoError(t, err)
	require.Equal(t, "Toyota", updated.Marca)

	_, err = testQueries.DeleteVeiculo(ctx, veiculo.Uuid)
	require.NoError(t, err)
}
