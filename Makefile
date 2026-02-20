# =============================================================================
#  goFarmacia — Makefile
#  Wails v2 · Go · Vue 3 · PostgreSQL
#
#  Uso rápido:
#    make              → muestra ayuda
#    make dev          → desarrollo con hot-reload (detecta SO actual)
#    make build        → binario/app nativo del SO actual
#    make build-mac    → .app universal macOS  [solo en Mac]
#    make build-win    → .exe Windows          [requiere mingw-w64 en Linux]
#    make dist         → ZIPs/DMG para todos los SO compilados
#    make clean        → limpia artefactos
# =============================================================================

# ── Detección de SO ───────────────────────────────────────────────────────────
UNAME := $(shell uname -s)
ARCH  := $(shell uname -m)

ifeq ($(UNAME), Darwin)
  # ── macOS ──
  HOST_OS       := mac
  GO_BUILD_TAGS :=                         # WebKit incluido en macOS, sin tags extra
  DEV_FLAGS     :=
  ifeq ($(ARCH), arm64)
    HOST_PLATFORM := darwin/arm64
  else
    HOST_PLATFORM := darwin/amd64
  endif
else
  # ── Linux (default) ──
  HOST_OS       := linux
  GO_BUILD_TAGS := webkit2_41              # Requiere webkit2gtk-4.1
  DEV_FLAGS     := -tags $(GO_BUILD_TAGS)
  HOST_PLATFORM := linux/amd64
endif

# ── Variables generales ───────────────────────────────────────────────────────
APP_NAME   := goFarmacia
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS    := -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)

# Directorios
BUILD_DIR  := build/bin
DIST_DIR   := dist

# Cross-compiler Windows (desde Linux/Mac con mingw-w64)
WIN_CC     := x86_64-w64-mingw32-gcc

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
	@echo "  Sistema de gestión farmacéutica · Wails v2"
	@echo ""
	@echo "$(BOLD)  Desarrollo$(RESET)"
	@echo "  $(GREEN)dev$(RESET)                   Hot-reload nativo ($(HOST_OS))"
	@echo "  $(GREEN)check-deps$(RESET)             Verifica herramientas instaladas"
	@echo ""
	@echo "$(BOLD)  Build — Nativo$(RESET)"
	@echo "  $(GREEN)build$(RESET)                  Compila para $(HOST_OS) ($(HOST_PLATFORM))"
	@echo "  $(GREEN)build-debug$(RESET)            Compila con devtools habilitados"
	@echo ""
	@echo "$(BOLD)  Build — macOS$(RESET)  $(YELLOW)[ejecutar en Mac]$(RESET)"
	@echo "  $(GREEN)build-mac$(RESET)              Universal .app (arm64 + amd64)"
	@echo "  $(GREEN)build-mac-arm64$(RESET)        .app solo Apple Silicon (M1–M4)"
	@echo "  $(GREEN)build-mac-amd64$(RESET)        .app solo Intel"
	@echo ""
	@echo "$(BOLD)  Build — Windows$(RESET)  $(YELLOW)[requiere mingw-w64]$(RESET)"
	@echo "  $(GREEN)build-win$(RESET)              .exe Windows/amd64"
	@echo ""
	@echo "$(BOLD)  Build — Todos$(RESET)"
	@echo "  $(GREEN)build-all$(RESET)              Nativo + Windows  (+ Mac si es Mac)"
	@echo ""
	@echo "$(BOLD)  Distribución$(RESET)"
	@echo "  $(GREEN)dist-linux$(RESET)             ZIP Linux"
	@echo "  $(GREEN)dist-mac$(RESET)               ZIP .app macOS  $(YELLOW)[Mac]$(RESET) / DMG si create-dmg disponible"
	@echo "  $(GREEN)dist-win$(RESET)               ZIP Windows"
	@echo "  $(GREEN)dist$(RESET)                   Todos los paquetes"
	@echo ""
	@echo "$(BOLD)  Base de datos$(RESET)"
	@echo "  $(GREEN)db-status$(RESET)              Estado de migraciones"
	@echo "  $(GREEN)db-users$(RESET)               Lista usuarios y roles"
	@echo "  $(GREEN)db-make-admin$(RESET)           Promover a admin: EMAIL=x@y.com"
	@echo "  $(GREEN)db-reset$(RESET)               Restaurar BD desde backup"
	@echo ""
	@echo "$(BOLD)  Utilidades$(RESET)"
	@echo "  $(GREEN)install-deps$(RESET)           Instala mingw-w64 + zip (Linux/apt)"
	@echo "  $(GREEN)install-deps-mac$(RESET)       Instala herramientas vía Homebrew"
	@echo "  $(GREEN)tidy$(RESET)                   go mod tidy + pnpm install"
	@echo "  $(GREEN)version$(RESET)                Muestra versión del proyecto"
	@echo "  $(GREEN)clean$(RESET)                  Limpia binarios y ZIPs"
	@echo "  $(GREEN)clean-all$(RESET)              + node_modules y dist frontend"
	@echo ""

