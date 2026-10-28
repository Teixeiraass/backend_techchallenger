CREATE TABLE "servicos" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "nome" varchar(120) UNIQUE NOT NULL,
  "descricao" text NOT NULL DEFAULT '',
  "preco_centavos" bigint NOT NULL,
  "tempo_estimado_minutos" integer NOT NULL DEFAULT 0,
  "ativo" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "servicos_preco_check" CHECK ("preco_centavos" >= 0),
  CONSTRAINT "servicos_tempo_check" CHECK ("tempo_estimado_minutos" >= 0)
);

-- Peças e insumos ficam no mesmo agregado, diferenciados pelo tipo.
CREATE TABLE "pecas" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "codigo" varchar(40) UNIQUE NOT NULL,
  "nome" varchar(120) NOT NULL,
  "descricao" text NOT NULL DEFAULT '',
  "tipo" varchar(10) NOT NULL,
  "unidade" varchar(10) NOT NULL DEFAULT 'UN',
  "preco_centavos" bigint NOT NULL,
  "quantidade_estoque" integer NOT NULL DEFAULT 0,
  "estoque_minimo" integer NOT NULL DEFAULT 0,
  "ativo" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "pecas_tipo_check" CHECK ("tipo" IN ('PECA', 'INSUMO')),
  CONSTRAINT "pecas_preco_check" CHECK ("preco_centavos" >= 0),
  -- Invariante do agregado: o saldo nunca fica negativo.
  CONSTRAINT "pecas_quantidade_estoque_check" CHECK ("quantidade_estoque" >= 0),
  CONSTRAINT "pecas_estoque_minimo_check" CHECK ("estoque_minimo" >= 0)
);

CREATE TABLE "movimentos_estoque" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "peca_id" bigint NOT NULL,
  "tipo" varchar(20) NOT NULL,
  "quantidade" integer NOT NULL,
  "saldo_anterior" integer NOT NULL,
  "saldo_posterior" integer NOT NULL,
  "ordem_servico_id" bigint,
  "usuario_id" bigint,
  "observacao" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "movimentos_estoque_tipo_check" CHECK ("tipo" IN ('ENTRADA', 'SAIDA', 'AJUSTE'))
);

ALTER TABLE "movimentos_estoque" ADD FOREIGN KEY ("peca_id") REFERENCES "pecas" ("id") ON DELETE CASCADE;
ALTER TABLE "movimentos_estoque" ADD FOREIGN KEY ("usuario_id") REFERENCES "usuarios" ("id") ON DELETE SET NULL;
CREATE INDEX ON "movimentos_estoque" ("peca_id");
