-- name: CreateMovimentoEstoque :one
INSERT INTO movimentos_estoque (
    peca_id,
    tipo,
    quantidade,
    saldo_anterior,
    saldo_posterior,
    ordem_servico_id,
    usuario_id,
    observacao
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListMovimentosEstoqueByPeca :many
SELECT * FROM movimentos_estoque
WHERE peca_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2
OFFSET $3;