# =============================================================================
#  CHECKS Y DEPENDENCIAS
# =============================================================================
.PHONY: check-deps
check-deps:
	@echo "$(BOLD)Verificando dependencias ($(HOST_OS))...$(RESET)"
	@command -v go    >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) go         $$(go version | awk '{print $$3}')" \
	  || (echo "  $(RED)✗$(RESET) go         → https://go.dev/dl" && exit 1)
	@command -v wails >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) wails      $$(wails version 2>/dev/null | head -1 | tr -d '[:space:]')" \
	  || (echo "  $(RED)✗$(RESET) wails      → go install github.com/wailsapp/wails/v2/cmd/wails@latest" && exit 1)
	@command -v pnpm  >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) pnpm       $$(pnpm --version)" \
	  || (echo "  $(RED)✗$(RESET) pnpm       → npm install -g pnpm" && exit 1)
	@command -v psql  >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) psql       $$(psql --version | head -1)" \
	  || echo "  $(YELLOW)!$(RESET) psql       — no encontrado (necesario para utilidades de BD)"
ifeq ($(UNAME), Darwin)
	@command -v create-dmg >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) create-dmg (paquete DMG)" \
	  || echo "  $(YELLOW)!$(RESET) create-dmg — brew install create-dmg (opcional, para dist-mac)"
	@command -v codesign >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) codesign   (firma de código)" \
	  || echo "  $(YELLOW)!$(RESET) codesign   — instalar Xcode Command Line Tools"
else
	@command -v $(WIN_CC) >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) mingw-w64  (cross-compile Windows)" \
	  || echo "  $(YELLOW)!$(RESET) mingw-w64  — make install-deps (solo para build-win)"
endif
	@command -v zip >/dev/null 2>&1 \
	  && echo "  $(GREEN)✓$(RESET) zip        (empaquetado dist)" \
	  || echo "  $(YELLOW)!$(RESET) zip        — necesario para make dist"
	@echo ""
	@echo "$(GREEN)Check completado.$(RESET)"

.PHONY: install-deps
install-deps:
	@echo "$(BOLD)Instalando dependencias del sistema (Linux/apt)...$(RESET)"
	sudo apt-get update -qq
	sudo apt-get install -y gcc-mingw-w64-x86-64 zip
	@echo "$(GREEN)✓ Listo.$(RESET)"
	@echo "  Instalar Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest"

.PHONY: install-deps-mac
install-deps-mac:
	@echo "$(BOLD)Instalando dependencias del sistema (macOS/Homebrew)...$(RESET)"
	@command -v brew >/dev/null 2>&1 || \
	  (echo "$(RED)Homebrew no encontrado.$(RESET) Instalar desde https://brew.sh" && exit 1)
	brew install create-dmg
	@echo "$(GREEN)✓ Listo.$(RESET)"
	@echo "  Instalar Xcode CLI si falta: xcode-select --install"
	@echo "  Instalar Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest"

