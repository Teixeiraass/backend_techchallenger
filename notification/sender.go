package notification

import "context"

// Message is a notice sent to a customer or to the workshop staff
// (e-mail or app push in production).
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers notifications. The MVP ships a log-based implementation;
// an e-mail or push provider only needs to implement this interface.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
