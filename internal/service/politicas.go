package service

import (
	"context"
	"fmt"
	"log"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/eventos"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
)

// Políticas do event storming que reagem a eventos ("sempre que X, então Y").
// São handlers assinados no despachante em cmd/server/main.go; rodam depois
// do commit da transação que gerou o evento, e a falha de um handler não
// desfaz a operação.
//
// A política "ao aprovar o orçamento, iniciar a execução" continua síncrona
// dentro de OrdemServicoService.AprovarOrcamento: ela faz parte da mesma
// requisição do cliente, que precisa receber o estado final.

// Politicas agrupa as dependências dos handlers.
type Politicas struct {
	catalogo Catalogo
	sender   notification.Sender
	config   util.Config
}

func NovasPoliticas(catalogo Catalogo, sender notification.Sender, config util.Config) *Politicas {
	return &Politicas{catalogo: catalogo, sender: sender, config: config}
}

// AssinarPoliticas liga cada política ao evento que a dispara.
func AssinarPoliticas(d *eventos.Despachante, p *Politicas) {
	d.Assinar(ordemservico.NomeOrcamentoEnviadoAoCliente, p.NotificarOrcamentoEnviado)
	d.Assinar(ordemservico.NomeReparoAdicionalIdentificado, p.NotificarReparoIdentificado)
	d.Assinar(ordemservico.NomeOSFinalizada, p.NotificarOSFinalizada)
	d.Assinar(estoque.NomeEstoqueInsuficienteDetectado, p.PedirReposicaoFaltante)
	d.Assinar(estoque.NomeEstoqueBaixoDetectado, p.PedirReposicaoEstoqueBaixo)
}

// NotificarOrcamentoEnviado: ao enviar o orçamento, o cliente é avisado para
// aprovar ou rejeitar pelo link de acompanhamento.
func (p *Politicas) NotificarOrcamentoEnviado(ctx context.Context, e domain.Evento) error {
	ev, ok := e.(ordemservico.OrcamentoEnviadoAoCliente)
	if !ok {
		return nil
	}
	return p.notificarCliente(ctx, ev.ClienteID, "Orçamento disponível para aprovação", fmt.Sprintf(
		"O orçamento da sua ordem de serviço é de %s. Aprove ou rejeite pelo link de acompanhamento: /api/v1/ordens-servico/%s/andamento",
		formatarDinheiro(ev.ValorTotalCentavos), ev.OSUUID,
	))
}

// NotificarReparoIdentificado: ao identificar um reparo adicional, o cliente
// é chamado a autorizar ou recusar.
func (p *Politicas) NotificarReparoIdentificado(ctx context.Context, e domain.Evento) error {
	ev, ok := e.(ordemservico.ReparoAdicionalIdentificado)
	if !ok {
		return nil
	}
	return p.notificarCliente(ctx, ev.ClienteID, "Reparo adicional aguardando autorização", fmt.Sprintf(
		"Durante a execução encontramos outro problema: %s. Autorize ou recuse pelo link de acompanhamento: /api/v1/ordens-servico/%s/andamento",
		ev.Descricao, ev.OSUUID,
	))
}

// NotificarOSFinalizada: ao finalizar a OS, o cliente é avisado de que o
// veículo está pronto.
func (p *Politicas) NotificarOSFinalizada(ctx context.Context, e domain.Evento) error {
	ev, ok := e.(ordemservico.OSFinalizada)
	if !ok {
		return nil
	}
	return p.notificarCliente(ctx, ev.ClienteID, "Seu veículo está pronto",
		fmt.Sprintf("A ordem de serviço %s foi finalizada. O veículo já pode ser retirado.", ev.OSUUID))
}

// PedirReposicaoFaltante: sem saldo para a OS, o administrador recebe o
// pedido de reposição; sem ela a OS aprovada não entra em execução.
func (p *Politicas) PedirReposicaoFaltante(ctx context.Context, e domain.Evento) error {
	ev, ok := e.(estoque.EstoqueInsuficienteDetectado)
	if !ok {
		return nil
	}
	return p.sender.Send(ctx, notification.Message{
		To:      p.config.AdminNotificationTo,
		Subject: "Pedido de reposição de estoque",
		Body: fmt.Sprintf("%s (%s) não tem saldo para a OS %d, que precisa de %d (saldo %d). A OS aguarda a reposição para entrar em execução.",
			ev.NomePeca, ev.Codigo, ev.OrdemServicoID, ev.Necessario, ev.Saldo),
	})
}

// PedirReposicaoEstoqueBaixo: saldo no mínimo ou abaixo dispara o pedido de
// reposição ao administrador.
func (p *Politicas) PedirReposicaoEstoqueBaixo(ctx context.Context, e domain.Evento) error {
	ev, ok := e.(estoque.EstoqueBaixoDetectado)
	if !ok {
		return nil
	}
	return p.sender.Send(ctx, notification.Message{
		To:      p.config.AdminNotificationTo,
		Subject: "Pedido de reposição de estoque",
		Body: fmt.Sprintf("%s (%s) está com saldo %d %s, abaixo ou igual ao mínimo de %d.",
			ev.NomePeca, ev.Codigo, ev.Saldo, ev.Unidade, ev.EstoqueMinimo),
	})
}

// notificarCliente envia o aviso por e-mail. O e-mail é obrigatório no
// cadastro; um cliente antigo sem e-mail não recebe aviso e o fato fica no log.
func (p *Politicas) notificarCliente(ctx context.Context, clienteID int64, assunto string, mensagem string) error {
	dono, err := p.catalogo.ClientePorID(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("carregar o cliente %d para o aviso: %w", clienteID, err)
	}
	if dono.Email == "" {
		log.Printf("cliente %d sem e-mail; aviso não enviado", clienteID)
		return nil
	}
	return p.sender.Send(ctx, notification.Message{To: dono.Email, Subject: assunto, Body: mensagem})
}
