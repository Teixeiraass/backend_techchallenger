-- name: CreateReparoAdicional :one
INSERT INTO reparos_adicionais (
    ordem_servico_id,
    descricao
) VALUES ($1, $2)
RETURNING *;

-- name: GetReparoAdicionalForUpdate :one
SELECT * FROM reparos_adicionais
WHERE uuid = $1 AND ordem_servico_id = $2
LIMIT 1
FOR NO KEY UPDATE;

-- name: ListReparosAdicionaisByOrdemServico :many
SELECT * FROM reparos_adicionais
WHERE ordem_servico_id = $1
ORDER BY created_at, id;

-- name: UpdateReparoAdicionalStatus :one
UPDATE reparos_adicionais
SET status = $2,
    decidido_em = now()
WHERE id = $1
RETURNING *;
