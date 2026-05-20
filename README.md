# Go Face Blur Detector

OpenCVを使用して顔のブレを検知するAPIサービス

## 概要

このプロジェクトは、アップロードされた画像から顔を検出し、その鮮明度スコアを返すREST APIです。OpenCVのラプラシアンフィルタと分散計算を使用してブレを検知します。

### 顔検出エンジン

商用レベルの顔検出を実現するため、以下の多段階検出パイプラインを実装しています：

1. **DNN（SSD ResNet-10）検出**（推奨・高精度）
   - OpenCVのDNNモジュールを使用した深層学習ベースの顔検出
   - 逆光、顔の向き変化、低コントラストに強い
   - 信頼度スコア付きの検出結果

2. **Haar Cascade検出**（フォールバック）
   - 従来型のカスケード分類器による検出
   - DNNモデルが利用できない場合の代替手段

3. **前処理パイプライン**
   - **適応的ガンマ補正**: 逆光や露出オーバーの画像を自動補正
   - **CLAHE（適応的ヒストグラム均等化）**: 局所的なコントラスト改善
   - **アンシャープマスキング**: ブレた画像のエッジを強調

4. **後処理**
   - **NMS（Non-Maximum Suppression）**: 重複検出の除去
   - **肌色フィルタ**: HSV色空間による偽陽性除去（多様な肌色対応）
   - **DNN/Cascade交差検証**: 複数手法の結果を照合

## 機能

- 画像アップロード
- 顔検出
- ブレ検知（鮮明度スコア計算）
- 顔領域の可視化（矩形描画・切り抜き）
- JSON形式でのレスポンス

## 必要な環境

- Go 1.21+
- OpenCV 4.x
- Docker（推奨）

## インストールと実行

### 前提条件

- Docker
- Docker Compose

### Docker環境の確認

```bash
# Docker環境をチェック
make check-docker
```

### DNNモデルのセットアップ（推奨）

顔検出の精度を最大化するため、DNNモデルファイルをダウンロードしてください：

```bash
# モデルファイルをダウンロード
make download-models

# または直接スクリプトを実行
bash scripts/download_models.sh
```

> **Note**: DNNモデルなしでも動作しますが、Haar Cascadeのみでの検出となり、精度が低下します。
> Docker環境でビルドする場合は、ビルド時に自動的にダウンロードされます。

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

## API エンドポイント

### GET /

APIのルートエンドポイント。サービス名とバージョンを返します。

**レスポンス:**
```json
{
  "message": "Face Blur Detector API",
  "version": "v1.0.0"
}
```

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

### POST /detect/face

画像をアップロードして顔領域の鮮明度スコアを返します。

**リクエスト:**
- Content-Type: multipart/form-data
- フィールド: `image` (画像ファイル)

**レスポンス:**
```json
{
  "sharpness_score": 123.45
}
```

### POST /detect/face/visualize

アップロードされた画像から顔を検出し、加工して返します。`output`クエリパラメータで、`box`（顔の周りに四角を描画）または`crop`（顔の部分を切り出す）を指定できます。デフォルトは`box`です。

**リクエスト:**
- Content-Type: multipart/form-data
- フィールド: `image` (画像ファイル)
- クエリパラメータ (オプション): `output` (`box` or `crop`)

**レスポンス:**
- Content-Type: image/png
- ボディ: 加工された画像データ

### GET /health

ヘルスチェック用エンドポイント

### APIのテスト

`curl`コマンドを使用して、APIエンドポイントをテストできます。プロジェクトのルートディレクトリにいることを確認してください。

**鮮明な画像のテスト:**

```bash
curl -X POST -F "image=@internal/facedetector/testdata/test.png" http://localhost:8080/detect
```

**ぼやけた画像のテスト:**

```bash
curl -X POST -F "image=@internal/facedetector/testdata/test_blurred.png" http://localhost:8080/detect
```

**顔画像の鮮明度テスト:**

```bash
curl -X POST -F "image=@internal/facedetector/testdata/face.jpg" http://localhost:8080/detect/face
```

**顔を四角で囲む (デフォルト):**
```bash
curl -X POST -F "image=@internal/facedetector/testdata/face.jpg" "http://localhost:8080/detect/face/visualize?output=box" -o visualized_face_box.png
```

**顔を切り出す:**
```bash
curl -X POST -F "image=@internal/facedetector/testdata/face.jpg" "http://localhost:8080/detect/face/visualize?output=crop" -o visualized_face_crop.png
```

## 使用可能なコマンド

```bash
make help            # 利用可能なコマンドを表示
make build           # Docker Composeでアプリケーションをビルド
make run             # Docker Composeでアプリケーションを実行
make dev             # 開発モード（ホットリロード）でアプリケーションを実行
make up              # バックグラウンドでサービスを起動
make down            # サービスを停止
make logs            # ログを表示
make restart         # サービスを再起動
make test            # テストを実行
make test-coverage   # テストカバレッジを取得
make download-models # DNNモデルファイルをダウンロード
make clean           # ビルド成果物をクリーンアップ
make check-docker    # Docker環境の確認
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

### テストの実行

```bash
# Docker Compose経由でテスト実行
make test

# テストカバレッジ取得
make test-coverage
```

## ライセンス

このプロジェクトはMITライセンスの下で公開されています。
