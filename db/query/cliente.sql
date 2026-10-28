-- name: CreateCliente :one
INSERT INTO clientes (
    nome,
    documento,
    tipo_documento,
    email,
    telefone
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCliente :one
SELECT * FROM clientes
WHERE uuid = $1 LIMIT 1;

-- name: GetClienteByID :one
SELECT * FROM clientes
WHERE id = $1 LIMIT 1;

-- name: GetClienteByDocumento :one
SELECT * FROM clientes
WHERE documento = $1 LIMIT 1;

-- name: ListClientes :many
SELECT * FROM clientes
WHERE @busca::text = ''
   OR nome ILIKE '%' || @busca::text || '%'
   OR documento LIKE '%' || @busca::text || '%'
ORDER BY nome
LIMIT @limite
OFFSET @deslocamento;

-- name: UpdateCliente :one
UPDATE clientes
SET nome = $2,
    email = $3,
    telefone = $4,
    updated_at = now()
WHERE uuid = $1
RETURNING *;

-- name: DeleteCliente :one
DELETE FROM clientes
WHERE uuid = $1
RETURNING id;
