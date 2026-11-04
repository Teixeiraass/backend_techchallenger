package estoque

import (
	"context"
	"strings"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/google/uuid"
)

// Peca é a raiz do agregado de estoque: uma peça ou insumo do catálogo com o
// saldo e o estoque mínimo. Toda mudança de saldo passa por um método da
// raiz, que valida e devolve o MovimentoEstoque a registrar.
type Peca struct {
	domain.Registrador

	// ID é a identidade de persistência (0 = ainda não gravada).
	ID            int64
	UUID          uuid.UUID
	Codigo        string
	Nome          string
	Descricao     string
	Tipo          Tipo
	Unidade       string
	PrecoCentavos int64
	Saldo         int32
	EstoqueMinimo int32
	Ativa         bool
	CriadaEm      time.Time
	AtualizadaEm  time.Time
}

// MovimentoEstoque é a entidade interna que registra cada mudança de saldo.
type MovimentoEstoque struct {
	ID             int64
	PecaID         int64
	Tipo           TipoMovimento
	Quantidade     int32
	SaldoAnterior  int32
	SaldoPosterior int32
	// OrdemServicoID é a OS que motivou a saída (0 quando não há).
	OrdemServicoID int64
	// UsuarioID é quem fez o movimento (0 para o sistema/cliente).
	UsuarioID  int64
	Observacao string
	CriadoEm   time.Time
}

// Cadastro são os dados informados ao criar ou atualizar uma peça.
type Cadastro struct {
	Codigo        string
	Nome          string
	Descricao     string
	Tipo          string
	Unidade       string
	PrecoCentavos int64
	EstoqueMinimo int32
}

// NovaPeca cria a peça com o saldo inicial. Quando o saldo inicial é maior
// que zero, devolve também o movimento de entrada "Saldo inicial".
func NovaPeca(cadastro Cadastro, saldoInicial int32) (*Peca, *MovimentoEstoque, error) {
	if err := ValidarCadastro(cadastro.Tipo, cadastro.PrecoCentavos, saldoInicial, cadastro.EstoqueMinimo); err != nil {
		return nil, nil, err
	}
	p := &Peca{Ativa: true, Saldo: saldoInicial}
	p.aplicarCadastro(cadastro)

	var movimento *MovimentoEstoque
	if saldoInicial > 0 {
		movimento = &MovimentoEstoque{
			Tipo:           MovimentoEntrada,
			Quantidade:     saldoInicial,
			SaldoAnterior:  0,
			SaldoPosterior: saldoInicial,
			Observacao:     "Saldo inicial",
		}
	}
	return p, movimento, nil
}

// Atualizar troca os dados de cadastro (o saldo só muda por movimento).
func (p *Peca) Atualizar(cadastro Cadastro, ativa bool) error {
	if err := ValidarCadastro(cadastro.Tipo, cadastro.PrecoCentavos, 0, cadastro.EstoqueMinimo); err != nil {
		return err
	}
	p.aplicarCadastro(cadastro)
	p.Ativa = ativa
	return nil
}

func (p *Peca) aplicarCadastro(cadastro Cadastro) {
	unidade := strings.ToUpper(strings.TrimSpace(cadastro.Unidade))
	if unidade == "" {
		unidade = "UN"
	}
	p.Codigo = strings.ToUpper(strings.TrimSpace(cadastro.Codigo))
	p.Nome = strings.TrimSpace(cadastro.Nome)
	p.Descricao = cadastro.Descricao
	p.Tipo = Tipo(cadastro.Tipo)
	p.Unidade = unidade
	p.PrecoCentavos = cadastro.PrecoCentavos
	p.EstoqueMinimo = cadastro.EstoqueMinimo
}

// Inativar tira a peça do catálogo sem apagar o histórico.
func (p *Peca) Inativar() {
	p.Ativa = false
}

