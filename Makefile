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

.PHONY: all build clean test run docker-build docker-run docker-stop docker-clean help deps tidy fmt lint

# デフォルトターゲット
all: clean deps tidy fmt test build

# ヘルプ表示
help:
	@echo "利用可能なコマンド:"
	@echo "  build          - アプリケーションをビルド"
	@echo "  clean          - ビルド成果物をクリーンアップ"
	@echo "  test           - テストを実行"
	@echo "  run            - アプリケーションを実行"
	@echo "  deps           - 依存関係を更新"
	@echo "  tidy           - go.modを整理"
	@echo "  fmt            - コードをフォーマット"
	@echo "  lint           - コードをリント"
	@echo "  docker-build   - Dockerイメージをビルド"
	@echo "  docker-run     - Dockerコンテナを実行"
	@echo "  docker-stop    - Dockerコンテナを停止"
	@echo "  docker-clean   - Dockerイメージとコンテナをクリーンアップ"
	@echo "  compose-up     - Docker Composeでサービスを起動"
	@echo "  compose-down   - Docker Composeでサービスを停止"

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

# ビルド
build:
	@echo "アプリケーションをビルド中..."
	@mkdir -p bin
	CGO_ENABLED=1 $(GOBUILD) -o $(BINARY_PATH) $(SRC_DIR)

# ローカル実行
run: build
	@echo "アプリケーションを実行中..."
	$(BINARY_PATH)

# クリーンアップ
clean:
	@echo "ビルド成果物をクリーンアップ中..."
	$(GOCLEAN)
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Dockerイメージをビルド
docker-build:
	@echo "Dockerイメージをビルド中..."
	docker build -t $(DOCKER_IMAGE) -t $(DOCKER_IMAGE_LATEST) .

# Dockerコンテナを実行
docker-run: docker-build
	@echo "Dockerコンテナを実行中..."
	docker run --rm -p 8080:8080 --name $(PROJECT_NAME) $(DOCKER_IMAGE_LATEST)

# Dockerコンテナを停止
docker-stop:
	@echo "Dockerコンテナを停止中..."
	@docker stop $(PROJECT_NAME) 2>/dev/null || true

# Dockerイメージとコンテナをクリーンアップ
docker-clean: docker-stop
	@echo "Dockerイメージとコンテナをクリーンアップ中..."
	@docker rm $(PROJECT_NAME) 2>/dev/null || true
	@docker rmi $(DOCKER_IMAGE) $(DOCKER_IMAGE_LATEST) 2>/dev/null || true
	@docker system prune -f

# Docker Composeでサービスを起動
compose-up:
	@echo "Docker Composeでサービスを起動中..."
	@if [ -f $(DOCKER_COMPOSE_FILE) ]; then \
		docker-compose up -d; \
	else \
		echo "$(DOCKER_COMPOSE_FILE)が見つかりません"; \
	fi

# Docker Composeでサービスを停止
compose-down:
	@echo "Docker Composeでサービスを停止中..."
	@if [ -f $(DOCKER_COMPOSE_FILE) ]; then \
		docker-compose down; \
	else \
		echo "$(DOCKER_COMPOSE_FILE)が見つかりません"; \
	fi

# 開発環境の初期化
dev-init:
	@echo "開発環境を初期化中..."
	$(GOMOD) init $(PROJECT_NAME) 2>/dev/null || true
	$(MAKE) deps
	$(MAKE) tidy

# プロダクションビルド
build-prod:
	@echo "プロダクション用ビルドを実行中..."
	@mkdir -p bin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="-w -s" -o $(BINARY_PATH) $(SRC_DIR)

# バージョン情報表示
version:
	@echo "プロジェクト: $(PROJECT_NAME)"
	@echo "バージョン: $(VERSION)"
	@echo "Dockerイメージ: $(DOCKER_IMAGE)"
