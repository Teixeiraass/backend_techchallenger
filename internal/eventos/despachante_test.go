package eventos

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/stretchr/testify/require"
)

type eventoTeste struct {
	nome string
	em   time.Time
}

func (e eventoTeste) Nome() string          { return e.nome }
func (e eventoTeste) OcorridoEm() time.Time { return e.em }

func TestDespachanteEntregaNaOrdemAosAssinantes(t *testing.T) {
	d := NovoDespachante()
	var recebidos []string
	d.Assinar("A", func(_ context.Context, e domain.Evento) error {
		recebidos = append(recebidos, "a1:"+e.Nome())
		return nil
	})
	d.Assinar("A", func(_ context.Context, e domain.Evento) error {
		recebidos = append(recebidos, "a2:"+e.Nome())
		return nil
	})
	d.Assinar("B", func(_ context.Context, e domain.Evento) error {
		recebidos = append(recebidos, "b:"+e.Nome())
		return nil
	})

	d.Publicar(context.Background(), []domain.Evento{eventoTeste{nome: "A"}, eventoTeste{nome: "C"}, eventoTeste{nome: "B"}})
	require.Equal(t, []string{"a1:A", "a2:A", "b:B"}, recebidos, "C não tem assinante e é ignorado")
}

func TestDespachanteFalhaDeHandlerNaoInterrompe(t *testing.T) {
	d := NovoDespachante()
	var logs []string
	d.logf = func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }
	chamados := 0
	d.Assinar("A", func(context.Context, domain.Evento) error { return errors.New("smtp fora do ar") })
	d.Assinar("A", func(context.Context, domain.Evento) error { chamados++; return nil })

	d.Publicar(context.Background(), []domain.Evento{eventoTeste{nome: "A"}})
	require.Equal(t, 1, chamados)
	require.Len(t, logs, 1)
	require.Contains(t, logs[0], "smtp fora do ar")
}

func TestDespachanteSemEventos(t *testing.T) {
	d := NovoDespachante()
	d.Publicar(context.Background(), nil)
}
