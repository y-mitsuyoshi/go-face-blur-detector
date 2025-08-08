# Go Face Blur Detector

OpenCVを使用して顔のブレを検知するAPIサービス

## 概要

このプロジェクトは、アップロードされた画像から顔を検出し、その鮮明度スコアを返すREST APIです。OpenCVのラプラシアンフィルタと分散計算を使用してブレを検知します。

## 機能

- 画像アップロード
- 顔検出
- ブレ検知（鮮明度スコア計算）
- JSON形式でのレスポンス

## 必要な環境

- Go 1.21+
- OpenCV 4.x
- Docker（オプション）

## インストールと実行

### 前提条件

- Docker
- Docker Compose（または `docker compose`）

### Docker環境の確認

```bash
# Docker環境をチェック
make check-docker
```

### Docker Compose使用（推奨）

```bash
# アプリケーションのビルド
make build

# アプリケーションの実行
make run

# バックグラウンドで起動
make up

# 開発モード（ホットリロード）
make dev
```

### ローカル開発（Dockerなし）

```bash
# 依存関係のインストール
make deps

# アプリケーションのビルドと実行
make run-local
```

## API エンドポイント

### POST /detect

画像をアップロードして顔のブレを検知します。

**リクエスト:**
- Content-Type: multipart/form-data
- フィールド: `image` (画像ファイル)

**レスポンス:**
```json
{
  "sharpness_score": 85.6
}
```

### GET /health

ヘルスチェック用エンドポイント

### APIのテスト

`curl`コマンドを使用して、APIエンドポイントをテストできます。プロジェクトのルートディレクトリにいることを確認してください。
`test.png`と`test_blurred.png`は`internal/facedetector/testdata/`にあります。

**鮮明な画像のテスト:**

```bash
curl -X POST -F "image=@internal/facedetector/testdata/test.png" http://localhost:8080/detect
```

**ぼやけた画像のテスト:**

```bash
curl -X POST -F "image=@internal/facedetector/testdata/test_blurred.png" http://localhost:8080/detect
```

**期待されるレスポンス:**

鮮明な画像は、ぼやけた画像よりも高い`sharpness_score`を返すはずです。

```json
// 鮮明な画像の例
{
  "sharpness_score": 234.56
}

// ぼやけた画像の例
{
  "sharpness_score": 78.90
}
```

## 使用可能なコマンド

```bash
make help           # 利用可能なコマンドを表示
make build          # Docker Composeでアプリケーションをビルド
make run            # Docker Composeでアプリケーションを実行
make dev            # 開発モード（ホットリロード）でアプリケーションを実行
make up             # バックグラウンドでサービスを起動
make down           # サービスを停止
make logs           # ログを表示
make restart        # サービスを再起動
make build-local    # ローカルでアプリケーションをビルド
make run-local      # ローカルでアプリケーションを実行
make test           # テストを実行
make clean          # ビルド成果物をクリーンアップ
make check-docker   # Docker環境の確認
```

## 開発

### 必要なツール

- golangci-lint（リント用）
- air（ホットリロード用、開発時）

### 開発用コンテナの使用

```bash
# 開発用コンテナでホットリロード
make dev

# または直接Docker Composeを使用
docker compose --profile dev up face-blur-detector-dev
```

## ライセンス

このプロジェクトはMITライセンスの下で公開されています。
