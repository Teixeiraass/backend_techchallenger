-- name: CreateUsuario :one
INSERT INTO usuarios (
    nome,
    email,
    hashed_password,
    papel
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUsuario :one
SELECT * FROM usuarios
WHERE uuid = $1 LIMIT 1;

-- name: GetUsuarioByEmail :one
SELECT * FROM usuarios
WHERE email = $1 LIMIT 1;

-- name: ListUsuarios :many
SELECT * FROM usuarios
ORDER BY nome
LIMIT $1
OFFSET $2;

-- name: UpdateUsuarioAtivo :one
UPDATE usuarios
SET ativo = $2,
    updated_at = now()
WHERE uuid = $1
RETURNING *;
