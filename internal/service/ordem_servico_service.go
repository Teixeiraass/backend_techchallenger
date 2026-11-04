package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/google/uuid"
)

// OrdemServicoService orquestra os casos de uso da OS. Cada comando
// (ordem_servico_comandos.go) abre uma unidade de trabalho, carrega o
// agregado pelo repositório (porta do domínio), chama o método da raiz que
// aplica as invariantes (internal/domain/ordemservico) e salva. As consultas
// (ordem_servico_consultas.go) leem pelo repositório e pelo read model.
type OrdemServicoService interface {
	AbrirOrdemServico(ctx context.Context, req dto.AbrirOrdemServicoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error)
	BuscarOrdemServico(ctx context.Context, osUUID uuid.UUID) (*dto.OrdemServicoResponse, error)
	ListarOrdensServico(ctx context.Context, query dto.ListOrdensServicoQuery) ([]dto.OrdemServicoResumoResponse, error)
	AdicionarServico(ctx context.Context, osUUID uuid.UUID, req dto.ItemServicoRequest) (*dto.OrdemServicoResponse, error)
	RemoverServico(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error)
	AdicionarPeca(ctx context.Context, osUUID uuid.UUID, req dto.ItemPecaRequest) (*dto.OrdemServicoResponse, error)
	RemoverPeca(ctx context.Context, osUUID uuid.UUID, itemUUID uuid.UUID) (*dto.OrdemServicoResponse, error)
	IniciarDiagnostico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	EnviarOrcamento(ctx context.Context, osUUID uuid.UUID, req dto.EnviarOrcamentoRequest, usuarioID int64) (*dto.OrdemServicoResponse, error)
	AprovarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	RejeitarOrcamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	IniciarExecucao(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	FinalizarOrdemServico(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	EntregarVeiculo(ctx context.Context, osUUID uuid.UUID, usuarioID int64) (*dto.OrdemServicoResponse, error)
	RegistrarReparoAdicional(ctx context.Context, osUUID uuid.UUID, req dto.RegistrarReparoAdicionalRequest) (*dto.OrdemServicoResponse, error)
	AutorizarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	RecusarReparoAdicional(ctx context.Context, osUUID uuid.UUID, reparoUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	ConsultarAndamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error)
	TempoMedioExecucao(ctx context.Context, query dto.TempoMedioQuery) (*dto.TempoMedioExecucaoResponse, error)
}

type ordemServicoService struct {
	uow UnidadeDeTrabalho
	// store é o read model das listagens e do tempo médio (SQL direto).
	store  db.Store
	sender notification.Sender
	config util.Config
	now    func() time.Time
}

func NewOrdemServicoService(uow UnidadeDeTrabalho, store db.Store, sender notification.Sender, config util.Config) OrdemServicoService {
	return &ordemServicoService{uow: uow, store: store, sender: sender, config: config, now: time.Now}
}

// Erros do agregado reexportados por compatibilidade com quem importa o serviço.
var (
	ErrServicoInativo      = ordemservico.ErrServicoInativo
	ErrItemNaoEncontrado   = ordemservico.ErrItemNaoEncontrado
	ErrReparoNaoEncontrado = ordemservico.ErrReparoNaoEncontrado
)

// validarTitular confere se o CPF/CNPJ informado é o do dono da OS. Se não
// conferir, a resposta é a mesma de OS inexistente, para não revelar quais
// OS existem.
func validarTitular(ctx context.Context, catalogo Catalogo, clienteID int64, documento string) error {
	dono, err := catalogo.ClientePorID(ctx, clienteID)
	if err != nil {
		return err
	}
	if cliente.Normalizar(documento) != dono.Documento {
		return ErrOSNaoEncontrada
	}
	return nil
}

// notificarCliente envia o aviso por e-mail depois do commit; falha de envio
// não desfaz a operação. O e-mail é obrigatório no cadastro; um cliente antigo
// sem e-mail não recebe aviso e o fato fica no log.
func (s *ordemServicoService) notificarCliente(ctx context.Context, clienteID int64, assunto string, mensagem string) {
	dono, err := s.uow.Leitura().Catalogo().ClientePorID(ctx, clienteID)
	if err != nil {
		log.Printf("não foi possível carregar o cliente %d para o aviso: %v", clienteID, err)
		return
	}

	if dono.Email == "" {
		log.Printf("cliente %d sem e-mail; aviso não enviado", clienteID)
		return
	}

	if err := s.sender.Send(ctx, notification.Message{To: dono.Email, Subject: assunto, Body: mensagem}); err != nil {
		log.Printf("não foi possível avisar o cliente %d: %v", clienteID, err)
	}
}

// baixarEstoque dá saída das peças da OS pelo agregado Peça e pelo repositório
// de estoque (UPDATE condicional, garantia de concorrência). Devolve as peças
// que ficaram no estoque mínimo para o pedido de reposição.
func baixarEstoque(ctx context.Context, pecas estoque.Repositorio, osID int64, itens []ordemservico.Item, usuarioID int64) ([]estoque.Peca, error) {
	estoqueBaixo := make([]estoque.Peca, 0)
	for _, item := range itens {
		peca, err := pecas.CarregarPorID(ctx, item.CatalogoID)
		if err != nil {
			return nil, err
		}
		movimento, err := peca.Baixar(item.Quantidade, osID, usuarioID)
		if err == nil {
			err = pecas.AplicarMovimento(ctx, peca, movimento)
		}
		if err != nil {
			if errors.Is(err, estoque.ErrEstoqueInsuficiente) {
				return nil, &faltaPeca{err: err, item: item}
			}
			return nil, err
		}
		if peca.EstoqueBaixo() {
			estoqueBaixo = append(estoqueBaixo, *peca)
		}
	}
	return estoqueBaixo, nil
}

// faltaPeca é o erro de saldo insuficiente na baixa. Guarda o item da OS para
// o pedido de reposição; errors.Is continua achando ErrEstoqueInsuficiente.
type faltaPeca struct {
	err  error
	item ordemservico.Item
}

func (f *faltaPeca) Error() string { return f.err.Error() }
func (f *faltaPeca) Unwrap() error { return f.err }

// pedirReposicaoFaltante avisa o administrador quando a baixa falhou por falta
// de saldo. Sem a reposição, a OS aprovada não entra em execução.
func pedirReposicaoFaltante(ctx context.Context, sender notification.Sender, to string, osUUID uuid.UUID, err error) {
	var falta *faltaPeca
	if !errors.As(err, &falta) {
		return
	}
	if err := sender.Send(ctx, notification.Message{
		To:      to,
		Subject: "Pedido de reposição de estoque",
		Body: fmt.Sprintf("%s (%s) não tem saldo para a OS %s, que precisa de %d. A OS aguarda a reposição para entrar em execução.",
			falta.item.Nome, falta.item.Codigo, osUUID, falta.item.Quantidade),
	}); err != nil {
		log.Printf("não foi possível enviar o pedido de reposição: %v", err)
	}
}

func formatarDuracao(segundos int64) string {
	horas := segundos / 3600
	minutos := (segundos % 3600) / 60
	if horas > 0 {
		return fmt.Sprintf("%dh %02dmin", horas, minutos)
	}
	if minutos > 0 {
		return fmt.Sprintf("%dmin", minutos)
	}
	return fmt.Sprintf("%ds", segundos)
}

func formatarDinheiro(centavos int64) string {
	return fmt.Sprintf("R$ %d,%02d", centavos/100, centavos%100)
}
