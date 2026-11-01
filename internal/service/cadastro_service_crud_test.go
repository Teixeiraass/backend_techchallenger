package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/cliente"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/veiculo"
	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------- clientes

func TestCriarCliente(t *testing.T) {
	ctx := context.Background()
	req := dto.CriarClienteRequest{Nome: "Maria", Documento: "529.982.247-25", Email: "m@x.com"}

	t.Run("OK", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().CreateCliente(gomock.Any(), db.CreateClienteParams{
			Nome: "Maria", Documento: "52998224725", TipoDocumento: "CPF", Email: "m@x.com",
		}).Return(db.Cliente{Documento: "52998224725"}, nil)
		c, err := service.NewClienteService(store).CriarCliente(ctx, req)
		require.NoError(t, err)
		require.Equal(t, "52998224725", c.Documento)
	})
	t.Run("CNPJ", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().CreateCliente(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.CreateClienteParams) (db.Cliente, error) {
				require.Equal(t, "CNPJ", arg.TipoDocumento)
				return db.Cliente{}, nil
			})
		_, err := service.NewClienteService(store).CriarCliente(ctx, dto.CriarClienteRequest{Nome: "X", Documento: "11.222.333/0001-81"})
		require.NoError(t, err)
	})
	t.Run("InvalidDocument", func(t *testing.T) {
		store := crudNewStore(t)
		_, err := service.NewClienteService(store).CriarCliente(ctx, dto.CriarClienteRequest{Documento: "12345678900"})
		require.ErrorIs(t, err, cliente.ErrDocumentoInvalido)
	})
	t.Run("Duplicate", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().CreateCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, crudUniqueErr)
		_, err := service.NewClienteService(store).CriarCliente(ctx, req)
		require.ErrorIs(t, err, service.ErrDocumentoDuplicado)
	})
	t.Run("DBError", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().CreateCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrConnDone)
		_, err := service.NewClienteService(store).CriarCliente(ctx, req)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestBuscarListarAtualizarRemoverCliente(t *testing.T) {
	ctx := context.Background()
	c := crudCliente()
	store := crudNewStore(t)
	svc := service.NewClienteService(store)

	store.EXPECT().GetCliente(gomock.Any(), c.Uuid).Return(c, nil)
	got, err := svc.BuscarCliente(ctx, c.Uuid)
	require.NoError(t, err)
	require.Equal(t, c.Uuid, got.Uuid)

	store.EXPECT().GetCliente(gomock.Any(), c.Uuid).Return(db.Cliente{}, sql.ErrNoRows)
	_, err = svc.BuscarCliente(ctx, c.Uuid)
	require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)

	store.EXPECT().GetClienteByDocumento(gomock.Any(), c.Documento).Return(c, nil)
	_, err = svc.BuscarClientePorDocumento(ctx, c.Documento)
	require.NoError(t, err)

	store.EXPECT().GetClienteByDocumento(gomock.Any(), c.Documento).Return(db.Cliente{}, sql.ErrNoRows)
	_, err = svc.BuscarClientePorDocumento(ctx, c.Documento)
	require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)

	_, err = svc.BuscarClientePorDocumento(ctx, "123")
	require.ErrorIs(t, err, cliente.ErrDocumentoInvalido)

	store.EXPECT().ListClientes(gomock.Any(), db.ListClientesParams{Busca: "ma", Limite: 20, Deslocamento: 0}).Return([]db.Cliente{c}, nil)
	list, err := svc.ListarClientes(ctx, dto.ListClientesQuery{Busca: "ma", PaginacaoQuery: dto.PaginacaoQuery{Pagina: 1, Tamanho: 20}})
	require.NoError(t, err)
	require.Len(t, list, 1)

	upd := dto.AtualizarClienteRequest{Nome: "Novo", Email: "n@x.com", Telefone: "1"}
	store.EXPECT().UpdateCliente(gomock.Any(), db.UpdateClienteParams{Uuid: c.Uuid, Nome: "Novo", Email: "n@x.com", Telefone: "1"}).Return(c, nil)
	_, err = svc.AtualizarCliente(ctx, c.Uuid, upd)
	require.NoError(t, err)

	store.EXPECT().UpdateCliente(gomock.Any(), gomock.Any()).Return(db.Cliente{}, sql.ErrNoRows)
	_, err = svc.AtualizarCliente(ctx, c.Uuid, upd)
	require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)

	store.EXPECT().DeleteCliente(gomock.Any(), c.Uuid).Return(c.ID, nil)
	require.NoError(t, svc.RemoverCliente(ctx, c.Uuid))

	store.EXPECT().DeleteCliente(gomock.Any(), c.Uuid).Return(int64(0), crudFKErr)
	require.ErrorIs(t, svc.RemoverCliente(ctx, c.Uuid), service.ErrClienteEmUso)

	store.EXPECT().DeleteCliente(gomock.Any(), c.Uuid).Return(int64(0), sql.ErrNoRows)
	require.ErrorIs(t, svc.RemoverCliente(ctx, c.Uuid), service.ErrClienteNaoEncontrado)
}

