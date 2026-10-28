ALTER TABLE "movimentos_estoque" DROP CONSTRAINT IF EXISTS "movimentos_estoque_ordem_servico_id_fkey";
DROP TABLE IF EXISTS "historico_status_os";
DROP TABLE IF EXISTS "itens_peca";
DROP TABLE IF EXISTS "itens_servico";
DROP TABLE IF EXISTS "reparos_adicionais";
DROP TABLE IF EXISTS "ordens_servico";
