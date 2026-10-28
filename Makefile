DB_URL=postgresql://root:secret@localhost:5433/oficina?sslmode=disable
TEST_DB_URL=postgresql://root:secret@localhost:5433/oficina_test?sslmode=disable
CRITICAL_PKGS=./internal/domain/...,./internal/service/...

postgres:
	docker run --name postgres16 -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16.13-alpine

createdb:
	docker exec -it postgres16 createdb --username=root --owner=root oficina

dropdb:
	docker exec -it postgres16 dropdb oficina

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down -all

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

# Testes completos: unitários + integração com o banco oficina_test.
test:
	migrate -path db/migration -database "$(TEST_DB_URL)" up
	DB_SOURCE="$(TEST_DB_URL)" go test -count=1 -cover ./...

test-unit:
	go test -short -cover ./...

coverage:
	go test -short -coverpkg=$(CRITICAL_PKGS) -coverprofile=coverage.out ./internal/domain/... ./internal/service/...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html

coverage-all:
	DB_SOURCE="$(TEST_DB_URL)" go test -coverpkg=./... -coverprofile=coverage-all.out ./...
	go tool cover -func=coverage-all.out | tail -1

server:
	go run cmd/server/main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/Teixeiraass/backend_techchallenger/db/sqlc Store

swag:
	swag init -g cmd/server/main.go

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

gosec:
	go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude-generated -exclude-dir=db/mock ./...

trivy:
	docker build -t oficina-api:scan .
	trivy image --severity HIGH,CRITICAL oficina-api:scan

up:
	docker compose up --build -d

down:
	docker compose down

.PHONY: postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc test test-unit coverage coverage-all server mock swag vulncheck gosec trivy up down
