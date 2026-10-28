CREATE TABLE "usuarios" (
  "id" bigserial PRIMARY KEY,
  "uuid" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "nome" varchar(150) NOT NULL,
  "email" varchar(150) UNIQUE NOT NULL,
  "hashed_password" varchar NOT NULL,
  "papel" varchar(20) NOT NULL,
  "ativo" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  CONSTRAINT "usuarios_papel_check" CHECK ("papel" IN ('ADMINISTRADOR', 'ATENDENTE', 'MECANICO'))
);

CREATE TABLE "sessoes" (
  "id" uuid PRIMARY KEY,
  "usuario_id" bigint NOT NULL,
  "refresh_token" varchar NOT NULL,
  "user_agent" varchar NOT NULL,
  "client_ip" varchar NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "sessoes" ADD FOREIGN KEY ("usuario_id") REFERENCES "usuarios" ("id") ON DELETE CASCADE;
CREATE INDEX ON "sessoes" ("usuario_id");
