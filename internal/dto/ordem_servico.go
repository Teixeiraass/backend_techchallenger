package dto

import (
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/google/uuid"
)

type ItemServicoRequest struct {
	ServicoUuid uuid.UUID `json:"servico_uuid" binding:"required" swaggertype:"string" format:"uuid"`
	Quantidade  int32     `json:"quantidade" binding:"omitempty,min=1" example:"1"`
}

type ItemPecaRequest struct {
	PecaUuid   uuid.UUID `json:"peca_uuid" binding:"required" swaggertype:"string" format:"uuid"`
	Quantidade int32     `json:"quantidade" binding:"required,min=1" example:"1"`
}

type AbrirOrdemServicoRequest struct {
	// CPF ou CNPJ do cliente já cadastrado (com ou sem pontuação).
	Documento string `json:"documento" binding:"required" example:"52998224725"`
	// Placa de um veículo do cliente.
	Placa             string               `json:"placa" binding:"required" example:"ABC1D23"`
	DescricaoProblema string               `json:"descricao_problema" example:"Barulho ao frear"`
	Servicos          []ItemServicoRequest `json:"servicos" binding:"omitempty,dive"`
	Pecas             []ItemPecaRequest    `json:"pecas" binding:"omitempty,dive"`
}

type EnviarOrcamentoRequest struct {
	Diagnostico string `json:"diagnostico" example:"Pastilhas gastas, fluido contaminado"`
}

// DecisaoClienteRequest is used by the public tracking routes: the customer
// confirms who they are with the CPF/CNPJ of the OS owner.
type DecisaoClienteRequest struct {
	Documento string `json:"documento" binding:"required" example:"52998224725"`
}

type RegistrarReparoAdicionalRequest struct {
	Descricao string               `json:"descricao" binding:"required" example:"Disco de freio empenado"`
	Servicos  []ItemServicoRequest `json:"servicos" binding:"omitempty,dive"`
	Pecas     []ItemPecaRequest    `json:"pecas" binding:"omitempty,dive"`
}

type ListOrdensServicoQuery struct {
	PaginacaoQuery
	Status      string `form:"status" binding:"omitempty,oneof=RECEBIDA EM_DIAGNOSTICO AGUARDANDO_APROVACAO EM_EXECUCAO FINALIZADA ENTREGUE"`
	ClienteUuid string `form:"cliente_uuid"`
}

type AndamentoQuery struct {
	Documento string `form:"documento" binding:"required"`
}

type TempoMedioQuery struct {
	DataInicio string `form:"data_inicio" example:"2026-01-01"`
	DataFim    string `form:"data_fim" example:"2026-12-31"`
}

type ClienteResumo struct {
	Uuid      uuid.UUID `json:"uuid"`
	Nome      string    `json:"nome"`
	Documento string    `json:"documento"`
}

type VeiculoResumo struct {
	Uuid   uuid.UUID `json:"uuid"`
	Placa  string    `json:"placa"`
	Marca  string    `json:"marca"`
	Modelo string    `json:"modelo"`
	Ano    int32     `json:"ano"`
}

type OrcamentoResponse struct {
	Status                string     `json:"status"`
	TotalServicosCentavos int64      `json:"total_servicos_centavos"`
	TotalPecasCentavos    int64      `json:"total_pecas_centavos"`
	TotalCentavos         int64      `json:"total_centavos"`
	EnviadoEm             *time.Time `json:"enviado_em"`
	DecididoEm            *time.Time `json:"decidido_em"`
}

type ItemServicoResponse struct {
	Uuid                  uuid.UUID  `json:"uuid"`
	ServicoUuid           uuid.UUID  `json:"servico_uuid"`
	ServicoNome           string     `json:"servico_nome"`
	Quantidade            int32      `json:"quantidade"`
	ValorUnitarioCentavos int64      `json:"valor_unitario_centavos"`
	SubtotalCentavos      int64      `json:"subtotal_centavos"`
	ReparoAdicionalUuid   *uuid.UUID `json:"reparo_adicional_uuid,omitempty"`
}

type ItemPecaResponse struct {
	Uuid                  uuid.UUID  `json:"uuid"`
	PecaUuid              uuid.UUID  `json:"peca_uuid"`
	PecaCodigo            string     `json:"peca_codigo"`
	PecaNome              string     `json:"peca_nome"`
	PecaTipo              string     `json:"peca_tipo"`
	Quantidade            int32      `json:"quantidade"`
	ValorUnitarioCentavos int64      `json:"valor_unitario_centavos"`
	SubtotalCentavos      int64      `json:"subtotal_centavos"`
	ReparoAdicionalUuid   *uuid.UUID `json:"reparo_adicional_uuid,omitempty"`
}

