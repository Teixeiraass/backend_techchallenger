# Débito técnico

Itens da avaliação crítica da `main` `e6d2bad` (03/11/2026) que **não** foram corrigidos pelos lotes de correção, mais os limites que os próprios lotes deixaram registrados. Cada item tem impacto, esforço estimado e prioridade; a lista é revisada a cada entrega.

Prioridade: **P1** corrigir antes da próxima fase · **P2** corrigir quando o fluxo for tocado · **P3** registrar e aceitar.

| # | Item | Origem | Impacto | Esforço | Prioridade |
|---|---|---|---|---|---|
| 1 | **Envio de e-mail real.** `notification.LogSender` só escreve no stdout; o cliente não recebe o link de acompanhamento de verdade. | Crítica 1.2 | Funcional: a "disponibilização" do orçamento existe pela API, mas o aviso não chega ao cliente | 2 h (adaptador para um provedor HTTP atrás de `notification.Sender`, envio assíncrono) | P1 |
| 2 | **Access token sem revogação.** Um access token roubado vale até 15 min mesmo depois do logout. | ADR 0002 | Segurança (janela curta) | 3 h (lista de revogação por `jti` em memória/cache, consultada no middleware) | P2 |
| 3 | **Link de acompanhamento sem expiração.** UUID da OS + CPF/CNPJ valem para sempre; e-mail encaminhado dá acesso às decisões. | ADR 0005, crítica 5.3 | Segurança/privacidade | 4 h (token de acompanhamento com expiração ou código de uso único por decisão) | P2 |
| 4 | **Limite de tentativas em memória.** O `RateLimiter` é por processo: não vale entre réplicas e zera no restart. | Lote A | Segurança em ambiente com mais de uma réplica | 2 h (backend em Redis) | P3 |
| 5 | **Eventos sem outbox.** O despachante é síncrono em processo; queda entre o commit e a publicação perde a notificação. | ADR 0006 | Confiabilidade das notificações | 1 dia (tabela outbox gravada na mesma transação + publicador) | P2 |
| 6 | **Sem camada anticorrupção entre os contextos internos.** As portas (`Catalogo`, `estoque.Repositorio`) devolvem o modelo do outro contexto sem tradução. | Crítica 3.3 (parcial), mapa de contexto | Acoplamento entre contextos; só dói ao separar em serviços | 1 dia (tradutores por porta quando houver um segundo deploy) | P3 |
| 7 | **Carga da OS em quatro consultas.** O repositório faz 1 + 3 selects por OS; a listagem usa SQL próprio. | Lote E2 | Desempenho em volume alto | 2 h (uma consulta com `json_agg` ou carga em lote) | P3 |
| 8 | **Sem observabilidade estruturada.** Log em texto, sem métricas, tracing nem id de correlação. | RNF | Operação: diagnóstico de incidentes | 1 dia (log JSON com request id, `/metrics` Prometheus, OpenTelemetry) | P2 |
| 9 | **Sem teste de carga nem metas de latência.** | RNF | Não se sabe o limite do MVP | 4 h (k6 contra o compose com metas por rota) | P3 |
| 10 | **Cobertura do serviço medida com mocks.** Os 99% de `internal/service` vêm dos mocks dos repositórios; a persistência real é coberta por `db/sqlc` (67%) e pela integração. | Crítica 4.1 | Confiança do número de cobertura | 2 h (medir `-coverpkg` a partir dos testes de integração e publicar no CI) | P3 |
| 11 | **Sem evidência de TDD nos commits.** Testes e código entram juntos. | Crítica 4.4 | Processo (não exigido pelo enunciado) | Mudança de hábito: commit do teste vermelho antes do código | P3 |
| 12 | **Notação do Event Storming no board.** Três cores fora da notação da aula (legenda explica). | Crítica 2.4 | Documentação | 30 min no Miro (grupo) | P3 |
| 13 | **Justificativa curta de "estoque como suporte".** | Crítica 2.5 | Documentação | 15 min no mapa de contexto (uma frase sobre por que não é principal) | P3 |
| 14 | **Dado pessoal em texto no banco, sem retenção.** | RNF (LGPD) | Privacidade | 1 dia (política de retenção, anonimização de OS antigas) | P2 |
| 15 | **Um banco, uma réplica, sem orquestrador.** | RNF (disponibilidade) | Disponibilidade em produção | Depende do ambiente alvo (fase futura) | P3 |

Itens que dependem só do grupo (board do Miro, PDF, vídeo, tag, proteção da `main`) estão na seção de pendências do relatório de correções, não aqui.
