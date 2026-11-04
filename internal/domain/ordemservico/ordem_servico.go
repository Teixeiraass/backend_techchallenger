package ordemservico

import (
	"fmt"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/google/uuid"
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
	ErrReparoNaoEncontrado    = domain.NewNotFoundError("REPARO_NAO_ENCONTRADO", "reparo adicional não encontrado")
	ErrEntregaNaoPermitida    = domain.NewConflictError("ENTREGA_NAO_PERMITIDA", "o veículo só pode ser entregue com a OS Finalizada ou com o orçamento rejeitado")
	ErrQuantidadeInvalida     = domain.NewValidationError("QUANTIDADE_INVALIDA", "a quantidade deve ser maior que zero")
	ErrServicoInativo         = domain.NewConflictError("SERVICO_INATIVO", "só serviços ativos no catálogo podem entrar na OS")
	ErrItemNaoEncontrado      = domain.NewNotFoundError("ITEM_NAO_ENCONTRADO", "item não encontrado na ordem de serviço")
)

// OrdemServico é a raiz do agregado. Guarda a identidade, o cliente e o
// veículo, o ciclo de vida (status e datas), o orçamento e as entidades
// internas: os itens (serviços e peças) e os reparos adicionais. Toda
// mudança de estado passa por um método da raiz, que aplica as invariantes
// e recalcula o orçamento.
type OrdemServico struct {
	domain.Registrador

	// ID é a identidade de persistência (0 = ainda não gravada).
	ID                    int64
	UUID                  uuid.UUID
	ClienteID             int64
	VeiculoID             int64
	DescricaoProblema     string
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
	CriadaEm              time.Time
	AtualizadaEm          time.Time
	Itens                 []Item
	Reparos               []ReparoAdicional

	alteracoes       Alteracoes
	statusAoCarregar Status
}

// Nova cria a OS no status inicial Recebida para o cliente e o veículo.
func Nova(clienteID int64, veiculoID int64, descricaoProblema string) OrdemServico {
	os := OrdemServico{
		ClienteID:         clienteID,
		VeiculoID:         veiculoID,
		DescricaoProblema: descricaoProblema,
		Status:            StatusRecebida,
		StatusOrcamento:   OrcamentoNaoEnviado,
	}
	os.Registrar(OSRecebida{EventoOS: os.eventoOS(time.Now())})
	return os
}

// Eventos devolve os eventos registrados, com o UUID da OS preenchido (uma
// OS nova só ganha UUID ao ser salva, depois de registrar "OS recebida").
func (os *OrdemServico) Eventos() []domain.Evento {
	eventos := os.Registrador.Eventos()
	for i, e := range eventos {
		eventos[i] = comOS(e, os.UUID, os.ClienteID)
	}
	return eventos
}

