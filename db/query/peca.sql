-- name: CreatePeca :one
INSERT INTO pecas (
    codigo,
    nome,
    descricao,
    tipo,
    unidade,
    preco_centavos,
    quantidade_estoque,
    estoque_minimo
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPeca :one
SELECT * FROM pecas
WHERE uuid = $1 LIMIT 1;

-- name: GetPecaForUpdate :one
SELECT * FROM pecas
WHERE uuid = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListPecas :many
SELECT * FROM pecas
WHERE (NOT @apenas_ativas::boolean OR ativo = true)
  AND (@tipo::text = '' OR tipo = @tipo::text)
  AND (@busca::text = '' OR nome ILIKE '%' || @busca::text || '%' OR codigo ILIKE '%' || @busca::text || '%')
ORDER BY nome
LIMIT @limite
OFFSET @deslocamento;

-- name: ListPecasEstoqueBaixo :many
SELECT * FROM pecas
WHERE ativo = true
  AND quantidade_estoque <= estoque_minimo
ORDER BY (quantidade_estoque - estoque_minimo), nome;

-- name: UpdatePeca :one
UPDATE pecas
SET codigo = $2,
    nome = $3,
    descricao = $4,
    tipo = $5,
    unidade = $6,
    preco_centavos = $7,
    estoque_minimo = $8,
    ativo = $9,
    updated_at = now()
WHERE uuid = $1
RETURNING *;

-- name: InativarPeca :one
UPDATE pecas
SET ativo = false,
    updated_at = now()
WHERE uuid = $1
RETURNING *;

-- name: BaixarEstoquePeca :one
-- A condição no WHERE garante que o saldo nunca fica negativo, mesmo com
-- duas OS baixando a mesma peça ao mesmo tempo: sem linha retornada = sem saldo.
UPDATE pecas
SET quantidade_estoque = quantidade_estoque - @quantidade::integer,
    updated_at = now()
WHERE id = @id
  AND quantidade_estoque >= @quantidade::integer
RETURNING *;

-- name: ReporEstoquePeca :one
UPDATE pecas
SET quantidade_estoque = quantidade_estoque + @quantidade::integer,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: AjustarEstoquePeca :one
UPDATE pecas
SET quantidade_estoque = @quantidade::integer,
    updated_at = now()
WHERE id = @id
RETURNING *;
