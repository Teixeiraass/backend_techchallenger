package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

// PecaService gerencia peças e insumos e o controle de estoque.
type PecaService interface {
	CriarPeca(ctx context.Context, req dto.CriarPecaRequest, usuarioID int64) (*db.Peca, error)
	BuscarPeca(ctx context.Context, pecaUUID uuid.UUID) (*db.Peca, error)
	ListarPecas(ctx context.Context, query dto.ListPecasQuery) ([]db.Peca, error)
	ListarEstoqueBaixo(ctx context.Context) ([]db.Peca, error)
	AtualizarPeca(ctx context.Context, pecaUUID uuid.UUID, req dto.AtualizarPecaRequest) (*db.Peca, error)
	InativarPeca(ctx context.Context, pecaUUID uuid.UUID) (*db.Peca, error)
	ReporEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.ReporEstoqueRequest, usuarioID int64) (*db.Peca, error)
	AjustarEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.AjustarEstoqueRequest, usuarioID int64) (*db.Peca, error)
	ListarMovimentos(ctx context.Context, pecaUUID uuid.UUID, paginacao dto.PaginacaoQuery) ([]db.MovimentoEstoque, error)
}

type pecaService struct {
	store  db.Store
	sender notification.Sender
	config util.Config
}

func NewPecaService(store db.Store, sender notification.Sender, config util.Config) PecaService {
	return &pecaService{store: store, sender: sender, config: config}
}

func (s *pecaService) CriarPeca(ctx context.Context, req dto.CriarPecaRequest, usuarioID int64) (*db.Peca, error) {
	if err := estoque.ValidarCadastro(req.Tipo, req.PrecoCentavos, req.QuantidadeEstoque, req.EstoqueMinimo); err != nil {
		return nil, err
	}

	unidade := strings.ToUpper(strings.TrimSpace(req.Unidade))
	if unidade == "" {
		unidade = "UN"
	}

	var peca db.Peca
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		peca, err = q.CreatePeca(ctx, db.CreatePecaParams{
			Codigo:            strings.ToUpper(strings.TrimSpace(req.Codigo)),
			Nome:              strings.TrimSpace(req.Nome),
			Descricao:         req.Descricao,
			Tipo:              req.Tipo,
			Unidade:           unidade,
			PrecoCentavos:     req.PrecoCentavos,
			QuantidadeEstoque: req.QuantidadeEstoque,
			EstoqueMinimo:     req.EstoqueMinimo,
		})
		if err != nil {
			return err
		}

		if req.QuantidadeEstoque == 0 {
			return nil
		}
		_, err = q.CreateMovimentoEstoque(ctx, db.CreateMovimentoEstoqueParams{
			PecaID:         peca.ID,
			Tipo:           string(estoque.MovimentoEntrada),
			Quantidade:     req.QuantidadeEstoque,
			SaldoAnterior:  0,
			SaldoPosterior: req.QuantidadeEstoque,
			UsuarioID:      util.ToNullInt64(usuarioID),
			Observacao:     "Saldo inicial",
		})
		return err
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCodigoPecaDuplicado
		}
		return nil, err
	}
	return &peca, nil
}

func (s *pecaService) BuscarPeca(ctx context.Context, pecaUUID uuid.UUID) (*db.Peca, error) {
	peca, err := s.store.GetPeca(ctx, pecaUUID)
	if err != nil {
		return nil, notFound(err, ErrPecaNaoEncontrada)
	}
	return &peca, nil
}

func (s *pecaService) ListarPecas(ctx context.Context, query dto.ListPecasQuery) ([]db.Peca, error) {
	return s.store.ListPecas(ctx, db.ListPecasParams{
		ApenasAtivas: query.ApenasAtivas,
		Tipo:         query.Tipo,
		Busca:        query.Busca,
		Limite:       query.Limite(),
		Deslocamento: query.Deslocamento(),
	})
}

func (s *pecaService) ListarEstoqueBaixo(ctx context.Context) ([]db.Peca, error) {
	return s.store.ListPecasEstoqueBaixo(ctx)
}

func (s *pecaService) AtualizarPeca(ctx context.Context, pecaUUID uuid.UUID, req dto.AtualizarPecaRequest) (*db.Peca, error) {
	if err := estoque.ValidarCadastro(req.Tipo, req.PrecoCentavos, 0, req.EstoqueMinimo); err != nil {
		return nil, err
	}

	unidade := strings.ToUpper(strings.TrimSpace(req.Unidade))
	if unidade == "" {
		unidade = "UN"
	}

	peca, err := s.store.UpdatePeca(ctx, db.UpdatePecaParams{
		Uuid:          pecaUUID,
		Codigo:        strings.ToUpper(strings.TrimSpace(req.Codigo)),
		Nome:          strings.TrimSpace(req.Nome),
		Descricao:     req.Descricao,
		Tipo:          req.Tipo,
		Unidade:       unidade,
		PrecoCentavos: req.PrecoCentavos,
		EstoqueMinimo: req.EstoqueMinimo,
		Ativo:         *req.Ativo,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCodigoPecaDuplicado
		}
		return nil, notFound(err, ErrPecaNaoEncontrada)
	}
	return &peca, nil
}

