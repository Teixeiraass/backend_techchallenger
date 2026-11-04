package repository

import (
	"context"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/google/uuid"
)

// Catalogo implementa service.Catalogo: leituras de cadastro e de serviços
// que a OS precisa.
type Catalogo struct {
	q db.Querier
}

func (c *Catalogo) Servico(ctx context.Context, servicoUUID uuid.UUID) (ordemservico.CatalogoServico, error) {
	s, err := c.q.GetServico(ctx, servicoUUID)
	if err != nil {
		return ordemservico.CatalogoServico{}, naoEncontrado(err, service.ErrServicoNaoEncontrado)
	}
	return ordemservico.CatalogoServico{ID: s.ID, UUID: s.Uuid, Nome: s.Nome, PrecoCentavos: s.PrecoCentavos, Ativo: s.Ativo}, nil
}

func (c *Catalogo) ClientePorDocumento(ctx context.Context, documento string) (readmodel.Cliente, error) {
	cli, err := c.q.GetClienteByDocumento(ctx, documento)
	if err != nil {
		return readmodel.Cliente{}, naoEncontrado(err, service.ErrClienteNaoEncontrado)
	}
	return readmodel.DeCliente(cli), nil
}

func (c *Catalogo) ClientePorID(ctx context.Context, clienteID int64) (readmodel.Cliente, error) {
	cli, err := c.q.GetClienteByID(ctx, clienteID)
	if err != nil {
		return readmodel.Cliente{}, naoEncontrado(err, service.ErrClienteNaoEncontrado)
	}
	return readmodel.DeCliente(cli), nil
}

func (c *Catalogo) VeiculoPorPlaca(ctx context.Context, placa string) (readmodel.Veiculo, error) {
	v, err := c.q.GetVeiculoByPlaca(ctx, placa)
	if err != nil {
		return readmodel.Veiculo{}, naoEncontrado(err, service.ErrVeiculoNaoEncontrado)
	}
	return readmodel.DeVeiculo(v, uuid.Nil), nil
}

func (c *Catalogo) VeiculoPorID(ctx context.Context, veiculoID int64) (readmodel.Veiculo, error) {
	v, err := c.q.GetVeiculoByID(ctx, veiculoID)
	if err != nil {
		return readmodel.Veiculo{}, naoEncontrado(err, service.ErrVeiculoNaoEncontrado)
	}
	return readmodel.DeVeiculo(v, uuid.Nil), nil
}

func (c *Catalogo) HistoricoDaOS(ctx context.Context, osID int64) ([]readmodel.HistoricoStatus, error) {
	rows, err := c.q.ListHistoricoStatusOSByOrdemServico(ctx, osID)
	if err != nil {
		return nil, err
	}
	historico := make([]readmodel.HistoricoStatus, 0, len(rows))
	for _, h := range rows {
		historico = append(historico, readmodel.DeHistoricoStatus(h))
	}
	return historico, nil
}
