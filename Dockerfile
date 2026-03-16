FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway cmd/gateway/main.go

FROM scratch

COPY --from=builder /app/gateway /gateway

EXPOSE 8080
ENTRYPOINT ["/gateway"]
