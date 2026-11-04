package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

// PecaRepositorio implementa estoque.Repositorio sobre as tabelas pecas e
// movimentos_estoque.
type PecaRepositorio struct {
	q db.Querier
}

func NewPecaRepositorio(q db.Querier) *PecaRepositorio {
	return &PecaRepositorio{q: q}
}

func (r *PecaRepositorio) Carregar(ctx context.Context, pecaUUID uuid.UUID) (*estoque.Peca, error) {
	row, err := r.q.GetPeca(ctx, pecaUUID)
	if err != nil {
		return nil, naoEncontrado(err, service.ErrPecaNaoEncontrada)
	}
	return paraPeca(row), nil
}

func (r *PecaRepositorio) CarregarPorID(ctx context.Context, pecaID int64) (*estoque.Peca, error) {
	row, err := r.q.GetPecaByID(ctx, pecaID)
	if err != nil {
		return nil, naoEncontrado(err, service.ErrPecaNaoEncontrada)
	}
	return paraPeca(row), nil
}

func (r *PecaRepositorio) CarregarParaAtualizar(ctx context.Context, pecaUUID uuid.UUID) (*estoque.Peca, error) {
	row, err := r.q.GetPecaForUpdate(ctx, pecaUUID)
	if err != nil {
		return nil, naoEncontrado(err, service.ErrPecaNaoEncontrada)
	}
	return paraPeca(row), nil
}

// ParaPeca converte a linha da tabela no agregado (usada também pelas
// consultas de listagem).
func ParaPeca(row db.Peca) estoque.Peca {
	return *paraPeca(row)
}

func paraPeca(row db.Peca) *estoque.Peca {
	return &estoque.Peca{
		ID:            row.ID,
		UUID:          row.Uuid,
		Codigo:        row.Codigo,
		Nome:          row.Nome,
		Descricao:     row.Descricao,
		Tipo:          estoque.Tipo(row.Tipo),
		Unidade:       row.Unidade,
		PrecoCentavos: row.PrecoCentavos,
		Saldo:         row.QuantidadeEstoque,
		EstoqueMinimo: row.EstoqueMinimo,
		Ativa:         row.Ativo,
		CriadaEm:      row.CreatedAt,
		AtualizadaEm:  row.UpdatedAt,
	}
}

// Salvar cria a peça (ID zero) ou atualiza o cadastro. O saldo só muda por
// AplicarMovimento; na criação ele é o saldo inicial.
func (r *PecaRepositorio) Salvar(ctx context.Context, peca *estoque.Peca) error {
	var row db.Peca
	var err error
	if peca.ID == 0 {
		row, err = r.q.CreatePeca(ctx, db.CreatePecaParams{
			Codigo:            peca.Codigo,
			Nome:              peca.Nome,
			Descricao:         peca.Descricao,
			Tipo:              string(peca.Tipo),
			Unidade:           peca.Unidade,
			PrecoCentavos:     peca.PrecoCentavos,
			QuantidadeEstoque: peca.Saldo,
			EstoqueMinimo:     peca.EstoqueMinimo,
		})
	} else {
		row, err = r.q.UpdatePeca(ctx, db.UpdatePecaParams{
			Uuid:          peca.UUID,
			Codigo:        peca.Codigo,
			Nome:          peca.Nome,
			Descricao:     peca.Descricao,
			Tipo:          string(peca.Tipo),
			Unidade:       peca.Unidade,
			PrecoCentavos: peca.PrecoCentavos,
			EstoqueMinimo: peca.EstoqueMinimo,
			Ativo:         peca.Ativa,
		})
	}
	if err != nil {
		if service.IsUniqueViolation(err) {
			return service.ErrCodigoPecaDuplicado
		}
		return naoEncontrado(err, service.ErrPecaNaoEncontrada)
	}
	*peca = *paraPeca(row)
	return nil
}

// AplicarMovimento grava a mudança de saldo e a linha do movimento. A saída
// usa o UPDATE condicional (quantidade_estoque >= qtd): sem linha afetada, o
// saldo acabou entre a leitura e a gravação e o erro é ErrEstoqueInsuficiente.
func (r *PecaRepositorio) AplicarMovimento(ctx context.Context, peca *estoque.Peca, movimento estoque.MovimentoEstoque) error {
	var row db.Peca
	var err error
	switch movimento.Tipo {
	case estoque.MovimentoSaida:
		row, err = r.q.BaixarEstoquePeca(ctx, db.BaixarEstoquePecaParams{ID: peca.ID, Quantidade: -movimento.Quantidade})
		if errors.Is(err, sql.ErrNoRows) {
			return estoque.ErrEstoqueInsuficientePara(peca.Nome, peca.Codigo)
		}
	case estoque.MovimentoEntrada:
		row, err = r.q.ReporEstoquePeca(ctx, db.ReporEstoquePecaParams{ID: peca.ID, Quantidade: movimento.Quantidade})
	case estoque.MovimentoAjuste:
		row, err = r.q.AjustarEstoquePeca(ctx, db.AjustarEstoquePecaParams{ID: peca.ID, Quantidade: movimento.SaldoPosterior})
	}
	if err != nil {
		return err
	}
	peca.Saldo, peca.AtualizadaEm = row.QuantidadeEstoque, row.UpdatedAt

	movimento.PecaID = peca.ID
	return r.RegistrarMovimento(ctx, movimento)
}

func (r *PecaRepositorio) RegistrarMovimento(ctx context.Context, movimento estoque.MovimentoEstoque) error {
	_, err := r.q.CreateMovimentoEstoque(ctx, db.CreateMovimentoEstoqueParams{
		PecaID:         movimento.PecaID,
		Tipo:           string(movimento.Tipo),
		Quantidade:     movimento.Quantidade,
		SaldoAnterior:  movimento.SaldoAnterior,
		SaldoPosterior: movimento.SaldoPosterior,
		OrdemServicoID: util.ToNullInt64(movimento.OrdemServicoID),
		UsuarioID:      util.ToNullInt64(movimento.UsuarioID),
		Observacao:     movimento.Observacao,
	})
	return err
}
