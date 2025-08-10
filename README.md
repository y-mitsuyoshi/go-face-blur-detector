# Go Image Sharpness Detector

OpenCVを使用して画像の鮮明度（ブレ）を判定するAPIサービス

## 概要

このプロジェクトは、アップロードされた画像の鮮明度を評価するREST APIです。画像全体の鮮明度を計算するだけでなく、画像内に顔が検出された場合は、その顔領域に特化して鮮明度を計算することも可能です。ブレの検知には、OpenCVのラプラシアン法（分散）を利用しています。

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
- Docker Compose

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

### GET /
APIのルートエンドポイント。サービス名とバージョンを返します。
- **レスポンス (200 OK):**
  ```json
  {
    "message": "Face Blur Detector API",
    "version": "v1.0.0"
  }
  ```

### GET /health
サービスの稼働状況を確認するためのヘルスチェックエンドポイント。
- **レスポンス (200 OK):**
  ```json
  {
    "status": "ok",
    "service": "go-face-blur-detector"
  }
  ```

### POST /detect
画像全体を対象に、鮮明度スコアを計算します。
- **リクエスト:**
  - `Content-Type: multipart/form-data`
  - `image`: 画像ファイル
- **レスポンス (200 OK):**
  ```json
  {
    "sharpness_score": 150.75
  }
  ```
- **エラーレスポンス (例: 400 Bad Request):**
  ```json
  {
    "error": "画像ファイルの取得に失敗しました:..."
  }
  ```

### POST /detect/face
画像から顔を検出し、その**顔領域だけ**を対象に鮮明度スコアを計算します。複数の顔が検出された場合は、最も鮮明な顔のスコアを返します。
- **リクエスト:**
  - `Content-Type: multipart/form-data`
  - `image`: 画像ファイル
- **レスポンス (200 OK):**
  ```json
  {
    "sharpness_score": 123.45
  }
  ```
- **エラーレスポンス (例: 500 Internal Server Error):**
  ```json
  {
    "error": "鮮明度の計算に失敗しました: 顔が検出されませんでした"
  }
  ```

### POST /detect/face/visualize
アップロードされた画像から最も大きな顔を検出し、加工した画像を返します。
- **リクエスト:**
  - `Content-Type: multipart/form-data`
  - `image`: 画像ファイル
  - `output` (クエリパラメータ, オプション):
    - `box` (デフォルト): 顔の周りに四角い枠を描画します。
    - `crop`: 顔の部分を切り抜きます。
- **レスポンス (200 OK):**
  - `Content-Type: image/png`
  - ボディ: 加工された画像データ
- **エラーレスポンス (例: 500 Internal Server Error):**
  ```json
  {
    "error": "顔検出または画像処理に失敗しました: 顔が検出されませんでした"
  }
  ```

## APIのテスト

`curl`コマンドを使用してAPIをテストできます。プロジェクトのルートディレクトリから以下のコマンドを実行してください。テスト用の画像は`internal/facedetector/testdata/`にあります。

### 画像全体の鮮明度をテスト
```bash
# 鮮明な画像
curl -X POST -F "image=@internal/facedetector/testdata/test.png" http://localhost:8080/detect
# ぼやけた画像
curl -X POST -F "image=@internal/facedetector/testdata/test_blurred.png" http://localhost:8080/detect
```
**期待される結果:** 鮮明な画像は、ぼやけた画像よりも高い`sharpness_score`を返します。

### 顔の鮮明度をテスト
```bash
# 鮮明な顔画像
curl -X POST -F "image=@internal/facedetector/testdata/face.jpg" http://localhost:8080/detect/face
# ぼやけた顔画像
curl -X POST -F "image=@internal/facedetector/testdata/face_blurred.jpg" http://localhost:8080/detect/face
```
**期待される結果:** 鮮明な顔画像は、ぼやけた顔画像よりも高い`sharpness_score`を返します。

### 顔検出の可視化をテスト
```bash
# 顔を四角で囲む (デフォルト)
curl -X POST -F "image=@internal/facedetector/testdata/face.jpg" "http://localhost:8080/detect/face/visualize?output=box" -o visualized_face_box.png

# 顔を切り抜く
curl -X POST -F "image=@internal/facedetector/testdata/face.jpg" "http://localhost:8080/detect/face/visualize?output=crop" -o visualized_face_crop.png
```
**期待される結果:** `visualized_face_box.png`と`visualized_face_crop.png`が生成されます。

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