func (s *pecaService) InativarPeca(ctx context.Context, pecaUUID uuid.UUID) (*db.Peca, error) {
	peca, err := s.store.InativarPeca(ctx, pecaUUID)
	if err != nil {
		return nil, notFound(err, ErrPecaNaoEncontrada)
	}
	return &peca, nil
}

// ReporEstoque registra a entrada de unidades e o movimento na mesma transação.
func (s *pecaService) ReporEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.ReporEstoqueRequest, usuarioID int64) (*db.Peca, error) {
	if err := estoque.ValidarReposicao(req.Quantidade); err != nil {
		return nil, err
	}

	var peca db.Peca
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		atual, err := q.GetPecaForUpdate(ctx, pecaUUID)
		if err != nil {
			return notFound(err, ErrPecaNaoEncontrada)
		}

		peca, err = q.ReporEstoquePeca(ctx, db.ReporEstoquePecaParams{ID: atual.ID, Quantidade: req.Quantidade})
		if err != nil {
			return err
		}

		_, err = q.CreateMovimentoEstoque(ctx, db.CreateMovimentoEstoqueParams{
			PecaID:         atual.ID,
			Tipo:           string(estoque.MovimentoEntrada),
			Quantidade:     req.Quantidade,
			SaldoAnterior:  atual.QuantidadeEstoque,
			SaldoPosterior: peca.QuantidadeEstoque,
			UsuarioID:      util.ToNullInt64(usuarioID),
			Observacao:     req.Observacao,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return &peca, nil
}

// AjustarEstoque define o saldo contado no inventário e registra a diferença.
func (s *pecaService) AjustarEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.AjustarEstoqueRequest, usuarioID int64) (*db.Peca, error) {
	if err := estoque.ValidarAjuste(*req.Quantidade); err != nil {
		return nil, err
	}

	var peca db.Peca
	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		atual, err := q.GetPecaForUpdate(ctx, pecaUUID)
		if err != nil {
			return notFound(err, ErrPecaNaoEncontrada)
		}

		peca, err = q.AjustarEstoquePeca(ctx, db.AjustarEstoquePecaParams{ID: atual.ID, Quantidade: *req.Quantidade})
		if err != nil {
			return err
		}

		_, err = q.CreateMovimentoEstoque(ctx, db.CreateMovimentoEstoqueParams{
			PecaID:         atual.ID,
			Tipo:           string(estoque.MovimentoAjuste),
			Quantidade:     peca.QuantidadeEstoque - atual.QuantidadeEstoque,
			SaldoAnterior:  atual.QuantidadeEstoque,
			SaldoPosterior: peca.QuantidadeEstoque,
			UsuarioID:      util.ToNullInt64(usuarioID),
			Observacao:     req.Observacao,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	if estoque.EstoqueBaixo(peca.QuantidadeEstoque, peca.EstoqueMinimo) {
		notificarEstoqueBaixo(ctx, s.sender, s.config.AdminNotificationTo, []db.Peca{peca})
	}
	return &peca, nil
}

func (s *pecaService) ListarMovimentos(ctx context.Context, pecaUUID uuid.UUID, paginacao dto.PaginacaoQuery) ([]db.MovimentoEstoque, error) {
	peca, err := s.store.GetPeca(ctx, pecaUUID)
	if err != nil {
		return nil, notFound(err, ErrPecaNaoEncontrada)
	}

	return s.store.ListMovimentosEstoqueByPeca(ctx, db.ListMovimentosEstoqueByPecaParams{
		PecaID: peca.ID,
		Limit:  paginacao.Limite(),
		Offset: paginacao.Deslocamento(),
	})
}

// notificarEstoqueBaixo envia o pedido de reposição ao administrador. Falha de
// envio só é registrada no log: a operação de estoque já foi concluída.
func notificarEstoqueBaixo(ctx context.Context, sender notification.Sender, to string, pecas []db.Peca) {
	for _, peca := range pecas {
		err := sender.Send(ctx, notification.Message{
			To:      to,
			Subject: "Pedido de reposição de estoque",
			Body: fmt.Sprintf("%s (%s) está com saldo %d %s, abaixo ou igual ao mínimo de %d.",
				peca.Nome, peca.Codigo, peca.QuantidadeEstoque, peca.Unidade, peca.EstoqueMinimo),
		})
		if err != nil {
			log.Printf("cannot send low stock notification: %v", err)
		}
	}
}
