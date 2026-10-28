-- name: CreateVeiculo :one
INSERT INTO veiculos (
    cliente_id,
    placa,
    marca,
    modelo,
    ano
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetVeiculo :one
SELECT sqlc.embed(v), c.uuid AS cliente_uuid
FROM veiculos v
JOIN clientes c ON c.id = v.cliente_id
WHERE v.uuid = $1 LIMIT 1;

-- name: GetVeiculoByPlaca :one
SELECT * FROM veiculos
WHERE placa = $1 LIMIT 1;

-- name: GetVeiculoByID :one
SELECT * FROM veiculos
WHERE id = $1 LIMIT 1;

-- name: ListVeiculos :many
SELECT sqlc.embed(v), c.uuid AS cliente_uuid
FROM veiculos v
JOIN clientes c ON c.id = v.cliente_id
WHERE (sqlc.narg('cliente_id')::bigint IS NULL OR v.cliente_id = sqlc.narg('cliente_id'))
ORDER BY v.placa
LIMIT @limite
OFFSET @deslocamento;

-- name: UpdateVeiculo :one
UPDATE veiculos
SET placa = $2,
    marca = $3,
    modelo = $4,
    ano = $5,
    updated_at = now()
WHERE uuid = $1
RETURNING *;

-- name: DeleteVeiculo :one
DELETE FROM veiculos
WHERE uuid = $1
RETURNING id;
