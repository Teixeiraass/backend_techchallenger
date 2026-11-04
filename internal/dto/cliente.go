package dto

import (
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/google/uuid"
)

type CriarClienteRequest struct {
	Nome      string `json:"nome" binding:"required,max=150" example:"Maria Silva"`
	Documento string `json:"documento" binding:"required,max=18" example:"529.982.247-25"`
	Email     string `json:"email" binding:"required,email,max=150" example:"maria@email.com"`
	Telefone  string `json:"telefone" binding:"max=20" example:"11999990000"`
}

type AtualizarClienteRequest struct {
	Nome     string `json:"nome" binding:"required,max=150"`
	Email    string `json:"email" binding:"required,email,max=150"`
	Telefone string `json:"telefone" binding:"max=20"`
}

type ListClientesQuery struct {
	PaginacaoQuery
	Busca string `form:"busca"`
}

type ClienteResponse struct {
	Uuid               uuid.UUID `json:"uuid"`
	Nome               string    `json:"nome"`
	Documento          string    `json:"documento"`
	DocumentoFormatado string    `json:"documento_formatado"`
	TipoDocumento      string    `json:"tipo_documento"`
	Email              string    `json:"email"`
	Telefone           string    `json:"telefone"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func NewClienteResponse(c readmodel.Cliente) ClienteResponse {
	formatado := c.Documento
	if documento, err := cliente.NovoDocumento(c.Documento); err == nil {
		formatado = documento.Formatado()
	}

	return ClienteResponse{
		Uuid:               c.UUID,
		Nome:               c.Nome,
		Documento:          c.Documento,
		DocumentoFormatado: formatado,
		TipoDocumento:      c.TipoDocumento,
		Email:              c.Email,
		Telefone:           c.Telefone,
		CreatedAt:          c.CriadoEm,
		UpdatedAt:          c.AtualizadoEm,
	}
}
