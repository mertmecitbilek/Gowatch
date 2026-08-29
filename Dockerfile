# ================================
# Stage 1: Builder
# ================================
FROM golang:1.25-alpine AS builder

# CGO için gcc gerekli (SQLite)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Bağımlılıkları önce kopyala (layer cache için)
COPY go.mod go.sum ./
RUN GOTOOLCHAIN=auto go mod download

# Kaynak kodları kopyala
COPY . .

# Binary derle
RUN GOTOOLCHAIN=auto CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o gowatch ./cmd/server/main.go

# ================================
# Stage 2: Runtime (minimal image)
# ================================
FROM alpine:3.19

# Timezone ve CA sertifikalar
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Derlenmiş binary'yi kopyala
COPY --from=builder /app/gowatch .

# Web dosyalarını kopyala
COPY --from=builder /app/web ./web

# Veri dizini (SQLite DB için)
RUN mkdir -p /app/data

# Port aç
EXPOSE 8080

# SQLite DB'yi /app/data içinde tut (volume mount için)
ENV DATABASE_PATH=/app/data/gowatch.db

# Non-root user (güvenlik)
RUN addgroup -S gowatch && adduser -S gowatch -G gowatch
RUN chown -R gowatch:gowatch /app
USER gowatch

# Uygulama çalıştır
CMD ["./gowatch"]
