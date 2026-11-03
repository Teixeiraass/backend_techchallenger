package service

import (
	"context"
	"database/sql"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/google/uuid"
)

// Consultas da OS: leitura do detalhe, acompanhamento público, listagem e
// tempo médio. Não alteram estado e não abrem transação.

func (s *ordemServicoService) BuscarOrdemServico(ctx context.Context, osUUID uuid.UUID) (*dto.OrdemServicoResponse, error) {
	os, err := s.store.GetOrdemServico(ctx, osUUID)
	if err != nil {
		return nil, notFound(err, ErrOSNaoEncontrada)
	}
	return s.montarResposta(ctx, os)
}

// ConsultarAndamento é a consulta pública do cliente. O CPF/CNPJ funciona
// como segundo fator junto com o UUID da OS; se não conferir, a resposta é a
// mesma de OS inexistente para não revelar quais OS existem.
func (s *ordemServicoService) ConsultarAndamento(ctx context.Context, osUUID uuid.UUID, documento string) (*dto.OrdemServicoResponse, error) {
	os, err := s.store.GetOrdemServico(ctx, osUUID)
	if err != nil {
		return nil, notFound(err, ErrOSNaoEncontrada)
	}
	if err := validarTitular(ctx, s.store, os.ClienteID, documento); err != nil {
		return nil, err
	}
	return s.montarResposta(ctx, os)
}

func (s *ordemServicoService) ListarOrdensServico(ctx context.Context, query dto.ListOrdensServicoQuery) ([]dto.OrdemServicoResumoResponse, error) {
	var clienteID sql.NullInt64
	if query.ClienteUuid != "" {
		clienteUUID, err := uuid.Parse(query.ClienteUuid)
		if err != nil {
			return nil, ErrClienteNaoEncontrado
		}
		dono, err := s.store.GetCliente(ctx, clienteUUID)
		if err != nil {
			return nil, notFound(err, ErrClienteNaoEncontrado)
		}
		clienteID = sql.NullInt64{Int64: dono.ID, Valid: true}
	}

	rows, err := s.store.ListOrdensServico(ctx, db.ListOrdensServicoParams{
		Status:       query.Status,
		ClienteID:    clienteID,
		Limite:       query.Limite(),
		Deslocamento: query.Deslocamento(),
	})
	if err != nil {
		return nil, err
	}

	response := make([]dto.OrdemServicoResumoResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, dto.NewOrdemServicoResumoResponse(row))
	}
	return response, nil
}

// TempoMedioExecucao calcula a média entre o início da execução e a
// finalização das OS finalizadas, no geral e por serviço.
func (s *ordemServicoService) TempoMedioExecucao(ctx context.Context, query dto.TempoMedioQuery) (*dto.TempoMedioExecucaoResponse, error) {
	dataInicio, err := parseData(query.DataInicio)
	if err != nil {
		return nil, err
	}
	dataFim, err := parseData(query.DataFim)
	if err != nil {
		return nil, err
	}
	if dataFim.Valid {
		// data_fim é inclusiva: soma um dia para usar "<" na consulta.
		dataFim.Time = dataFim.Time.AddDate(0, 0, 1)
	}
	if dataInicio.Valid && dataFim.Valid && !dataInicio.Time.Before(dataFim.Time) {
		return nil, ErrIntervaloDatasInvalido
	}

	geral, err := s.store.GetTempoMedioExecucao(ctx, db.GetTempoMedioExecucaoParams{DataInicio: dataInicio, DataFim: dataFim})
	if err != nil {
		return nil, err
	}

	porServico, err := s.store.ListTempoMedioExecucaoPorServico(ctx, db.ListTempoMedioExecucaoPorServicoParams{DataInicio: dataInicio, DataFim: dataFim})
	if err != nil {
		return nil, err
	}

	servicos := make([]dto.TempoMedioServicoResponse, 0, len(porServico))
	for _, row := range porServico {
		media := int64(row.MediaSegundos)
		servicos = append(servicos, dto.TempoMedioServicoResponse{
			ServicoUuid:          row.ServicoUuid,
			ServicoNome:          row.ServicoNome,
			TempoEstimadoMinutos: row.TempoEstimadoMinutos,
			TotalOrdens:          row.TotalOrdens,
			MediaSegundos:        media,
			MediaFormatada:       formatarDuracao(media),
		})
	}

	media := int64(geral.MediaSegundos)
	return &dto.TempoMedioExecucaoResponse{
		TotalOrdens:    geral.TotalOrdens,
		MediaSegundos:  media,
		MediaFormatada: formatarDuracao(media),
		MinimoSegundos: int64(geral.MinimoSegundos),
		MaximoSegundos: int64(geral.MaximoSegundos),
		PorServico:     servicos,
	}, nil
}

func (s *ordemServicoService) montarResposta(ctx context.Context, os db.OrdemServico) (*dto.OrdemServicoResponse, error) {
	dono, err := s.store.GetClienteByID(ctx, os.ClienteID)
	if err != nil {
		return nil, err
	}
	v, err := s.store.GetVeiculoByID(ctx, os.VeiculoID)
	if err != nil {
		return nil, err
	}
	servicos, err := s.store.ListItensServicoByOrdemServico(ctx, os.ID)
	if err != nil {
		return nil, err
	}
	pecas, err := s.store.ListItensPecaByOrdemServico(ctx, os.ID)
	if err != nil {
		return nil, err
	}
	reparos, err := s.store.ListReparosAdicionaisByOrdemServico(ctx, os.ID)
	if err != nil {
		return nil, err
	}
	historico, err := s.store.ListHistoricoStatusOSByOrdemServico(ctx, os.ID)
	if err != nil {
		return nil, err
	}

	response := dto.NewOrdemServicoResponse(dto.OrdemServicoDetalhe{
		OrdemServico: os,
		Cliente:      dono,
		Veiculo:      v,
		Servicos:     servicos,
		Pecas:        pecas,
		Reparos:      reparos,
		Historico:    historico,
	})
	return &response, nil
}

// parseData interpreta a data do filtro (AAAA-MM-DD) no fuso horário do
// servidor (time.Local). O docker-compose.yaml e o Dockerfile fixam
// TZ=America/Sao_Paulo; sem TZ o container usaria UTC e um filtro
// data_fim=2026-11-03 terminaria às 21h de Brasília.
func parseData(valor string) (sql.NullTime, error) {
	if valor == "" {
		return sql.NullTime{}, nil
	}
	data, err := time.ParseInLocation("2006-01-02", valor, time.Local)
	if err != nil {
		return sql.NullTime{}, ErrDataInvalida
	}
	return sql.NullTime{Time: data, Valid: true}, nil
}
