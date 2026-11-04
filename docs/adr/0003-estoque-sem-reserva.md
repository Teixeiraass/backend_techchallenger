# ADR 0003 · Estoque sem reserva: baixa ao iniciar a execução

**Status:** Aceita · **Data:** 2026-11-02

## Contexto

Quando uma peça entra no orçamento da OS, o estoque ainda não deveria sair: o cliente pode rejeitar. Mas se o estoque só sai depois, duas OS podem prometer a mesma peça. As opções eram: reservar ao incluir na OS (com liberação na rejeição e expiração), ou não reservar e baixar na execução.

## Decisão

**Não há reserva.** O estoque sai ao **iniciar a execução**, na mesma transação da mudança de status para Em execução, pelo agregado `Peca` (`Baixar`) e pelo `UPDATE` condicional do repositório.

- Ao incluir a peça na OS, a raiz confere o saldo somando o que a própria OS já pediu da mesma peça e ainda não baixou (`AdicionarPeca`); é uma conferência, não uma reserva.
- Na baixa, se faltar saldo, a transação é desfeita: a OS continua Aguardando aprovação com orçamento `APROVADO`, a resposta traz `aviso`, e o evento `Estoque insuficiente detectado` dispara o pedido de reposição ao administrador. Depois da reposição, `POST /ordens-servico/{uuid}/execucao` inicia a execução.
- Peças de reparo adicional são baixadas na autorização do cliente (a OS já está em execução).

## Consequências

- Modelo simples: nenhum estado de "reservado" para expirar ou liberar; o saldo do catálogo é o saldo real.
- Uma OS aprovada pode esperar reposição. O fluxo é explícito (aviso, e-mail, rota de início da execução) e está nos testes de integração.
- A conferência na inclusão não impede que outra OS consuma o saldo antes; só a baixa é definitiva.