type ReparoAdicionalResponse struct {
	Uuid       uuid.UUID  `json:"uuid"`
	Descricao  string     `json:"descricao"`
	Status     string     `json:"status"`
	DecididoEm *time.Time `json:"decidido_em"`
	CreatedAt  time.Time  `json:"created_at"`
}

type HistoricoStatusResponse struct {
	StatusAnterior string    `json:"status_anterior"`
	StatusNovo     string    `json:"status_novo"`
	Observacao     string    `json:"observacao"`
	CreatedAt      time.Time `json:"created_at"`
}

type OrdemServicoResponse struct {
	Uuid                  uuid.UUID                 `json:"uuid"`
	Status                string                    `json:"status"`
	StatusDescricao       string                    `json:"status_descricao"`
	DescricaoProblema     string                    `json:"descricao_problema"`
	Diagnostico           string                    `json:"diagnostico"`
	Cliente               ClienteResumo             `json:"cliente"`
	Veiculo               VeiculoResumo             `json:"veiculo"`
	Orcamento             OrcamentoResponse         `json:"orcamento"`
	Servicos              []ItemServicoResponse     `json:"servicos"`
	Pecas                 []ItemPecaResponse        `json:"pecas"`
	ReparosAdicionais     []ReparoAdicionalResponse `json:"reparos_adicionais"`
	Historico             []HistoricoStatusResponse `json:"historico"`
	DiagnosticoIniciadoEm *time.Time                `json:"diagnostico_iniciado_em"`
	ExecucaoIniciadaEm    *time.Time                `json:"execucao_iniciada_em"`
	FinalizadaEm          *time.Time                `json:"finalizada_em"`
	EntregueEm            *time.Time                `json:"entregue_em"`
	TempoExecucaoSegundos *int64                    `json:"tempo_execucao_segundos"`
	CreatedAt             time.Time                 `json:"created_at"`
	UpdatedAt             time.Time                 `json:"updated_at"`
	// Aviso is filled when the customer approved the budget but the system
	// could not start the execution yet (e.g. missing stock).
	Aviso string `json:"aviso,omitempty"`
}

// NewOrdemServicoResponse monta a resposta a partir do agregado (itens e
// reparos vêm da raiz), do cliente e veículo (leitura) e do histórico.
func NewOrdemServicoResponse(os *ordemservico.OrdemServico, dono readmodel.Cliente, v readmodel.Veiculo, historico []readmodel.HistoricoStatus) OrdemServicoResponse {
	servicos := make([]ItemServicoResponse, 0)
	pecas := make([]ItemPecaResponse, 0)
	for _, item := range os.Itens {
		switch item.Tipo {
		case ordemservico.ItemServico:
			servicos = append(servicos, ItemServicoResponse{
				Uuid:                  item.UUID,
				ServicoUuid:           item.CatalogoUUID,
				ServicoNome:           item.Nome,
				Quantidade:            item.Quantidade,
				ValorUnitarioCentavos: item.ValorUnitarioCentavos,
				SubtotalCentavos:      item.Subtotal(),
				ReparoAdicionalUuid:   nullUUIDPtr(item.ReparoUUID),
			})
		case ordemservico.ItemPeca:
			pecas = append(pecas, ItemPecaResponse{
				Uuid:                  item.UUID,
				PecaUuid:              item.CatalogoUUID,
				PecaCodigo:            item.Codigo,
				PecaNome:              item.Nome,
				PecaTipo:              item.TipoPeca,
				Quantidade:            item.Quantidade,
				ValorUnitarioCentavos: item.ValorUnitarioCentavos,
				SubtotalCentavos:      item.Subtotal(),
				ReparoAdicionalUuid:   nullUUIDPtr(item.ReparoUUID),
			})
		}
	}

	reparos := make([]ReparoAdicionalResponse, 0, len(os.Reparos))
	for _, r := range os.Reparos {
		reparos = append(reparos, ReparoAdicionalResponse{
			Uuid:       r.UUID,
			Descricao:  r.Descricao,
			Status:     string(r.Status),
			DecididoEm: r.DecididoEm,
			CreatedAt:  r.CriadoEm,
		})
	}

	linhas := make([]HistoricoStatusResponse, 0, len(historico))
	for _, h := range historico {
		linhas = append(linhas, HistoricoStatusResponse{
			StatusAnterior: h.StatusAnterior,
			StatusNovo:     h.StatusNovo,
			Observacao:     h.Observacao,
			CreatedAt:      h.CriadoEm,
		})
	}

	orcamento := os.Orcamento()

	var tempoExecucao *int64
	if duracao, ok := os.TempoExecucao(); ok {
		segundos := int64(duracao.Seconds())
		tempoExecucao = &segundos
	}

	return OrdemServicoResponse{
		Uuid:              os.UUID,
		Status:            string(os.Status),
		StatusDescricao:   os.Status.Descricao(),
		DescricaoProblema: os.DescricaoProblema,
		Diagnostico:       os.Diagnostico,
		Cliente: ClienteResumo{
			Uuid:      dono.UUID,
			Nome:      dono.Nome,
			Documento: dono.Documento,
		},
		Veiculo: VeiculoResumo{
			Uuid:   v.UUID,
			Placa:  v.Placa,
			Marca:  v.Marca,
			Modelo: v.Modelo,
			Ano:    v.Ano,
		},
		Orcamento: OrcamentoResponse{
			Status:                string(os.StatusOrcamento),
			TotalServicosCentavos: orcamento.TotalServicosCentavos,
			TotalPecasCentavos:    orcamento.TotalPecasCentavos,
			TotalCentavos:         os.ValorTotalCentavos,
			EnviadoEm:             os.OrcamentoEnviadoEm,
			DecididoEm:            os.OrcamentoDecididoEm,
		},
		Servicos:              servicos,
		Pecas:                 pecas,
		ReparosAdicionais:     reparos,
		Historico:             linhas,
		DiagnosticoIniciadoEm: os.DiagnosticoIniciadoEm,
		ExecucaoIniciadaEm:    os.ExecucaoIniciadaEm,
		FinalizadaEm:          os.FinalizadaEm,
		EntregueEm:            os.EntregueEm,
		TempoExecucaoSegundos: tempoExecucao,
		CreatedAt:             os.CriadaEm,
		UpdatedAt:             os.AtualizadaEm,
	}
}

