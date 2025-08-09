# マルチステージビルド - ビルドステージ
FROM ubuntu:24.04 AS builder

# パッケージリストを更新し、必要なライブラリをインストール
RUN apt-get update && apt-get install -y \
    golang-1.21 \
    git \
    build-essential \
    pkg-config \
    libopencv-dev \
    libopencv-contrib-dev \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Goのパスを設定
ENV PATH="/usr/lib/go-1.21/bin:${PATH}"
ENV GOPATH="/go"
ENV GOROOT="/usr/lib/go-1.21"

# 作業ディレクトリを設定
WORKDIR /app

# Go modulesファイルをコピーして依存関係をダウンロード
COPY go.mod ./
RUN go mod download

# ソースコードをコピー
COPY . .

# 依存関係を整理
RUN go mod tidy

# アプリケーションをビルド（Arucoコンポーネントを無効にして）
ENV CGO_CPPFLAGS="-I/usr/include/opencv4"
ENV CGO_LDFLAGS="-lopencv_core -lopencv_imgproc -lopencv_imgcodecs -lopencv_objdetect"
ENV PKG_CONFIG_PATH="/usr/lib/x86_64-linux-gnu/pkgconfig"
RUN CGO_ENABLED=1 go build -tags "!aruco" -ldflags "-s -w" -o face-blur-detector ./cmd/api

# 実行ステージ
FROM ubuntu:24.04

# パッケージリストを更新し、必要なライブラリをインストール
RUN apt-get update && apt-get install -y \
    libopencv-dev \
    libopencv-contrib-dev \
    ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 作業ディレクトリを設定
WORKDIR /app

# ビルドステージからバイナリとカスケードファイルをコピー
COPY --from=builder /app/face-blur-detector /usr/local/bin/
COPY --from=builder /app/internal/facedetector/cascade ./cascade

# 実行権限を付与
RUN chmod +x /usr/local/bin/face-blur-detector

# ポート8080を公開
EXPOSE 8080

# 環境変数のデフォルト値を設定
ENV PORT=8080
ENV LOG_LEVEL=INFO

# アプリケーションを実行
CMD ["face-blur-detector"]
