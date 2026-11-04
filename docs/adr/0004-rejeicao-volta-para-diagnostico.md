# ADR 0004 · Orçamento rejeitado volta a OS para Em diagnóstico

**Status:** Aceita · **Data:** 2026-11-02

## Contexto

O enunciado define seis status (Recebida, Em diagnóstico, Aguardando aprovação, Em execução, Finalizada, Entregue) e não diz o que acontece quando o cliente rejeita o orçamento. As opções eram criar um sétimo status ("Cancelada"), encerrar a OS, ou devolvê-la a um status existente.

## Decisão

A rejeição leva a OS de **Aguardando aprovação para Em diagnóstico**, com `orcamento_status = REJEITADO`. É a única transição "para trás" da máquina de estados. Dali, a oficina pode ajustar os itens e enviar um novo orçamento, ou **entregar o veículo sem execução** (Em diagnóstico → Entregue, só permitido com orçamento rejeitado). Não existe status "Cancelada".

## Consequências

- Os seis status do enunciado são preservados e a máquina de estados continua explícita (`status.go`, teste da matriz 6×6).
- O histórico mostra a volta e a nova rodada de orçamento; a OS não "some".
- `Entregar` precisa de uma regra a mais (só com orçamento rejeitado quando Em diagnóstico), coberta pelos testes de domínio.
- Uma OS rejeitada e entregue sem execução não entra no tempo médio (não tem execução).