# =============================================================================
#  DESARROLLO
# =============================================================================
.PHONY: dev
dev:
	@echo "$(BOLD)$(CYAN)→ Dev server ($(HOST_OS))...$(RESET)"
ifeq ($(GO_BUILD_TAGS),)
	wails dev
else
	wails dev -tags $(GO_BUILD_TAGS)
endif

# =============================================================================
#  BUILD — NATIVO (detecta SO actual)
# =============================================================================
.PHONY: build
build:
	@echo "$(BOLD)$(CYAN)→ Compilando para $(HOST_OS) ($(HOST_PLATFORM))...$(RESET)"
	@echo "  Versión: $(VERSION)  Fecha: $(BUILD_DATE)"
ifeq ($(UNAME), Darwin)
	$(MAKE) build-mac
else
	wails build \
		-platform linux/amd64 \
		-tags "$(GO_BUILD_TAGS)" \
		-ldflags "$(LDFLAGS)" \
		-clean \
		-o $(APP_NAME)
	@echo ""
	@echo "$(GREEN)✓ Binario:$(RESET) $(BUILD_DIR)/$(APP_NAME)"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)
endif

.PHONY: build-debug
build-debug:
	@echo "$(BOLD)$(YELLOW)→ Build debug con devtools ($(HOST_OS))...$(RESET)"
ifeq ($(UNAME), Darwin)
	wails build \
		-platform $(HOST_PLATFORM) \
		-ldflags "$(LDFLAGS)" \
		-debug -devtools \
		-o $(APP_NAME)-debug.app
else
	wails build \
		-platform linux/amd64 \
		-tags "$(GO_BUILD_TAGS)" \
		-ldflags "$(LDFLAGS)" \
		-debug -devtools \
		-o $(APP_NAME)-debug
endif

# =============================================================================
#  BUILD — macOS  [debe ejecutarse en una Mac]
# =============================================================================
.PHONY: _guard-mac
_guard-mac:
	@[ "$(UNAME)" = "Darwin" ] || \
	  (echo "$(RED)Error:$(RESET) Los targets build-mac* deben ejecutarse en macOS." && \
	   echo "       macOS no permite cross-compilación desde otros sistemas operativos." && exit 1)

.PHONY: build-mac
build-mac: _guard-mac
	@echo "$(BOLD)$(CYAN)→ Compilando macOS universal (.app)...$(RESET)"
	@echo "  Versión: $(VERSION)  Fecha: $(BUILD_DATE)"
	wails build \
		-platform darwin/universal \
		-ldflags "$(LDFLAGS)" \
		-clean
	@echo ""
	@echo "$(GREEN)✓ Bundle:$(RESET) $(BUILD_DIR)/$(APP_NAME).app"
	@du -sh $(BUILD_DIR)/$(APP_NAME).app

.PHONY: build-mac-arm64
build-mac-arm64: _guard-mac
	@echo "$(BOLD)$(CYAN)→ Compilando macOS arm64 (Apple Silicon)...$(RESET)"
	wails build \
		-platform darwin/arm64 \
		-ldflags "$(LDFLAGS)" \
		-clean \
		-o $(APP_NAME)-arm64.app
	@echo "$(GREEN)✓ Bundle:$(RESET) $(BUILD_DIR)/$(APP_NAME)-arm64.app"

.PHONY: build-mac-amd64
build-mac-amd64: _guard-mac
	@echo "$(BOLD)$(CYAN)→ Compilando macOS amd64 (Intel)...$(RESET)"
	wails build \
		-platform darwin/amd64 \
		-ldflags "$(LDFLAGS)" \
		-clean \
		-o $(APP_NAME)-amd64.app
	@echo "$(GREEN)✓ Bundle:$(RESET) $(BUILD_DIR)/$(APP_NAME)-amd64.app"

