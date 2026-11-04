# ADR 0001 · PostgreSQL como banco único

**Status:** Aceita · **Data:** 2026-11-01

## Contexto

O domínio é relacional e transacional: a OS referencia cliente e veículo, tem itens que copiam o preço do catálogo, e a aprovação do orçamento precisa mudar o status da OS e baixar o estoque das peças de forma atômica. Duas OS podem disputar a mesma peça ao mesmo tempo. A gestão quer o tempo médio de execução por serviço, que é uma agregação. O enunciado pede a justificativa do banco.

A alternativa considerada foi um banco de documentos (MongoDB), que facilitaria guardar a OS inteira como um documento.

## Decisão

Usamos o **PostgreSQL 16** como único banco, acessado por SQL tipado (sqlc) e versionado por migrations (golang-migrate).

- Consistência forte entre agregados: iniciar a execução e baixar o estoque acontecem em uma transação; a OS fica travada com `SELECT … FOR NO KEY UPDATE`.
- Concorrência no estoque resolvida no banco: `UPDATE pecas SET quantidade_estoque = quantidade_estoque - $qtd WHERE id = $id AND quantidade_estoque >= $qtd`, mais `CHECK (quantidade_estoque >= 0)`.
- Integridade por constraints: chaves estrangeiras, `UNIQUE` (CPF/CNPJ, placa, código da peça), `CHECK` nos status.
- Tempo médio por agregação SQL (`AVG`, `EXTRACT(EPOCH …)`, `GROUP BY`).

## Consequências

- O domínio pode confiar nas invariantes também no banco: um bug na aplicação não deixa saldo negativo nem OS com status inválido.
- Há um teste de integração com 10 baixas simultâneas que prova a garantia de concorrência; ele precisa continuar passando em qualquer mudança nas queries de estoque.
- Perde-se a conveniência do documento único: a OS é carregada de quatro tabelas (OS, itens de serviço, itens de peça, reparos) pelo repositório.
- Toda mudança de schema passa por migration; o seed de demonstração fica fora das migrations (`db/seed`).
