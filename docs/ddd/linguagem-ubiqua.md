# Linguagem Ubíqua, Glossário e relação com o código

Sistema de gestão de oficina mecânica (Tech Challenge, Fase 1).
Esta página define os termos que o time de negócio e o time técnico usam da mesma forma: nos quadros de Event Storming, nos diagramas, na API e no código.

A coluna "No código" aponta onde cada termo aparece na implementação (Go). Caminhos relativos à raiz do repositório.

---

## 1. Linguagem ubíqua

### 1.1 Pessoas (atores)

| Termo | Definição | No código |
|---|---|---|
| **Cliente** | Pessoa física ou jurídica dona do veículo. É identificado por CPF ou CNPJ. Acompanha a OS e aprova ou rejeita o orçamento. | tabela `clientes`, `db.Cliente`, `service.ClienteService` |
| **Atendente** | Funcionário que recebe o cliente, cadastra cliente e veículo, abre a OS, envia o orçamento e entrega o veículo. | papel `ATENDENTE` (`usuarios.papel`, `middleware.PapelAtendente`) |
| **Mecânico** | Funcionário que faz o diagnóstico, executa os serviços, identifica reparo adicional e finaliza a OS. | papel `MECANICO` (`middleware.PapelMecanico`) |
| **Administrador** | Gerencia o cadastro de peças, insumos e serviços, o estoque e consulta o tempo médio de execução. | papel `ADMINISTRADOR` (`middleware.PapelAdministrador`), passa em todas as rotas |
| **Sistema** | O próprio software: executa as reações automáticas (gerar orçamento, mudar status, baixar estoque, notificar). | `internal/service` (casos de uso que reagem aos comandos) |

### 1.2 Conceitos do domínio

| Termo | Definição | No código |
|---|---|---|
| **Veículo** | Carro que entra na oficina. Identificado pela **placa** e sempre pertence a um cliente. | tabela `veiculos`, `db.Veiculo`, `service.VeiculoService` |
| **Ordem de Serviço (OS)** | Registro de tudo que será feito em um veículo: serviços, peças, orçamento, status e tempos. É o centro do domínio. | `ordemservico.OrdemServico` (agregado em `internal/domain/ordemservico`), tabela `ordens_servico` |
| **Serviço** | Trabalho oferecido pela oficina (ex.: troca de óleo) e cadastrado no catálogo com preço. Só serviço **ativo** entra em OS nova. | tabela `servicos`, `db.Servico`, `service.ServicoService` |
| **Peça** | Item físico que é trocado e cobrado (ex.: filtro). Tem código, preço e saldo em estoque. | tabela `pecas` com `tipo = PECA`, `service.PecaService` |
| **Insumo** | Material consumido no serviço (ex.: óleo). Segue as mesmas regras da peça. | tabela `pecas` com `tipo = INSUMO` (mesmo agregado, `estoque.TipoInsumo`) |
| **Item de serviço** | Serviço dentro de uma OS, com o valor da época. | tabela `itens_servico`, `db.ItemServico` |
| **Item de peça** | Peça dentro de uma OS, com quantidade e valor da época. | tabela `itens_peca`, `db.ItemPeca` |
| **Orçamento** | Valor total da OS: soma dos serviços e das peças. É gerado pelo sistema e recalculado quando os itens mudam. | `ordemservico.Orcamento` + `CalcularOrcamento`; total em `ordens_servico.valor_total_centavos` |
| **Aprovação / Rejeição** | Decisão do cliente sobre o orçamento, pelo link de acompanhamento ou aplicativo. | `OrdemServico.AprovarOrcamento()` / `RejeitarOrcamento()`; campo `orcamento_status` |
| **Status da OS** | Etapa em que a OS está. Os seis valores são os do enunciado (seção 1.3). | `ordemservico.Status` |
| **Reparo adicional** | Problema novo que o mecânico encontra durante a execução. Só entra na OS se o cliente autorizar. | tabela `reparos_adicionais`; itens com `reparo_adicional_id` |
| **Tempo de execução** | Período entre o início da execução e a finalização da OS. Base do tempo médio. | `OrdemServico.TempoExecucao()` (`execucao_iniciada_em` a `finalizada_em`) |
| **Estoque / saldo** | Quantidade disponível de uma peça ou insumo. Nunca fica negativo. | `pecas.quantidade_estoque` (CHECK >= 0) |
| **Estoque mínimo** | Saldo abaixo do qual a peça precisa de reposição. | `pecas.estoque_minimo`, `estoque.EstoqueBaixo()` |
| **Reposição** | Entrada de unidades no estoque. | `PecaService.ReporEstoque()` |
| **Baixa de estoque** | Saída das unidades usadas, feita pelo sistema ao iniciar a execução. | `baixarEstoque()` em `ordem_servico_service.go`, query `BaixarEstoquePeca` |
| **Movimento de estoque** | Registro de cada reposição, ajuste ou baixa, com data e responsável. | tabela `movimentos_estoque` (ENTRADA, SAIDA, AJUSTE) |
| **Notificação** | Aviso ao cliente (e-mail ou app) quando o orçamento é enviado e quando a OS é finalizada. | porta `notification.Sender`, adaptador `notification.LogSender` |
| **Link de acompanhamento** | Endereço pelo qual o cliente consulta a OS e decide o orçamento, sem precisar de login. | `GET /api/v1/ordens-servico/{uuid}/andamento?documento=` |

