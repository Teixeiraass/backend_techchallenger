# ADR 0006 · Eventos de domínio com despachante síncrono em processo

**Status:** Proposta (implementada no PR do lote E3, aguardando merge) · **Data:** 2026-11-03

## Contexto

O Event Storming do grupo é orientado a eventos ("Orçamento aprovado", "Estoque insuficiente detectado"…) e a políticas ("ao enviar o orçamento, notificar o cliente"). No código, até o lote E, as políticas eram chamadas diretas dentro dos serviços, e nenhum tipo de evento existia: o board e o código contavam histórias diferentes. Uma fila externa (RabbitMQ, SQS) traria infraestrutura e eventual consistência que o MVP não precisa.

## Decisão

- Os eventos pivotais existem como tipos de `internal/domain` (`domain.Evento`), com os nomes da linguagem ubíqua. As raízes (`OrdemServico`, `Peca`) os registram ao executar os comandos.
- Os serviços publicam os eventos **depois do commit** da transação por um **despachante síncrono em processo** (`internal/eventos`). As políticas são handlers assinados em `cmd/server/main.go`: notificar o cliente (orçamento enviado, reparo identificado, OS finalizada) e pedir reposição ao administrador (estoque insuficiente, estoque baixo).
- A falha de um handler fica no log e não desfaz a operação nem muda a resposta.
- A política "ao aprovar o orçamento, iniciar a execução" continua **síncrona no serviço**: faz parte da mesma requisição do cliente, que precisa receber o estado final.
- `Estoque insuficiente detectado` é publicado mesmo com a transação desfeita, porque sem o pedido de reposição a OS aprovada ficaria parada.

## Consequências

- O código conta a mesma história do board: cada evento do event storming é um tipo, cada política é um handler nomeado.
- Nenhuma infraestrutura nova; o despachante são 50 linhas testadas.
- Sem outbox nem fila: se o processo cair entre o commit e a publicação, a notificação se perde (o estado da OS, não). Quando os contextos virarem serviços, o despachante é trocado por um publicador com outbox transacional, sem mexer no domínio.
- Os handlers rodam na mesma requisição: uma política lenta (provedor de e-mail) aumenta a latência da resposta. O envio real deve ser assíncrono dentro do `Sender`.
