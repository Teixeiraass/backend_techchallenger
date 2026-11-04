package ordemservico

import (
	"time"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/google/uuid"
)

// Nomes dos eventos da OS, iguais aos da linguagem ubíqua (§2).
const (
	NomeOSRecebida                  = "OS recebida"
	NomeDiagnosticoIniciado         = "Diagnóstico iniciado"
	NomeOrcamentoEnviadoAoCliente   = "Orçamento enviado ao cliente"
	NomeOrcamentoAprovado           = "Orçamento aprovado"
	NomeOrcamentoRejeitado          = "Orçamento rejeitado"
	NomeExecucaoIniciada            = "Execução iniciada"
	NomeReparoAdicionalIdentificado = "Reparo adicional identificado"
	NomeReparoAdicionalAutorizado   = "Reparo adicional autorizado"
	NomeReparoAdicionalRecusado     = "Reparo adicional recusado"
	NomeOSFinalizada                = "OS finalizada"
	NomeVeiculoEntregue             = "Veículo entregue"
)

// EventoOS é a parte comum dos eventos da OS: qual OS, de qual cliente e
// quando. O UUID é preenchido ao ler os eventos (uma OS nova só ganha UUID
// ao ser salva).
type EventoOS struct {
	OSUUID    uuid.UUID
	ClienteID int64
	Em        time.Time
}

func (e EventoOS) OcorridoEm() time.Time { return e.Em }

func (OSRecebida) Nome() string                    { return NomeOSRecebida }
func (DiagnosticoIniciado) Nome() string           { return NomeDiagnosticoIniciado }
func (OrcamentoEnviadoAoCliente) Nome() string     { return NomeOrcamentoEnviadoAoCliente }
func (OrcamentoAprovadoPeloCliente) Nome() string  { return NomeOrcamentoAprovado }
func (OrcamentoRejeitadoPeloCliente) Nome() string { return NomeOrcamentoRejeitado }
func (ExecucaoIniciada) Nome() string              { return NomeExecucaoIniciada }
func (ReparoAdicionalIdentificado) Nome() string   { return NomeReparoAdicionalIdentificado }
func (ReparoAdicionalAutorizado) Nome() string     { return NomeReparoAdicionalAutorizado }
func (ReparoAdicionalRecusado) Nome() string       { return NomeReparoAdicionalRecusado }
func (OSFinalizada) Nome() string                  { return NomeOSFinalizada }
func (VeiculoEntregue) Nome() string               { return NomeVeiculoEntregue }

type OSRecebida struct{ EventoOS }
type DiagnosticoIniciado struct{ EventoOS }
type OrcamentoEnviadoAoCliente struct {
	EventoOS
	ValorTotalCentavos int64
}

// OrcamentoAprovadoPeloCliente e OrcamentoRejeitadoPeloCliente têm o sufixo
// porque OrcamentoAprovado/OrcamentoRejeitado já são os valores de StatusOrcamento.
type OrcamentoAprovadoPeloCliente struct{ EventoOS }
type OrcamentoRejeitadoPeloCliente struct{ EventoOS }
type ExecucaoIniciada struct{ EventoOS }
type ReparoAdicionalIdentificado struct {
	EventoOS
	ReparoUUID uuid.UUID
	Descricao  string
}
type ReparoAdicionalAutorizado struct {
	EventoOS
	ReparoUUID uuid.UUID
	Descricao  string
}
type ReparoAdicionalRecusado struct {
	EventoOS
	ReparoUUID uuid.UUID
	Descricao  string
}
type OSFinalizada struct {
	EventoOS
	TempoExecucao time.Duration
}
type VeiculoEntregue struct{ EventoOS }

func (os *OrdemServico) eventoOS(agora time.Time) EventoOS {
	return EventoOS{OSUUID: os.UUID, ClienteID: os.ClienteID, Em: agora}
}

// comOS preenche OS e cliente no evento (valor), devolvendo a cópia.
func comOS(e domain.Evento, osUUID uuid.UUID, clienteID int64) domain.Evento {
	switch ev := e.(type) {
	case OSRecebida:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case DiagnosticoIniciado:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case OrcamentoEnviadoAoCliente:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case OrcamentoAprovadoPeloCliente:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case OrcamentoRejeitadoPeloCliente:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case ExecucaoIniciada:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case ReparoAdicionalIdentificado:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case ReparoAdicionalAutorizado:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case ReparoAdicionalRecusado:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case OSFinalizada:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	case VeiculoEntregue:
		ev.OSUUID, ev.ClienteID = osUUID, clienteID
		return ev
	}
	return e
}
