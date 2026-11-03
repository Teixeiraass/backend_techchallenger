// Package ordemservico contém o agregado OrdemServico: ciclo de vida
// (status e transições), orçamento e tempo de execução.
package ordemservico

// Status é a etapa em que a OS está. Os seis valores são os do enunciado.
type Status string

const (
	StatusRecebida            Status = "RECEBIDA"
	StatusEmDiagnostico       Status = "EM_DIAGNOSTICO"
	StatusAguardandoAprovacao Status = "AGUARDANDO_APROVACAO"
	StatusEmExecucao          Status = "EM_EXECUCAO"
	StatusFinalizada          Status = "FINALIZADA"
	StatusEntregue            Status = "ENTREGUE"
)

// StatusOrcamento acompanha a decisão do cliente sobre o orçamento.
type StatusOrcamento string

const (
	OrcamentoNaoEnviado StatusOrcamento = "NAO_ENVIADO"
	OrcamentoAguardando StatusOrcamento = "AGUARDANDO"
	OrcamentoAprovado   StatusOrcamento = "APROVADO"
	OrcamentoRejeitado  StatusOrcamento = "REJEITADO"
)

// StatusReparo é a decisão do cliente sobre um reparo adicional.
type StatusReparo string

const (
	ReparoPendente   StatusReparo = "PENDENTE"
	ReparoAutorizado StatusReparo = "AUTORIZADO"
	ReparoRecusado   StatusReparo = "RECUSADO"
)

// transicoes lista para onde cada status pode ir. O fluxo normal só avança;
// a única volta é Aguardando aprovação -> Em diagnóstico quando o cliente
// rejeita o orçamento, e a única saída antecipada é Em diagnóstico -> Entregue
// para entregar o veículo sem execução depois de um orçamento rejeitado.
var transicoes = map[Status][]Status{
	StatusRecebida:            {StatusEmDiagnostico},
	StatusEmDiagnostico:       {StatusAguardandoAprovacao, StatusEntregue},
	StatusAguardandoAprovacao: {StatusEmExecucao, StatusEmDiagnostico},
	StatusEmExecucao:          {StatusFinalizada},
	StatusFinalizada:          {StatusEntregue},
	StatusEntregue:            {},
}

// Todos devolve os status na ordem do ciclo de vida.
func Todos() []Status {
	return []Status{
		StatusRecebida,
		StatusEmDiagnostico,
		StatusAguardandoAprovacao,
		StatusEmExecucao,
		StatusFinalizada,
		StatusEntregue,
	}
}

// Valido indica se o valor é um dos seis status conhecidos.
func (s Status) Valido() bool {
	_, ok := transicoes[s]
	return ok
}

// PodeIrPara indica se a transição de s para destino é permitida.
func (s Status) PodeIrPara(destino Status) bool {
	for _, permitido := range transicoes[s] {
		if permitido == destino {
			return true
		}
	}
	return false
}

// Descricao devolve o nome do status como aparece para o cliente.
func (s Status) Descricao() string {
	switch s {
	case StatusRecebida:
		return "Recebida"
	case StatusEmDiagnostico:
		return "Em diagnóstico"
	case StatusAguardandoAprovacao:
		return "Aguardando aprovação"
	case StatusEmExecucao:
		return "Em execução"
	case StatusFinalizada:
		return "Finalizada"
	case StatusEntregue:
		return "Entregue"
	default:
		return string(s)
	}
}
