# プロジェクト名とバージョン
PROJECT_NAME := go-face-blur-detector
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
DOCKER_IMAGE := $(PROJECT_NAME):$(VERSION)
DOCKER_IMAGE_LATEST := $(PROJECT_NAME):latest

# Goの設定
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# バイナリ名
BINARY_NAME := face-blur-detector
BINARY_PATH := ./bin/$(BINARY_NAME)

# ソースディレクトリ
SRC_DIR := ./cmd/api

# Docker compose ファイル
DOCKER_COMPOSE_FILE := docker-compose.yml

.PHONY: all build run build-local run-local clean test help deps tidy fmt lint dev up down logs restart check-docker

# デフォルトターゲット
all: build

# ヘルプ表示
help:
	@echo "利用可能なコマンド:"
	@echo "  build          - Docker Composeでアプリケーションをビルド"
	@echo "  run            - Docker Composeでアプリケーションを実行"
	@echo "  dev            - 開発モードでアプリケーションを実行"
	@echo "  up             - Docker Composeでサービスを起動（バックグラウンド）"
	@echo "  down           - Docker Composeでサービスを停止"
	@echo "  logs           - Docker Composeのログを表示"
	@echo "  restart        - Docker Composeでサービスを再起動"
	@echo "  build-local    - ローカルでアプリケーションをビルド"
	@echo "  run-local      - ローカルでアプリケーションを実行"
	@echo "  clean          - ビルド成果物をクリーンアップ"
	@echo "  test           - テストを実行"
	@echo "  deps           - 依存関係を更新"
	@echo "  tidy           - go.modを整理"
	@echo "  fmt            - コードをフォーマット"
	@echo "  lint           - コードをリント"
	@echo "  check-docker   - Docker環境の確認"

# 依存関係の更新
deps:
	@echo "依存関係を更新中..."
	$(GOGET) -u ./...

# go.modを整理
tidy:
	@echo "go.modを整理中..."
	$(GOMOD) tidy

# コードフォーマット
fmt:
	@echo "コードをフォーマット中..."
	$(GOCMD) fmt ./...

# コードリント（golangci-lintが必要）
lint:
	@echo "コードをリント中..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lintがインストールされていません。'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'でインストールしてください。"; \
	fi

# テスト実行
test:
	@echo "テストを実行中..."
	$(GOTEST) -v ./...

# テストカバレッジ
test-coverage:
	@echo "テストカバレッジを取得中..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "カバレッジレポートがcoverage.htmlに生成されました"

# Docker設定
DOCKER_COMPOSE_CMD := docker compose

# Dockerの利用可能性をチェック
check-docker:
	@docker --version >/dev/null 2>&1 || (echo "Dockerがインストールされていません。Dockerをインストールしてください。" && exit 1)
	@docker info >/dev/null 2>&1 || (echo "Dockerが実行されていません。Dockerを起動してください。" && exit 1)
	@echo "Docker環境: OK"

# Docker Composeでビルド
build: check-docker
	@echo "Docker Composeでアプリケーションをビルド中..."
	$(DOCKER_COMPOSE_CMD) build

# Docker Composeで実行
run: check-docker
	@echo "Docker Composeでアプリケーションを実行中..."
	$(DOCKER_COMPOSE_CMD) up

# 開発モードで実行
dev: check-docker
	@echo "開発モードでアプリケーションを実行中..."
	$(DOCKER_COMPOSE_CMD) --profile dev up face-blur-detector-dev

# Docker Composeでサービスを起動（バックグラウンド）
up: check-docker
	@echo "Docker Composeでサービスを起動中..."
	$(DOCKER_COMPOSE_CMD) up -d

# Docker Composeでサービスを停止
down:
	@echo "Docker Composeでサービスを停止中..."
	$(DOCKER_COMPOSE_CMD) down

# Docker Composeのログを表示
logs:
	@echo "Docker Composeのログを表示中..."
	$(DOCKER_COMPOSE_CMD) logs -f

# Docker Composeでサービスを再起動
restart:
	@echo "Docker Composeでサービスを再起動中..."
	$(DOCKER_COMPOSE_CMD) restart

# ローカルビルド（Dockerなし）
build-local:
	@echo "ローカルでアプリケーションをビルド中..."
	@mkdir -p bin
	CGO_ENABLED=1 $(GOBUILD) -o $(BINARY_PATH) $(SRC_DIR)

# ローカル実行（Dockerなし）
run-local: build-local
	@echo "ローカルでアプリケーションを実行中..."
	$(BINARY_PATH)

# クリーンアップ
clean:
	@echo "ビルド成果物をクリーンアップ中..."
	$(GOCLEAN)
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	$(DOCKER_COMPOSE_CMD) down --rmi all --volumes --remove-orphans 2>/dev/null || true

# 開発環境の初期化
dev-init:
	@echo "開発環境を初期化中..."
	$(GOMOD) init $(PROJECT_NAME) 2>/dev/null || true
	$(MAKE) deps
	$(MAKE) tidy
	$(DOCKER_COMPOSE_CMD) build

# バージョン情報表示
version:
	@echo "プロジェクト: $(PROJECT_NAME)"
	@echo "バージョン: $(VERSION)"
	@echo "Dockerイメージ: $(DOCKER_IMAGE)"
