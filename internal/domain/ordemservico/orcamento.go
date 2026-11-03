package ordemservico

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
