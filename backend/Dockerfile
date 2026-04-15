FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api/
RUN CGO_ENABLED=0 GOOS=linux go build -o /sync ./cmd/sync/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /api .
COPY --from=builder /sync .
COPY migrations ./migrations
EXPOSE 8080
CMD ["./api"]
