# =============================================================================
#  goFarmacia — Makefile
#  Echo HTTP · Go · Vue 3 · PostgreSQL · Gmail API (DIAN)
#
#  Uso rápido:
#    make              → muestra ayuda
#    make dev          → desarrollo con hot-reload (Go + Vite en paralelo)
#    make build        → compila frontend y binario Go con frontend embebido
#    make run          → build + ejecutar (libera el puerto si está ocupado)
#    make start        → ejecutar el binario ya compilado (libera puerto si ocupado)
#    make clean        → limpia artefactos
# =============================================================================

# ── Detección de SO ───────────────────────────────────────────────────────────
UNAME := $(shell uname -s)
ARCH  := $(shell uname -m)

ifeq ($(UNAME), Darwin)
  HOST_OS       := mac
  GO_BUILD_TAGS :=
else
  HOST_OS       := linux
  GO_BUILD_TAGS := webkit2_41
endif

# ── Variables generales ───────────────────────────────────────────────────────
APP_NAME   := goFarmacia
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)

PORT       := 6969
BUILD_DIR  := build/bin
DIST_DIR   := dist
BINARY     := $(BUILD_DIR)/$(APP_NAME)

FRONTEND_DIR := frontend
FRONTEND_DIST := $(FRONTEND_DIR)/dist

# Config de usuario en tiempo de ejecución
CONFIG_DIR    := $(HOME)/.config/goFarmacia
ENV_FILE      := .env
ENV_EXAMPLE   := .env.example
CREDS_EXAMPLE := credentials.example.json
CREDS_DEST    := $(CONFIG_DIR)/credentials.json

# Cross-compiler Windows
WIN_CC := x86_64-w64-mingw32-gcc

