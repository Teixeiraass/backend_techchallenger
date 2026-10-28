-- name: CreateServico :one
INSERT INTO servicos (
    nome,
    descricao,
    preco_centavos,
    tempo_estimado_minutos
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetServico :one
SELECT * FROM servicos
WHERE uuid = $1 LIMIT 1;

-- name: ListServicos :many
SELECT * FROM servicos
WHERE (NOT @apenas_ativos::boolean OR ativo = true)
  AND (@busca::text = '' OR nome ILIKE '%' || @busca::text || '%')
ORDER BY nome
LIMIT @limite
OFFSET @deslocamento;

-- name: UpdateServico :one
UPDATE servicos
SET nome = $2,
    descricao = $3,
    preco_centavos = $4,
    tempo_estimado_minutos = $5,
    ativo = $6,
    updated_at = now()
WHERE uuid = $1
RETURNING *;

-- name: InativarServico :one
UPDATE servicos
SET ativo = false,
    updated_at = now()
WHERE uuid = $1
RETURNING *;
