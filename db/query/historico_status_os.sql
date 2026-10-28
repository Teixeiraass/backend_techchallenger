-- name: CreateHistoricoStatusOS :one
INSERT INTO historico_status_os (
    ordem_servico_id,
    status_anterior,
    status_novo,
    usuario_id,
    observacao
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListHistoricoStatusOSByOrdemServico :many
SELECT * FROM historico_status_os
WHERE ordem_servico_id = $1
ORDER BY created_at, id;
