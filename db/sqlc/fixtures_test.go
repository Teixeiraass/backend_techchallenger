package db

import (
	"context"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/stretchr/testify/require"
)

func createRandomCliente(t *testing.T) Cliente {
	arg := CreateClienteParams{
		Nome:          util.RandomName(),
		Documento:     util.RandomCPF(),
		TipoDocumento: "CPF",
		Email:         util.RandomEmail(),
		Telefone:      "11999990000",
	}

	cliente, err := testQueries.CreateCliente(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, arg.Documento, cliente.Documento)
	require.NotZero(t, cliente.Uuid)
	return cliente
}

func createRandomVeiculo(t *testing.T, clienteID int64) Veiculo {
	arg := CreateVeiculoParams{
		ClienteID: clienteID,
		Placa:     util.RandomPlaca(),
		Marca:     "Fiat",
		Modelo:    util.RandomName(),
		Ano:       2020,
	}

	veiculo, err := testQueries.CreateVeiculo(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, arg.Placa, veiculo.Placa)
	return veiculo
}

func createRandomServico(t *testing.T, preco int64) Servico {
	servico, err := testQueries.CreateServico(context.Background(), CreateServicoParams{
		Nome:                 "Serviço " + util.RandomString(10),
		PrecoCentavos:        preco,
		TempoEstimadoMinutos: 30,
	})
	require.NoError(t, err)
	return servico
}

func createRandomPeca(t *testing.T, preco int64, saldo int32, minimo int32) Peca {
	peca, err := testQueries.CreatePeca(context.Background(), CreatePecaParams{
		Codigo:            "T-" + util.RandomString(10),
		Nome:              "Peça " + util.RandomString(6),
		Tipo:              "PECA",
		Unidade:           "UN",
		PrecoCentavos:     preco,
		QuantidadeEstoque: saldo,
		EstoqueMinimo:     minimo,
	})
	require.NoError(t, err)
	return peca
}

func createRandomOrdemServico(t *testing.T) (OrdemServico, Cliente, Veiculo) {
	cliente := createRandomCliente(t)
	veiculo := createRandomVeiculo(t, cliente.ID)

	os, err := testQueries.CreateOrdemServico(context.Background(), CreateOrdemServicoParams{
		ClienteID:         cliente.ID,
		VeiculoID:         veiculo.ID,
		DescricaoProblema: "Barulho no motor",
	})
	require.NoError(t, err)
	require.Equal(t, "RECEBIDA", os.Status)
	require.Equal(t, "NAO_ENVIADO", os.OrcamentoStatus)
	return os, cliente, veiculo
}