### 1.3 Status da OS (ciclo de vida)

| Ordem | Status | Quando entra | Quem causa |
|---|---|---|---|
| 1 | **Recebida** | OS aberta | Atendente |
| 2 | **Em diagnóstico** | Mecânico começa a avaliar | Mecânico |
| 3 | **Aguardando aprovação** | Orçamento enviado ao cliente | Sistema (automático) |
| 4 | **Em execução** | Orçamento aprovado e trabalho começa (estoque baixado) | Sistema |
| 5 | **Finalizada** | Mecânico conclui | Mecânico |
| 6 | **Entregue** | Veículo devolvido ao cliente | Atendente |

Regra de transição: só avança na ordem acima. Qualquer transição fora da ordem é recusada (HTTP 409, código `TRANSICAO_INVALIDA`). Há duas exceções, ambas causadas pela rejeição do orçamento:

- Aguardando aprovação → Em diagnóstico: o cliente rejeitou o orçamento e a oficina pode montar um novo.
- Em diagnóstico → Entregue: o veículo é devolvido sem execução, só quando o último orçamento foi rejeitado.

A tabela de transições está em `internal/domain/ordemservico/status.go`.

### 1.4 Termos a evitar

| Use | Evite | Motivo |
|---|---|---|
| Ordem de Serviço (OS) | chamado, ticket, pedido | "Pedido" confunde com pedido de reposição |
| Cliente | usuário, comprador | "Usuário" é quem usa o sistema (atendente, mecânico, administrador) |
| Orçamento | cotação, proposta | um só nome em todo lugar |
| Peça / Insumo | produto, item de estoque | "Item" fica reservado para o item dentro da OS |
| Reposição | compra, pedido de compra | o sistema só avisa que falta, não compra |
| Finalizada | concluída, pronta | é o valor oficial do status |

---

## 2. Glossário de eventos e comandos

Comandos são intenções (no imperativo). Eventos são fatos já ocorridos (no passado). O detalhamento completo está no `event-storming.pdf`.

| Contexto | Comando | Evento de sucesso | Evento de falha |
|---|---|---|---|
| Cadastro | Cadastrar cliente | Cliente cadastrado | Cadastro de cliente recusado |
| Cadastro | Cadastrar veículo | Veículo cadastrado | Cadastro de veículo recusado |
| Ordem de Serviço | Abrir OS | **OS recebida** | Abertura de OS recusada |
| Ordem de Serviço | Adicionar serviço | Serviço adicionado à OS | Serviço recusado |
| Ordem de Serviço | Adicionar peça | Peça adicionada à OS | Peça sem estoque |
| Ordem de Serviço | Gerar orçamento | Orçamento gerado | - |
| Ordem de Serviço | Enviar orçamento | **Orçamento enviado ao cliente** | Envio recusado |
| Ordem de Serviço | Decidir orçamento | **Orçamento aprovado** / Orçamento rejeitado | - |
| Ordem de Serviço | Iniciar diagnóstico | Diagnóstico iniciado | Transição de status recusada |
| Ordem de Serviço | Iniciar execução | Execução iniciada | Transição de status recusada |
| Ordem de Serviço | Finalizar OS | OS finalizada, Tempo de execução registrado | Transição de status recusada |
| Ordem de Serviço | Entregar veículo | **Veículo entregue** | Transição de status recusada |
| Ordem de Serviço | Identificar reparo adicional | Reparo adicional identificado | - |
| Ordem de Serviço | Decidir reparo adicional | Reparo adicional autorizado / recusado | - |
| Catálogo e estoque | Cadastrar, atualizar, inativar peça | Peça cadastrada, atualizada, inativada | Recusa ou peça não encontrada |
| Catálogo e estoque | Repor estoque | Estoque reposto | Reposição recusada |
| Catálogo e estoque | Ajustar estoque | Estoque ajustado | Ajuste recusado |
| Catálogo e estoque | Baixar estoque | Estoque baixado | Estoque insuficiente detectado |
| Catálogo e estoque | Conferir estoque mínimo | Estoque baixo detectado | - |
| Catálogo e estoque | Solicitar reposição | Reposição de estoque solicitada | - |

