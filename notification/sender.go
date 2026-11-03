package notification

import "context"

// Message é um aviso por e-mail ao cliente ou ao administrador (pedido de
// reposição de estoque).
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender entrega as notificações por e-mail (sistema externo). O MVP usa uma
// implementação que só registra em log; um provedor de e-mail só precisa
// implementar esta interface.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
