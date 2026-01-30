FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/eventbooker ./cmd/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/eventbooker /usr/local/bin/eventbooker
COPY .env .
COPY internal/web /app/internal/web
COPY internal/migrations/ /app/migrations/
EXPOSE 8080
CMD ["./eventbooker"]
