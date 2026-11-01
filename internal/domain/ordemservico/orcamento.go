package ordemservico

// Item é a linha de serviço ou peça usada no cálculo do orçamento, com o
// valor unitário congelado no momento em que entrou na OS.
type Item struct {
	Quantidade            int32
	ValorUnitarioCentavos int64
	// StatusReparo vem preenchido só quando o item pertence a um reparo
	// adicional; itens de reparo pendente ou recusado não entram no total.
	StatusReparo StatusReparo
}

// Subtotal é quantidade x valor unitário.
func (i Item) Subtotal() int64 {
	return int64(i.Quantidade) * i.ValorUnitarioCentavos
}

// EntraNoOrcamento indica se o item conta no valor total.
func (i Item) EntraNoOrcamento() bool {
	return i.StatusReparo == "" || i.StatusReparo == ReparoAutorizado
}

// Orcamento é o objeto de valor com o valor da OS: soma dos serviços e das peças.
type Orcamento struct {
	TotalServicosCentavos int64 `json:"total_servicos_centavos"`
	TotalPecasCentavos    int64 `json:"total_pecas_centavos"`
	TotalCentavos         int64 `json:"total_centavos"`
}

// CalcularOrcamento soma os itens que entram no orçamento.
func CalcularOrcamento(servicos []Item, pecas []Item) Orcamento {
	var orcamento Orcamento
	for _, item := range servicos {
		if item.EntraNoOrcamento() {
			orcamento.TotalServicosCentavos += item.Subtotal()
		}
	}
	for _, item := range pecas {
		if item.EntraNoOrcamento() {
			orcamento.TotalPecasCentavos += item.Subtotal()
		}
	}
	orcamento.TotalCentavos = orcamento.TotalServicosCentavos + orcamento.TotalPecasCentavos
	return orcamento
}

// QuantidadeItens conta os itens que entram no orçamento.
func QuantidadeItens(servicos []Item, pecas []Item) int {
	total := 0
	for _, item := range append(append([]Item{}, servicos...), pecas...) {
		if item.EntraNoOrcamento() {
			total++
		}
	}
	return total
}
