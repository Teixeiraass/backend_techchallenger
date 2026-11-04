package estoque

import "time"

// Nomes dos eventos de estoque, iguais aos da linguagem ubíqua (§2).
const (
	NomeEstoqueBaixado               = "Estoque baixado"
	NomeEstoqueBaixoDetectado        = "Estoque baixo detectado"
	NomeEstoqueInsuficienteDetectado = "Estoque insuficiente detectado"
)

// EventoPeca é a parte comum dos eventos de estoque: qual peça e quando.
type EventoPeca struct {
	PecaID   int64
	Codigo   string
	NomePeca string
	Unidade  string
	Em       time.Time
}

func (EstoqueBaixado) Nome() string               { return NomeEstoqueBaixado }
func (EstoqueBaixoDetectado) Nome() string        { return NomeEstoqueBaixoDetectado }
func (EstoqueInsuficienteDetectado) Nome() string { return NomeEstoqueInsuficienteDetectado }
func (e EventoPeca) OcorridoEm() time.Time        { return e.Em }

// EstoqueBaixado: saída de peças para uma OS.
type EstoqueBaixado struct {
	EventoPeca
	OrdemServicoID int64
	Quantidade     int32
	Saldo          int32
}

// EstoqueBaixoDetectado: o saldo chegou ao estoque mínimo ou abaixo.
type EstoqueBaixoDetectado struct {
	EventoPeca
	Saldo         int32
	EstoqueMinimo int32
}

// EstoqueInsuficienteDetectado: uma OS pediu mais do que há em estoque.
type EstoqueInsuficienteDetectado struct {
	EventoPeca
	OrdemServicoID int64
	Necessario     int32
	Saldo          int32
}

func (p *Peca) eventoPeca() EventoPeca {
	return EventoPeca{PecaID: p.ID, Codigo: p.Codigo, NomePeca: p.Nome, Unidade: p.Unidade, Em: time.Now()}
}
