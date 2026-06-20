.PHONY: format run back migrate migrate-down seed swagger gen gobash

GOBASH = docker exec -it url-shortener /bin/sh

-include .env

format:
	@echo "Formatting code..."
	go fmt ./...

run:
	air -c .air.toml

back:
	docker compose -f deployments/local-dev/compose.yaml up

migrate:
	@echo "Running database migrations..."
	$(GOBASH) -c 'migrate -database "postgres://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=disable" -path ./migrations up'

migrate-down:
	@echo "Running database migrations..."
	$(GOBASH) -c 'migrate -database "postgres://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=disable" -path ./migrations down'

seed:
	@echo "Seeding database with default data..."
	$(GOBASH) -c 'go run cmd/seed/seed.go'

swagger:
	@echo "Generating swagger docs..."
	$(GOBASH) -c 'swag init -g cmd/url-shortener/main.go --parseDependency'

gen:
	@echo "Generating go structs from sql schema..."
	sqlc generate

gobash:
	$(GOBASH)