// Baixar dá saída de quantidade unidades para uma OS. O saldo nunca fica
// negativo: sem saldo devolve ErrEstoqueInsuficiente com o nome da peça. A
// persistência repete a conferência no UPDATE condicional (concorrência).
func (p *Peca) Baixar(quantidade int32, ordemServicoID int64, usuarioID int64) (MovimentoEstoque, error) {
	if quantidade <= 0 {
		return MovimentoEstoque{}, ErrQuantidadeInvalida
	}
	if p.Saldo < quantidade {
		p.Registrar(EstoqueInsuficienteDetectado{EventoPeca: p.eventoPeca(), OrdemServicoID: ordemServicoID, Necessario: quantidade, Saldo: p.Saldo})
		return MovimentoEstoque{}, ErrEstoqueInsuficientePara(p.Nome, p.Codigo)
	}
	anterior := p.Saldo
	p.Saldo -= quantidade
	p.Registrar(EstoqueBaixado{EventoPeca: p.eventoPeca(), OrdemServicoID: ordemServicoID, Quantidade: quantidade, Saldo: p.Saldo})
	p.conferirEstoqueMinimo()
	return MovimentoEstoque{
		PecaID:         p.ID,
		Tipo:           MovimentoSaida,
		Quantidade:     -quantidade,
		SaldoAnterior:  anterior,
		SaldoPosterior: p.Saldo,
		OrdemServicoID: ordemServicoID,
		UsuarioID:      usuarioID,
		Observacao:     "Baixa por ordem de serviço",
	}, nil
}

// Repor dá entrada de quantidade unidades (reposição de estoque).
func (p *Peca) Repor(quantidade int32, usuarioID int64, observacao string) (MovimentoEstoque, error) {
	if err := ValidarReposicao(quantidade); err != nil {
		return MovimentoEstoque{}, err
	}
	anterior := p.Saldo
	p.Saldo += quantidade
	return MovimentoEstoque{
		PecaID:         p.ID,
		Tipo:           MovimentoEntrada,
		Quantidade:     quantidade,
		SaldoAnterior:  anterior,
		SaldoPosterior: p.Saldo,
		UsuarioID:      usuarioID,
		Observacao:     observacao,
	}, nil
}

// Ajustar define o saldo contado no inventário e registra a diferença.
func (p *Peca) Ajustar(novoSaldo int32, usuarioID int64, observacao string) (MovimentoEstoque, error) {
	if err := ValidarAjuste(novoSaldo); err != nil {
		return MovimentoEstoque{}, err
	}
	anterior := p.Saldo
	p.Saldo = novoSaldo
	p.conferirEstoqueMinimo()
	return MovimentoEstoque{
		PecaID:         p.ID,
		Tipo:           MovimentoAjuste,
		Quantidade:     novoSaldo - anterior,
		SaldoAnterior:  anterior,
		SaldoPosterior: p.Saldo,
		UsuarioID:      usuarioID,
		Observacao:     observacao,
	}, nil
}

// EstoqueBaixo indica se o saldo chegou ao estoque mínimo (dispara pedido de reposição).
func (p *Peca) EstoqueBaixo() bool {
	return EstoqueBaixo(p.Saldo, p.EstoqueMinimo)
}

// conferirEstoqueMinimo registra "Estoque baixo detectado" quando o saldo
// chega ao mínimo: a política de pedir reposição reage a esse evento.
func (p *Peca) conferirEstoqueMinimo() {
	if p.EstoqueBaixo() {
		p.Registrar(EstoqueBaixoDetectado{EventoPeca: p.eventoPeca(), Saldo: p.Saldo, EstoqueMinimo: p.EstoqueMinimo})
	}
}

// Repositorio é a porta do agregado Peça para a persistência.
type Repositorio interface {
	Carregar(ctx context.Context, pecaUUID uuid.UUID) (*Peca, error)
	CarregarPorID(ctx context.Context, pecaID int64) (*Peca, error)
	// CarregarParaAtualizar trava a linha até o fim da transação.
	CarregarParaAtualizar(ctx context.Context, pecaUUID uuid.UUID) (*Peca, error)
	// Salvar grava os dados de cadastro (cria quando ainda não tem identidade).
	Salvar(ctx context.Context, peca *Peca) error
	// AplicarMovimento grava a mudança de saldo e o movimento. A saída usa
	// UPDATE ... WHERE quantidade_estoque >= qtd: sem linha afetada, devolve
	// ErrEstoqueInsuficiente (garantia de concorrência) e a peça não muda.
	AplicarMovimento(ctx context.Context, peca *Peca, movimento MovimentoEstoque) error
	// RegistrarMovimento grava só a linha do movimento (saldo já gravado,
	// como no saldo inicial da peça nova).
	RegistrarMovimento(ctx context.Context, movimento MovimentoEstoque) error
}
