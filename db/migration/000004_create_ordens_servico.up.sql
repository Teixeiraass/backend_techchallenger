CREATE TABLE "ordens_servico" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "cliente_id" bigint NOT NULL,
  "veiculo_id" bigint NOT NULL,
  "status" varchar(30) NOT NULL DEFAULT 'RECEBIDA',
  "descricao_problema" text NOT NULL DEFAULT '',
  "diagnostico" text NOT NULL DEFAULT '',
  "orcamento_status" varchar(20) NOT NULL DEFAULT 'NAO_ENVIADO',
  "valor_total_centavos" bigint NOT NULL DEFAULT 0,
  "orcamento_enviado_em" timestamptz,
  "orcamento_decidido_em" timestamptz,
  "diagnostico_iniciado_em" timestamptz,
  "execucao_iniciada_em" timestamptz,
  "finalizada_em" timestamptz,
  "entregue_em" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "ordens_servico_status_check" CHECK ("status" IN (
    'RECEBIDA', 'EM_DIAGNOSTICO', 'AGUARDANDO_APROVACAO', 'EM_EXECUCAO', 'FINALIZADA', 'ENTREGUE'
  )),
  CONSTRAINT "ordens_servico_orcamento_status_check" CHECK ("orcamento_status" IN (
    'NAO_ENVIADO', 'AGUARDANDO', 'APROVADO', 'REJEITADO'
  ))
);

ALTER TABLE "ordens_servico" ADD FOREIGN KEY ("cliente_id") REFERENCES "clientes" ("id") ON DELETE RESTRICT;
ALTER TABLE "ordens_servico" ADD FOREIGN KEY ("veiculo_id") REFERENCES "veiculos" ("id") ON DELETE RESTRICT;
CREATE INDEX ON "ordens_servico" ("status");
CREATE INDEX ON "ordens_servico" ("cliente_id");
CREATE INDEX ON "ordens_servico" ("veiculo_id");

CREATE TABLE "reparos_adicionais" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "ordem_servico_id" bigint NOT NULL,
  "descricao" text NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'PENDENTE',
  "decidido_em" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "reparos_adicionais_status_check" CHECK ("status" IN ('PENDENTE', 'AUTORIZADO', 'RECUSADO'))
);

ALTER TABLE "reparos_adicionais" ADD FOREIGN KEY ("ordem_servico_id") REFERENCES "ordens_servico" ("id") ON DELETE CASCADE;
CREATE INDEX ON "reparos_adicionais" ("ordem_servico_id");

CREATE TABLE "itens_servico" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "ordem_servico_id" bigint NOT NULL,
  "servico_id" bigint NOT NULL,
  "reparo_adicional_id" bigint,
  "quantidade" integer NOT NULL DEFAULT 1,
  "valor_unitario_centavos" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "itens_servico_quantidade_check" CHECK ("quantidade" > 0)
);

ALTER TABLE "itens_servico" ADD FOREIGN KEY ("ordem_servico_id") REFERENCES "ordens_servico" ("id") ON DELETE CASCADE;
ALTER TABLE "itens_servico" ADD FOREIGN KEY ("servico_id") REFERENCES "servicos" ("id") ON DELETE RESTRICT;
ALTER TABLE "itens_servico" ADD FOREIGN KEY ("reparo_adicional_id") REFERENCES "reparos_adicionais" ("id") ON DELETE CASCADE;
CREATE INDEX ON "itens_servico" ("ordem_servico_id");
CREATE INDEX ON "itens_servico" ("servico_id");

CREATE TABLE "itens_peca" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "ordem_servico_id" bigint NOT NULL,
  "peca_id" bigint NOT NULL,
  "reparo_adicional_id" bigint,
  "quantidade" integer NOT NULL,
  "valor_unitario_centavos" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "itens_peca_quantidade_check" CHECK ("quantidade" > 0)
);

ALTER TABLE "itens_peca" ADD FOREIGN KEY ("ordem_servico_id") REFERENCES "ordens_servico" ("id") ON DELETE CASCADE;
ALTER TABLE "itens_peca" ADD FOREIGN KEY ("peca_id") REFERENCES "pecas" ("id") ON DELETE RESTRICT;
ALTER TABLE "itens_peca" ADD FOREIGN KEY ("reparo_adicional_id") REFERENCES "reparos_adicionais" ("id") ON DELETE CASCADE;
CREATE INDEX ON "itens_peca" ("ordem_servico_id");
CREATE INDEX ON "itens_peca" ("peca_id");

CREATE TABLE "historico_status_os" (
  "id" bigserial PRIMARY KEY,
  "ordem_servico_id" bigint NOT NULL,
  "status_anterior" varchar(30) NOT NULL DEFAULT '',
  "status_novo" varchar(30) NOT NULL,
  "usuario_id" bigint,
  "observacao" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "historico_status_os" ADD FOREIGN KEY ("ordem_servico_id") REFERENCES "ordens_servico" ("id") ON DELETE CASCADE;
ALTER TABLE "historico_status_os" ADD FOREIGN KEY ("usuario_id") REFERENCES "usuarios" ("id") ON DELETE SET NULL;
CREATE INDEX ON "historico_status_os" ("ordem_servico_id");

ALTER TABLE "movimentos_estoque" ADD FOREIGN KEY ("ordem_servico_id") REFERENCES "ordens_servico" ("id") ON DELETE SET NULL;
