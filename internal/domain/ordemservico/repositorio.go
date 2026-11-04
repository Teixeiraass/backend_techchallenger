package ordemservico

import (
	"context"

	"github.com/google/uuid"
)

// Repositorio é a porta do agregado para a persistência. A implementação
// fica na infraestrutura (internal/repository); o domínio e a aplicação só
// conhecem esta interface, então a dependência aponta para dentro.
type Repositorio interface {
	// Carregar reconstitui a OS com itens e reparos, sem travar a linha.
	Carregar(ctx context.Context, osUUID uuid.UUID) (*OrdemServico, error)
	// CarregarParaAtualizar reconstitui a OS e trava a linha até o fim da
	// transação, para dois comandos concorrentes não se sobreporem.
	CarregarParaAtualizar(ctx context.Context, osUUID uuid.UUID) (*OrdemServico, error)
	// Salvar grava a raiz (cria quando ainda não tem identidade), insere e
	// remove itens e reparos pela diferença e registra as entradas de
	// histórico anotadas na raiz.
	Salvar(ctx context.Context, os *OrdemServico) error
}

// EntradaHistorico é uma linha de historico_status_os pendente de gravação:
// mudança de status ou decisão do cliente (status anterior = status novo).
type EntradaHistorico struct {
	StatusAnterior Status
	StatusNovo     Status
	UsuarioID      int64
	Observacao     string
}

// Reconstituir devolve a raiz carregada da persistência, guardando o status
// com que ela chegou (usado nas entradas de histórico).
func Reconstituir(os OrdemServico) *OrdemServico {
	os.statusAoCarregar = os.Status
	return &os
}

// AnotarHistorico registra que o comando atual deve deixar uma linha no
// histórico, do status de carregamento até o status atual.
func (os *OrdemServico) AnotarHistorico(usuarioID int64, observacao string) {
	os.alteracoes.Historico = append(os.alteracoes.Historico, EntradaHistorico{
		StatusAnterior: os.statusAoCarregar,
		StatusNovo:     os.Status,
		UsuarioID:      usuarioID,
		Observacao:     observacao,
	})
}

// LimparAlteracoes zera o que já foi persistido.
func (os *OrdemServico) LimparAlteracoes() {
	os.alteracoes = Alteracoes{}
	os.statusAoCarregar = os.Status
}