# =============================================================================
#  BUILD — Windows  [desde Linux/Mac requiere mingw-w64]
# =============================================================================
.PHONY: build-win
build-win:
	@command -v $(WIN_CC) >/dev/null 2>&1 || \
	  (echo "$(RED)Error:$(RESET) mingw-w64 no encontrado." && \
	   echo "  Linux: make install-deps" && \
	   echo "  Mac:   brew install mingw-w64" && exit 1)
	@echo "$(BOLD)$(CYAN)→ Compilando Windows/amd64...$(RESET)"
	@echo "  Versión: $(VERSION)  Fecha: $(BUILD_DATE)"
	CC=$(WIN_CC) wails build \
		-platform windows/amd64 \
		-ldflags "$(LDFLAGS) -H windowsgui" \
		-clean \
		-o $(APP_NAME).exe
	@echo ""
	@echo "$(GREEN)✓ Binario:$(RESET) $(BUILD_DIR)/$(APP_NAME).exe"
	@ls -lh $(BUILD_DIR)/$(APP_NAME).exe

# =============================================================================
#  BUILD — TODOS
# =============================================================================
.PHONY: build-all
build-all:
ifeq ($(UNAME), Darwin)
	$(MAKE) build-mac build-win
else
	$(MAKE) build build-win
endif

# =============================================================================
#  DISTRIBUCIÓN
# =============================================================================
_ENV_EXAMPLE := .env.example

.PHONY: _copy-common
_copy-common:
	@cp $(_ENV_EXAMPLE) "$(TARGET_DIR)/" 2>/dev/null || \
	  printf "DATABASE_URL=postgresql://user:pass@localhost:5432/farmacia_db?sslmode=disable\nJWT_SECRET_KEY=cambia_esto\n" \
	  > "$(TARGET_DIR)/.env.example"
	@cp README.md    "$(TARGET_DIR)/"
	@cp CHANGELOG.md "$(TARGET_DIR)/"

# ── Linux ─────────────────────────────────────────────────────────────────────
.PHONY: dist-linux
dist-linux: build
	@echo "$(BOLD)$(CYAN)→ Empaquetando Linux...$(RESET)"
	@mkdir -p $(DIST_DIR)/linux
	@cp $(BUILD_DIR)/$(APP_NAME) $(DIST_DIR)/linux/
	@$(MAKE) _copy-common TARGET_DIR=$(DIST_DIR)/linux
	@cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-linux-amd64.zip linux/
	@echo "$(GREEN)✓$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.zip"
	@ls -lh $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.zip

# ── macOS ─────────────────────────────────────────────────────────────────────
.PHONY: dist-mac
dist-mac: build-mac
	@echo "$(BOLD)$(CYAN)→ Empaquetando macOS...$(RESET)"
	@mkdir -p $(DIST_DIR)/mac
	@cp -r $(BUILD_DIR)/$(APP_NAME).app $(DIST_DIR)/mac/
	@$(MAKE) _copy-common TARGET_DIR=$(DIST_DIR)/mac
	@if command -v create-dmg >/dev/null 2>&1; then \
	  echo "  Creando DMG con create-dmg..."; \
	  create-dmg \
	    --volname "$(APP_NAME) $(VERSION)" \
	    --window-size 540 380 \
	    --icon-size 96 \
	    --icon "$(APP_NAME).app" 130 190 \
	    --app-drop-link 410 190 \
	    --no-internet-enable \
	    "$(DIST_DIR)/$(APP_NAME)-$(VERSION)-macos-universal.dmg" \
	    "$(DIST_DIR)/mac/" ; \
	  echo "$(GREEN)✓$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-macos-universal.dmg" ; \
	  ls -lh "$(DIST_DIR)/$(APP_NAME)-$(VERSION)-macos-universal.dmg" ; \
	else \
	  echo "  create-dmg no disponible — generando ZIP..."; \
	  cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-macos-universal.zip mac/ ; \
	  echo "$(GREEN)✓$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-macos-universal.zip" ; \
	  ls -lh "$(DIST_DIR)/$(APP_NAME)-$(VERSION)-macos-universal.zip" ; \
	fi

