-- name: CreateSessao :one
INSERT INTO sessoes (
    id,
    usuario_id,
    refresh_token,
    user_agent,
    client_ip,
    is_blocked,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetSessao :one
SELECT * FROM sessoes
WHERE id = $1 LIMIT 1;

-- name: BlockSessao :exec
UPDATE sessoes
SET is_blocked = true
WHERE id = $1;
