package dto

import (
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/util"
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

// OrdemServicoDetalhe groups everything loaded from the database to render an OS.
type OrdemServicoDetalhe struct {
	OrdemServico db.OrdemServico
	Cliente      db.Cliente
	Veiculo      db.Veiculo
	Servicos     []db.ListItensServicoByOrdemServicoRow
	Pecas        []db.ListItensPecaByOrdemServicoRow
	Reparos      []db.ReparoAdicional
	Historico    []db.HistoricoStatusOS
}

func NewOrdemServicoResponse(d OrdemServicoDetalhe) OrdemServicoResponse {
	os := d.OrdemServico
	status := ordemservico.Status(os.Status)

	servicos := make([]ItemServicoResponse, 0, len(d.Servicos))
	itensServico := make([]ordemservico.Item, 0, len(d.Servicos))
	for _, item := range d.Servicos {
		servicos = append(servicos, ItemServicoResponse{
			Uuid:                  item.Uuid,
			ServicoUuid:           item.ServicoUuid,
			ServicoNome:           item.ServicoNome,
			Quantidade:            item.Quantidade,
			ValorUnitarioCentavos: item.ValorUnitarioCentavos,
			SubtotalCentavos:      int64(item.Quantidade) * item.ValorUnitarioCentavos,
			ReparoAdicionalUuid:   nullUUIDPtr(item.ReparoAdicionalUuid),
		})
		itensServico = append(itensServico, ItemServicoParaDominio(item))
	}

	pecas := make([]ItemPecaResponse, 0, len(d.Pecas))
	itensPeca := make([]ordemservico.Item, 0, len(d.Pecas))
	for _, item := range d.Pecas {
		pecas = append(pecas, ItemPecaResponse{
			Uuid:                  item.Uuid,
			PecaUuid:              item.PecaUuid,
			PecaCodigo:            item.PecaCodigo,
			PecaNome:              item.PecaNome,
			PecaTipo:              item.PecaTipo,
			Quantidade:            item.Quantidade,
			ValorUnitarioCentavos: item.ValorUnitarioCentavos,
			SubtotalCentavos:      int64(item.Quantidade) * item.ValorUnitarioCentavos,
			ReparoAdicionalUuid:   nullUUIDPtr(item.ReparoAdicionalUuid),
		})
		itensPeca = append(itensPeca, ItemPecaParaDominio(item))
	}

	reparos := make([]ReparoAdicionalResponse, 0, len(d.Reparos))
	for _, r := range d.Reparos {
		reparos = append(reparos, ReparoAdicionalResponse{
			Uuid:       r.Uuid,
			Descricao:  r.Descricao,
			Status:     r.Status,
			DecididoEm: util.FromNullTime(r.DecididoEm),
			CreatedAt:  r.CreatedAt,
		})
	}

	historico := make([]HistoricoStatusResponse, 0, len(d.Historico))
	for _, h := range d.Historico {
		historico = append(historico, HistoricoStatusResponse{
			StatusAnterior: h.StatusAnterior,
			StatusNovo:     h.StatusNovo,
			Observacao:     h.Observacao,
			CreatedAt:      h.CreatedAt,
		})
	}

	orcamento := ordemservico.CalcularOrcamento(itensServico, itensPeca)

	var tempoExecucao *int64
	if os.ExecucaoIniciadaEm.Valid && os.FinalizadaEm.Valid {
		segundos := int64(os.FinalizadaEm.Time.Sub(os.ExecucaoIniciadaEm.Time).Seconds())
		tempoExecucao = &segundos
	}

	return OrdemServicoResponse{
		Uuid:              os.Uuid,
		Status:            os.Status,
		StatusDescricao:   status.Descricao(),
		DescricaoProblema: os.DescricaoProblema,
		Diagnostico:       os.Diagnostico,
		Cliente: ClienteResumo{
			Uuid:      d.Cliente.Uuid,
			Nome:      d.Cliente.Nome,
			Documento: d.Cliente.Documento,
		},
		Veiculo: VeiculoResumo{
			Uuid:   d.Veiculo.Uuid,
			Placa:  d.Veiculo.Placa,
			Marca:  d.Veiculo.Marca,
			Modelo: d.Veiculo.Modelo,
			Ano:    d.Veiculo.Ano,
		},
		Orcamento: OrcamentoResponse{
			Status:                os.OrcamentoStatus,
			TotalServicosCentavos: orcamento.TotalServicosCentavos,
			TotalPecasCentavos:    orcamento.TotalPecasCentavos,
			TotalCentavos:         os.ValorTotalCentavos,
			EnviadoEm:             util.FromNullTime(os.OrcamentoEnviadoEm),
			DecididoEm:            util.FromNullTime(os.OrcamentoDecididoEm),
		},
		Servicos:              servicos,
		Pecas:                 pecas,
		ReparosAdicionais:     reparos,
		Historico:             historico,
		DiagnosticoIniciadoEm: util.FromNullTime(os.DiagnosticoIniciadoEm),
		ExecucaoIniciadaEm:    util.FromNullTime(os.ExecucaoIniciadaEm),
		FinalizadaEm:          util.FromNullTime(os.FinalizadaEm),
		EntregueEm:            util.FromNullTime(os.EntregueEm),
		TempoExecucaoSegundos: tempoExecucao,
		CreatedAt:             os.CreatedAt,
		UpdatedAt:             os.UpdatedAt,
	}
}

// ItemServicoParaDominio converts a service line into the item used by the budget.
func ItemServicoParaDominio(item db.ListItensServicoByOrdemServicoRow) ordemservico.Item {
	return ordemservico.Item{
		Quantidade:            item.Quantidade,
		ValorUnitarioCentavos: item.ValorUnitarioCentavos,
		StatusReparo:          ordemservico.StatusReparo(item.ReparoAdicionalStatus.String),
	}
}

// ItemPecaParaDominio converts a part line into the item used by the budget.
func ItemPecaParaDominio(item db.ListItensPecaByOrdemServicoRow) ordemservico.Item {
	return ordemservico.Item{
		Quantidade:            item.Quantidade,
		ValorUnitarioCentavos: item.ValorUnitarioCentavos,
		StatusReparo:          ordemservico.StatusReparo(item.ReparoAdicionalStatus.String),
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

func NewOrdemServicoResumoResponse(row db.ListOrdensServicoRow) OrdemServicoResumoResponse {
	return OrdemServicoResumoResponse{
		Uuid:               row.OrdemServico.Uuid,
		Status:             row.OrdemServico.Status,
		StatusDescricao:    ordemservico.Status(row.OrdemServico.Status).Descricao(),
		OrcamentoStatus:    row.OrdemServico.OrcamentoStatus,
		ValorTotalCentavos: row.OrdemServico.ValorTotalCentavos,
		ClienteUuid:        row.ClienteUuid,
		ClienteNome:        row.ClienteNome,
		ClienteDocumento:   row.ClienteDocumento,
		VeiculoUuid:        row.VeiculoUuid,
		VeiculoPlaca:       row.VeiculoPlaca,
		CreatedAt:          row.OrdemServico.CreatedAt,
		UpdatedAt:          row.OrdemServico.UpdatedAt,
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
