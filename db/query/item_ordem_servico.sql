-- name: CreateItemServico :one
INSERT INTO itens_servico (
    ordem_servico_id,
    servico_id,
    reparo_adicional_id,
    quantidade,
    valor_unitario_centavos
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListItensServicoByOrdemServico :many
SELECT i.id,
       i.uuid,
       i.ordem_servico_id,
       i.quantidade,
       i.valor_unitario_centavos,
       i.created_at,
       s.uuid AS servico_uuid,
       s.nome AS servico_nome,
       r.uuid AS reparo_adicional_uuid,
       r.status AS reparo_adicional_status
FROM itens_servico i
JOIN servicos s ON s.id = i.servico_id
LEFT JOIN reparos_adicionais r ON r.id = i.reparo_adicional_id
WHERE i.ordem_servico_id = $1
ORDER BY i.created_at, i.id;

-- name: DeleteItemServico :one
DELETE FROM itens_servico
WHERE uuid = $1
  AND ordem_servico_id = $2
  AND reparo_adicional_id IS NULL
RETURNING id;

-- name: CreateItemPeca :one
INSERT INTO itens_peca (
    ordem_servico_id,
    peca_id,
    reparo_adicional_id,
    quantidade,
    valor_unitario_centavos
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListItensPecaByOrdemServico :many
SELECT i.id,
       i.uuid,
       i.ordem_servico_id,
       i.peca_id,
       i.quantidade,
       i.valor_unitario_centavos,
       i.created_at,
       p.uuid AS peca_uuid,
       p.codigo AS peca_codigo,
       p.nome AS peca_nome,
       p.tipo AS peca_tipo,
       r.uuid AS reparo_adicional_uuid,
       r.status AS reparo_adicional_status
FROM itens_peca i
JOIN pecas p ON p.id = i.peca_id
LEFT JOIN reparos_adicionais r ON r.id = i.reparo_adicional_id
WHERE i.ordem_servico_id = $1
ORDER BY i.created_at, i.id;

-- name: DeleteItemPeca :one
DELETE FROM itens_peca
WHERE uuid = $1
  AND ordem_servico_id = $2
  AND reparo_adicional_id IS NULL
RETURNING id;
