// Package eventos tem o despachante síncrono, em processo, dos eventos de
// domínio. Os serviços publicam os eventos depois do commit da transação;
// as políticas (notificar o cliente, pedir reposição) são handlers
// assinados em cmd/server/main.go.
package eventos

import (
	"context"
	"log"
	"sync"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
)

// Handler reage a um evento. Um erro é registrado no log e não interrompe
// os demais handlers nem desfaz a operação que gerou o evento.
type Handler func(ctx context.Context, evento domain.Evento) error

// Despachante entrega cada evento, na ordem, aos handlers assinados pelo
// nome do evento. É síncrono: Publicar só volta depois de todos os handlers.
type Despachante struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	logf     func(format string, args ...any)
}

func NovoDespachante() *Despachante {
	return &Despachante{handlers: make(map[string][]Handler), logf: log.Printf}
}

// Assinar registra um handler para o evento com o nome informado.
func (d *Despachante) Assinar(nome string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[nome] = append(d.handlers[nome], handler)
}

// Publicar entrega os eventos aos handlers assinados. Falha em um handler
// fica no log; os outros continuam.
func (d *Despachante) Publicar(ctx context.Context, eventos []domain.Evento) {
	for _, evento := range eventos {
		d.mu.RLock()
		handlers := append([]Handler(nil), d.handlers[evento.Nome()]...)
		d.mu.RUnlock()
		for _, handler := range handlers {
			if err := handler(ctx, evento); err != nil {
				d.logf("evento %q: handler falhou: %v", evento.Nome(), err)
			}
		}
	}
}
