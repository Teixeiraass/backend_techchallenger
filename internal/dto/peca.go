package dto

import (
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/google/uuid"
)

type CriarPecaRequest struct {
	Codigo            string `json:"codigo" binding:"required,max=40" example:"FLT-OLEO-02"`
	Nome              string `json:"nome" binding:"required,max=120" example:"Filtro de ar"`
	Descricao         string `json:"descricao"`
	Tipo              string `json:"tipo" binding:"required,oneof=PECA INSUMO" example:"PECA"`
	Unidade           string `json:"unidade" binding:"max=10" example:"UN"`
	PrecoCentavos     int64  `json:"preco_centavos" binding:"gte=0" example:"4500"`
	QuantidadeEstoque int32  `json:"quantidade_estoque" binding:"gte=0" example:"10"`
	EstoqueMinimo     int32  `json:"estoque_minimo" binding:"gte=0" example:"2"`
}

type AtualizarPecaRequest struct {
	Codigo        string `json:"codigo" binding:"required,max=40"`
	Nome          string `json:"nome" binding:"required,max=120"`
	Descricao     string `json:"descricao"`
	Tipo          string `json:"tipo" binding:"required,oneof=PECA INSUMO"`
	Unidade       string `json:"unidade" binding:"max=10"`
	PrecoCentavos int64  `json:"preco_centavos" binding:"gte=0"`
	EstoqueMinimo int32  `json:"estoque_minimo" binding:"gte=0"`
	Ativo         *bool  `json:"ativo" binding:"required"`
}

type ReporEstoqueRequest struct {
	Quantidade int32  `json:"quantidade" binding:"required,min=1" example:"10"`
	Observacao string `json:"observacao" example:"Nota fiscal 123"`
}

type AjustarEstoqueRequest struct {
	// Novo saldo contado no inventário.
	Quantidade *int32 `json:"quantidade" binding:"required,gte=0" example:"7"`
	Observacao string `json:"observacao" binding:"required" example:"Inventário mensal"`
}

type ListPecasQuery struct {
	PaginacaoQuery
	Busca        string `form:"busca"`
	Tipo         string `form:"tipo" binding:"omitempty,oneof=PECA INSUMO"`
	ApenasAtivas bool   `form:"apenas_ativas"`
}

type PecaResponse struct {
	Uuid              uuid.UUID `json:"uuid"`
	Codigo            string    `json:"codigo"`
	Nome              string    `json:"nome"`
	Descricao         string    `json:"descricao"`
	Tipo              string    `json:"tipo"`
	Unidade           string    `json:"unidade"`
	PrecoCentavos     int64     `json:"preco_centavos"`
	QuantidadeEstoque int32     `json:"quantidade_estoque"`
	EstoqueMinimo     int32     `json:"estoque_minimo"`
	EstoqueBaixo      bool      `json:"estoque_baixo"`
	Ativo             bool      `json:"ativo"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewPecaResponse(p db.Peca) PecaResponse {
	return PecaResponse{
		Uuid:              p.Uuid,
		Codigo:            p.Codigo,
		Nome:              p.Nome,
		Descricao:         p.Descricao,
		Tipo:              p.Tipo,
		Unidade:           p.Unidade,
		PrecoCentavos:     p.PrecoCentavos,
		QuantidadeEstoque: p.QuantidadeEstoque,
		EstoqueMinimo:     p.EstoqueMinimo,
		EstoqueBaixo:      estoque.EstoqueBaixo(p.QuantidadeEstoque, p.EstoqueMinimo),
		Ativo:             p.Ativo,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

type MovimentoEstoqueResponse struct {
	Uuid           uuid.UUID `json:"uuid"`
	Tipo           string    `json:"tipo"`
	Quantidade     int32     `json:"quantidade"`
	SaldoAnterior  int32     `json:"saldo_anterior"`
	SaldoPosterior int32     `json:"saldo_posterior"`
	Observacao     string    `json:"observacao"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewMovimentoEstoqueResponse(m db.MovimentoEstoque) MovimentoEstoqueResponse {
	return MovimentoEstoqueResponse{
		Uuid:           m.Uuid,
		Tipo:           m.Tipo,
		Quantidade:     m.Quantidade,
		SaldoAnterior:  m.SaldoAnterior,
		SaldoPosterior: m.SaldoPosterior,
		Observacao:     m.Observacao,
		CreatedAt:      m.CreatedAt,
	}
}
