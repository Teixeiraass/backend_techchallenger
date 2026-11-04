#!/bin/sh

set -e

echo "run db migration"
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

# Dados de demonstração (usuários admin/atendente/mecânico, catálogo e clientes
# de exemplo). Só em ambiente local: em qualquer outro, SEED_DEMO fica desligado.
if [ "$SEED_DEMO" = "true" ]; then
  echo "run demo seed"
  /app/migrate -path /app/seed -database "${DB_SOURCE}&x-migrations-table=seed_migrations" -verbose up
fi

echo "start the app"
exec "$@"
