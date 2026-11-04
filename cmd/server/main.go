package main

import (
	"database/sql"
	"log"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	_ "github.com/Teixeiraass/backend_techchallenger/docs"
	"github.com/Teixeiraass/backend_techchallenger/internal/eventos"
	"github.com/Teixeiraass/backend_techchallenger/internal/handler"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	_ "github.com/lib/pq"
)

// @title           Oficina Mecânica API
// @version         1.0
// @description     MVP do back-end do Sistema Integrado de Atendimento e Execução de Serviços de uma oficina mecânica (Tech Challenge SOAT - Fase 1).
// @description     Gestão de clientes, veículos, catálogo de serviços, peças e insumos com controle de estoque, e o ciclo de vida completo da Ordem de Serviço (orçamento, aprovação, execução, reparo adicional e entrega).
// @description     Rotas administrativas exigem token Bearer (POST /usuarios/login). As rotas de acompanhamento do cliente são públicas e usam o CPF/CNPJ do dono da OS.
// @contact.name    Guilherme Teixeira
// @license.name    MIT
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Informe: Bearer {access_token}
func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	if err := conn.Ping(); err != nil {
		log.Fatal("cannot reach db:", err)
	}

	store := db.NewStore(conn)

	// Eventos de domínio: os serviços publicam depois do commit e as políticas
	// do event storming reagem como handlers (notificar o cliente, pedir
	// reposição ao administrador). O envio de e-mail fica atrás da porta
	// notification.Sender.
	despachante := eventos.NovoDespachante()
	server, err := handler.NewServer(config, store, despachante)
	if err != nil {
		log.Fatal("cannot create server: ", err)
	}
	service.AssinarPoliticas(despachante, service.NovasPoliticas(server.Catalogo(), notification.NewLogSender(), config))

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
