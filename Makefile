# =============================================================================
#  goFarmacia — Makefile
#  Wails v2 · Go · Vue 3 · PostgreSQL
#
#  Uso rápido:
#    make              → muestra esta ayuda
#    make dev          → desarrollo con hot-reload (Linux)
#    make build        → binario de producción para Linux
#    make build-win    → binario de producción para Windows (requiere mingw-w64)
#    make dist         → paquetes ZIP para Linux y Windows
#    make clean        → limpia artefactos de build
# =============================================================================

# ── Variables ─────────────────────────────────────────────────────────────────
APP_NAME      := goFarmacia
VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_BUILD_TAGS := webkit2_41

# Directorios
BUILD_DIR     := build/bin
DIST_DIR      := dist

# Cross-compiler para Windows (desde Linux)
WIN_CC        := x86_64-w64-mingw32-gcc

# Wails flags comunes
WAILS_FLAGS   := -tags "$(GO_BUILD_TAGS)" \
                 -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"

# ── Colores para la terminal ──────────────────────────────────────────────────
BOLD  := \033[1m
GREEN := \033[32m
CYAN  := \033[36m
YELLOW:= \033[33m
RED   := \033[31m
RESET := \033[0m

# =============================================================================
#  DEFAULT — Ayuda
# =============================================================================
.DEFAULT_GOAL := help

.PHONY: help
help: ## Muestra esta ayuda
	@echo ""
	@echo "$(BOLD)$(CYAN)  goFarmacia $(VERSION)$(RESET)"
	@echo "  Sistema de gestión farmacéutica · Wails v2"
	@echo ""
	@echo "$(BOLD)  Desarrollo$(RESET)"
	@grep -E '^(dev|check-deps).*:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*##"}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)  Build$(RESET)"
	@grep -E '^build.*:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*##"}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)  Distribución$(RESET)"
	@grep -E '^dist.*:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*##"}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)  Base de datos$(RESET)"
	@grep -E '^(db|migrate).*:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*##"}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)  Utilidades$(RESET)"
	@grep -E '^(clean|install-deps|version).*:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*##"}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""

# =============================================================================
#  CHECKS
# =============================================================================
.PHONY: check-deps
check-deps: ## Verifica que todas las herramientas estén instaladas
	@echo "$(BOLD)Verificando dependencias...$(RESET)"
	@command -v go        >/dev/null 2>&1 && echo "  $(GREEN)✓$(RESET) go        $$(go version | awk '{print $$3}')" \
	  || (echo "  $(RED)✗$(RESET) go        — instalar desde https://go.dev"; exit 1)
	@command -v wails     >/dev/null 2>&1 && echo "  $(GREEN)✓$(RESET) wails     $$(wails version 2>/dev/null | head -1)" \
	  || (echo "  $(RED)✗$(RESET) wails     — go install github.com/wailsapp/wails/v2/cmd/wails@latest"; exit 1)
	@command -v pnpm      >/dev/null 2>&1 && echo "  $(GREEN)✓$(RESET) pnpm      $$(pnpm --version)" \
	  || (echo "  $(RED)✗$(RESET) pnpm      — npm install -g pnpm"; exit 1)
	@command -v psql      >/dev/null 2>&1 && echo "  $(GREEN)✓$(RESET) psql      $$(psql --version)" \
	  || echo "  $(YELLOW)!$(RESET) psql      — no encontrado (necesario para utilidades de BD)"
	@command -v $(WIN_CC) >/dev/null 2>&1 && echo "  $(GREEN)✓$(RESET) mingw-w64 (build Windows)" \
	  || echo "  $(YELLOW)!$(RESET) mingw-w64 — no instalado (solo necesario para 'make build-win')"
	@command -v zip       >/dev/null 2>&1 && echo "  $(GREEN)✓$(RESET) zip       (empaquetado dist)" \
	  || echo "  $(YELLOW)!$(RESET) zip       — no instalado (necesario para 'make dist')"
	@echo ""
	@echo "$(GREEN)Check completado.$(RESET)"

.PHONY: install-deps
install-deps: ## Instala mingw-w64 y zip (requiere sudo en Linux)
	@echo "$(BOLD)Instalando dependencias del sistema...$(RESET)"
	sudo apt-get update -qq
	sudo apt-get install -y gcc-mingw-w64-x86-64 zip
	@echo "$(GREEN)Dependencias instaladas.$(RESET)"
	@echo "  Instalar Wails CLI si falta:  go install github.com/wailsapp/wails/v2/cmd/wails@latest"

# =============================================================================
#  DESARROLLO
# =============================================================================
.PHONY: dev
dev: ## Inicia el servidor de desarrollo con hot-reload (Linux)
	@echo "$(BOLD)$(CYAN)→ Iniciando dev server...$(RESET)"
	wails dev -tags $(GO_BUILD_TAGS)

# =============================================================================
#  BUILD
# =============================================================================
.PHONY: build
build: ## Compila para Linux (nativo) → build/bin/goFarmacia
	@echo "$(BOLD)$(CYAN)→ Compilando para Linux ($(shell go env GOARCH))...$(RESET)"
	@echo "  Versión: $(VERSION)  Fecha: $(BUILD_DATE)"
	wails build \
		-platform linux/amd64 \
		-tags "$(GO_BUILD_TAGS)" \
		-ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)" \
		-clean \
		-o $(APP_NAME)
	@echo ""
	@echo "$(GREEN)✓ Binario:$(RESET) $(BUILD_DIR)/$(APP_NAME)"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)

.PHONY: build-win
build-win: ## Compila para Windows x64 → build/bin/goFarmacia.exe  [requiere mingw-w64]
	@command -v $(WIN_CC) >/dev/null 2>&1 || \
		(echo "$(RED)Error:$(RESET) mingw-w64 no encontrado. Ejecuta 'make install-deps' primero." && exit 1)
	@echo "$(BOLD)$(CYAN)→ Compilando para Windows/amd64...$(RESET)"
	@echo "  Versión: $(VERSION)  Fecha: $(BUILD_DATE)"
	CC=$(WIN_CC) wails build \
		-platform windows/amd64 \
		-tags "$(GO_BUILD_TAGS)" \
		-ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE) -H windowsgui" \
		-clean \
		-o $(APP_NAME).exe
	@echo ""
	@echo "$(GREEN)✓ Binario:$(RESET) $(BUILD_DIR)/$(APP_NAME).exe"
	@ls -lh $(BUILD_DIR)/$(APP_NAME).exe

.PHONY: build-all
build-all: build build-win ## Compila para Linux y Windows

.PHONY: build-debug
build-debug: ## Compila Linux con devtools habilitados (debug)
	@echo "$(BOLD)$(YELLOW)→ Compilando en modo debug (con devtools)...$(RESET)"
	wails build \
		-platform linux/amd64 \
		-tags "$(GO_BUILD_TAGS)" \
		-debug \
		-devtools \
		-o $(APP_NAME)-debug

# =============================================================================
#  DISTRIBUCIÓN
# =============================================================================
.PHONY: dist-dirs
dist-dirs:
	@mkdir -p $(DIST_DIR)/linux $(DIST_DIR)/windows

.PHONY: dist-linux
dist-linux: build dist-dirs ## Empaqueta distribución Linux (.zip)
	@echo "$(BOLD)$(CYAN)→ Empaquetando Linux...$(RESET)"
	@cp $(BUILD_DIR)/$(APP_NAME) $(DIST_DIR)/linux/
	@cp .env.example              $(DIST_DIR)/linux/ 2>/dev/null || \
		echo "DATABASE_URL=postgresql://user:pass@localhost:5432/farmacia_db?sslmode=disable\nJWT_SECRET_KEY=cambia_esto" \
		> $(DIST_DIR)/linux/.env.example
	@cp README.md  $(DIST_DIR)/linux/
	@cp CHANGELOG.md $(DIST_DIR)/linux/
	@cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-linux-amd64.zip linux/
	@echo "$(GREEN)✓ Paquete:$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.zip"
	@ls -lh $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.zip

.PHONY: dist-win
dist-win: build-win dist-dirs ## Empaqueta distribución Windows (.zip)
	@echo "$(BOLD)$(CYAN)→ Empaquetando Windows...$(RESET)"
	@cp $(BUILD_DIR)/$(APP_NAME).exe $(DIST_DIR)/windows/
	@cp .env.example                  $(DIST_DIR)/windows/ 2>/dev/null || \
		printf "DATABASE_URL=postgresql://user:pass@localhost:5432/farmacia_db?sslmode=disable\nJWT_SECRET_KEY=cambia_esto\n" \
		> $(DIST_DIR)/windows/.env.example
	@cp README.md    $(DIST_DIR)/windows/
	@cp CHANGELOG.md $(DIST_DIR)/windows/
	@cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-windows-amd64.zip windows/
	@echo "$(GREEN)✓ Paquete:$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.zip"
	@ls -lh $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.zip

.PHONY: dist
dist: dist-linux dist-win ## Empaqueta distribuciones para Linux y Windows
	@echo ""
	@echo "$(BOLD)$(GREEN)✓ Distribuciones generadas en $(DIST_DIR)/$(RESET)"
	@ls -lh $(DIST_DIR)/*.zip 2>/dev/null

# =============================================================================
#  BASE DE DATOS
# =============================================================================
.PHONY: db-reset
db-reset: ## Restaura la BD desde backups de Supabase (backend/python/)
	@echo "$(BOLD)$(YELLOW)→ Restaurando base de datos desde backup...$(RESET)"
	@echo "  $(YELLOW)Advertencia: esto borrará todos los datos actuales.$(RESET)"
	@read -p "  ¿Continuar? [s/N] " confirm && [ "$$confirm" = "s" ] || exit 0
	cd backend/python && bash reset_and_import.sh

.PHONY: db-status
db-status: ## Muestra el estado de las migraciones
	@echo "$(BOLD)Estado de migraciones:$(RESET)"
	@if [ -f .env ]; then \
		export $$(grep -v '^#' .env | xargs) && \
		psql "$$DATABASE_URL" -c "SELECT version, dirty FROM schema_migrations ORDER BY version;" 2>/dev/null \
		|| echo "$(RED)No se pudo conectar a la BD. Verifica .env$(RESET)"; \
	else \
		echo "$(RED)Archivo .env no encontrado.$(RESET)"; \
	fi

.PHONY: db-users
db-users: ## Lista los usuarios registrados y sus roles
	@if [ -f .env ]; then \
		export $$(grep -v '^#' .env | xargs) && \
		psql "$$DATABASE_URL" -c \
			"SELECT nombre, apellido, email, role, created_at::date FROM vendedors WHERE deleted_at IS NULL ORDER BY role DESC, nombre;" \
		2>/dev/null || echo "$(RED)Error de conexión.$(RESET)"; \
	else \
		echo "$(RED)Archivo .env no encontrado.$(RESET)"; \
	fi

.PHONY: db-make-admin
db-make-admin: ## Promueve un usuario a admin: make db-make-admin EMAIL=user@example.com
	@[ -n "$(EMAIL)" ] || (echo "$(RED)Uso: make db-make-admin EMAIL=user@example.com$(RESET)" && exit 1)
	@if [ -f .env ]; then \
		export $$(grep -v '^#' .env | xargs) && \
		psql "$$DATABASE_URL" -c \
			"UPDATE vendedors SET role='admin' WHERE email='$(EMAIL)' AND deleted_at IS NULL RETURNING nombre, email, role;" \
		2>/dev/null || echo "$(RED)Error de conexión o usuario no encontrado.$(RESET)"; \
	else \
		echo "$(RED)Archivo .env no encontrado.$(RESET)"; \
	fi

# =============================================================================
#  UTILIDADES
# =============================================================================
.PHONY: version
version: ## Muestra la versión actual del proyecto
	@echo "$(APP_NAME) $(VERSION) (build $(BUILD_DATE))"

.PHONY: tidy
tidy: ## Ejecuta go mod tidy y pnpm install
	@echo "$(BOLD)→ go mod tidy$(RESET)"
	go mod tidy
	@echo "$(BOLD)→ pnpm install$(RESET)"
	cd frontend && pnpm install

.PHONY: clean
clean: ## Elimina binarios y artefactos de build
	@echo "$(BOLD)$(YELLOW)→ Limpiando...$(RESET)"
	@rm -f  $(BUILD_DIR)/$(APP_NAME)
	@rm -f  $(BUILD_DIR)/$(APP_NAME).exe
	@rm -f  $(BUILD_DIR)/$(APP_NAME)-debug
	@rm -rf $(DIST_DIR)/linux $(DIST_DIR)/windows
	@rm -f  $(DIST_DIR)/$(APP_NAME)-*.zip
	@echo "$(GREEN)✓ Limpio.$(RESET)"

.PHONY: clean-all
clean-all: clean ## Limpia también node_modules y cachés del frontend
	@echo "$(BOLD)$(YELLOW)→ Limpiando frontend...$(RESET)"
	@rm -rf frontend/node_modules frontend/dist
	@echo "$(GREEN)✓ Frontend limpio.$(RESET)"
