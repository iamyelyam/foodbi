FROM golang:alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api/
RUN CGO_ENABLED=0 GOOS=linux go build -o /sync ./cmd/sync/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /api .
COPY --from=builder /sync .
COPY backend/migrations ./migrations
EXPOSE 8080
CMD ["./api"]