// ---------------------------------------------------------------- veículos

func TestCriarVeiculo(t *testing.T) {
	ctx := context.Background()
	dono := crudCliente()
	req := dto.CriarVeiculoRequest{ClienteUuid: dono.Uuid, Placa: "abc-1d23", Marca: " VW ", Modelo: "Gol ", Ano: 2019}

	t.Run("OK", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetCliente(gomock.Any(), dono.Uuid).Return(dono, nil)
		store.EXPECT().CreateVeiculo(gomock.Any(), db.CreateVeiculoParams{
			ClienteID: dono.ID, Placa: "ABC1D23", Marca: "VW", Modelo: "Gol", Ano: 2019,
		}).Return(db.Veiculo{Placa: "ABC1D23"}, nil)
		v, err := service.NewVeiculoService(store).CriarVeiculo(ctx, req)
		require.NoError(t, err)
		require.Equal(t, dono.Uuid, v.ClienteUuid)
		require.Equal(t, "ABC1D23", v.Placa)
	})
	t.Run("InvalidPlate", func(t *testing.T) {
		bad := req
		bad.Placa = "12345"
		_, err := service.NewVeiculoService(crudNewStore(t)).CriarVeiculo(ctx, bad)
		require.ErrorIs(t, err, veiculo.ErrPlacaInvalida)
	})
	t.Run("InvalidYear", func(t *testing.T) {
		bad := req
		bad.Ano = int32(time.Now().Year() + 5)
		_, err := service.NewVeiculoService(crudNewStore(t)).CriarVeiculo(ctx, bad)
		require.ErrorIs(t, err, veiculo.ErrAnoInvalido)
	})
	t.Run("ClienteNotFound", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetCliente(gomock.Any(), dono.Uuid).Return(db.Cliente{}, sql.ErrNoRows)
		_, err := service.NewVeiculoService(store).CriarVeiculo(ctx, req)
		require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)
	})
	t.Run("DuplicatePlate", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetCliente(gomock.Any(), dono.Uuid).Return(dono, nil)
		store.EXPECT().CreateVeiculo(gomock.Any(), gomock.Any()).Return(db.Veiculo{}, crudUniqueErr)
		_, err := service.NewVeiculoService(store).CriarVeiculo(ctx, req)
		require.ErrorIs(t, err, service.ErrPlacaDuplicada)
	})
	t.Run("DBError", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetCliente(gomock.Any(), dono.Uuid).Return(dono, nil)
		store.EXPECT().CreateVeiculo(gomock.Any(), gomock.Any()).Return(db.Veiculo{}, sql.ErrConnDone)
		_, err := service.NewVeiculoService(store).CriarVeiculo(ctx, req)
		require.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestBuscarListarVeiculo(t *testing.T) {
	ctx := context.Background()
	dono := crudCliente()
	v := db.Veiculo{ID: 1, Uuid: uuid.New(), Placa: "ABC1D23"}
	store := crudNewStore(t)
	svc := service.NewVeiculoService(store)

	store.EXPECT().GetVeiculo(gomock.Any(), v.Uuid).Return(db.GetVeiculoRow{Veiculo: v, ClienteUuid: dono.Uuid}, nil)
	got, err := svc.BuscarVeiculo(ctx, v.Uuid)
	require.NoError(t, err)
	require.Equal(t, dono.Uuid, got.ClienteUuid)

	store.EXPECT().GetVeiculo(gomock.Any(), v.Uuid).Return(db.GetVeiculoRow{}, sql.ErrNoRows)
	_, err = svc.BuscarVeiculo(ctx, v.Uuid)
	require.ErrorIs(t, err, service.ErrVeiculoNaoEncontrado)

	// sem filtro
	store.EXPECT().ListVeiculos(gomock.Any(), db.ListVeiculosParams{Limite: 20}).
		Return([]db.ListVeiculosRow{{Veiculo: v, ClienteUuid: dono.Uuid}}, nil)
	list, err := svc.ListarVeiculos(ctx, dto.ListVeiculosQuery{})
	require.NoError(t, err)
	require.Len(t, list, 1)

	// filtro por cliente
	store.EXPECT().GetCliente(gomock.Any(), dono.Uuid).Return(dono, nil)
	store.EXPECT().ListVeiculos(gomock.Any(), db.ListVeiculosParams{
		ClienteID: sql.NullInt64{Int64: dono.ID, Valid: true}, Limite: 20,
	}).Return([]db.ListVeiculosRow{}, nil)
	list, err = svc.ListarVeiculos(ctx, dto.ListVeiculosQuery{ClienteUuid: dono.Uuid.String()})
	require.NoError(t, err)
	require.Empty(t, list)

	_, err = svc.ListarVeiculos(ctx, dto.ListVeiculosQuery{ClienteUuid: "not-a-uuid"})
	require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)

	store.EXPECT().GetCliente(gomock.Any(), dono.Uuid).Return(db.Cliente{}, sql.ErrNoRows)
	_, err = svc.ListarVeiculos(ctx, dto.ListVeiculosQuery{ClienteUuid: dono.Uuid.String()})
	require.ErrorIs(t, err, service.ErrClienteNaoEncontrado)

	store.EXPECT().ListVeiculos(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
	_, err = svc.ListarVeiculos(ctx, dto.ListVeiculosQuery{})
	require.ErrorIs(t, err, sql.ErrConnDone)
}

func TestAtualizarRemoverVeiculo(t *testing.T) {
	ctx := context.Background()
	dono := crudCliente()
	v := db.Veiculo{ID: 1, Uuid: uuid.New(), Placa: "ABC1D23"}
	req := dto.AtualizarVeiculoRequest{Placa: "XYZ1234", Marca: "Fiat", Modelo: "Uno", Ano: 2010}

	t.Run("OK", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetVeiculo(gomock.Any(), v.Uuid).Return(db.GetVeiculoRow{Veiculo: v, ClienteUuid: dono.Uuid}, nil)
		store.EXPECT().UpdateVeiculo(gomock.Any(), db.UpdateVeiculoParams{Uuid: v.Uuid, Placa: "XYZ1234", Marca: "Fiat", Modelo: "Uno", Ano: 2010}).
			Return(db.Veiculo{Uuid: v.Uuid, Placa: "XYZ1234"}, nil)
		got, err := service.NewVeiculoService(store).AtualizarVeiculo(ctx, v.Uuid, req)
		require.NoError(t, err)
		require.Equal(t, dono.Uuid, got.ClienteUuid)
	})
	t.Run("InvalidPlate", func(t *testing.T) {
		bad := req
		bad.Placa = "?"
		_, err := service.NewVeiculoService(crudNewStore(t)).AtualizarVeiculo(ctx, v.Uuid, bad)
		require.ErrorIs(t, err, veiculo.ErrPlacaInvalida)
	})
	t.Run("InvalidYear", func(t *testing.T) {
		bad := req
		bad.Ano = 1800
		_, err := service.NewVeiculoService(crudNewStore(t)).AtualizarVeiculo(ctx, v.Uuid, bad)
		require.ErrorIs(t, err, veiculo.ErrAnoInvalido)
	})
	t.Run("NotFound", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetVeiculo(gomock.Any(), v.Uuid).Return(db.GetVeiculoRow{}, sql.ErrNoRows)
		_, err := service.NewVeiculoService(store).AtualizarVeiculo(ctx, v.Uuid, req)
		require.ErrorIs(t, err, service.ErrVeiculoNaoEncontrado)
	})
	t.Run("DuplicatePlate", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetVeiculo(gomock.Any(), v.Uuid).Return(db.GetVeiculoRow{Veiculo: v}, nil)
		store.EXPECT().UpdateVeiculo(gomock.Any(), gomock.Any()).Return(db.Veiculo{}, crudUniqueErr)
		_, err := service.NewVeiculoService(store).AtualizarVeiculo(ctx, v.Uuid, req)
		require.ErrorIs(t, err, service.ErrPlacaDuplicada)
	})
	t.Run("UpdateNotFound", func(t *testing.T) {
		store := crudNewStore(t)
		store.EXPECT().GetVeiculo(gomock.Any(), v.Uuid).Return(db.GetVeiculoRow{Veiculo: v}, nil)
		store.EXPECT().UpdateVeiculo(gomock.Any(), gomock.Any()).Return(db.Veiculo{}, sql.ErrNoRows)
		_, err := service.NewVeiculoService(store).AtualizarVeiculo(ctx, v.Uuid, req)
		require.ErrorIs(t, err, service.ErrVeiculoNaoEncontrado)
	})
	t.Run("Remove", func(t *testing.T) {
		store := crudNewStore(t)
		svc := service.NewVeiculoService(store)
		store.EXPECT().DeleteVeiculo(gomock.Any(), v.Uuid).Return(v.ID, nil)
		require.NoError(t, svc.RemoverVeiculo(ctx, v.Uuid))
		store.EXPECT().DeleteVeiculo(gomock.Any(), v.Uuid).Return(int64(0), crudFKErr)
		require.ErrorIs(t, svc.RemoverVeiculo(ctx, v.Uuid), service.ErrVeiculoEmUso)
		store.EXPECT().DeleteVeiculo(gomock.Any(), v.Uuid).Return(int64(0), sql.ErrNoRows)
		require.ErrorIs(t, svc.RemoverVeiculo(ctx, v.Uuid), service.ErrVeiculoNaoEncontrado)
	})
}

