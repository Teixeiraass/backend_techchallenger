# Oficina Mecânica — Back-end (Tech Challenge SOAT, Fase 1)

MVP do back-end do **Sistema Integrado de Atendimento e Execução de Serviços** de uma oficina mecânica de médio porte. Feito em Go, com Domain-Driven Design, PostgreSQL, Docker e testes automatizados.

- **Documentação DDD (Miro):** _[inserir link público do Miro]_
- **Documentação DDD exportada:** [docs/ddd/](docs/ddd/) — [linguagem ubíqua e relação com o código](docs/ddd/linguagem-ubiqua.md), [mapa de contexto](docs/ddd/mapa-de-contexto.md), [diagramas](docs/ddd/diagramas.pdf), [event storming](docs/ddd/event-storming.pdf)
- **Vídeo da entrega:** _[inserir link do YouTube]_
- **Análise de vulnerabilidades:** [docs/security/README.md](docs/security/README.md)
- **Evidência de testes e cobertura:** [docs/testes/](docs/testes/)

---

## Objetivo

Hoje a oficina controla atendimento, diagnóstico, execução e entrega com anotações e planilhas. Isso causa erros de priorização, falhas no controle de peças, dificuldade para acompanhar os serviços, perda de histórico e um fluxo de orçamento ineficiente. Este projeto entrega a primeira versão do back-end que organiza esse processo:

- a oficina registra clientes, veículos, serviços, peças e insumos, com controle de estoque;
- cada atendimento vira uma **Ordem de Serviço (OS)** com orçamento automático e ciclo de vida controlado;
- o cliente acompanha a OS e aprova o orçamento (e reparos adicionais) pela API, sem login;
- a gestão consulta o tempo médio de execução dos serviços.

## Solução em resumo

1. O **atendente** cadastra o cliente (CPF/CNPJ validado e e-mail obrigatório) e o veículo (placa validada) e abre a OS com os serviços e peças. A OS nasce **Recebida** e o orçamento é calculado na hora.
2. O **mecânico** inicia o **diagnóstico**, ajusta os itens e envia o orçamento. A OS passa para **Aguardando aprovação** e o cliente é notificado por e-mail.
3. O **cliente** aprova ou rejeita pelo link de acompanhamento, informando o CPF/CNPJ:
   - **Aprovou:** o sistema coloca a OS **Em execução** e baixa o estoque das peças na mesma transação. Se faltar peça, a aprovação fica registrada, o administrador recebe um pedido de reposição e, depois da reposição, o atendente ou o mecânico inicia a execução.
   - **Rejeitou:** a OS volta para diagnóstico. Dali sai um novo orçamento ou o veículo é entregue sem execução.
4. Durante a execução, o mecânico pode registrar um **reparo adicional**. Ele só entra na OS se o cliente autorizar.
5. O mecânico **finaliza** a OS, e o tempo de execução fica registrado. O atendente **entrega** o veículo.

Status: `RECEBIDA → EM_DIAGNOSTICO → AGUARDANDO_APROVACAO → EM_EXECUCAO → FINALIZADA → ENTREGUE`. Transições fora da ordem são recusadas com HTTP 409. As únicas exceções vêm da rejeição do orçamento (detalhes em [linguagem-ubiqua.md §1.3](docs/ddd/linguagem-ubiqua.md)).

## Arquitetura

**Monólito em camadas**, com o domínio isolado de banco e HTTP:

```
HTTP (Gin)                    internal/routes + internal/handler   rotas, binding, mapeamento erro → status
   │                          internal/middleware                  autenticação (JWT/PASETO) e papéis
   ▼
Aplicação (casos de uso)      internal/service                     um método por comando; abre a transação,
   │                                                               carrega dados, aplica o domínio, persiste
   ▼
Domínio (DDD)                 internal/domain/...                  agregados, objetos de valor e regras puras
   │                                                               (sem dependência de banco ou framework)
   ▼
Infraestrutura                db/sqlc (sqlc + Store.ExecTx)        acesso ao PostgreSQL e transações
                              db/migration (golang-migrate)        schema versionado + dados de demonstração
                              notification/                        porta de notificação (adaptador de log)
                              token/                               JWT HS256 (padrão) e PASETO v2
```

Elementos de DDD no código:

