package ordemservico

import (
	"fmt"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
)

var (
	ErrTransicaoInvalida      = domain.NewConflictError("TRANSICAO_INVALIDA", "transição de status inválida")
	ErrItensNaoEditaveis      = domain.NewConflictError("ITENS_NAO_EDITAVEIS", "os itens da OS só podem ser alterados nos status Recebida ou Em diagnóstico")
	ErrOrcamentoSemItens      = domain.NewConflictError("ORCAMENTO_SEM_ITENS", "inclua ao menos um serviço ou peça antes de enviar o orçamento")
	ErrOrcamentoNaoAguardando = domain.NewConflictError("ORCAMENTO_NAO_AGUARDANDO", "não há orçamento aguardando decisão do cliente")
	ErrOrcamentoNaoAprovado   = domain.NewConflictError("ORCAMENTO_NAO_APROVADO", "a execução só começa com o orçamento aprovado")
	ErrReparoForaDeExecucao   = domain.NewConflictError("REPARO_FORA_DE_EXECUCAO", "reparo adicional só pode ser registrado com a OS Em execução")
	ErrReparoSemItens         = domain.NewValidationError("REPARO_SEM_ITENS", "o reparo adicional precisa de ao menos um serviço ou peça")
	ErrReparoJaDecidido       = domain.NewConflictError("REPARO_JA_DECIDIDO", "o reparo adicional já foi decidido pelo cliente")
	ErrReparoPendente         = domain.NewConflictError("REPARO_PENDENTE", "há reparo adicional aguardando autorização do cliente")
	ErrEntregaNaoPermitida    = domain.NewConflictError("ENTREGA_NAO_PERMITIDA", "o veículo só pode ser entregue com a OS Finalizada ou com o orçamento rejeitado")
	ErrQuantidadeInvalida     = domain.NewValidationError("QUANTIDADE_INVALIDA", "a quantidade deve ser maior que zero")
)

// OrdemServico é a raiz do agregado. Guarda o estado que muda ao longo do
// ciclo de vida; itens e reparos ficam em entidades próprias e chegam aqui
// já convertidos em Item para o cálculo do orçamento.
type OrdemServico struct {
	Status                Status
	StatusOrcamento       StatusOrcamento
	ValorTotalCentavos    int64
	Diagnostico           string
	OrcamentoEnviadoEm    *time.Time
	OrcamentoDecididoEm   *time.Time
	DiagnosticoIniciadoEm *time.Time
	ExecucaoIniciadaEm    *time.Time
	FinalizadaEm          *time.Time
	EntregueEm            *time.Time
}

// Nova cria a OS no status inicial Recebida.
func Nova() OrdemServico {
	return OrdemServico{
		Status:          StatusRecebida,
		StatusOrcamento: OrcamentoNaoEnviado,
	}
}

func (os *OrdemServico) transitar(destino Status) error {
	if !os.Status.PodeIrPara(destino) {
		return domain.NewConflictError(
			ErrTransicaoInvalida.Code,
			fmt.Sprintf("transição de status inválida: %s -> %s", os.Status, destino),
		)
	}
	os.Status = destino
	return nil
}

// IniciarDiagnostico: Recebida -> Em diagnóstico (mecânico começa a avaliar).
func (os *OrdemServico) IniciarDiagnostico(agora time.Time) error {
	// Aguardando aprovação -> Em diagnóstico só acontece pela rejeição do
	// cliente (RejeitarOrcamento), nunca por este comando.
	if os.Status != StatusRecebida {
		return domain.NewConflictError(
			ErrTransicaoInvalida.Code,
			fmt.Sprintf("transição de status inválida: %s -> %s", os.Status, StatusEmDiagnostico),
		)
	}
	os.Status = StatusEmDiagnostico
	os.DiagnosticoIniciadoEm = &agora
	return nil
}

// PodeAlterarItens: serviços e peças só mudam antes do envio do orçamento.
func (os *OrdemServico) PodeAlterarItens() error {
	if os.Status != StatusRecebida && os.Status != StatusEmDiagnostico {
		return ErrItensNaoEditaveis
	}
	return nil
}

// AtualizarOrcamento recalcula o valor total sempre que os itens mudam.
func (os *OrdemServico) AtualizarOrcamento(servicos []Item, pecas []Item) Orcamento {
	orcamento := CalcularOrcamento(servicos, pecas)
	os.ValorTotalCentavos = orcamento.TotalCentavos
	return orcamento
}

