# マルチステージビルド用のベースイメージ
FROM golang:1.21-alpine AS builder

# 必要なパッケージをインストール
RUN apk add --no-cache \
    gcc \
    g++ \
    musl-dev \
    pkgconfig \
    opencv-dev \
    ca-certificates

# 作業ディレクトリを設定
WORKDIR /app

# Go modulesファイルをコピー
COPY go.mod ./

# 依存関係をダウンロード
RUN go mod download && go mod tidy

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# 本番用イメージ
FROM alpine:latest

# OpenCVランタイムライブラリをインストール
RUN apk add --no-cache \
    opencv \
    ca-certificates

# 非rootユーザーを作成
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 作業ディレクトリを作成
WORKDIR /app

# ビルダーステージからバイナリをコピー
COPY --from=builder /app/main .

# アプリケーションディレクトリの所有者を変更
RUN chown -R appuser:appgroup /app

# 非rootユーザーに切り替え
USER appuser

# ポート8080を公開
EXPOSE 8080

# ヘルスチェック
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# アプリケーションを実行
CMD ["./main"]
