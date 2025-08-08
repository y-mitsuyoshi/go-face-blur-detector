# プロジェクト名とバージョン
PROJECT_NAME := go-face-blur-detector
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
DOCKER_IMAGE := $(PROJECT_NAME):$(VERSION)
DOCKER_IMAGE_LATEST := $(PROJECT_NAME):latest

# Docker compose ファイル
DOCKER_COMPOSE_FILE := docker-compose.yml

.PHONY: all build run clean test help deps tidy fmt lint dev up down stop logs restart check-docker test-coverage

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
	@echo "  stop           - Docker Composeでサービスを停止"
	@echo "  logs           - Docker Composeのログを表示"
	@echo "  restart        - Docker Composeでサービスを再起動"
	@echo "  test           - Docker Composeでテストを実行"
	@echo "  test-coverage  - Docker Composeでテストカバレッジを取得"
	@echo "  deps           - Docker Composeで依存関係を更新"
	@echo "  tidy           - Docker Composeでgo.modを整理"
	@echo "  fmt            - Docker Composeでコードをフォーマット"
	@echo "  lint           - Docker Composeでコードをリント"
	@echo "  clean          - ビルド成果物をクリーンアップ"
	@echo "  check-docker   - Docker環境の確認"

# 依存関係の更新
deps: check-docker
	@echo "Docker Composeで依存関係を更新中..."
	$(DOCKER_COMPOSE_CMD) --profile dev run --rm face-blur-detector-dev go get -u ./...

# go.modを整理
tidy: check-docker
	@echo "Docker Composeでgo.modを整理中..."
	$(DOCKER_COMPOSE_CMD) --profile dev run --rm face-blur-detector-dev go mod tidy

# コードフォーマット
fmt: check-docker
	@echo "Docker Composeでコードをフォーマット中..."
	$(DOCKER_COMPOSE_CMD) --profile dev run --rm face-blur-detector-dev go fmt ./...

# コードリント（golangci-lintが必要）
lint: check-docker
	@echo "Docker Composeでコードをリント中..."
	$(DOCKER_COMPOSE_CMD) --profile dev run --rm face-blur-detector-dev sh -c "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && golangci-lint run"

# テスト実行
test: check-docker
	@echo "Docker Composeでテストを実行中..."
	$(DOCKER_COMPOSE_CMD) --profile test run --rm test

# テストカバレッジ
test-coverage: check-docker
	@echo "Docker Composeでテストカバレッジを取得中..."
	$(DOCKER_COMPOSE_CMD) --profile test run --rm test sh -c "go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html"
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

# Docker Composeでサービスを停止（stopコマンド）
stop:
	@echo "Docker Composeでサービスを停止中..."
	$(DOCKER_COMPOSE_CMD) stop

# Docker Composeのログを表示
logs:
	@echo "Docker Composeのログを表示中..."
	$(DOCKER_COMPOSE_CMD) logs -f

# Docker Composeでサービスを再起動
restart:
	@echo "Docker Composeでサービスを再起動中..."
	$(DOCKER_COMPOSE_CMD) restart

# クリーンアップ
clean: check-docker
	@echo "ビルド成果物をクリーンアップ中..."
	@rm -f coverage.out coverage.html
	$(DOCKER_COMPOSE_CMD) down --rmi all --volumes --remove-orphans 2>/dev/null || true

# 開発環境の初期化
dev-init: check-docker
	@echo "開発環境を初期化中..."
	$(DOCKER_COMPOSE_CMD) --profile dev run --rm face-blur-detector-dev sh -c "go mod init $(PROJECT_NAME) 2>/dev/null || true"
	$(MAKE) deps
	$(MAKE) tidy
	$(DOCKER_COMPOSE_CMD) build

# バージョン情報表示
version:
	@echo "プロジェクト: $(PROJECT_NAME)"
	@echo "バージョン: $(VERSION)"
	@echo "Dockerイメージ: $(DOCKER_IMAGE)"
