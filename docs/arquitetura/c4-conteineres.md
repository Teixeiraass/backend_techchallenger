# C4 · Nível 2 · Contêineres

O que roda dentro do sistema da oficina. Um contêiner no C4 é uma unidade executável ou de dados, não necessariamente um contêiner Docker (aqui, por coincidência, cada um é um serviço do `docker-compose.yaml`).

```mermaid
C4Container
    title Oficina Mecânica · Contêineres

    Person(cliente, "Cliente", "Acompanha a OS pelo link, sem login")
    Person(oficina_pessoas, "Atendente / Mecânico / Administrador", "Operam a oficina com JWT e papéis")

    System_Boundary(sistema, "Sistema da Oficina") {
        Container(api, "API da Oficina", "Go 1.26, Gin, sqlc", "Monólito em camadas com DDD: rotas → middleware → handlers → serviços (casos de uso) → domínio (agregados OrdemServico e Peca, objetos de valor, eventos) → repositórios. Expõe REST + Swagger em /api/v1.")
        ContainerDb(db, "Banco de dados", "PostgreSQL 16", "Clientes, veículos, serviços, peças e movimentos de estoque, ordens de serviço, itens, reparos adicionais, histórico de status, usuários e sessões. Schema versionado por golang-migrate; seed de demonstração separado (db/seed).")
        Container(migrate, "Migrações", "golang-migrate (no start.sh da imagem)", "Aplica db/migration no boot e, só com SEED_DEMO=true, o seed de demonstração.")
    }

    System_Ext(email, "E-mail", "Provedor atrás da porta notification.Sender (LogSender no MVP)")

    Rel(cliente, api, "GET andamento (X-Documento), POST aprovar/rejeitar, autorizar/recusar", "HTTPS/JSON")
    Rel(oficina_pessoas, api, "Login, cadastros, OS, catálogo, estoque", "HTTPS/JSON + Bearer JWT")
    Rel(api, db, "Lê e grava em transações (SELECT … FOR NO KEY UPDATE na OS, UPDATE condicional no estoque)", "database/sql + lib/pq, TCP 5432")
    Rel(migrate, db, "Cria e evolui o schema", "SQL")
    Rel(api, email, "Políticas de notificação reagem a eventos de domínio depois do commit", "notification.Sender")

    UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")
```

## Dentro da API (camadas)

| Camada | Pacotes | Responsabilidade |
|---|---|---|
| HTTP | `internal/routes`, `internal/middleware`, `internal/handler`, `internal/dto` | Rotas e papéis por rota, autenticação JWT, limite de tentativas, limite de corpo, log de acesso sem dado pessoal, binding e mapeamento erro → status, Swagger |
| Aplicação | `internal/service` | Um método por comando: abre a unidade de trabalho, carrega o agregado pelo repositório, chama a raiz, salva, publica os eventos; consultas pelo read model; políticas como handlers de eventos |
| Domínio | `internal/domain/...` | Agregados `OrdemServico` (itens, reparos, máquina de estados, orçamento) e `Peca` (saldo, movimentos), objetos de valor (`Documento`, `Placa`, `Status`), eventos de domínio e as portas de repositório |
| Infraestrutura | `internal/repository`, `internal/readmodel`, `internal/eventos`, `db/sqlc`, `db/migration`, `notification`, `token` | Implementação das portas com sqlc, projeções de leitura, despachante de eventos, acesso ao PostgreSQL e transações, e-mail (log), JWT |

## Decisões relacionadas

- [ADR 0001 · PostgreSQL](../adr/0001-postgresql.md)
- [ADR 0002 · JWT como autenticação](../adr/0002-jwt-com-opcao-paseto.md)
- [ADR 0003 · Estoque sem reserva](../adr/0003-estoque-sem-reserva.md)
- [ADR 0004 · Rejeição volta para diagnóstico](../adr/0004-rejeicao-volta-para-diagnostico.md)
- [ADR 0005 · Acompanhamento sem login](../adr/0005-acompanhamento-sem-login.md)
- [ADR 0006 · Eventos de domínio em processo](../adr/0006-eventos-de-dominio-em-processo.md)

Nível anterior: [contexto](c4-contexto.md). Requisitos não funcionais: [requisitos-nao-funcionais.md](requisitos-nao-funcionais.md). Débito técnico: [debito-tecnico.md](debito-tecnico.md).
