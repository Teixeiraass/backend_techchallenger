package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/eventos"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	mockrepo "github.com/Teixeiraass/backend_techchallenger/internal/repository/mock"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fxPoliticas struct {
	catalogo    *mockrepo.MockCatalogo
	sender      *fxSender
	despachante *eventos.Despachante
	cliente     readmodel.Cliente
}

func novasPoliticas(t *testing.T) *fxPoliticas {
	f := &fxPoliticas{
		catalogo:    mockrepo.NewMockCatalogo(gomock.NewController(t)),
		sender:      &fxSender{},
		despachante: eventos.NovoDespachante(),
		cliente:     readmodel.Cliente{ID: 1, Email: "maria@email.com"},
	}
	service.AssinarPoliticas(f.despachante, service.NovasPoliticas(f.catalogo, f.sender, fxConfig()))
	return f
}

func (f *fxPoliticas) publicar(e ...domain.Evento) {
	f.despachante.Publicar(context.Background(), e)
}

func eventoOS(clienteID int64) ordemservico.EventoOS {
	return ordemservico.EventoOS{OSUUID: uuid.New(), ClienteID: clienteID}
}

func TestPoliticasNotificamOCliente(t *testing.T) {
	f := novasPoliticas(t)
	f.catalogo.EXPECT().ClientePorID(gomock.Any(), int64(1)).Times(3).Return(f.cliente, nil)

	f.publicar(
		ordemservico.OrcamentoEnviadoAoCliente{EventoOS: eventoOS(1), ValorTotalCentavos: 15000},
		ordemservico.ReparoAdicionalIdentificado{EventoOS: eventoOS(1), Descricao: "Disco empenado"},
		ordemservico.OSFinalizada{EventoOS: eventoOS(1)},
		ordemservico.VeiculoEntregue{EventoOS: eventoOS(1)}, // sem política
	)
	require.Len(t, f.sender.msgs, 3)
	require.Equal(t, "maria@email.com", f.sender.msgs[0].To)
	require.Equal(t, "Orçamento disponível para aprovação", f.sender.msgs[0].Subject)
	require.Contains(t, f.sender.msgs[0].Body, "R$ 150,00")
	require.Equal(t, "Reparo adicional aguardando autorização", f.sender.msgs[1].Subject)
	require.Contains(t, f.sender.msgs[1].Body, "Disco empenado")
	require.Equal(t, "Seu veículo está pronto", f.sender.msgs[2].Subject)
}

// A notificação é por e-mail (linguagem ubíqua §1.2): cliente sem e-mail não
// recebe aviso, e a operação que gerou o evento continua válida.
func TestPoliticasClienteSemEmailNaoNotifica(t *testing.T) {
	f := novasPoliticas(t)
	f.cliente.Email = ""
	f.catalogo.EXPECT().ClientePorID(gomock.Any(), int64(1)).Return(f.cliente, nil)

	f.publicar(ordemservico.OSFinalizada{EventoOS: eventoOS(1)})
	require.Empty(t, f.sender.msgs)
}

func TestPoliticasFalhasFicamNoLog(t *testing.T) {
	f := novasPoliticas(t)
	f.catalogo.EXPECT().ClientePorID(gomock.Any(), int64(1)).Return(readmodel.Cliente{}, sql.ErrConnDone)
	f.publicar(ordemservico.OSFinalizada{EventoOS: eventoOS(1)})
	require.Empty(t, f.sender.msgs)

	f.sender.err = errors.New("smtp fora do ar")
	f.catalogo.EXPECT().ClientePorID(gomock.Any(), int64(1)).Return(f.cliente, nil)
	f.publicar(ordemservico.OSFinalizada{EventoOS: eventoOS(1)})
	require.Len(t, f.sender.msgs, 1, "a tentativa aconteceu; o erro fica no log")
}

func TestPoliticasPedemReposicaoAoAdministrador(t *testing.T) {
	f := novasPoliticas(t)
	peca := estoque.EventoPeca{PecaID: 5, Codigo: "FLT-01", NomePeca: "Filtro", Unidade: "UN"}

	f.publicar(
		estoque.EstoqueInsuficienteDetectado{EventoPeca: peca, OrdemServicoID: 77, Necessario: 3, Saldo: 1},
		estoque.EstoqueBaixoDetectado{EventoPeca: peca, Saldo: 1, EstoqueMinimo: 2},
		estoque.EstoqueBaixado{EventoPeca: peca}, // sem política
	)
	require.Len(t, f.sender.msgs, 2)
	require.Equal(t, "admin@oficina.com", f.sender.msgs[0].To)
	require.Equal(t, "Pedido de reposição de estoque", f.sender.msgs[0].Subject)
	require.Contains(t, f.sender.msgs[0].Body, "FLT-01")
	require.Contains(t, f.sender.msgs[0].Body, "OS 77")
	require.Contains(t, f.sender.msgs[1].Body, "saldo 1 UN")
}

// Um evento de outro tipo com o mesmo nome é ignorado pelo handler.
func TestPoliticasIgnoramEventosDeOutroTipo(t *testing.T) {
	f := novasPoliticas(t)
	p := service.NovasPoliticas(f.catalogo, f.sender, fxConfig())
	outro := ordemservico.VeiculoEntregue{EventoOS: eventoOS(1)}
	require.NoError(t, p.NotificarOrcamentoEnviado(context.Background(), outro))
	require.NoError(t, p.NotificarReparoIdentificado(context.Background(), outro))
	require.NoError(t, p.NotificarOSFinalizada(context.Background(), outro))
	require.NoError(t, p.PedirReposicaoFaltante(context.Background(), outro))
	require.NoError(t, p.PedirReposicaoEstoqueBaixo(context.Background(), outro))
	require.Empty(t, f.sender.msgs)
}
