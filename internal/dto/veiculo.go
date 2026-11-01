package dto

import (
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/google/uuid"
)

type CriarVeiculoRequest struct {
	ClienteUuid uuid.UUID `json:"cliente_uuid" binding:"required" swaggertype:"string" format:"uuid"`
	Placa       string    `json:"placa" binding:"required,max=8" example:"ABC1D23"`
	Marca       string    `json:"marca" binding:"required,max=60" example:"Volkswagen"`
	Modelo      string    `json:"modelo" binding:"required,max=80" example:"Gol"`
	Ano         int32     `json:"ano" binding:"required" example:"2019"`
}

type AtualizarVeiculoRequest struct {
	Placa  string `json:"placa" binding:"required,max=8" example:"ABC1D23"`
	Marca  string `json:"marca" binding:"required,max=60"`
	Modelo string `json:"modelo" binding:"required,max=80"`
	Ano    int32  `json:"ano" binding:"required"`
}

type ListVeiculosQuery struct {
	PaginacaoQuery
	ClienteUuid string `form:"cliente_uuid"`
}

type VeiculoResponse struct {
	Uuid        uuid.UUID `json:"uuid"`
	ClienteUuid uuid.UUID `json:"cliente_uuid"`
	Placa       string    `json:"placa"`
	Marca       string    `json:"marca"`
	Modelo      string    `json:"modelo"`
	Ano         int32     `json:"ano"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewVeiculoResponse(v db.Veiculo, clienteUuid uuid.UUID) VeiculoResponse {
	return VeiculoResponse{
		Uuid:        v.Uuid,
		ClienteUuid: clienteUuid,
		Placa:       v.Placa,
		Marca:       v.Marca,
		Modelo:      v.Modelo,
		Ano:         v.Ano,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}
