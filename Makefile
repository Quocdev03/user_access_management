.PHONY: help dev run build migrate-up migrate-down migrate-create swagger swagger-install test test-coverage docker-up docker-down

# Database URL for golang-migrate
DB_URL="mysql://uam_user:uam_password@tcp(localhost:3306)/uam_db"

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "  dev              Run with air (hot reload)"
	@echo "  run              Run server"
	@echo "  build            Build binary to bin/server"
	@echo "  migrate-up       Apply DB migrations"
	@echo "  migrate-down     Rollback last migration"
	@echo "  migrate-create   Create migration (name=...)"
	@echo "  swagger          Generate docs/docs.go + swagger.json/yaml"
	@echo "  swagger-install  Install swag CLI (go install)"
	@echo "  test             Run tests"
	@echo "  test-coverage    Tests + coverage HTML"
	@echo "  docker-up        docker compose up -d"
	@echo "  docker-down      docker compose down"

dev:
	air

run:
	go run cmd/server/main.go

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

migrate-up:
	migrate -path migrations -database $(DB_URL) -verbose up

migrate-down:
	migrate -path migrations -database $(DB_URL) -verbose down

migrate-create:
	@if [ -z "$(name)" ]; then echo "Error: name is required. Usage: make migrate-create name=create_xxx_table"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(name)

# Regenerate Swagger from // @Router annotations in handlers
swagger:
	@command -v swag >/dev/null 2>&1 || { echo "swag not found. Run: make swagger-install"; exit 1; }
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
	@echo "Swagger generated: docs/docs.go, docs/swagger.json, docs/swagger.yaml"

swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installed swag. Ensure \$$(go env GOPATH)/bin is on PATH."

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

docker-up:
	docker compose up -d

docker-down:
	docker compose down
