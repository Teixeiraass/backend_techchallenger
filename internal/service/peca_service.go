package service

import (
	"context"
	"fmt"
	"log"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

// PecaService gerencia peças e insumos e o controle de estoque. Os comandos
// passam pelo agregado estoque.Peca e pelo repositório de estoque; as
// listagens leem o read model.
type PecaService interface {
	CriarPeca(ctx context.Context, req dto.CriarPecaRequest, usuarioID int64) (*estoque.Peca, error)
	BuscarPeca(ctx context.Context, pecaUUID uuid.UUID) (*estoque.Peca, error)
	ListarPecas(ctx context.Context, query dto.ListPecasQuery) ([]estoque.Peca, error)
	ListarEstoqueBaixo(ctx context.Context) ([]estoque.Peca, error)
	AtualizarPeca(ctx context.Context, pecaUUID uuid.UUID, req dto.AtualizarPecaRequest) (*estoque.Peca, error)
	InativarPeca(ctx context.Context, pecaUUID uuid.UUID) (*estoque.Peca, error)
	ReporEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.ReporEstoqueRequest, usuarioID int64) (*estoque.Peca, error)
	AjustarEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.AjustarEstoqueRequest, usuarioID int64) (*estoque.Peca, error)
	ListarMovimentos(ctx context.Context, pecaUUID uuid.UUID, paginacao dto.PaginacaoQuery) ([]db.MovimentoEstoque, error)
}

type pecaService struct {
	uow    UnidadeDeTrabalho
	store  db.Store
	sender notification.Sender
	config util.Config
}

func NewPecaService(uow UnidadeDeTrabalho, store db.Store, sender notification.Sender, config util.Config) PecaService {
	return &pecaService{uow: uow, store: store, sender: sender, config: config}
}

func cadastroDe(codigo, nome, descricao, tipo, unidade string, preco int64, minimo int32) estoque.Cadastro {
	return estoque.Cadastro{Codigo: codigo, Nome: nome, Descricao: descricao, Tipo: tipo, Unidade: unidade, PrecoCentavos: preco, EstoqueMinimo: minimo}
}

func (s *pecaService) CriarPeca(ctx context.Context, req dto.CriarPecaRequest, usuarioID int64) (*estoque.Peca, error) {
	peca, movimento, err := estoque.NovaPeca(cadastroDe(req.Codigo, req.Nome, req.Descricao, req.Tipo, req.Unidade, req.PrecoCentavos, req.EstoqueMinimo), req.QuantidadeEstoque)
	if err != nil {
		return nil, err
	}

	err = s.uow.Executar(ctx, func(r Repositorios) error {
		if err := r.Pecas().Salvar(ctx, peca); err != nil {
			return err
		}
		if movimento == nil {
			return nil
		}
		// O saldo inicial já foi gravado com a peça: aqui entra só a linha do movimento.
		movimento.PecaID, movimento.UsuarioID = peca.ID, usuarioID
		return r.Pecas().RegistrarMovimento(ctx, *movimento)
	})
	if err != nil {
		return nil, err
	}
	return peca, nil
}

func (s *pecaService) BuscarPeca(ctx context.Context, pecaUUID uuid.UUID) (*estoque.Peca, error) {
	return s.uow.Leitura().Pecas().Carregar(ctx, pecaUUID)
}

func (s *pecaService) ListarPecas(ctx context.Context, query dto.ListPecasQuery) ([]estoque.Peca, error) {
	rows, err := s.store.ListPecas(ctx, db.ListPecasParams{
		ApenasAtivas: query.ApenasAtivas,
		Tipo:         query.Tipo,
		Busca:        query.Busca,
		Limite:       query.Limite(),
		Deslocamento: query.Deslocamento(),
	})
	if err != nil {
		return nil, err
	}
	return pecasDe(rows), nil
}

func (s *pecaService) ListarEstoqueBaixo(ctx context.Context) ([]estoque.Peca, error) {
	rows, err := s.store.ListPecasEstoqueBaixo(ctx)
	if err != nil {
		return nil, err
	}
	return pecasDe(rows), nil
}

func pecasDe(rows []db.Peca) []estoque.Peca {
	pecas := make([]estoque.Peca, 0, len(rows))
	for _, row := range rows {
		pecas = append(pecas, estoque.Peca{
			ID: row.ID, UUID: row.Uuid, Codigo: row.Codigo, Nome: row.Nome, Descricao: row.Descricao,
			Tipo: estoque.Tipo(row.Tipo), Unidade: row.Unidade, PrecoCentavos: row.PrecoCentavos,
			Saldo: row.QuantidadeEstoque, EstoqueMinimo: row.EstoqueMinimo, Ativa: row.Ativo,
			CriadaEm: row.CreatedAt, AtualizadaEm: row.UpdatedAt,
		})
	}
	return pecas
}