// Alteracoes devolve o que mudou desde o carregamento e precisa ser persistido.
func (os *OrdemServico) Alteracoes() Alteracoes {
	return os.alteracoes
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

// ---------------------------------------------------------------------------
// Itens e orçamento
// ---------------------------------------------------------------------------

// PodeAlterarItens: serviços e peças só mudam antes do envio do orçamento.
func (os *OrdemServico) PodeAlterarItens() error {
	if os.Status != StatusRecebida && os.Status != StatusEmDiagnostico {
		return ErrItensNaoEditaveis
	}
	return nil
}

// Servicos devolve os itens de serviço; Pecas, os itens de peça.
func (os *OrdemServico) Servicos() []Item { return os.itensDoTipo(ItemServico) }
func (os *OrdemServico) Pecas() []Item    { return os.itensDoTipo(ItemPeca) }

func (os *OrdemServico) itensDoTipo(tipo TipoItem) []Item {
	itens := make([]Item, 0, len(os.Itens))
	for _, item := range os.Itens {
		if item.Tipo == tipo {
			itens = append(itens, item)
		}
	}
	return itens
}

// Orcamento soma os itens que entram no orçamento (serviços e peças do
// orçamento original mais os reparos autorizados).
func (os *OrdemServico) Orcamento() Orcamento {
	return CalcularOrcamento(os.Servicos(), os.Pecas())
}

// recalcular atualiza o valor total sempre que os itens mudam.
func (os *OrdemServico) recalcular() {
	os.ValorTotalCentavos = os.Orcamento().TotalCentavos
}

// AdicionarServico inclui um serviço ativo do catálogo com o preço atual.
// Quantidade zero vale 1.
func (os *OrdemServico) AdicionarServico(servico CatalogoServico, quantidade int32) error {
	if err := os.PodeAlterarItens(); err != nil {
		return err
	}
	item, err := novoItemServico(servico, quantidade)
	if err != nil {
		return err
	}
	os.Itens = append(os.Itens, item)
	os.recalcular()
	return nil
}

// AdicionarPeca inclui a peça se houver saldo para ela, contando o que a OS
// já pediu da mesma peça e ainda não foi baixado. Não há reserva: o saldo é
// conferido de novo na baixa, ao iniciar a execução.
func (os *OrdemServico) AdicionarPeca(peca CatalogoPeca, quantidade int32, saldoDisponivel int32) error {
	if err := os.PodeAlterarItens(); err != nil {
		return err
	}
	item, err := os.novoItemPeca(peca, quantidade, saldoDisponivel)
	if err != nil {
		return err
	}
	os.Itens = append(os.Itens, item)
	os.recalcular()
	return nil
}

// RemoverItem tira um serviço ou peça do orçamento. Itens de reparo
// adicional não são removidos por aqui: o cliente decide sobre o reparo.
func (os *OrdemServico) RemoverItem(itemUUID uuid.UUID) error {
	if err := os.PodeAlterarItens(); err != nil {
		return err
	}
	for i, item := range os.Itens {
		if item.UUID == itemUUID && !item.ReparoUUID.Valid {
			os.Itens = append(os.Itens[:i], os.Itens[i+1:]...)
			os.alteracoes.ItensRemovidos = append(os.alteracoes.ItensRemovidos, item)
			os.recalcular()
			return nil
		}
	}
	return ErrItemNaoEncontrado
}

func novoItemServico(servico CatalogoServico, quantidade int32) (Item, error) {
	if quantidade == 0 {
		quantidade = 1
	}
	if err := ValidarQuantidade(quantidade); err != nil {
		return Item{}, err
	}
	if !servico.Ativo {
		return Item{}, ErrServicoInativo
	}
	return Item{
		Tipo:                  ItemServico,
		CatalogoID:            servico.ID,
		CatalogoUUID:          servico.UUID,
		Nome:                  servico.Nome,
		Quantidade:            quantidade,
		ValorUnitarioCentavos: servico.PrecoCentavos,
	}, nil
}

func (os *OrdemServico) novoItemPeca(peca CatalogoPeca, quantidade int32, saldoDisponivel int32) (Item, error) {
	if err := estoque.ValidarDisponibilidade(saldoDisponivel, os.quantidadeAindaNaoBaixada(peca.ID), quantidade, peca.Ativa); err != nil {
		if err == estoque.ErrEstoqueInsuficiente {
			return Item{}, estoque.ErrEstoqueInsuficientePara(peca.Nome, peca.Codigo)
		}
		return Item{}, err
	}
	return Item{
		Tipo:                  ItemPeca,
		CatalogoID:            peca.ID,
		CatalogoUUID:          peca.UUID,
		Nome:                  peca.Nome,
		Codigo:                peca.Codigo,
		TipoPeca:              peca.Tipo,
		Quantidade:            quantidade,
		ValorUnitarioCentavos: peca.PrecoCentavos,
	}, nil
}

// quantidadeAindaNaoBaixada soma o que a OS já pediu da peça e ainda vai
// sair do estoque: itens do orçamento antes da execução e itens de reparo
// pendente. O que já foi baixado não conta de novo.
func (os *OrdemServico) quantidadeAindaNaoBaixada(pecaID int64) int32 {
	var total int32
	for _, item := range os.Itens {
		if item.Tipo != ItemPeca || item.CatalogoID != pecaID {
			continue
		}
		aindaNaoBaixado := !item.ReparoUUID.Valid && os.Status != StatusEmExecucao
		reparoPendente := item.StatusReparo == ReparoPendente
		if aindaNaoBaixado || reparoPendente {
			total += item.Quantidade
		}
	}
	return total
}

// PecasParaBaixaNaExecucao são as peças do orçamento original (sem reparo),
// baixadas do estoque quando a execução começa.
func (os *OrdemServico) PecasParaBaixaNaExecucao() []Item {
	itens := make([]Item, 0)
	for _, item := range os.Itens {
		if item.Tipo == ItemPeca && !item.ReparoUUID.Valid {
			itens = append(itens, item)
		}
	}
	return itens
}

// PecasDoReparo são as peças de um reparo adicional, baixadas na autorização.
func (os *OrdemServico) PecasDoReparo(reparoUUID uuid.UUID) []Item {
	itens := make([]Item, 0)
	for _, item := range os.Itens {
		if item.Tipo == ItemPeca && item.DoReparo(reparoUUID) {
			itens = append(itens, item)
		}
	}
	return itens
}

// ---------------------------------------------------------------------------
// Ciclo de vida
// ---------------------------------------------------------------------------

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
	os.Registrar(DiagnosticoIniciado{EventoOS: os.eventoOS(agora)})
	return nil
}

