FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/warehousecontrol ./cmd/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/warehousecontrol /usr/local/bin/warehousecontrol
COPY .env .
COPY internal/web /app/internal/web
COPY internal/migrations/ /app/migrations/
EXPOSE 8080
CMD ["./warehousecontrol"]
