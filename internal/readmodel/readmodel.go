// Package readmodel contém as projeções de leitura usadas pelas respostas da
// API: dados planos, sem regra de negócio, montados a partir das linhas do
// banco. Os DTOs dependem destes tipos (e do domínio), nunca do sqlc.
package readmodel

import (
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/google/uuid"
)

type Cliente struct {
	ID            int64
	UUID          uuid.UUID
	Nome          string
	Documento     string
	TipoDocumento string
	Email         string
	Telefone      string
	CriadoEm      time.Time
	AtualizadoEm  time.Time
}

func DeCliente(c db.Cliente) Cliente {
	return Cliente{ID: c.ID, UUID: c.Uuid, Nome: c.Nome, Documento: c.Documento, TipoDocumento: c.TipoDocumento, Email: c.Email, Telefone: c.Telefone, CriadoEm: c.CreatedAt, AtualizadoEm: c.UpdatedAt}
}

type Veiculo struct {
	ID           int64
	UUID         uuid.UUID
	ClienteID    int64
	ClienteUUID  uuid.UUID
	Placa        string
	Marca        string
	Modelo       string
	Ano          int32
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

func DeVeiculo(v db.Veiculo, clienteUUID uuid.UUID) Veiculo {
	return Veiculo{ID: v.ID, UUID: v.Uuid, ClienteID: v.ClienteID, ClienteUUID: clienteUUID, Placa: v.Placa, Marca: v.Marca, Modelo: v.Modelo, Ano: v.Ano, CriadoEm: v.CreatedAt, AtualizadoEm: v.UpdatedAt}
}

type Servico struct {
	ID                   int64
	UUID                 uuid.UUID
	Nome                 string
	Descricao            string
	PrecoCentavos        int64
	TempoEstimadoMinutos int32
	Ativo                bool
	CriadoEm             time.Time
	AtualizadoEm         time.Time
}

func DeServico(s db.Servico) Servico {
	return Servico{ID: s.ID, UUID: s.Uuid, Nome: s.Nome, Descricao: s.Descricao, PrecoCentavos: s.PrecoCentavos, TempoEstimadoMinutos: s.TempoEstimadoMinutos, Ativo: s.Ativo, CriadoEm: s.CreatedAt, AtualizadoEm: s.UpdatedAt}
}

type MovimentoEstoque struct {
	ID             int64
	UUID           uuid.UUID
	PecaID         int64
	Tipo           string
	Quantidade     int32
	SaldoAnterior  int32
	SaldoPosterior int32
	OrdemServicoID *int64
	UsuarioID      *int64
	Observacao     string
	CriadoEm       time.Time
}

func DeMovimentoEstoque(m db.MovimentoEstoque) MovimentoEstoque {
	mov := MovimentoEstoque{ID: m.ID, UUID: m.Uuid, PecaID: m.PecaID, Tipo: m.Tipo, Quantidade: m.Quantidade, SaldoAnterior: m.SaldoAnterior, SaldoPosterior: m.SaldoPosterior, Observacao: m.Observacao, CriadoEm: m.CreatedAt}
	if m.OrdemServicoID.Valid {
		id := m.OrdemServicoID.Int64
		mov.OrdemServicoID = &id
	}
	if m.UsuarioID.Valid {
		id := m.UsuarioID.Int64
		mov.UsuarioID = &id
	}
	return mov
}

type Usuario struct {
	ID       int64
	UUID     uuid.UUID
	Nome     string
	Email    string
	Papel    string
	Ativo    bool
	CriadoEm time.Time
}

func DeUsuario(u db.Usuario) Usuario {
	return Usuario{ID: u.ID, UUID: u.Uuid, Nome: u.Nome, Email: u.Email, Papel: u.Papel, Ativo: u.Ativo, CriadoEm: u.CreatedAt}
}

// HistoricoStatus é uma linha de historico_status_os.
type HistoricoStatus struct {
	StatusAnterior string
	StatusNovo     string
	Observacao     string
	CriadoEm       time.Time
}

func DeHistoricoStatus(h db.HistoricoStatusOS) HistoricoStatus {
	return HistoricoStatus{StatusAnterior: h.StatusAnterior, StatusNovo: h.StatusNovo, Observacao: h.Observacao, CriadoEm: h.CreatedAt}
}

// OrdemServicoResumo é a linha da listagem administrativa de OS.
type OrdemServicoResumo struct {
	UUID               uuid.UUID
	Status             string
	OrcamentoStatus    string
	ValorTotalCentavos int64
	ClienteUUID        uuid.UUID
	ClienteNome        string
	ClienteDocumento   string
	VeiculoUUID        uuid.UUID
	VeiculoPlaca       string
	CriadaEm           time.Time
	AtualizadaEm       time.Time
}

func DeOrdemServicoResumo(row db.ListOrdensServicoRow) OrdemServicoResumo {
	return OrdemServicoResumo{
		UUID:               row.OrdemServico.Uuid,
		Status:             row.OrdemServico.Status,
		OrcamentoStatus:    row.OrdemServico.OrcamentoStatus,
		ValorTotalCentavos: row.OrdemServico.ValorTotalCentavos,
		ClienteUUID:        row.ClienteUuid,
		ClienteNome:        row.ClienteNome,
		ClienteDocumento:   row.ClienteDocumento,
		VeiculoUUID:        row.VeiculoUuid,
		VeiculoPlaca:       row.VeiculoPlaca,
		CriadaEm:           row.OrdemServico.CreatedAt,
		AtualizadaEm:       row.OrdemServico.UpdatedAt,
	}
}