func (s *pecaService) AtualizarPeca(ctx context.Context, pecaUUID uuid.UUID, req dto.AtualizarPecaRequest) (*estoque.Peca, error) {
	var peca *estoque.Peca
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		peca, err = r.Pecas().CarregarParaAtualizar(ctx, pecaUUID)
		if err != nil {
			return err
		}
		if err := peca.Atualizar(cadastroDe(req.Codigo, req.Nome, req.Descricao, req.Tipo, req.Unidade, req.PrecoCentavos, req.EstoqueMinimo), *req.Ativo); err != nil {
			return err
		}
		return r.Pecas().Salvar(ctx, peca)
	})
	if err != nil {
		return nil, err
	}
	return peca, nil
}

func (s *pecaService) InativarPeca(ctx context.Context, pecaUUID uuid.UUID) (*estoque.Peca, error) {
	var peca *estoque.Peca
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		peca, err = r.Pecas().CarregarParaAtualizar(ctx, pecaUUID)
		if err != nil {
			return err
		}
		peca.Inativar()
		return r.Pecas().Salvar(ctx, peca)
	})
	if err != nil {
		return nil, err
	}
	return peca, nil
}

// ReporEstoque registra a entrada de unidades e o movimento na mesma transação.
func (s *pecaService) ReporEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.ReporEstoqueRequest, usuarioID int64) (*estoque.Peca, error) {
	if err := estoque.ValidarReposicao(req.Quantidade); err != nil {
		return nil, err
	}
	return s.movimentar(ctx, pecaUUID, func(peca *estoque.Peca) (estoque.MovimentoEstoque, error) {
		return peca.Repor(req.Quantidade, usuarioID, req.Observacao)
	})
}

// AjustarEstoque define o saldo contado no inventário e registra a diferença.
func (s *pecaService) AjustarEstoque(ctx context.Context, pecaUUID uuid.UUID, req dto.AjustarEstoqueRequest, usuarioID int64) (*estoque.Peca, error) {
	if err := estoque.ValidarAjuste(*req.Quantidade); err != nil {
		return nil, err
	}
	peca, err := s.movimentar(ctx, pecaUUID, func(peca *estoque.Peca) (estoque.MovimentoEstoque, error) {
		return peca.Ajustar(*req.Quantidade, usuarioID, req.Observacao)
	})
	if err != nil {
		return nil, err
	}

	// Política: saldo no mínimo ou abaixo dispara o pedido de reposição.
	if peca.EstoqueBaixo() {
		notificarEstoqueBaixo(ctx, s.sender, s.config.AdminNotificationTo, []estoque.Peca{*peca})
	}
	return peca, nil
}

// movimentar carrega a peça travada, aplica o movimento pelo agregado e
// grava saldo e movimento pelo repositório.
func (s *pecaService) movimentar(ctx context.Context, pecaUUID uuid.UUID, mover func(*estoque.Peca) (estoque.MovimentoEstoque, error)) (*estoque.Peca, error) {
	var peca *estoque.Peca
	err := s.uow.Executar(ctx, func(r Repositorios) error {
		var err error
		peca, err = r.Pecas().CarregarParaAtualizar(ctx, pecaUUID)
		if err != nil {
			return err
		}
		movimento, err := mover(peca)
		if err != nil {
			return err
		}
		return r.Pecas().AplicarMovimento(ctx, peca, movimento)
	})
	if err != nil {
		return nil, err
	}
	return peca, nil
}

func (s *pecaService) ListarMovimentos(ctx context.Context, pecaUUID uuid.UUID, paginacao dto.PaginacaoQuery) ([]db.MovimentoEstoque, error) {
	peca, err := s.uow.Leitura().Pecas().Carregar(ctx, pecaUUID)
	if err != nil {
		return nil, err
	}

	return s.store.ListMovimentosEstoqueByPeca(ctx, db.ListMovimentosEstoqueByPecaParams{
		PecaID: peca.ID,
		Limit:  paginacao.Limite(),
		Offset: paginacao.Deslocamento(),
	})
}

// notificarEstoqueBaixo envia o pedido de reposição ao administrador. Falha de
// envio só é registrada no log: a operação de estoque já foi concluída.
func notificarEstoqueBaixo(ctx context.Context, sender notification.Sender, to string, pecas []estoque.Peca) {
	for _, peca := range pecas {
		err := sender.Send(ctx, notification.Message{
			To:      to,
			Subject: "Pedido de reposição de estoque",
			Body: fmt.Sprintf("%s (%s) está com saldo %d %s, abaixo ou igual ao mínimo de %d.",
				peca.Nome, peca.Codigo, peca.Saldo, peca.Unidade, peca.EstoqueMinimo),
		})
		if err != nil {
			log.Printf("cannot send low stock notification: %v", err)
		}
	}
}