// ---------------------------------------------------------------- serviços

func TestServicoService(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	criar := dto.CriarServicoRequest{Nome: " Troca de óleo ", PrecoCentavos: 8000, TempoEstimadoMinutos: 30}
	atualizar := dto.AtualizarServicoRequest{Nome: "Troca", PrecoCentavos: 9000, Ativo: crudBool(false)}

	store := crudNewStore(t)
	svc := service.NewServicoService(store)

	store.EXPECT().CreateServico(gomock.Any(), db.CreateServicoParams{Nome: "Troca de óleo", PrecoCentavos: 8000, TempoEstimadoMinutos: 30}).
		Return(db.Servico{Nome: "Troca de óleo"}, nil)
	s, err := svc.CriarServico(ctx, criar)
	require.NoError(t, err)
	require.Equal(t, "Troca de óleo", s.Nome)

	store.EXPECT().CreateServico(gomock.Any(), gomock.Any()).Return(db.Servico{}, crudUniqueErr)
	_, err = svc.CriarServico(ctx, criar)
	require.ErrorIs(t, err, service.ErrServicoDuplicado)

	store.EXPECT().CreateServico(gomock.Any(), gomock.Any()).Return(db.Servico{}, sql.ErrConnDone)
	_, err = svc.CriarServico(ctx, criar)
	require.ErrorIs(t, err, sql.ErrConnDone)

	store.EXPECT().GetServico(gomock.Any(), id).Return(db.Servico{Uuid: id}, nil)
	_, err = svc.BuscarServico(ctx, id)
	require.NoError(t, err)

	store.EXPECT().GetServico(gomock.Any(), id).Return(db.Servico{}, sql.ErrNoRows)
	_, err = svc.BuscarServico(ctx, id)
	require.ErrorIs(t, err, service.ErrServicoNaoEncontrado)

	store.EXPECT().ListServicos(gomock.Any(), db.ListServicosParams{ApenasAtivos: true, Busca: "tr", Limite: 20}).Return([]db.Servico{{}}, nil)
	list, err := svc.ListarServicos(ctx, dto.ListServicosQuery{ApenasAtivos: true, Busca: "tr"})
	require.NoError(t, err)
	require.Len(t, list, 1)

	store.EXPECT().UpdateServico(gomock.Any(), db.UpdateServicoParams{Uuid: id, Nome: "Troca", PrecoCentavos: 9000, Ativo: false}).
		Return(db.Servico{Uuid: id}, nil)
	_, err = svc.AtualizarServico(ctx, id, atualizar)
	require.NoError(t, err)

	store.EXPECT().UpdateServico(gomock.Any(), gomock.Any()).Return(db.Servico{}, crudUniqueErr)
	_, err = svc.AtualizarServico(ctx, id, atualizar)
	require.ErrorIs(t, err, service.ErrServicoDuplicado)

	store.EXPECT().UpdateServico(gomock.Any(), gomock.Any()).Return(db.Servico{}, sql.ErrNoRows)
	_, err = svc.AtualizarServico(ctx, id, atualizar)
	require.ErrorIs(t, err, service.ErrServicoNaoEncontrado)

	store.EXPECT().InativarServico(gomock.Any(), id).Return(db.Servico{Uuid: id, Ativo: false}, nil)
	inativo, err := svc.InativarServico(ctx, id)
	require.NoError(t, err)
	require.False(t, inativo.Ativo)

	store.EXPECT().InativarServico(gomock.Any(), id).Return(db.Servico{}, sql.ErrNoRows)
	_, err = svc.InativarServico(ctx, id)
	require.ErrorIs(t, err, service.ErrServicoNaoEncontrado)
}
