package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/google/uuid"
)

// OrdemServicoService orquestra os casos de uso da OS. Cada comando
// (ordem_servico_comandos.go) abre uma unidade de trabalho, carrega o
// agregado pelo repositório (porta do domínio), chama o método da raiz que
// aplica as invariantes (internal/domain/ordemservico), salva e, depois do
// commit, publica os eventos registrados pela raiz (as políticas de
// notificação e reposição são handlers; ver politicas.go). As consultas
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
	store   db.Store
	eventos Publicador
	now     func() time.Time
}

func NewOrdemServicoService(uow UnidadeDeTrabalho, store db.Store, eventos Publicador) OrdemServicoService {
	return &ordemServicoService{uow: uow, store: store, eventos: eventos, now: time.Now}
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

// baixarEstoque dá saída das peças da OS pelo agregado Peça e pelo repositório
// de estoque (UPDATE condicional, garantia de concorrência). Devolve os
// eventos de estoque registrados pelas peças (baixa, estoque baixo).
func baixarEstoque(ctx context.Context, pecas estoque.Repositorio, osID int64, itens []ordemservico.Item, usuarioID int64) ([]domain.Evento, error) {
	eventos := make([]domain.Evento, 0)
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
				// A transação será desfeita, mas o fato "estoque insuficiente
				// detectado" precisa chegar à política de reposição.
				return nil, &faltaPeca{err: err, eventos: peca.Eventos()}
			}
			return nil, err
		}
		eventos = append(eventos, peca.Eventos()...)
	}
	return eventos, nil
}

// faltaPeca é o erro de saldo insuficiente na baixa. Carrega os eventos da
// peça (estoque insuficiente detectado) para serem publicados mesmo com a
// transação desfeita; errors.Is continua achando ErrEstoqueInsuficiente.
type faltaPeca struct {
	err     error
	eventos []domain.Evento
}

func (f *faltaPeca) Error() string { return f.err.Error() }
func (f *faltaPeca) Unwrap() error { return f.err }

// publicarFaltaDePeca publica os eventos de uma baixa que falhou por falta de
// saldo (a transação foi desfeita; só o aviso ao administrador sobrevive).
func (s *ordemServicoService) publicarFaltaDePeca(ctx context.Context, err error) {
	var falta *faltaPeca
	if errors.As(err, &falta) {
		s.eventos.Publicar(ctx, falta.eventos)
	}
}

// publicar entrega os eventos do agregado (e os de estoque) depois do commit.
func (s *ordemServicoService) publicar(ctx context.Context, os *ordemservico.OrdemServico, extras []domain.Evento) {
	eventos := append(os.Eventos(), extras...)
	os.LimparEventos()
	s.eventos.Publicar(ctx, eventos)
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