| Bounded context | Agregados | Objetos de valor / regras |
|---|---|---|
| Cadastro | Cliente, Veículo | `cliente.Documento` (CPF/CNPJ), `veiculo.Placa` |
| Ordem de Serviço | OrdemServico (itens, reparos, histórico) | `ordemservico.Status` (máquina de estados), `Orcamento`, `StatusOrcamento` |
| Catálogo e estoque | Peça (peça ou insumo), Serviço | `estoque` (saldo nunca negativo, reposição > 0, estoque mínimo) |
| Identidade e acesso | Usuário | papéis `ADMINISTRADOR`, `ATENDENTE`, `MECANICO` (`middleware.RequireRole`) |

Tipos de subdomínio e padrões de integração entre os contextos: [mapa de contexto](docs/ddd/mapa-de-contexto.md). Fluxos de negócio: [event storming](docs/ddd/event-storming.pdf).

**Transações:** todo comando que altera mais de um registro roda dentro de `Store.ExecTx`, como abrir a OS com itens, aprovar e baixar estoque, ou autorizar um reparo. A OS fica travada com `SELECT ... FOR NO KEY UPDATE` durante o comando. A baixa usa `UPDATE ... WHERE quantidade_estoque >= qtd`, e a tabela tem `CHECK (quantidade_estoque >= 0)`. Assim, duas OS concorrentes nunca deixam o saldo negativo; há um teste de integração com 10 baixas simultâneas que prova isso.

## Tecnologias

