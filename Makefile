# GoWatch Makefile

APP_NAME=gowatch
BUILD_DIR=./bin
MAIN=./cmd/server/main.go

.PHONY: run build clean dev

# Geliştirme modunda çalıştır
run:
	go run $(MAIN)

# Production binary oluştur
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)
	@echo "✅ Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Binary'yi çalıştır
start: build
	$(BUILD_DIR)/$(APP_NAME)

# Temizle
clean:
	rm -rf $(BUILD_DIR) gowatch.db

# Test
test:
	go test ./...

# Lint
lint:
	go vet ./...

# Docker imajı oluştur
docker-build:
	docker compose build

# Docker ile başlat (arka planda)
docker-up:
	docker compose up -d

# Docker logları
docker-logs:
	docker compose logs -f

# Docker durdur
docker-down:
	docker compose down

# Docker yeniden başlat
docker-restart:
	docker compose restart

# Docker durdur + imajı sil (sıfırdan başlamak için)
docker-reset:
	docker compose down --rmi local
	docker compose build
	docker compose up -d

# Docker durumu
docker-status:
	docker compose ps
