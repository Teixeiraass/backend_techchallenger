package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/veiculo"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/google/uuid"
)

type VeiculoService interface {
	CriarVeiculo(ctx context.Context, req dto.CriarVeiculoRequest) (*dto.VeiculoResponse, error)
	BuscarVeiculo(ctx context.Context, veiculoUUID uuid.UUID) (*dto.VeiculoResponse, error)
	ListarVeiculos(ctx context.Context, query dto.ListVeiculosQuery) ([]dto.VeiculoResponse, error)
	AtualizarVeiculo(ctx context.Context, veiculoUUID uuid.UUID, req dto.AtualizarVeiculoRequest) (*dto.VeiculoResponse, error)
	RemoverVeiculo(ctx context.Context, veiculoUUID uuid.UUID) error
}

type veiculoService struct {
	store db.Store
	now   func() time.Time
}

func NewVeiculoService(store db.Store) VeiculoService {
	return &veiculoService{store: store, now: time.Now}
}

func (s *veiculoService) CriarVeiculo(ctx context.Context, req dto.CriarVeiculoRequest) (*dto.VeiculoResponse, error) {
	placa, err := veiculo.NovaPlaca(req.Placa)
	if err != nil {
		return nil, err
	}
	if err := veiculo.ValidarAno(req.Ano, s.now()); err != nil {
		return nil, err
	}

	dono, err := s.store.GetCliente(ctx, req.ClienteUuid)
	if err != nil {
		return nil, notFound(err, ErrClienteNaoEncontrado)
	}

	v, err := s.store.CreateVeiculo(ctx, db.CreateVeiculoParams{
		ClienteID: dono.ID,
		Placa:     placa.String(),
		Marca:     strings.TrimSpace(req.Marca),
		Modelo:    strings.TrimSpace(req.Modelo),
		Ano:       req.Ano,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPlacaDuplicada
		}
		return nil, err
	}

	response := dto.NewVeiculoResponse(v, dono.Uuid)
	return &response, nil
}

func (s *veiculoService) BuscarVeiculo(ctx context.Context, veiculoUUID uuid.UUID) (*dto.VeiculoResponse, error) {
	row, err := s.store.GetVeiculo(ctx, veiculoUUID)
	if err != nil {
		return nil, notFound(err, ErrVeiculoNaoEncontrado)
	}

	response := dto.NewVeiculoResponse(row.Veiculo, row.ClienteUuid)
	return &response, nil
}

func (s *veiculoService) ListarVeiculos(ctx context.Context, query dto.ListVeiculosQuery) ([]dto.VeiculoResponse, error) {
	var clienteID sql.NullInt64
	if query.ClienteUuid != "" {
		clienteUUID, err := uuid.Parse(query.ClienteUuid)
		if err != nil {
			return nil, ErrClienteNaoEncontrado
		}
		dono, err := s.store.GetCliente(ctx, clienteUUID)
		if err != nil {
			return nil, notFound(err, ErrClienteNaoEncontrado)
		}
		clienteID = sql.NullInt64{Int64: dono.ID, Valid: true}
	}

	rows, err := s.store.ListVeiculos(ctx, db.ListVeiculosParams{
		ClienteID:    clienteID,
		Limite:       query.Limite(),
		Deslocamento: query.Deslocamento(),
	})
	if err != nil {
		return nil, err
	}

	response := make([]dto.VeiculoResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, dto.NewVeiculoResponse(row.Veiculo, row.ClienteUuid))
	}
	return response, nil
}

func (s *veiculoService) AtualizarVeiculo(ctx context.Context, veiculoUUID uuid.UUID, req dto.AtualizarVeiculoRequest) (*dto.VeiculoResponse, error) {
	placa, err := veiculo.NovaPlaca(req.Placa)
	if err != nil {
		return nil, err
	}
	if err := veiculo.ValidarAno(req.Ano, s.now()); err != nil {
		return nil, err
	}

	atual, err := s.store.GetVeiculo(ctx, veiculoUUID)
	if err != nil {
		return nil, notFound(err, ErrVeiculoNaoEncontrado)
	}

	v, err := s.store.UpdateVeiculo(ctx, db.UpdateVeiculoParams{
		Uuid:   veiculoUUID,
		Placa:  placa.String(),
		Marca:  strings.TrimSpace(req.Marca),
		Modelo: strings.TrimSpace(req.Modelo),
		Ano:    req.Ano,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPlacaDuplicada
		}
		return nil, notFound(err, ErrVeiculoNaoEncontrado)
	}

	response := dto.NewVeiculoResponse(v, atual.ClienteUuid)
	return &response, nil
}

func (s *veiculoService) RemoverVeiculo(ctx context.Context, veiculoUUID uuid.UUID) error {
	_, err := s.store.DeleteVeiculo(ctx, veiculoUUID)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrVeiculoEmUso
		}
		return notFound(err, ErrVeiculoNaoEncontrado)
	}
	return nil
}
