#!/bin/sh

echo "Running migrations..."
migrate -database "postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable" -path ./migrations up

echo "Generating Swagger docs..."
swag init -g cmd/url-shortener/main.go --parseDependency

cd /app && air -c .air.toml

