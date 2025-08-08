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

### ローカル開発

```bash
# 依存関係のインストール
make deps

# アプリケーションのビルド
make build

# アプリケーションの実行
make run
```

### Docker使用

```bash
# Dockerイメージのビルドと実行
make docker-run

# または Docker Composeを使用
make compose-up
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
  "faces": [
    {
      "x": 100,
      "y": 150,
      "width": 120,
      "height": 120,
      "sharpness_score": 85.6,
      "is_blurred": false
    }
  ],
  "total_faces": 1
}
```

### GET /health

ヘルスチェック用エンドポイント

## 使用可能なコマンド

```bash
make help          # 利用可能なコマンドを表示
make build         # アプリケーションをビルド
make test          # テストを実行
make docker-build  # Dockerイメージをビルド
make docker-run    # Dockerコンテナを実行
make clean         # ビルド成果物をクリーンアップ
```

## 開発

### 必要なツール

- golangci-lint（リント用）
- air（ホットリロード用、開発時）

### 開発用コンテナの使用

```bash
# 開発用コンテナでホットリロード
docker-compose --profile dev up face-blur-detector-dev
```

## ライセンス

このプロジェクトはMITライセンスの下で公開されています。