Em negrito: eventos **pivotais**, os que marcam virada de fase.

---

## 3. Bounded contexts, agregados, entidades e objetos de valor

### 3.1 Bounded contexts

| Contexto | Responsabilidade | Agregados |
|---|---|---|
| **Cadastro** | Quem é o cliente e qual é o veículo | Cliente, Veículo |
| **Ordem de Serviço** | Ciclo de vida da OS, orçamento, status e tempos | OrdemServiço |
| **Catálogo e estoque** | Serviços oferecidos, peças, insumos e saldo | Peça, Serviço |

Relações entre contextos: a OS referencia Cliente, Veículo, Serviço e Peça **só pelo identificador**. A baixa de estoque é pedida pela OS (evento Execução iniciada) e executada pelo contexto de estoque.

### 3.2 Agregados, entidades e objetos de valor

| Agregado (raiz) | Entidades internas | Objetos de valor | Invariantes principais |
|---|---|---|---|
| **Cliente** | - | `cliente.Documento` (CpfCnpj), `Nome`, `Contato` | CPF/CNPJ válido e único |
| **Veículo** | - | `veiculo.Placa`, `Modelo`, `Ano` | Placa válida e única; sempre tem um dono |
| **OrdemServiço** | `ItemServico`, `ItemPeca`, `ReparoAdicional`, `HistoricoStatusOS` | `Status`, `StatusOrcamento`, `Orcamento`, `Dinheiro` (centavos) | Status só avança na ordem; orçamento = serviços + peças; peça só entra com estoque; só executa depois de aprovada |
| **Peça** (peça ou insumo) | tabela `movimentos_estoque` (ENTRADA, SAIDA, AJUSTE) | `CodigoPeca`, `Quantidade`, `EstoqueMinimo`, `Dinheiro` | Código único; saldo nunca negativo; reposição maior que zero |
| **Serviço** | - | `Dinheiro`, `Descricao` | Só serviço ativo entra em OS nova |

Valores em dinheiro são inteiros em **centavos** (`preco_centavos`, `valor_total_centavos`), para evitar erro de arredondamento.

Diferença usada aqui: **entidade** tem identidade e histórico (muda ao longo do tempo); **objeto de valor** é definido só pelos seus valores e é imutável (`Placa`, `Dinheiro`).

---

## 4. Regras de negócio (políticas)

1. CPF/CNPJ é validado e não se repete.
2. A placa é validada e pertence a um cliente já cadastrado.
3. A OS só abre com cliente e veículo existentes e nasce como **Recebida**.
4. Só serviço ativo no catálogo entra na OS.
5. Peça só entra na OS se houver estoque. Sem estoque, o sistema registra a falta e solicita reposição.
6. O orçamento é gerado e recalculado automaticamente sempre que os itens mudam.
7. Ao enviar o orçamento, o cliente é notificado e a OS passa a **Aguardando aprovação**.
8. A execução só começa com o orçamento aprovado.
9. Ao iniciar a execução, o sistema baixa o estoque das peças, na mesma transação da mudança de status.
10. O saldo de uma peça nunca fica negativo.
11. Reparo adicional só entra na OS com autorização do cliente pelo aplicativo.
12. Ao finalizar, o sistema registra o tempo de execução e notifica o cliente.
13. O status só muda na ordem definida; transição fora da ordem é recusada (exceções da seção 1.3).
14. O cliente consulta o andamento pela API pública; as demais rotas administrativas exigem JWT.
15. Operações que alteram mais de um registro (OS + estoque) acontecem em transação.

---

## 5. Relação entre a documentação e o código

### 5.1 Conceito → implementação

