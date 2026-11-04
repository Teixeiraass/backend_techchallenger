package domain

import "time"

// Evento é um fato do domínio que já aconteceu, com o nome da linguagem
// ubíqua (§2 do event storming). As raízes dos agregados registram os
// eventos enquanto executam os comandos; a aplicação os publica depois do
// commit da transação, para que nenhuma política reaja a um estado que
// acabou desfeito.
type Evento interface {
	Nome() string
	OcorridoEm() time.Time
}

// Registrador acumula os eventos de uma raiz de agregado. É embutido nas
// raízes (OrdemServico, Peca).
type Registrador struct {
	eventos []Evento
}

// Registrar guarda um evento ocorrido nesta operação.
func (r *Registrador) Registrar(e Evento) {
	r.eventos = append(r.eventos, e)
}

// Eventos devolve os eventos registrados e ainda não publicados.
func (r *Registrador) Eventos() []Evento {
	return append([]Evento(nil), r.eventos...)
}

// LimparEventos esquece os eventos já publicados.
func (r *Registrador) LimparEventos() {
	r.eventos = nil
}