// EnviarOrcamento: Em diagnóstico -> Aguardando aprovação. O orçamento é
// recalculado no envio e precisa de ao menos um item.
func (os *OrdemServico) EnviarOrcamento(diagnostico string, agora time.Time) error {
	if QuantidadeItens(os.Servicos(), os.Pecas()) == 0 {
		return ErrOrcamentoSemItens
	}
	if err := os.transitar(StatusAguardandoAprovacao); err != nil {
		return err
	}
	os.recalcular()
	if diagnostico != "" {
		os.Diagnostico = diagnostico
	}
	os.StatusOrcamento = OrcamentoAguardando
	os.OrcamentoEnviadoEm = &agora
	os.OrcamentoDecididoEm = nil
	os.Registrar(OrcamentoEnviadoAoCliente{EventoOS: os.eventoOS(agora), ValorTotalCentavos: os.ValorTotalCentavos})
	return nil
}

// AprovarOrcamento registra a aprovação do cliente. A OS continua em
// Aguardando aprovação até IniciarExecucao, que o sistema chama em seguida.
//
// A operação é idempotente: se o orçamento já estiver aprovado (a OS pode
// estar Aguardando aprovação, por falta de peça, ou já Em execução e além),
// nada muda e nenhum erro é devolvido. Assim o cliente pode repetir a
// aprovação depois de uma falha de rede sem receber conflito. O chamador
// descobre se houve mudança por OrcamentoJaAprovado antes da chamada.
func (os *OrdemServico) AprovarOrcamento(agora time.Time) error {
	if os.OrcamentoJaAprovado() {
		return nil
	}
	if os.Status != StatusAguardandoAprovacao || os.StatusOrcamento != OrcamentoAguardando {
		return ErrOrcamentoNaoAguardando
	}
	os.StatusOrcamento = OrcamentoAprovado
	os.OrcamentoDecididoEm = &agora
	os.Registrar(OrcamentoAprovadoPeloCliente{EventoOS: os.eventoOS(agora)})
	return nil
}

// OrcamentoJaAprovado informa se a aprovação do cliente já está registrada.
func (os *OrdemServico) OrcamentoJaAprovado() bool {
	return os.StatusOrcamento == OrcamentoAprovado
}