| Conceito da documentação | Tipo | Onde está no código |
|---|---|---|
| Cliente | Agregado | `db/migration/000002_*`, `internal/service/cliente_service.go` |
| CPF/CNPJ | Objeto de valor | `internal/domain/cliente/documento.go` (`NovoDocumento`) |
| Veículo | Agregado | `db/migration/000002_*`, `internal/service/veiculo_service.go` |
| Placa | Objeto de valor | `internal/domain/veiculo/placa.go` (`NovaPlaca`, `ValidarAno`) |
| Ordem de Serviço | Agregado (raiz) | `internal/domain/ordemservico/ordem_servico.go` |
| Status da OS e transições | Objeto de valor | `internal/domain/ordemservico/status.go` |
| Orçamento | Objeto de valor | `internal/domain/ordemservico/orcamento.go` |
| Item de serviço, item de peça, reparo adicional, histórico | Entidades | `db/migration/000004_create_ordens_servico.up.sql` |
| Peça e insumo | Agregado | `db/migration/000003_*`, `internal/domain/estoque/estoque.go`, `internal/service/peca_service.go` |
| Movimento de estoque | Entidade | tabela `movimentos_estoque` |
| Serviço (catálogo) | Agregado | `internal/service/servico_service.go` |
| Comandos | Casos de uso | um método por comando em `internal/service/*_service.go` |
| Políticas automáticas | Casos de uso | `internal/service/ordem_servico_service.go` (orçamento recalculado, início da execução após aprovação, baixa de estoque, pedido de reposição, notificações) |
| Notificação | Porta e adaptador | `notification/sender.go`, `notification/log_sender.go` |
| Autenticação (PASETO/JWT) e papéis | Infraestrutura | `token/`, `internal/middleware/auth.go` |
| Transações | Infraestrutura | `db/sqlc/store.go` (`ExecTx`) |

### 5.2 Comandos e consultas → rotas da API

Prefixo: `/api/v1`. "Token" = Bearer (PASETO ou JWT). Entre parênteses, os papéis aceitos; o administrador passa em todas.

| Comando ou consulta | Rota | Acesso |
|---|---|---|
| Login / renovar token / logout | `POST /usuarios/login`, `POST /tokens/refresh`, `POST /usuarios/logout` | público / público / token |
| Cadastrar, listar, detalhar, atualizar e remover cliente | `/clientes`, `/clientes/{uuid}`, `GET /clientes/documento/{documento}` | token (escrita: atendente) |
| Cadastrar, listar, detalhar, atualizar e remover veículo | `/veiculos`, `/veiculos/{uuid}` | token (escrita: atendente) |
| CRUD de serviços (inativar = DELETE) | `/servicos`, `/servicos/{uuid}` | token (escrita: administrador) |
| CRUD de peças e insumos (inativar = DELETE) | `/pecas`, `/pecas/{uuid}` | token (escrita: administrador) |
| Repor / ajustar estoque / movimentos | `POST /pecas/{uuid}/reposicao`, `POST /pecas/{uuid}/ajuste`, `GET /pecas/{uuid}/movimentos` | token (administrador) |
| Conferir estoque mínimo | `GET /pecas/estoque-baixo` | token |
| Abrir OS | `POST /ordens-servico` | token (atendente) |
| Adicionar / remover serviço e peça | `POST /ordens-servico/{uuid}/servicos`, `/pecas` e `DELETE .../{item_uuid}` | token (atendente, mecânico) |
| Iniciar diagnóstico | `POST /ordens-servico/{uuid}/diagnostico` | token (mecânico) |
| Enviar orçamento | `POST /ordens-servico/{uuid}/orcamento/enviar` | token (atendente, mecânico) |
| Aprovar ou rejeitar orçamento | `POST /ordens-servico/{uuid}/orcamento/aprovar`, `/rejeitar` | **público** + CPF/CNPJ |
| Iniciar execução (nova tentativa após reposição) | `POST /ordens-servico/{uuid}/execucao` | token (atendente, mecânico) |
| Identificar reparo adicional | `POST /ordens-servico/{uuid}/reparos-adicionais` | token (mecânico) |
| Autorizar ou recusar reparo adicional | `POST /ordens-servico/{uuid}/reparos-adicionais/{reparo_uuid}/autorizar`, `/recusar` | **público** + CPF/CNPJ |
| Finalizar OS | `POST /ordens-servico/{uuid}/finalizar` | token (mecânico) |
| Entregar veículo | `POST /ordens-servico/{uuid}/entregar` | token (atendente) |
| Consultar andamento | `GET /ordens-servico/{uuid}/andamento?documento=` | **público** + CPF/CNPJ |
| Listar e detalhar OS | `GET /ordens-servico`, `GET /ordens-servico/{uuid}` | token |
| Tempo médio de execução | `GET /ordens-servico/tempo-medio` | token (administrador) |
| Gestão de usuários da oficina | `POST/GET /usuarios`, `PATCH /usuarios/{uuid}/ativo`, `GET /usuarios/me` | token (administrador; `me` para todos) |

