package service

import (
	"context"
	"strings"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/google/uuid"
)

// ServicoService gerencia o catálogo de serviços oferecidos pela oficina.
type ServicoService interface {
	CriarServico(ctx context.Context, req dto.CriarServicoRequest) (*db.Servico, error)
	BuscarServico(ctx context.Context, servicoUUID uuid.UUID) (*db.Servico, error)
	ListarServicos(ctx context.Context, query dto.ListServicosQuery) ([]db.Servico, error)
	AtualizarServico(ctx context.Context, servicoUUID uuid.UUID, req dto.AtualizarServicoRequest) (*db.Servico, error)
	InativarServico(ctx context.Context, servicoUUID uuid.UUID) (*db.Servico, error)
}

type servicoService struct {
	store db.Store
}

func NewServicoService(store db.Store) ServicoService {
	return &servicoService{store: store}
}

func (s *servicoService) CriarServico(ctx context.Context, req dto.CriarServicoRequest) (*db.Servico, error) {
	servico, err := s.store.CreateServico(ctx, db.CreateServicoParams{
		Nome:                 strings.TrimSpace(req.Nome),
		Descricao:            req.Descricao,
		PrecoCentavos:        req.PrecoCentavos,
		TempoEstimadoMinutos: req.TempoEstimadoMinutos,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrServicoDuplicado
		}
		return nil, err
	}
	return &servico, nil
}

func (s *servicoService) BuscarServico(ctx context.Context, servicoUUID uuid.UUID) (*db.Servico, error) {
	servico, err := s.store.GetServico(ctx, servicoUUID)
	if err != nil {
		return nil, notFound(err, ErrServicoNaoEncontrado)
	}
	return &servico, nil
}

func (s *servicoService) ListarServicos(ctx context.Context, query dto.ListServicosQuery) ([]db.Servico, error) {
	return s.store.ListServicos(ctx, db.ListServicosParams{
		ApenasAtivos: query.ApenasAtivos,
		Busca:        query.Busca,
		Limite:       query.Limite(),
		Deslocamento: query.Deslocamento(),
	})
}

func (s *servicoService) AtualizarServico(ctx context.Context, servicoUUID uuid.UUID, req dto.AtualizarServicoRequest) (*db.Servico, error) {
	servico, err := s.store.UpdateServico(ctx, db.UpdateServicoParams{
		Uuid:                 servicoUUID,
		Nome:                 strings.TrimSpace(req.Nome),
		Descricao:            req.Descricao,
		PrecoCentavos:        req.PrecoCentavos,
		TempoEstimadoMinutos: req.TempoEstimadoMinutos,
		Ativo:                *req.Ativo,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrServicoDuplicado
		}
		return nil, notFound(err, ErrServicoNaoEncontrado)
	}
	return &servico, nil
}

// InativarServico remove o serviço do catálogo sem apagar o histórico das OS
// que já o usaram: serviço inativo não entra em OS nova.
func (s *servicoService) InativarServico(ctx context.Context, servicoUUID uuid.UUID) (*db.Servico, error) {
	servico, err := s.store.InativarServico(ctx, servicoUUID)
	if err != nil {
		return nil, notFound(err, ErrServicoNaoEncontrado)
	}
	return &servico, nil
}