// RejeitarOrcamento: Aguardando aprovação -> Em diagnóstico. A OS não segue
// para execução; a oficina pode montar um novo orçamento ou entregar o veículo.
func (os *OrdemServico) RejeitarOrcamento(agora time.Time) error {
	if os.Status != StatusAguardandoAprovacao || os.StatusOrcamento != OrcamentoAguardando {
		return ErrOrcamentoNaoAguardando
	}
	if err := os.transitar(StatusEmDiagnostico); err != nil {
		return err
	}
	os.StatusOrcamento = OrcamentoRejeitado
	os.OrcamentoDecididoEm = &agora
	os.Registrar(OrcamentoRejeitadoPeloCliente{EventoOS: os.eventoOS(agora)})
	return nil
}

// IniciarExecucao: Aguardando aprovação -> Em execução, só com orçamento
// aprovado. A baixa das peças (PecasParaBaixaNaExecucao) acontece na mesma
// transação, pelo contexto de estoque.
func (os *OrdemServico) IniciarExecucao(agora time.Time) error {
	if os.StatusOrcamento != OrcamentoAprovado {
		return ErrOrcamentoNaoAprovado
	}
	if err := os.transitar(StatusEmExecucao); err != nil {
		return err
	}
	os.ExecucaoIniciadaEm = &agora
	os.Registrar(ExecucaoIniciada{EventoOS: os.eventoOS(agora)})
	return nil
}

// Finalizar: Em execução -> Finalizada. Bloqueia se houver reparo adicional
// aguardando o cliente; o tempo de execução fica registrado.
func (os *OrdemServico) Finalizar(agora time.Time) error {
	if os.Status == StatusEmExecucao && os.ReparosPendentes() > 0 {
		return ErrReparoPendente
	}
	if err := os.transitar(StatusFinalizada); err != nil {
		return err
	}
	os.FinalizadaEm = &agora
	duracao, _ := os.TempoExecucao()
	os.Registrar(OSFinalizada{EventoOS: os.eventoOS(agora), TempoExecucao: duracao})
	return nil
}

// Entregar: Finalizada -> Entregue, ou Em diagnóstico -> Entregue quando o
// cliente rejeitou o orçamento e o veículo é entregue sem execução.
func (os *OrdemServico) Entregar(agora time.Time) error {
	if os.Status == StatusEmDiagnostico && os.StatusOrcamento != OrcamentoRejeitado {
		return ErrEntregaNaoPermitida
	}
	if !os.Status.PodeIrPara(StatusEntregue) {
		return ErrEntregaNaoPermitida
	}
	os.Status = StatusEntregue
	os.EntregueEm = &agora
	os.Registrar(VeiculoEntregue{EventoOS: os.eventoOS(agora)})
	return nil
}

// TempoExecucao é o período entre o início da execução e a finalização.
func (os *OrdemServico) TempoExecucao() (time.Duration, bool) {
	if os.ExecucaoIniciadaEm == nil || os.FinalizadaEm == nil {
		return 0, false
	}
	return os.FinalizadaEm.Sub(*os.ExecucaoIniciadaEm), true
}

// ---------------------------------------------------------------------------
// Reparo adicional
// ---------------------------------------------------------------------------

// PodeRegistrarReparoAdicional: reparo adicional só durante a execução.
func (os *OrdemServico) PodeRegistrarReparoAdicional() error {
	if os.Status != StatusEmExecucao {
		return ErrReparoForaDeExecucao
	}
	return nil
}

