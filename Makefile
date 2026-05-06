.DEFAULT_GOAL := help

# 通过 -ldflags 注入版本信息。
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

BIN_NAME ?= astrolabe
BIN_OUT  ?= ./$(BIN_NAME)

GO_PKGS := ./...
WEB_DIR := web

.PHONY: help
help: ## 显示可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: vet
vet: ## go vet
	go vet $(GO_PKGS)

.PHONY: test
test: ## go test
	go test $(GO_PKGS)

.PHONY: smoke
smoke: ## 端到端 smoke（需要 ./astrolabe 已运行在 8080）
	go run ./tests/smoke

.PHONY: web-install
web-install: ## 安装前端依赖
	cd $(WEB_DIR) && pnpm install

.PHONY: web-typecheck
web-typecheck: ## 前端类型检查
	cd $(WEB_DIR) && pnpm type-check

.PHONY: web-lint
web-lint: ## 前端 ESLint
	cd $(WEB_DIR) && pnpm lint

.PHONY: web
web: ## 仅构建前端（产物输出到 internal/embed/dist）
	cd $(WEB_DIR) && pnpm build

.PHONY: build
build: web ## 构建单二进制（含嵌入前端）
	go build -ldflags "$(LDFLAGS)" -o $(BIN_OUT) ./cmd/astrolabe

.PHONY: build-go
build-go: ## 仅构建后端二进制（不重建前端）
	go build -ldflags "$(LDFLAGS)" -o $(BIN_OUT) ./cmd/astrolabe

.PHONY: dev-back
dev-back: ## 仅启动后端（监听 8080）
	go run ./cmd/astrolabe

.PHONY: dev-web
dev-web: ## 仅启动前端 dev server（端口 5173，代理 /ws -> 8080）
	cd $(WEB_DIR) && pnpm dev

.PHONY: lint
lint: vet web-lint ## 全部静态检查

.PHONY: clean
clean: ## 清理构建产物
	rm -f $(BIN_OUT)
	rm -rf $(WEB_DIR)/dist internal/embed/dist/assets
	@echo "保留 internal/embed/dist/index.html 占位文件"
	@git checkout -- internal/embed/dist/index.html 2>/dev/null || true
