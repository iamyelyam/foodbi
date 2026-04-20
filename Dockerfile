FROM golang:alpine AS builder

# CACHE_BUST forces Docker to invalidate all downstream layers whenever we
# bump this value — essential on Railway where BuildKit occasionally reuses
# stale Go build caches and ships binaries built from old source.
ARG CACHE_BUST=2026-04-20-fixedzone-v2

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# -a forces rebuild of all packages (even cached), -trimpath strips absolute
# paths for reproducible output. Together they guarantee the binary reflects
# the current source tree, never a stale cache.
RUN CGO_ENABLED=0 GOOS=linux go build -a -trimpath -o /api ./cmd/api/
RUN CGO_ENABLED=0 GOOS=linux go build -a -trimpath -o /sync ./cmd/sync/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /api .
COPY --from=builder /sync .
COPY backend/migrations ./migrations
COPY start-all.sh .
RUN chmod +x start-all.sh ./api ./sync
EXPOSE 8080
CMD ["./start-all.sh"]
