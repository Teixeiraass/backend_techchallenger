package notification

import "context"

// NoopSender discards notifications (used in tests).
type NoopSender struct{}

func NewNoopSender() Sender {
	return &NoopSender{}
}

func (s *NoopSender) Send(_ context.Context, _ Message) error {
	return nil
}
