package dto

import (
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/google/uuid"
)

type CriarServicoRequest struct {
	Nome                 string `json:"nome" binding:"required,max=120" example:"Troca de óleo"`
	Descricao            string `json:"descricao" example:"Troca do óleo do motor"`
	PrecoCentavos        int64  `json:"preco_centavos" binding:"gte=0" example:"8000"`
	TempoEstimadoMinutos int32  `json:"tempo_estimado_minutos" binding:"gte=0" example:"30"`
}

type AtualizarServicoRequest struct {
	Nome                 string `json:"nome" binding:"required,max=120"`
	Descricao            string `json:"descricao"`
	PrecoCentavos        int64  `json:"preco_centavos" binding:"gte=0"`
	TempoEstimadoMinutos int32  `json:"tempo_estimado_minutos" binding:"gte=0"`
	Ativo                *bool  `json:"ativo" binding:"required"`
}

type ListServicosQuery struct {
	PaginacaoQuery
	Busca        string `form:"busca"`
	ApenasAtivos bool   `form:"apenas_ativos"`
}

type ServicoResponse struct {
	Uuid                 uuid.UUID `json:"uuid"`
	Nome                 string    `json:"nome"`
	Descricao            string    `json:"descricao"`
	PrecoCentavos        int64     `json:"preco_centavos"`
	TempoEstimadoMinutos int32     `json:"tempo_estimado_minutos"`
	Ativo                bool      `json:"ativo"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func NewServicoResponse(s db.Servico) ServicoResponse {
	return ServicoResponse{
		Uuid:                 s.Uuid,
		Nome:                 s.Nome,
		Descricao:            s.Descricao,
		PrecoCentavos:        s.PrecoCentavos,
		TempoEstimadoMinutos: s.TempoEstimadoMinutos,
		Ativo:                s.Ativo,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}