| Camada | Tecnologia |
|---|---|
| Linguagem | Go 1.26 (toolchain 1.26.10) |
| HTTP | [Gin](https://github.com/gin-gonic/gin) |
| Banco de dados | PostgreSQL 16 |
| Acesso a dados | [sqlc](https://sqlc.dev/) (SQL tipado, sem ORM) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Autenticação | JWT HS256 (padrão) ou PASETO v2 (`TOKEN_TYPE`), senhas com bcrypt |
| Configuração | [Viper](https://github.com/spf13/viper) |
| Documentação da API | [Swaggo](https://github.com/swaggo/swag) (Swagger/OpenAPI 2.0) |
| Testes | `testing`, testify, gomock |
| Segurança | govulncheck, gosec, Trivy |
| Containers | Docker + Docker Compose |
| CI | GitHub Actions |

## Banco de dados: PostgreSQL

O domínio é relacional e transacional. Por isso a escolha foi o PostgreSQL:

- **Consistência forte (ACID):** aprovar o orçamento, mudar o status e baixar o estoque precisa acontecer de forma atômica. Se faltar uma peça, nada pode ser gravado. Transações e `SELECT ... FOR UPDATE` resolvem isso de forma nativa.
- **Integridade garantida pelo banco:** chaves estrangeiras (cliente → veículo → OS → itens), `UNIQUE` (CPF/CNPJ, placa, código da peça) e `CHECK` (status válidos, saldo ≥ 0) protegem as invariantes mesmo se a aplicação tiver um bug.
- **Concorrência no estoque:** o `UPDATE` condicional permite que várias OS disputem a mesma peça sem lock global e sem saldo negativo.
- **Consultas analíticas:** o tempo médio de execução sai de agregações (`AVG`, `EXTRACT(EPOCH ...)`, `GROUP BY`) direto em SQL.
- **Ecossistema:** casa bem com sqlc (queries tipadas, checadas na geração) e golang-migrate, é open source, roda em Docker e está disponível em qualquer nuvem.

Um banco de documentos (MongoDB, por exemplo) facilitaria guardar a OS como um único documento. Em troca, perderia as garantias entre agregados (OS × estoque), que são justamente o ponto crítico deste domínio.

## Pré-requisitos

- **Para rodar com Docker (recomendado):** Docker e Docker Compose.
- **Para rodar localmente:** Go 1.26+, Docker (para o PostgreSQL), [golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) e `make`.
- **Opcionais:** [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html), [mockgen](https://github.com/golang/mock) e [swag](https://github.com/swaggo/swag), só para gerar código de novo.

## Como executar

### Com Docker Compose (ambiente completo)

```bash
docker compose up --build -d
```

O compose sobe o PostgreSQL, espera ele ficar saudável, aplica as migrations (schema + dados de demonstração) e inicia a API em **http://localhost:8080**.

- Health check: `curl http://localhost:8080/api/v1/helper`
- Se a porta 8080 estiver ocupada: `API_PORT=8090 docker compose up --build -d`
- Para parar: `docker compose down`. Para zerar o banco: `docker compose down -v`

### Localmente (API fora do container)

```bash
cp app.env.example app.env        # configuração local
docker compose up -d postgres     # só o banco (porta 5433)
make migrateup                    # aplica as migrations
make server                       # API em http://localhost:8080
```

## Documentação da API (Swagger)

Com a API rodando, abra **http://localhost:8080/swagger/index.html**.

1. Chame `POST /usuarios/login` com uma das credenciais abaixo.
2. Copie o `access_token`, clique em **Authorize** e informe `Bearer <access_token>`.

Os arquivos OpenAPI ficam em [docs/swagger.yaml](docs/swagger.yaml) e [docs/swagger.json](docs/swagger.json). Para gerar de novo: `make swag`.

## Usuários de demonstração e autenticação

A migration `000005_seed_dados_demo` cria:

| Papel | E-mail | Senha | Pode |
|---|---|---|---|
| ADMINISTRADOR | `admin@oficina.com` | `admin123` | tudo, incluindo catálogo, estoque, usuários e tempo médio |
| ATENDENTE | `atendente@oficina.com` | `atendente123` | clientes, veículos, abrir OS, itens, iniciar execução após reposição, entregar |
| MECANICO | `mecanico@oficina.com` | `mecanico123` | diagnóstico, itens, enviar orçamento, iniciar execução após reposição, reparo adicional, finalizar |

Ela também cria 5 serviços, 4 peças/insumos (um deles já abaixo do estoque mínimo), a cliente **Maria Silva** (CPF `52998224725`, veículo `ABC1D23`) e a empresa **Transportes Exemplo LTDA** (CNPJ `11222333000181`, veículo `XYZ1234`).

**Como funciona a autenticação:**
- **Rotas administrativas** exigem `Authorization: Bearer <access_token>`.
- **Tokens:** o access token vale 15 min. O refresh token (`POST /tokens/refresh`) vale 24 h e é revogado em `POST /usuarios/logout`.
- **Formato:** o padrão é **JWT** (HS256), como pede o enunciado. O algoritmo é fixado na verificação. Como opção, `TOKEN_TYPE=paseto` emite **PASETO v2**, que tem o mesmo papel de um JWT, mas é cifrado e não tem negociação de algoritmo. Trocar o `TOKEN_TYPE` invalida os tokens já emitidos, e os usuários precisam fazer login de novo.
- **Rotas do cliente** são públicas e exigem o UUID da OS + o CPF/CNPJ do dono: `GET /ordens-servico/{uuid}/andamento`, aprovar/rejeitar orçamento e autorizar/recusar reparo.

### Roteiro rápido do fluxo (curl)

```bash
API=http://localhost:8080/api/v1
login() { curl -s -X POST $API/usuarios/login -H 'Content-Type: application/json' \
  -d "{\"email\":\"$1\",\"senha\":\"$2\"}" | sed -E 's/.*"access_token":"([^"]+)".*/\1/'; }
ATD=$(login atendente@oficina.com atendente123); MEC=$(login mecanico@oficina.com mecanico123)

SERVICO=$(curl -s $API/servicos -H "Authorization: Bearer $ATD" | sed -E 's/.*"uuid":"([^"]+)","nome":"Troca de óleo".*/\1/')
PECA=$(curl -s "$API/pecas?busca=FLT-OLEO" -H "Authorization: Bearer $ATD" | sed -E 's/^\[\{"uuid":"([^"]+)".*/\1/')

OS=$(curl -s -X POST $API/ordens-servico -H "Authorization: Bearer $ATD" -H 'Content-Type: application/json' \
  -d "{\"documento\":\"52998224725\",\"placa\":\"ABC1D23\",\"descricao_problema\":\"Revisão\",
       \"servicos\":[{\"servico_uuid\":\"$SERVICO\"}],\"pecas\":[{\"peca_uuid\":\"$PECA\",\"quantidade\":1}]}" \
  | sed -E 's/^\{"uuid":"([^"]+)".*/\1/')

curl -s -X POST $API/ordens-servico/$OS/diagnostico -H "Authorization: Bearer $MEC"
curl -s -X POST $API/ordens-servico/$OS/orcamento/enviar -H "Authorization: Bearer $MEC"
curl -s "$API/ordens-servico/$OS/andamento?documento=52998224725"                      # cliente, sem token
curl -s -X POST $API/ordens-servico/$OS/orcamento/aprovar -H 'Content-Type: application/json' \
  -d '{"documento":"52998224725"}'                                                     # -> EM_EXECUCAO
curl -s -X POST $API/ordens-servico/$OS/finalizar -H "Authorization: Bearer $MEC"
curl -s -X POST $API/ordens-servico/$OS/entregar  -H "Authorization: Bearer $ATD"
```

## Testes

| Tipo | Onde | Precisa de banco? |
|---|---|---|
| Unitários de domínio | `internal/domain/**` | não |
| Unitários de casos de uso (gomock) | `internal/service` | não |
| HTTP dos handlers, autenticação e papéis (gomock) | `internal/handler`, `internal/middleware` | não |
| Integração das queries, transações e concorrência | `db/sqlc` | sim |
| Integração dos fluxos principais pela API (aprovação, rejeição, falta de estoque, reparo adicional, segurança) | `internal/integration` | sim |

```bash
# Só unitários (sem Docker): os testes de banco são pulados
go test -short ./...

# Suíte completa: sobe o banco e usa o banco separado oficina_test
docker compose up -d postgres
make test

# Cobertura dos domínios críticos (internal/domain + internal/service) + coverage.html
make coverage
```

**Cobertura dos domínios críticos: 96,7%** (mínimo exigido: 80%). O CI falha se ficar abaixo de 80%.

| Pacote | Cobertura |
|---|---|
| internal/domain (todos os subpacotes) | 99–100% |
| internal/service | 95,9% |
| internal/handler | 99,4% |
| internal/middleware | 100% |

Evidências: [docs/testes/resultado-testes.txt](docs/testes/resultado-testes.txt) e [docs/testes/cobertura-dominios-criticos.txt](docs/testes/cobertura-dominios-criticos.txt).

## Análise de vulnerabilidades

```bash
make vulncheck   # govulncheck: dependências e stdlib
make gosec       # análise estática do código
make trivy       # imagem Docker (precisa do trivy instalado)
```

Resultado atual: govulncheck com 0 vulnerabilidades alcançáveis, gosec com 0 problemas e Trivy com 0 no Alpine e no `migrate`. Os detalhes, as correções feitas e o único risco aceito estão em [docs/security/README.md](docs/security/README.md).

## Estrutura de diretórios

```
.
├── cmd/server/            # main: carrega config, conecta no banco, sobe o servidor
├── db/
│   ├── migration/         # migrations versionadas (schema + seed de demonstração)
│   ├── query/             # SQL de entrada do sqlc
│   ├── sqlc/              # código gerado pelo sqlc + Store com ExecTx + testes de integração
│   ├── mock/              # mock do Store (mockgen)
│   └── init/              # script do Postgres que cria o banco de testes
├── docs/
│   ├── ddd/               # linguagem ubíqua, diagramas e event storming
│   ├── security/          # relatórios de vulnerabilidades
│   ├── testes/            # evidências de testes e cobertura
│   └── swagger.*          # OpenAPI gerado
├── internal/
│   ├── domain/            # DDD: cliente, veiculo, ordemservico, estoque (regras puras)
│   ├── service/           # casos de uso (comandos e consultas) e transações
│   ├── handler/           # handlers HTTP + anotações Swagger
│   ├── routes/            # registro de rotas e papéis por rota
│   ├── middleware/        # autenticação e autorização por papel
│   ├── dto/               # requests e responses da API
│   └── integration/       # testes ponta a ponta dos fluxos principais
├── notification/          # porta de notificação (adaptador de log)
├── token/                 # JWT (padrão) e PASETO
├── util/                  # config, senha, conversores, geradores de dados de teste
├── .github/workflows/     # CI: testes, cobertura mínima e análise de segurança
├── Dockerfile
├── docker-compose.yaml
├── Makefile
└── sqlc.yaml
```

## Comandos úteis (Makefile)

| Comando | O que faz |
|---|---|
| `make up` / `make down` | sobe ou derruba o ambiente completo |
| `make migrateup` / `make migratedown` | aplica ou desfaz as migrations no banco local |
| `make new_migration name=xxx` | cria uma nova migration |
| `make sqlc` / `make mock` / `make swag` | gera de novo o código do sqlc, o mock e o Swagger |
| `make server` | roda a API localmente |
| `make test` / `make coverage` | testes completos / cobertura dos domínios críticos |
| `make vulncheck` / `make gosec` / `make trivy` | análise de segurança |
