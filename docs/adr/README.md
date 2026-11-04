# Registros de decisão de arquitetura (ADR)

Cada decisão relevante tem um arquivo curto, numerado, com contexto, decisão, consequências e status. Um ADR não é editado depois de aceito: uma nova decisão supera a antiga com um ADR novo que aponta para ela.

| ADR | Decisão | Status |
|---|---|---|
| [0001](0001-postgresql.md) | PostgreSQL como banco único | Aceita |
| [0002](0002-jwt-com-opcao-paseto.md) | JWT HS256 nas rotas administrativas, com PASETO como opção | Aceita |
| [0003](0003-estoque-sem-reserva.md) | Estoque sem reserva: baixa ao iniciar a execução | Aceita |
| [0004](0004-rejeicao-volta-para-diagnostico.md) | Orçamento rejeitado volta a OS para Em diagnóstico | Aceita |
| [0005](0005-acompanhamento-sem-login.md) | Acompanhamento do cliente sem login (UUID da OS + CPF/CNPJ) | Aceita |
| [0006](0006-eventos-de-dominio-em-processo.md) | Eventos de domínio com despachante síncrono em processo | Proposta (PR do lote E3) |

## Template

```markdown
# ADR NNNN · Título

**Status:** Proposta | Aceita | Superada por ADR NNNN · **Data:** AAAA-MM-DD

## Contexto
O problema e as forças em jogo, em poucas frases.

## Decisão
O que foi decidido, no presente ("usamos", "a OS volta").

## Consequências
O que fica melhor, o que fica pior e o que passa a ser obrigatório.
```
