# syntax=docker/dockerfile:1

# Build Stage
FROM golang:1.27-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /smart-fridge-backend ./cmd/smart-fridge-backend && \
    CGO_ENABLED=0 GOBIN=/go/bin go install github.com/pressly/goose/v3/cmd/goose@v3.29.0

# Runtime stage
FROM gcr.io/distroless/static-debian13:latest-amd64 AS build-release-stage

WORKDIR /app

COPY --from=build-stage /smart-fridge-backend /smart-fridge-backend
COPY --from=build-stage /go/bin/goose /goose
COPY --from=build-stage /app/migrations /migrations

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/smart-fridge-backend"]