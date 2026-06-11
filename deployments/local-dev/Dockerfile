# Generator Stage
FROM golang:1.26.3-alpine AS generator
WORKDIR /app
COPY sqlc.yaml ./
COPY internal/ ./internal/
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.26.0
COPY migrations/ ./migrations/
RUN sqlc generate

# Build Stage
FROM golang:1.26.3-alpine AS build
WORKDIR /app

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOFLAGS="-buildvcs=false"

COPY go.* ./

RUN go mod download && go mod verify

RUN apk update && \
    apk add --no-cache git curl

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.11
RUN go install github.com/air-verse/air@v1.61.0

COPY . .

COPY deployments/local-dev/entrypoint.sh /bin/entrypoint.sh
RUN chmod +x /bin/entrypoint.sh

CMD ["entrypoint.sh"]