# ── Windows ───────────────────────────────────────────────────────────────────
.PHONY: dist-win
dist-win: build-win
	@echo "$(BOLD)$(CYAN)→ Empaquetando Windows...$(RESET)"
	@mkdir -p $(DIST_DIR)/windows
	@cp $(BUILD_DIR)/$(APP_NAME).exe $(DIST_DIR)/windows/
	@$(MAKE) _copy-common TARGET_DIR=$(DIST_DIR)/windows
	@cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-windows-amd64.zip windows/
	@echo "$(GREEN)✓$(RESET) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.zip"
	@ls -lh $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.zip

# ── Todos ─────────────────────────────────────────────────────────────────────
.PHONY: dist
dist:
ifeq ($(UNAME), Darwin)
	$(MAKE) dist-mac dist-win
else
	$(MAKE) dist-linux dist-win
endif
	@echo ""
	@echo "$(BOLD)$(GREEN)✓ Distribuciones en $(DIST_DIR)/$(RESET)"
	@ls -lh $(DIST_DIR)/*.zip $(DIST_DIR)/*.dmg 2>/dev/null || true

# =============================================================================
#  BASE DE DATOS
# =============================================================================
.PHONY: db-reset
db-reset:
	@echo "$(BOLD)$(YELLOW)→ Restaurando base de datos desde backup...$(RESET)"
	@echo "  $(YELLOW)ADVERTENCIA: borrará todos los datos actuales.$(RESET)"
	@read -p "  ¿Continuar? [s/N] " confirm && [ "$$confirm" = "s" ] || exit 0
	cd backend/python && bash reset_and_import.sh

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
	    "UPDATE vendedors SET role='admin' \
	     WHERE email='$(EMAIL)' AND deleted_at IS NULL \
	     RETURNING nombre, email, role;" \
	  2>/dev/null || echo "$(RED)Error o usuario no encontrado.$(RESET)"

# =============================================================================
#  UTILIDADES
# =============================================================================
.PHONY: version
version:
	@echo "$(APP_NAME) $(VERSION)  ($(HOST_OS)/$(ARCH) · $(BUILD_DATE))"

.PHONY: tidy
tidy:
	@echo "$(BOLD)→ go mod tidy$(RESET)"
	go mod tidy
	@echo "$(BOLD)→ pnpm install$(RESET)"
	cd frontend && pnpm install

.PHONY: clean
clean:
	@echo "$(BOLD)$(YELLOW)→ Limpiando artefactos...$(RESET)"
	@rm -f  $(BUILD_DIR)/$(APP_NAME)
	@rm -f  $(BUILD_DIR)/$(APP_NAME).exe
	@rm -f  $(BUILD_DIR)/$(APP_NAME)-debug
	@rm -rf $(BUILD_DIR)/$(APP_NAME).app
	@rm -rf $(BUILD_DIR)/$(APP_NAME)-arm64.app
	@rm -rf $(BUILD_DIR)/$(APP_NAME)-amd64.app
	@rm -rf $(BUILD_DIR)/$(APP_NAME)-debug.app
	@rm -rf $(DIST_DIR)/linux $(DIST_DIR)/windows $(DIST_DIR)/mac
	@rm -f  $(DIST_DIR)/$(APP_NAME)-*.zip
	@rm -f  $(DIST_DIR)/$(APP_NAME)-*.dmg
	@echo "$(GREEN)✓ Limpio.$(RESET)"

.PHONY: clean-all
clean-all: clean
	@echo "$(BOLD)$(YELLOW)→ Limpiando frontend...$(RESET)"
	@rm -rf frontend/node_modules frontend/dist
	@echo "$(GREEN)✓ Frontend limpio.$(RESET)"
