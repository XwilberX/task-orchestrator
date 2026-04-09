FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /orchestrator ./cmd/server

FROM golang:1.22-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
CMD ["air", "-c", ".air.toml"]

FROM gcr.io/distroless/static-debian12 AS prod
COPY --from=builder /orchestrator /orchestrator
EXPOSE 8080
ENTRYPOINT ["/orchestrator"]
