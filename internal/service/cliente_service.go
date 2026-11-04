package service

import (
	"context"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/google/uuid"
)

type ClienteService interface {
	CriarCliente(ctx context.Context, req dto.CriarClienteRequest) (*db.Cliente, error)
	BuscarCliente(ctx context.Context, clienteUUID uuid.UUID) (*db.Cliente, error)
	BuscarClientePorDocumento(ctx context.Context, documento string) (*db.Cliente, error)
	ListarClientes(ctx context.Context, query dto.ListClientesQuery) ([]db.Cliente, error)
	AtualizarCliente(ctx context.Context, clienteUUID uuid.UUID, req dto.AtualizarClienteRequest) (*db.Cliente, error)
	RemoverCliente(ctx context.Context, clienteUUID uuid.UUID) error
}

type clienteService struct {
	store db.Store
}

func NewClienteService(store db.Store) ClienteService {
	return &clienteService{store: store}
}

func (s *clienteService) CriarCliente(ctx context.Context, req dto.CriarClienteRequest) (*db.Cliente, error) {
	documento, err := cliente.NovoDocumento(req.Documento)
	if err != nil {
		return nil, err
	}

	c, err := s.store.CreateCliente(ctx, db.CreateClienteParams{
		Nome:          req.Nome,
		Documento:     documento.Numero(),
		TipoDocumento: string(documento.Tipo()),
		Email:         req.Email,
		Telefone:      req.Telefone,
	})
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrDocumentoDuplicado
		}
		return nil, err
	}
	return &c, nil
}

func (s *clienteService) BuscarCliente(ctx context.Context, clienteUUID uuid.UUID) (*db.Cliente, error) {
	c, err := s.store.GetCliente(ctx, clienteUUID)
	if err != nil {
		return nil, notFound(err, ErrClienteNaoEncontrado)
	}
	return &c, nil
}

func (s *clienteService) BuscarClientePorDocumento(ctx context.Context, valor string) (*db.Cliente, error) {
	documento, err := cliente.NovoDocumento(valor)
	if err != nil {
		return nil, err
	}

	c, err := s.store.GetClienteByDocumento(ctx, documento.Numero())
	if err != nil {
		return nil, notFound(err, ErrClienteNaoEncontrado)
	}
	return &c, nil
}

func (s *clienteService) ListarClientes(ctx context.Context, query dto.ListClientesQuery) ([]db.Cliente, error) {
	return s.store.ListClientes(ctx, db.ListClientesParams{
		Busca:        query.Busca,
		Limite:       query.Limite(),
		Deslocamento: query.Deslocamento(),
	})
}

func (s *clienteService) AtualizarCliente(ctx context.Context, clienteUUID uuid.UUID, req dto.AtualizarClienteRequest) (*db.Cliente, error) {
	c, err := s.store.UpdateCliente(ctx, db.UpdateClienteParams{
		Uuid:     clienteUUID,
		Nome:     req.Nome,
		Email:    req.Email,
		Telefone: req.Telefone,
	})
	if err != nil {
		return nil, notFound(err, ErrClienteNaoEncontrado)
	}
	return &c, nil
}

func (s *clienteService) RemoverCliente(ctx context.Context, clienteUUID uuid.UUID) error {
	_, err := s.store.DeleteCliente(ctx, clienteUUID)
	if err != nil {
		if IsForeignKeyViolation(err) {
			return ErrClienteEmUso
		}
		return notFound(err, ErrClienteNaoEncontrado)
	}
	return nil
}