// RegistrarReparo cria um reparo adicional pendente com os itens pedidos.
// Os itens ficam fora do orçamento até o cliente autorizar. As peças
// precisam de saldo, contando o que a OS ainda vai baixar.
func (os *OrdemServico) RegistrarReparo(descricao string, servicos []PedidoServico, pecas []PedidoPeca) (*ReparoAdicional, error) {
	if len(servicos)+len(pecas) == 0 {
		return nil, ErrReparoSemItens
	}
	if err := os.PodeRegistrarReparoAdicional(); err != nil {
		return nil, err
	}

	reparo := ReparoAdicional{UUID: uuid.New(), Descricao: descricao, Status: ReparoPendente}
	doReparo := uuid.NullUUID{UUID: reparo.UUID, Valid: true}

	itens := make([]Item, 0, len(servicos)+len(pecas))
	for _, pedido := range servicos {
		item, err := novoItemServico(pedido.Servico, pedido.Quantidade)
		if err != nil {
			return nil, err
		}
		item.ReparoUUID, item.StatusReparo = doReparo, ReparoPendente
		itens = append(itens, item)
	}
	for _, pedido := range pecas {
		// Os itens já montados entram na conta do saldo pedido pela OS.
		os.Itens = append(os.Itens, itens...)
		item, err := os.novoItemPeca(pedido.Peca, pedido.Quantidade, pedido.SaldoDisponivel)
		os.Itens = os.Itens[:len(os.Itens)-len(itens)]
		if err != nil {
			return nil, err
		}
		item.ReparoUUID, item.StatusReparo = doReparo, ReparoPendente
		itens = append(itens, item)
	}

	os.Reparos = append(os.Reparos, reparo)
	os.Itens = append(os.Itens, itens...)
	os.recalcular()
	os.Registrar(ReparoAdicionalIdentificado{EventoOS: os.eventoOS(time.Now()), ReparoUUID: reparo.UUID, Descricao: descricao})
	return &os.Reparos[len(os.Reparos)-1], nil
}

// ReparosPendentes conta os reparos ainda sem decisão do cliente.
func (os *OrdemServico) ReparosPendentes() int {
	total := 0
	for _, r := range os.Reparos {
		if r.Pendente() {
			total++
		}
	}
	return total
}

// Reparo localiza um reparo adicional pelo UUID.
func (os *OrdemServico) Reparo(reparoUUID uuid.UUID) (*ReparoAdicional, error) {
	for i := range os.Reparos {
		if os.Reparos[i].UUID == reparoUUID {
			return &os.Reparos[i], nil
		}
	}
	return nil, ErrReparoNaoEncontrado
}

// AutorizarReparo registra a autorização do cliente: os itens do reparo
// entram no orçamento. As peças dele (PecasDoReparo) são baixadas do estoque
// pelo chamador, na mesma transação.
func (os *OrdemServico) AutorizarReparo(reparoUUID uuid.UUID, agora time.Time) error {
	return os.decidirReparo(reparoUUID, ReparoAutorizado, agora)
}

// RecusarReparo registra a recusa do cliente: o reparo não entra na OS.
func (os *OrdemServico) RecusarReparo(reparoUUID uuid.UUID, agora time.Time) error {
	return os.decidirReparo(reparoUUID, ReparoRecusado, agora)
}

func (os *OrdemServico) decidirReparo(reparoUUID uuid.UUID, decisao StatusReparo, agora time.Time) error {
	reparo, err := os.Reparo(reparoUUID)
	if err != nil {
		return err
	}
	if !reparo.Pendente() {
		return ErrReparoJaDecidido
	}
	if os.Status != StatusEmExecucao {
		return ErrReparoForaDeExecucao
	}
	reparo.Status = decisao
	reparo.DecididoEm = &agora
	for i := range os.Itens {
		if os.Itens[i].DoReparo(reparoUUID) {
			os.Itens[i].StatusReparo = decisao
		}
	}
	os.alteracoes.ReparosDecididos = append(os.alteracoes.ReparosDecididos, reparoUUID)
	os.recalcular()
	if decisao == ReparoAutorizado {
		os.Registrar(ReparoAdicionalAutorizado{EventoOS: os.eventoOS(agora), ReparoUUID: reparoUUID, Descricao: reparo.Descricao})
	} else {
		os.Registrar(ReparoAdicionalRecusado{EventoOS: os.eventoOS(agora), ReparoUUID: reparoUUID, Descricao: reparo.Descricao})
	}
	return nil
}

// ValidarQuantidade garante quantidade positiva em itens da OS.
func ValidarQuantidade(quantidade int32) error {
	if quantidade <= 0 {
		return ErrQuantidadeInvalida
	}
	return nil
}
