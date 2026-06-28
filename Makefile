.PHONY: dev run build migrate-up migrate-down migrate-create swagger test test-coverage docker-up docker-down

# Database URL for golang-migrate
DB_URL="mysql://uam_user:uam_password@tcp(localhost:3306)/uam_db"

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

swagger:
	swag init -g cmd/server/main.go

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

docker-up:
	docker compose up -d

docker-down:
	docker compose down