// EnviarOrcamento: Em diagnóstico -> Aguardando aprovação. O orçamento é
// recalculado no envio e precisa de ao menos um item.
func (os *OrdemServico) EnviarOrcamento(servicos []Item, pecas []Item, diagnostico string, agora time.Time) error {
	if QuantidadeItens(servicos, pecas) == 0 {
		return ErrOrcamentoSemItens
	}
	if err := os.transitar(StatusAguardandoAprovacao); err != nil {
		return err
	}
	os.AtualizarOrcamento(servicos, pecas)
	if diagnostico != "" {
		os.Diagnostico = diagnostico
	}
	os.StatusOrcamento = OrcamentoAguardando
	os.OrcamentoEnviadoEm = &agora
	os.OrcamentoDecididoEm = nil
	return nil
}

// AprovarOrcamento registra a aprovação do cliente. A OS continua em
// Aguardando aprovação até IniciarExecucao, que o sistema chama em seguida.
func (os *OrdemServico) AprovarOrcamento(agora time.Time) error {
	if os.Status != StatusAguardandoAprovacao || os.StatusOrcamento != OrcamentoAguardando {
		return ErrOrcamentoNaoAguardando
	}
	os.StatusOrcamento = OrcamentoAprovado
	os.OrcamentoDecididoEm = &agora
	return nil
}

// RejeitarOrcamento: Aguardando aprovação -> Em diagnóstico. A OS não segue
// para execução; a oficina pode montar um novo orçamento ou devolver o veículo.
func (os *OrdemServico) RejeitarOrcamento(agora time.Time) error {
	if os.Status != StatusAguardandoAprovacao || os.StatusOrcamento != OrcamentoAguardando {
		return ErrOrcamentoNaoAguardando
	}
	if err := os.transitar(StatusEmDiagnostico); err != nil {
		return err
	}
	os.StatusOrcamento = OrcamentoRejeitado
	os.OrcamentoDecididoEm = &agora
	return nil
}

// IniciarExecucao: Aguardando aprovação -> Em execução, só com orçamento aprovado.
func (os *OrdemServico) IniciarExecucao(agora time.Time) error {
	if os.StatusOrcamento != OrcamentoAprovado {
		return ErrOrcamentoNaoAprovado
	}
	if err := os.transitar(StatusEmExecucao); err != nil {
		return err
	}
	os.ExecucaoIniciadaEm = &agora
	return nil
}

// PodeRegistrarReparoAdicional: reparo adicional só durante a execução.
func (os *OrdemServico) PodeRegistrarReparoAdicional() error {
	if os.Status != StatusEmExecucao {
		return ErrReparoForaDeExecucao
	}
	return nil
}

// Finalizar: Em execução -> Finalizada. Bloqueia se houver reparo adicional
// aguardando o cliente; o tempo de execução fica registrado.
func (os *OrdemServico) Finalizar(reparosPendentes int, agora time.Time) error {
	if os.Status == StatusEmExecucao && reparosPendentes > 0 {
		return ErrReparoPendente
	}
	if err := os.transitar(StatusFinalizada); err != nil {
		return err
	}
	os.FinalizadaEm = &agora
	return nil
}

// Entregar: Finalizada -> Entregue, ou Em diagnóstico -> Entregue quando o
// cliente rejeitou o orçamento e o veículo é devolvido sem execução.
func (os *OrdemServico) Entregar(agora time.Time) error {
	if os.Status == StatusEmDiagnostico && os.StatusOrcamento != OrcamentoRejeitado {
		return ErrEntregaNaoPermitida
	}
	if !os.Status.PodeIrPara(StatusEntregue) {
		return ErrEntregaNaoPermitida
	}
	os.Status = StatusEntregue
	os.EntregueEm = &agora
	return nil
}

// TempoExecucao é o período entre o início da execução e a finalização.
func (os *OrdemServico) TempoExecucao() (time.Duration, bool) {
	if os.ExecucaoIniciadaEm == nil || os.FinalizadaEm == nil {
		return 0, false
	}
	return os.FinalizadaEm.Sub(*os.ExecucaoIniciadaEm), true
}

// ValidarQuantidade garante quantidade positiva em itens da OS.
func ValidarQuantidade(quantidade int32) error {
	if quantidade <= 0 {
		return ErrQuantidadeInvalida
	}
	return nil
}

// DecidirReparo valida a decisão do cliente sobre um reparo adicional.
func DecidirReparo(atual StatusReparo, statusOS Status) error {
	if atual != ReparoPendente {
		return ErrReparoJaDecidido
	}
	if statusOS != StatusEmExecucao {
		return ErrReparoForaDeExecucao
	}
	return nil
}
