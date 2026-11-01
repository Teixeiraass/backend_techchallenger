package notification

import (
	"context"
	"log"
)

// LogSender writes notifications to the application log.
type LogSender struct{}

func NewLogSender() Sender {
	return &LogSender{}
}

func (s *LogSender) Send(_ context.Context, msg Message) error {
	log.Printf("[notificacao] para=%s assunto=%q mensagem=%q", msg.To, msg.Subject, msg.Body)
	return nil
}
