package notification

import "context"

// Message is an e-mail notice sent to a customer or to the administrator
// (stock replenishment request).
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers notifications by e-mail (external system). The MVP ships a
// log-based implementation; an e-mail provider only needs to implement this
// interface.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