Em vez de um `PATCH /status` genérico, cada transição tem sua rota de comando. Assim cada comando do Event Storming vira uma rota, e a regra de cada transição fica no agregado.

### 5.3 Eventos → onde acontecem no código

Os eventos ficam registrados na tabela `historico_status_os` (status anterior, novo, observação, usuário e data) e aparecem no campo `historico` da OS.

| Evento | Quem o produz | Quem reage |
|---|---|---|
| OS recebida | `OrdemServicoService.AbrirOrdemServico` | Orçamento calculado na mesma transação |
| Serviço / peça adicionados à OS | `AdicionarServico` / `AdicionarPeca` | Orçamento recalculado (`recalcularOrcamento`) |
| Diagnóstico iniciado | `OrdemServico.IniciarDiagnostico()` | - |
| Orçamento enviado ao cliente | `OrdemServico.EnviarOrcamento()` | Status muda para Aguardando aprovação; `notification.Sender` avisa o cliente |
| Orçamento aprovado | `OrdemServico.AprovarOrcamento()` | Sistema chama `IniciarExecucao` em seguida |
| Orçamento rejeitado | `OrdemServico.RejeitarOrcamento()` | OS volta para Em diagnóstico |
| Execução iniciada | `OrdemServico.IniciarExecucao()` | Baixa de estoque na mesma transação (`baixarEstoque`) |
| Estoque baixado / Estoque baixo detectado | query `BaixarEstoquePeca` + `estoque.EstoqueBaixo()` | Pedido de reposição ao administrador (`notificarEstoqueBaixo`) |
| Estoque insuficiente detectado | `BaixarEstoquePeca` sem linha retornada | Transação desfeita; a aprovação fica registrada e a resposta traz `aviso` |
| Reparo adicional identificado | `RegistrarReparoAdicional` | Cliente notificado |
| Reparo adicional autorizado | `AutorizarReparoAdicional` | Baixa das peças do reparo + orçamento recalculado |
| OS finalizada, Tempo de execução registrado | `OrdemServico.Finalizar()` | Cliente notificado; entra no cálculo do tempo médio |
| Veículo entregue | `OrdemServico.Entregar()` | - |

---

## 6. Decisões tomadas pelo grupo

1. **Orçamento rejeitado:** a OS volta para **Em diagnóstico** com `orcamento_status = REJEITADO`. A partir daí a oficina pode ajustar os itens e enviar um novo orçamento, ou devolver o veículo (Em diagnóstico → **Entregue**, sem execução). Não foi criado o status "Cancelada", para manter os seis status do enunciado.
2. **Peça e insumo:** um único agregado `Peca` com `tipo` (`PECA` ou `INSUMO`). As regras de estoque são as mesmas.
3. **Baixa de estoque:** acontece ao iniciar a execução, na mesma transação da mudança de status. **Não há reserva:** ao incluir a peça na OS, o sistema confere o saldo (somando o que a OS já pediu da mesma peça), e na baixa confere de novo.
4. **Falta de peça na baixa:** a execução fica bloqueada. A aprovação do cliente continua registrada, a OS permanece em Aguardando aprovação com orçamento APROVADO e a resposta traz um `aviso`. Depois da reposição, `POST /ordens-servico/{uuid}/execucao` inicia a execução.
5. **Estoque mínimo e ajuste de inventário:** entram no MVP. Há `GET /pecas/estoque-baixo` e `POST /pecas/{uuid}/ajuste`. Quando a baixa ou o ajuste deixam o saldo no mínimo ou abaixo dele, o administrador recebe um pedido de reposição.
6. **Cancelamento depois da baixa:** não se aplica, porque o MVP não tem cancelamento de OS.
7. **Reparo adicional:** itens de reparo pendente ou recusado não entram no orçamento. A OS não pode ser finalizada com reparo pendente. Na autorização, as peças do reparo são baixadas na hora (a OS já está em execução).
8. **Acompanhamento sem login:** o cliente usa o UUID da OS (link de acompanhamento) junto com o seu CPF/CNPJ.
