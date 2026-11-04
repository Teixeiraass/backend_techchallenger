package service

import (
	"context"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/google/uuid"
)

// Portas da camada de aplicação. Os comandos da OS e do estoque só conhecem
// estas interfaces; a implementação com sqlc fica em internal/repository e é
// injetada em cmd/server/main.go (e em handler.NewServer).

// UnidadeDeTrabalho abre uma transação e entrega os repositórios ligados a
// ela. Tudo o que fn grava é confirmado junto ou desfeito junto.
type UnidadeDeTrabalho interface {
	Executar(ctx context.Context, fn func(Repositorios) error) error
	// Leitura devolve os repositórios fora de transação, para consultas.
	Leitura() Repositorios
}

// Repositorios agrupa as portas usadas pelos casos de uso.
type Repositorios interface {
	OrdemServico() ordemservico.Repositorio
	Pecas() estoque.Repositorio
	Catalogo() Catalogo
}

// Publicador entrega os eventos de domínio às políticas assinadas. Os
// serviços publicam depois do commit da transação: uma política nunca reage
// a um estado que acabou desfeito, e a falha de um handler não desfaz a
// operação (internal/eventos).
type Publicador interface {
	Publicar(ctx context.Context, eventos []domain.Evento)
}

// Catalogo é a leitura dos outros contextos que a OS precisa: cadastro
// (cliente e veículo) e serviços do catálogo. A OS guarda só os ids e copia
// o preço do serviço no momento da inclusão.
type Catalogo interface {
	Servico(ctx context.Context, servicoUUID uuid.UUID) (ordemservico.CatalogoServico, error)
	ClientePorDocumento(ctx context.Context, documento string) (readmodel.Cliente, error)
	ClientePorID(ctx context.Context, clienteID int64) (readmodel.Cliente, error)
	VeiculoPorPlaca(ctx context.Context, placa string) (readmodel.Veiculo, error)
	VeiculoPorID(ctx context.Context, veiculoID int64) (readmodel.Veiculo, error)
	HistoricoDaOS(ctx context.Context, osID int64) ([]readmodel.HistoricoStatus, error)
}
