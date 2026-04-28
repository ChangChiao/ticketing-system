FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api
RUN CGO_ENABLED=0 go build -o /migrate ./cmd/migrate

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /api /api
COPY --from=builder /migrate /migrate
COPY migrations /migrations
EXPOSE 8080
CMD ["/api"]
