CREATE TABLE "clientes" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "nome" varchar(150) NOT NULL,
  "documento" varchar(14) UNIQUE NOT NULL,
  "tipo_documento" varchar(4) NOT NULL,
  "email" varchar(150) NOT NULL DEFAULT '',
  "telefone" varchar(20) NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "clientes_tipo_documento_check" CHECK ("tipo_documento" IN ('CPF', 'CNPJ'))
);

CREATE TABLE "veiculos" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "cliente_id" bigint NOT NULL,
  "placa" varchar(7) UNIQUE NOT NULL,
  "marca" varchar(60) NOT NULL,
  "modelo" varchar(80) NOT NULL,
  "ano" integer NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "veiculos" ADD FOREIGN KEY ("cliente_id") REFERENCES "clientes" ("id") ON DELETE RESTRICT;
CREATE INDEX ON "veiculos" ("cliente_id");