# ── Colores ───────────────────────────────────────────────────────────────────
BOLD   := \033[1m
GREEN  := \033[32m
CYAN   := \033[36m
YELLOW := \033[33m
RED    := \033[31m
BLUE   := \033[34m
RESET  := \033[0m

# =============================================================================
#  AYUDA
# =============================================================================
.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo ""
	@echo "$(BOLD)$(CYAN)  goFarmacia $(VERSION)$(RESET)  ·  $(HOST_OS)/$(ARCH)"
	@echo "  Sistema de gestión farmacéutica · Echo HTTP"
	@echo ""
	@echo "$(BOLD)  Desarrollo$(RESET)"
	@echo "  $(GREEN)dev$(RESET)              Go server + Vite hot-reload en paralelo"
	@echo "  $(GREEN)check-deps$(RESET)       Verifica herramientas instaladas"
	@echo "  $(GREEN)check-env$(RESET)        Verifica que .env y credenciales existen"
	@echo ""
	@echo "$(BOLD)  Build$(RESET)"
	@echo "  $(GREEN)build$(RESET)            Frontend (pnpm build) + binario Go embebido"
	@echo "  $(GREEN)build-frontend$(RESET)   Solo compila el frontend (pnpm build)"
	@echo "  $(GREEN)build-go$(RESET)         Solo compila el binario Go (requiere dist/ previo)"
	@echo "  $(GREEN)build-win$(RESET)        Binario Windows/amd64 (requiere mingw-w64)"
	@echo ""
	@echo "$(BOLD)  Ejecución$(RESET)"
	@echo "  $(GREEN)run$(RESET)              build + libera puerto $(PORT) + ejecuta"
	@echo "  $(GREEN)start$(RESET)            Libera puerto $(PORT) y ejecuta binario existente"
	@echo "  $(GREEN)stop$(RESET)             Mata el proceso en puerto $(PORT)"
	@echo ""
	@echo "$(BOLD)  Distribución$(RESET)"
	@echo "  $(GREEN)dist-linux$(RESET)       ZIP Linux autónomo"
	@echo "  $(GREEN)dist-win$(RESET)         ZIP Windows"
	@echo "  $(GREEN)dist$(RESET)             Todos los paquetes"
	@echo ""
	@echo "$(BOLD)  Base de datos$(RESET)"
	@echo "  $(GREEN)db-status$(RESET)        Estado de migraciones"
	@echo "  $(GREEN)db-users$(RESET)         Lista usuarios y roles"
	@echo "  $(GREEN)db-create-admin$(RESET)  Crear primer administrador"
	@echo "  $(GREEN)db-make-admin$(RESET)    Promover a admin: EMAIL=x@y.com"
	@echo "  $(GREEN)db-reset$(RESET)         Restaurar BD desde SQL"
	@echo ""
	@echo "$(BOLD)  Utilidades$(RESET)"
	@echo "  $(GREEN)tidy$(RESET)             go mod tidy + pnpm install"
	@echo "  $(GREEN)install-deps$(RESET)     Instala mingw-w64 + zip (Linux/apt)"
	@echo "  $(GREEN)version$(RESET)          Muestra versión del proyecto"
	@echo "  $(GREEN)clean$(RESET)            Limpia binarios y ZIPs"
	@echo "  $(GREEN)clean-all$(RESET)        + node_modules y dist frontend"
	@echo ""
	@echo "  Puerto por defecto: $(BOLD)$(PORT)$(RESET)  (override: make run PORT=8080)"
	@echo ""

# =============================================================================
#  CHECKS Y DEPENDENCIAS
# =============================================================================
.PHONY: check-deps
check-deps:
	@echo "$(BOLD)Verificando dependencias ($(HOST_OS))...$(RESET)"
	@command -v go   >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) go      $$(go version | awk '{print $$3}')" \
	  || (echo "  $(RED)✗$(RESET) go      → https://go.dev/dl" && exit 1)
	@command -v pnpm >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) pnpm    $$(pnpm --version)" \
	  || (echo "  $(RED)✗$(RESET) pnpm    → npm install -g pnpm" && exit 1)
	@command -v psql >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) psql    $$(psql --version | head -1)" \
	  || echo "  $(YELLOW)!$(RESET) psql    — no encontrado (necesario para utilidades de BD)"
	@command -v $(WIN_CC) >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) mingw-w64 (cross-compile Windows)" \
	  || echo "  $(YELLOW)!$(RESET) mingw-w64 — make install-deps (solo para build-win)"
	@command -v zip >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) zip     (empaquetado dist)" \
	  || echo "  $(YELLOW)!$(RESET) zip     — necesario para make dist"
	@echo ""
	@echo "$(GREEN)Check completado.$(RESET)"

.PHONY: check-env
check-env:
	@echo "$(BOLD)Verificando archivos de configuración...$(RESET)"
	@if [ -f "$(ENV_FILE)" ]; then \
	  echo "  $(GREEN)✓$(RESET) $(ENV_FILE)"; \
	  grep -qE "^DATABASE_URL=.+" $(ENV_FILE) \
	    && echo "  $(GREEN)✓$(RESET)   DATABASE_URL configurado" \
	    || echo "  $(YELLOW)!$(RESET)   DATABASE_URL no configurado en $(ENV_FILE)"; \
	  grep -qE "^JWT_SECRET_KEY=.+" $(ENV_FILE) \
	    && echo "  $(GREEN)✓$(RESET)   JWT_SECRET_KEY configurado" \
	    || echo "  $(YELLOW)!$(RESET)   JWT_SECRET_KEY no configurado en $(ENV_FILE)"; \
	else \
	  echo "  $(RED)✗$(RESET) $(ENV_FILE) no encontrado"; \
	  echo "      Copia $(ENV_EXAMPLE) → $(ENV_FILE) y ajusta los valores"; \
	fi
	@if [ -f "$(CREDS_DEST)" ]; then \
	  echo "  $(GREEN)✓$(RESET) credentials.json — Gmail OAuth2 ($(CONFIG_DIR))"; \
	else \
	  echo "  $(YELLOW)!$(RESET) credentials.json NO encontrado — $(CONFIG_DIR)/credentials.json"; \
	fi
	@echo ""

.PHONY: install-deps
install-deps:
	@echo "$(BOLD)Instalando dependencias del sistema (Linux/apt)...$(RESET)"
	sudo apt-get update -qq
	sudo apt-get install -y gcc-mingw-w64-x86-64 zip pkg-config
	@echo "$(GREEN)✓ Listo.$(RESET)"

# =============================================================================
#  PORT MANAGEMENT
# =============================================================================

# Libera el puerto si está ocupado; no falla si ya está libre.
.PHONY: _free-port
_free-port:
	@PID=$$(lsof -ti tcp:$(PORT) 2>/dev/null); \
	if [ -n "$$PID" ]; then \
	  echo "  $(YELLOW)!$(RESET) Puerto $(PORT) ocupado por PID $$PID — terminando proceso..."; \
	  kill -TERM $$PID 2>/dev/null || true; \
	  sleep 1; \
	  kill -9 $$PID 2>/dev/null || true; \
	  echo "  $(GREEN)✓$(RESET) Puerto $(PORT) liberado."; \
	else \
	  echo "  $(GREEN)✓$(RESET) Puerto $(PORT) disponible."; \
	fi

.PHONY: stop
stop:
	@echo "$(BOLD)Deteniendo servidor en puerto $(PORT)...$(RESET)"
	@$(MAKE) _free-port PORT=$(PORT)

# =============================================================================
#  DESARROLLO
# =============================================================================
.PHONY: dev
dev:
	@echo "$(BOLD)$(CYAN)→ Modo desarrollo: Go :$(PORT) + Vite hot-reload$(RESET)"
	@$(MAKE) _free-port PORT=$(PORT)
	@trap 'kill 0' INT TERM; \
	  PORT=$(PORT) go run -tags "$(GO_BUILD_TAGS)" . & \
	  sleep 1 && cd $(FRONTEND_DIR) && pnpm dev & \
	  wait

# =============================================================================
#  BUILD
# =============================================================================

# Compila el frontend (pnpm build)
.PHONY: build-frontend
build-frontend:
	@echo "$(BOLD)$(CYAN)→ Compilando frontend...$(RESET)"
	cd $(FRONTEND_DIR) && pnpm install --frozen-lockfile && pnpm build
	@echo "$(GREEN)✓ Frontend compilado en $(FRONTEND_DIST)/$(RESET)"

# Compila solo el binario Go (el frontend/dist debe existir para el embed)
.PHONY: build-go
build-go:
	@echo "$(BOLD)$(CYAN)→ Compilando binario Go ($(HOST_OS))...$(RESET)"
	@echo "  Versión: $(VERSION)  Fecha: $(BUILD_DATE)"
	@mkdir -p $(BUILD_DIR)
	go build \
		-tags "$(GO_BUILD_TAGS)" \
		-ldflags "$(LDFLAGS)" \
		-o $(BINARY) \
		.
	@echo "$(GREEN)✓ Binario:$(RESET) $(BINARY)  ($$(du -sh $(BINARY) | cut -f1))"

# Build completo: frontend embebido en el binario Go
.PHONY: build
build: build-frontend build-go
	@$(MAKE) _copy-env TARGET_DIR=$(BUILD_DIR)
	@echo ""
	@echo "$(BOLD)$(GREEN)✓ Build completo.$(RESET)"
	@echo "  Binario : $(BINARY)"
	@echo "  Ejecutar: make start  (o PORT=XXXX make start)"

# Build Windows desde Linux (requiere mingw-w64)
.PHONY: build-win
build-win: build-frontend
	@command -v $(WIN_CC) >/dev/null 2>&1 || \
	  (echo "$(RED)Error:$(RESET) mingw-w64 no encontrado — make install-deps" && exit 1)
	@echo "$(BOLD)$(CYAN)→ Compilando Windows/amd64...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	CC=$(WIN_CC) GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
	  go build \
	    -ldflags "$(LDFLAGS) -H windowsgui" \
	    -o $(BUILD_DIR)/$(APP_NAME).exe \
	    .
	@echo "$(GREEN)✓ $(BUILD_DIR)/$(APP_NAME).exe$(RESET)  ($$(du -sh $(BUILD_DIR)/$(APP_NAME).exe | cut -f1))"
	@$(MAKE) _copy-env TARGET_DIR=$(BUILD_DIR)

# =============================================================================
#  EJECUCIÓN
# =============================================================================

# build + free port + run
.PHONY: run
run: build
	@echo "$(BOLD)$(CYAN)→ Iniciando servidor en http://localhost:$(PORT)$(RESET)"
	@$(MAKE) _free-port PORT=$(PORT)
	@cd $(BUILD_DIR) && PORT=$(PORT) ./$(APP_NAME)

# free port + run (sin recompilar)
.PHONY: start
start:
	@[ -f "$(BINARY)" ] || (echo "$(RED)Binario no encontrado. Ejecuta 'make build' primero.$(RESET)" && exit 1)
	@echo "$(BOLD)$(CYAN)→ Iniciando servidor en http://localhost:$(PORT)$(RESET)"
	@$(MAKE) _free-port PORT=$(PORT)
	@cd $(BUILD_DIR) && PORT=$(PORT) ./$(APP_NAME)

# =============================================================================
#  HELPERS INTERNOS
# =============================================================================
.PHONY: _copy-env
_copy-env:
	@if [ -f "$(ENV_FILE)" ]; then \
	  cp "$(ENV_FILE)" "$(TARGET_DIR)/.env"; \
	  echo "  $(GREEN)✓$(RESET) .env copiado a $(TARGET_DIR)/"; \
	elif [ -f "$(ENV_EXAMPLE)" ]; then \
	  cp "$(ENV_EXAMPLE)" "$(TARGET_DIR)/.env.example"; \
	  echo "  $(YELLOW)!$(RESET) .env no encontrado — copiado .env.example (renómbralo y ajusta DATABASE_URL)"; \
	else \
	  printf 'DATABASE_URL=postgresql://usuario:password@localhost:5432/farmacia_db?sslmode=disable\nJWT_SECRET_KEY=cambia_esto_por_una_clave_segura_de_32_caracteres_minimo\nPORT=$(PORT)\n' \
	    > "$(TARGET_DIR)/.env.example"; \
	  echo "  $(YELLOW)!$(RESET) .env.example mínimo generado en $(TARGET_DIR)/"; \
	fi

.PHONY: _copy-creds-example
_copy-creds-example:
	@if [ -f "$(CREDS_EXAMPLE)" ]; then \
	  cp "$(CREDS_EXAMPLE)" "$(TARGET_DIR)/$(CREDS_EXAMPLE)"; \
	  echo "  $(GREEN)✓$(RESET) credentials.example.json copiado"; \
	else \
	  printf '{"installed":{"client_id":"","client_secret":"","redirect_uris":["http://localhost:8094/gmail/oauth2/callback"]}}\n' \
	    > "$(TARGET_DIR)/$(CREDS_EXAMPLE)"; \
	  echo "  $(YELLOW)!$(RESET) credentials.example.json generado en $(TARGET_DIR)/"; \
	fi

.PHONY: _copy-common
_copy-common:
	@$(MAKE) _copy-env TARGET_DIR="$(TARGET_DIR)"
	@$(MAKE) _copy-creds-example TARGET_DIR="$(TARGET_DIR)"
	@[ -f README.md    ] && cp README.md    "$(TARGET_DIR)/" || true
	@[ -f CHANGELOG.md ] && cp CHANGELOG.md "$(TARGET_DIR)/" || true

# =============================================================================
#  DISTRIBUCIÓN
# =============================================================================
.PHONY: dist-linux
dist-linux: build
	@echo "$(BOLD)$(CYAN)→ Empaquetando Linux...$(RESET)"
	@mkdir -p $(DIST_DIR)/linux
	@cp $(BINARY) $(DIST_DIR)/linux/
	@$(MAKE) _copy-common TARGET_DIR=$(DIST_DIR)/linux
	@cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-linux-amd64.zip linux/
	@echo "$(GREEN)✓$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.zip"
	@ls -lh $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.zip

.PHONY: dist-win
dist-win: build-win
	@echo "$(BOLD)$(CYAN)→ Empaquetando Windows...$(RESET)"
	@mkdir -p $(DIST_DIR)/windows
	@cp $(BUILD_DIR)/$(APP_NAME).exe $(DIST_DIR)/windows/
	@$(MAKE) _copy-common TARGET_DIR=$(DIST_DIR)/windows
	@cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-windows-amd64.zip windows/
	@echo "$(GREEN)✓$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.zip"
	@ls -lh $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.zip

.PHONY: dist
dist:
	$(MAKE) dist-linux dist-win
	@echo ""
	@echo "$(BOLD)$(GREEN)✓ Distribuciones en $(DIST_DIR)/$(RESET)"
	@ls -lh $(DIST_DIR)/*.zip 2>/dev/null || true

# =============================================================================
#  BASE DE DATOS
# =============================================================================
.PHONY: db-create-admin
db-create-admin:
	@([ -n "$(NOMBRE)" ] && [ -n "$(APELLIDO)" ] && [ -n "$(EMAIL)" ] && [ -n "$(CEDULA)" ] && [ -n "$(PASSWORD)" ]) || \
	  (echo "$(RED)Uso: make db-create-admin NOMBRE=Juan APELLIDO=Pérez EMAIL=admin@ejemplo.com CEDULA=12345678 PASSWORD=miClave$(RESET)" && exit 1)
	@[ -f .env ] || (echo "$(RED)Error: .env no encontrado.$(RESET)" && exit 1)
	cd cmd/initadmin && go run . \
		-nombre   "$(NOMBRE)" \
		-apellido "$(APELLIDO)" \
		-email    "$(EMAIL)" \
		-cedula   "$(CEDULA)" \
		-password "$(PASSWORD)"

.PHONY: db-reset
db-reset:
	@echo "$(BOLD)$(YELLOW)→ Restaurando base de datos...$(RESET)"
	@echo "  Destino : DATABASE_URL del .env"
ifdef SOURCE
	@echo "  Origen  : $(SOURCE)"
else
	@echo "  Origen  : backend/db/backup_supabase.sql (por defecto)"
endif
	@echo "  $(YELLOW)ADVERTENCIA: borrará todos los datos actuales.$(RESET)"
	@read -p "  ¿Continuar? [s/N] " confirm && [ "$$confirm" = "s" ] || exit 0
	@bash backend/python/reset_and_import.sh "$(SOURCE)"

.PHONY: db-status
db-status:
	@echo "$(BOLD)Estado de migraciones:$(RESET)"
	@[ -f .env ] || (echo "$(RED).env no encontrado.$(RESET)" && exit 1)
	@export $$(grep -v '^#' .env | xargs) && \
	  psql "$$DATABASE_URL" -c \
	    "SELECT version, dirty FROM schema_migrations ORDER BY version;" 2>/dev/null \
	  || echo "$(RED)No se pudo conectar a la BD.$(RESET)"

.PHONY: db-users
db-users:
	@[ -f .env ] || (echo "$(RED).env no encontrado.$(RESET)" && exit 1)
	@export $$(grep -v '^#' .env | xargs) && \
	  psql "$$DATABASE_URL" -c \
	    "SELECT nombre, apellido, email, role, created_at::date AS desde \
	     FROM vendedors WHERE deleted_at IS NULL ORDER BY role DESC, nombre;" \
	  2>/dev/null || echo "$(RED)Error de conexión.$(RESET)"

.PHONY: db-make-admin
db-make-admin:
	@[ -n "$(EMAIL)" ] || \
	  (echo "$(RED)Uso: make db-make-admin EMAIL=user@example.com$(RESET)" && exit 1)
	@[ -f .env ] || (echo "$(RED).env no encontrado.$(RESET)" && exit 1)
	@export $$(grep -v '^#' .env | xargs) && \
	  psql "$$DATABASE_URL" -c \
	    "UPDATE vendedors SET role='admin' WHERE email='$(EMAIL)' AND deleted_at IS NULL RETURNING nombre, email, role;" \
	  2>/dev/null || echo "$(RED)Error de conexión.$(RESET)"

# =============================================================================
#  GMAIL
# =============================================================================
.PHONY: gmail-setup
gmail-setup:
	@echo "$(BOLD)Configuración Gmail OAuth2:$(RESET)"
	@echo ""
	@echo "  1. Ve a https://console.cloud.google.com/apis/credentials"
	@echo "  2. Crea credenciales OAuth 2.0 → Aplicación de escritorio"
	@echo "  3. Descarga el JSON y guárdalo como credentials.json en la raíz del proyecto"
	@echo "     O en: $(CONFIG_DIR)/credentials.json"
	@echo ""
	@if [ -f "$(CREDS_DEST)" ]; then \
	  echo "  $(GREEN)✓$(RESET) credentials.json encontrado en $(CONFIG_DIR)/"; \
	else \
	  echo "  $(YELLOW)!$(RESET) credentials.json NO encontrado"; \
	fi

.PHONY: gmail-revoke
gmail-revoke:
	@[ -f "$(CONFIG_DIR)/gmail_token.json" ] && \
	  rm "$(CONFIG_DIR)/gmail_token.json" && \
	  echo "$(GREEN)✓$(RESET) Token Gmail eliminado — se pedirá re-autenticación." || \
	  echo "$(YELLOW)!$(RESET) gmail_token.json no encontrado."

# =============================================================================
#  UTILIDADES
# =============================================================================
.PHONY: tidy
tidy:
	@echo "$(BOLD)Actualizando dependencias...$(RESET)"
	go mod tidy
	cd $(FRONTEND_DIR) && pnpm install
	@echo "$(GREEN)✓ Listo.$(RESET)"

.PHONY: version
version:
	@echo "$(APP_NAME) $(VERSION) — $(BUILD_DATE)"
	@echo "Go: $$(go version)"
	@echo "OS: $(HOST_OS)/$(ARCH)"

.PHONY: clean
clean:
	@echo "$(BOLD)Limpiando artefactos...$(RESET)"
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	@echo "$(GREEN)✓ Limpiado.$(RESET)"

.PHONY: clean-all
clean-all: clean
	@echo "$(BOLD)Limpiando node_modules y dist frontend...$(RESET)"
	rm -rf $(FRONTEND_DIR)/node_modules $(FRONTEND_DIR)/dist
	@echo "$(GREEN)✓ Limpiado todo.$(RESET)"
