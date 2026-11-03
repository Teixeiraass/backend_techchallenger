package ordemservico

import (
	"time"

	"github.com/google/uuid"
)

// TipoItem diferencia as duas linhas da OS: serviço do catálogo ou peça/insumo.
type TipoItem string

const (
	ItemServico TipoItem = "SERVICO"
	ItemPeca    TipoItem = "PECA"
)

// Item é a entidade interna do agregado: uma linha de serviço ou de peça da
// OS, com o valor unitário congelado no momento em que entrou. Itens de um
// reparo adicional carregam o UUID e o status do reparo; só os autorizados
// entram no orçamento.
type Item struct {
	// ID é a identidade de persistência (0 = ainda não gravado).
	ID   int64
	UUID uuid.UUID
	Tipo TipoItem
	// CatalogoID e CatalogoUUID identificam o serviço ou a peça do catálogo.
	CatalogoID   int64
	CatalogoUUID uuid.UUID
	Nome         string
	// Codigo e TipoPeca só existem para peças (código do catálogo, PECA ou INSUMO).
	Codigo                string
	TipoPeca              string
	Quantidade            int32
	ValorUnitarioCentavos int64
	// ReparoUUID vem preenchido só quando o item pertence a um reparo adicional.
	ReparoUUID uuid.NullUUID
	// StatusReparo acompanha a decisão do cliente sobre o reparo; itens de
	// reparo pendente ou recusado não entram no total.
	StatusReparo StatusReparo
	CriadoEm     time.Time
}

// Subtotal é quantidade x valor unitário.
func (i Item) Subtotal() int64 {
	return int64(i.Quantidade) * i.ValorUnitarioCentavos
}

// EntraNoOrcamento indica se o item conta no valor total.
func (i Item) EntraNoOrcamento() bool {
	return i.StatusReparo == "" || i.StatusReparo == ReparoAutorizado
}

// DoReparo indica se o item pertence ao reparo adicional informado.
func (i Item) DoReparo(reparoUUID uuid.UUID) bool {
	return i.ReparoUUID.Valid && i.ReparoUUID.UUID == reparoUUID
}

// ReparoAdicional é a entidade interna que registra um problema encontrado
// durante a execução. Os itens dele ficam em OrdemServico.Itens, ligados
// pelo UUID, e só entram no orçamento depois da autorização do cliente.
type ReparoAdicional struct {
	ID         int64
	UUID       uuid.UUID
	Descricao  string
	Status     StatusReparo
	DecididoEm *time.Time
	CriadoEm   time.Time
}

// Pendente indica que o cliente ainda não decidiu sobre o reparo.
func (r ReparoAdicional) Pendente() bool {
	return r.Status == ReparoPendente
}

// CatalogoServico é o que o agregado precisa saber de um serviço do catálogo
// para incluí-lo na OS. O preço é copiado para o item (congelado).
type CatalogoServico struct {
	ID            int64
	UUID          uuid.UUID
	Nome          string
	PrecoCentavos int64
	Ativo         bool
}

// CatalogoPeca é o que o agregado precisa saber de uma peça ou insumo do
// estoque para incluí-la na OS.
type CatalogoPeca struct {
	ID            int64
	UUID          uuid.UUID
	Codigo        string
	Nome          string
	Tipo          string
	PrecoCentavos int64
	Ativa         bool
}

// PedidoServico e PedidoPeca são as linhas pedidas em um reparo adicional.
type PedidoServico struct {
	Servico    CatalogoServico
	Quantidade int32
}

type PedidoPeca struct {
	Peca       CatalogoPeca
	Quantidade int32
	// SaldoDisponivel é o saldo atual do estoque, lido na mesma transação.
	SaldoDisponivel int32
}

// Alteracoes é o que mudou no agregado desde o carregamento e ainda não foi
// persistido. Itens e reparos novos são os que têm ID zero.
type Alteracoes struct {
	ItensRemovidos   []Item
	ReparosDecididos []uuid.UUID
}