type OrdemServicoResumoResponse struct {
	Uuid               uuid.UUID `json:"uuid"`
	Status             string    `json:"status"`
	StatusDescricao    string    `json:"status_descricao"`
	OrcamentoStatus    string    `json:"orcamento_status"`
	ValorTotalCentavos int64     `json:"valor_total_centavos"`
	ClienteUuid        uuid.UUID `json:"cliente_uuid"`
	ClienteNome        string    `json:"cliente_nome"`
	ClienteDocumento   string    `json:"cliente_documento"`
	VeiculoUuid        uuid.UUID `json:"veiculo_uuid"`
	VeiculoPlaca       string    `json:"veiculo_placa"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func NewOrdemServicoResumoResponse(row readmodel.OrdemServicoResumo) OrdemServicoResumoResponse {
	return OrdemServicoResumoResponse{
		Uuid:               row.UUID,
		Status:             row.Status,
		StatusDescricao:    ordemservico.Status(row.Status).Descricao(),
		OrcamentoStatus:    row.OrcamentoStatus,
		ValorTotalCentavos: row.ValorTotalCentavos,
		ClienteUuid:        row.ClienteUUID,
		ClienteNome:        row.ClienteNome,
		ClienteDocumento:   row.ClienteDocumento,
		VeiculoUuid:        row.VeiculoUUID,
		VeiculoPlaca:       row.VeiculoPlaca,
		CreatedAt:          row.CriadaEm,
		UpdatedAt:          row.AtualizadaEm,
	}
}

type TempoMedioServicoResponse struct {
	ServicoUuid          uuid.UUID `json:"servico_uuid"`
	ServicoNome          string    `json:"servico_nome"`
	TempoEstimadoMinutos int32     `json:"tempo_estimado_minutos"`
	TotalOrdens          int64     `json:"total_ordens"`
	MediaSegundos        int64     `json:"media_segundos"`
	MediaFormatada       string    `json:"media_formatada"`
}

type TempoMedioExecucaoResponse struct {
	TotalOrdens    int64                       `json:"total_ordens"`
	MediaSegundos  int64                       `json:"media_segundos"`
	MediaFormatada string                      `json:"media_formatada"`
	MinimoSegundos int64                       `json:"minimo_segundos"`
	MaximoSegundos int64                       `json:"maximo_segundos"`
	PorServico     []TempoMedioServicoResponse `json:"por_servico"`
}

func nullUUIDPtr(value uuid.NullUUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := value.UUID
	return &id
}
