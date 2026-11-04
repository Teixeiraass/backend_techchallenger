# ADR 0005 · Acompanhamento do cliente sem login (UUID da OS + CPF/CNPJ)

**Status:** Aceita · **Data:** 2026-11-02

## Contexto

O cliente precisa acompanhar a OS e aprovar ou rejeitar o orçamento e os reparos adicionais. Criar conta e senha para cada cliente de oficina é atrito demais para o MVP e para o público. O enunciado aceita "envio ou disponibilização" do orçamento.

## Decisão

As cinco rotas do cliente (andamento, aprovar/rejeitar orçamento, autorizar/recusar reparo) são **públicas** e validadas por **UUID da OS + CPF/CNPJ do dono** (`validarTitular`). O link de acompanhamento chega por e-mail. Documento errado recebe 404, igual a OS inexistente, para não revelar quais OS existem.

Proteções que acompanham a decisão: o CPF/CNPJ vai no header `X-Documento` (consulta) ou no corpo (decisões), nunca na URL; o log de acesso não registra query string; há limite de tentativas por IP nas rotas públicas e no login (429).

## Consequências

- Zero atrito para o cliente e nenhuma senha de cliente para guardar.
- O UUID (128 bits aleatórios) mais o documento são a credencial. Quem tiver os dois (e-mail encaminhado, print) pode decidir pela OS. O limite de tentativas impede adivinhar o documento; o risco residual é aceito para o MVP e registrado no débito técnico (link com expiração ou código de uso único como evolução).
