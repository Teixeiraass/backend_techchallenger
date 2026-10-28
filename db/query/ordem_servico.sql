-- name: CreateOrdemServico :one
INSERT INTO ordens_servico (
    cliente_id,
    veiculo_id,
    descricao_problema
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOrdemServico :one
SELECT * FROM ordens_servico
WHERE uuid = $1 LIMIT 1;

-- name: GetOrdemServicoForUpdate :one
-- Trava a linha da OS durante a transação para que duas ações concorrentes
-- (ex: aprovação do cliente e alteração de itens) não se sobreponham.
SELECT * FROM ordens_servico
WHERE uuid = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListOrdensServico :many
SELECT sqlc.embed(os),
       c.uuid AS cliente_uuid,
       c.nome AS cliente_nome,
       c.documento AS cliente_documento,
       v.uuid AS veiculo_uuid,
       v.placa AS veiculo_placa
FROM ordens_servico os
JOIN clientes c ON c.id = os.cliente_id
JOIN veiculos v ON v.id = os.veiculo_id
WHERE (@status::text = '' OR os.status = @status::text)
  AND (sqlc.narg('cliente_id')::bigint IS NULL OR os.cliente_id = sqlc.narg('cliente_id'))
ORDER BY os.created_at DESC, os.id DESC
LIMIT @limite
OFFSET @deslocamento;

-- name: UpdateOrdemServicoEstado :one
-- Persiste o estado do agregado OrdemServico depois que as regras de domínio
-- (transição de status, orçamento e tempos) foram aplicadas em memória.
UPDATE ordens_servico
SET status = $2,
    diagnostico = $3,
    orcamento_status = $4,
    valor_total_centavos = $5,
    orcamento_enviado_em = $6,
    orcamento_decidido_em = $7,
    diagnostico_iniciado_em = $8,
    execucao_iniciada_em = $9,
    finalizada_em = $10,
    entregue_em = $11,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateOrdemServicoValorTotal :one
UPDATE ordens_servico
SET valor_total_centavos = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetTempoMedioExecucao :one
SELECT COUNT(*)::bigint AS total_ordens,
       COALESCE(AVG(EXTRACT(EPOCH FROM (finalizada_em - execucao_iniciada_em))), 0)::float8 AS media_segundos,
       COALESCE(MIN(EXTRACT(EPOCH FROM (finalizada_em - execucao_iniciada_em))), 0)::float8 AS minimo_segundos,
       COALESCE(MAX(EXTRACT(EPOCH FROM (finalizada_em - execucao_iniciada_em))), 0)::float8 AS maximo_segundos
FROM ordens_servico
WHERE finalizada_em IS NOT NULL
  AND execucao_iniciada_em IS NOT NULL
  AND (sqlc.narg('data_inicio')::timestamptz IS NULL OR finalizada_em >= sqlc.narg('data_inicio'))
  AND (sqlc.narg('data_fim')::timestamptz IS NULL OR finalizada_em < sqlc.narg('data_fim'));

-- name: ListTempoMedioExecucaoPorServico :many
SELECT s.uuid AS servico_uuid,
       s.nome AS servico_nome,
       s.tempo_estimado_minutos,
       COUNT(DISTINCT os.id)::bigint AS total_ordens,
       COALESCE(AVG(EXTRACT(EPOCH FROM (os.finalizada_em - os.execucao_iniciada_em))), 0)::float8 AS media_segundos
FROM itens_servico i
JOIN ordens_servico os ON os.id = i.ordem_servico_id
JOIN servicos s ON s.id = i.servico_id
LEFT JOIN reparos_adicionais r ON r.id = i.reparo_adicional_id
WHERE os.finalizada_em IS NOT NULL
  AND os.execucao_iniciada_em IS NOT NULL
  AND (r.id IS NULL OR r.status = 'AUTORIZADO')
  AND (sqlc.narg('data_inicio')::timestamptz IS NULL OR os.finalizada_em >= sqlc.narg('data_inicio'))
  AND (sqlc.narg('data_fim')::timestamptz IS NULL OR os.finalizada_em < sqlc.narg('data_fim'))
GROUP BY s.id, s.uuid, s.nome, s.tempo_estimado_minutos
ORDER BY s.nome;
