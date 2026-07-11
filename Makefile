# linkpulse-dashboard — Makefile

# SPA собирается keg-версией Node (>=20 нужен для Vite 7); системный node может быть старее.
NODE_BIN ?= /opt/homebrew/opt/node@23/bin

.DEFAULT_GOAL := help

.PHONY: help
help: ## Показать список целей
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Собрать бинарник в bin/dashboard
	CGO_ENABLED=0 go build -o bin/dashboard ./cmd/dashboard

.PHONY: run
run: ## Запустить WS-сервер (нужен linkpulse-analytics на :50051)
	go run ./cmd/dashboard

.PHONY: test
test: ## Юнит-тесты с гонками
	go test -race -count=1 ./...

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run

.PHONY: web-install
web-install: ## Установить зависимости SPA
	cd web && PATH="$(NODE_BIN):$$PATH" npm ci

.PHONY: web-dev
web-dev: ## Dev-сервер SPA (Vite, проксирует /ws и /api на бэкенды)
	cd web && PATH="$(NODE_BIN):$$PATH" npm run dev

.PHONY: web-build
web-build: ## Продакшен-сборка SPA в web/dist
	cd web && PATH="$(NODE_BIN):$$PATH" npm run build

.PHONY: docker
docker: ## Собрать Docker-образ
	docker build -t linkpulse-dashboard .
