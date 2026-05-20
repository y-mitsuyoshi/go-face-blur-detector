# Go Face Blur Detector

OpenCVを使用して顔のブレを検知するAPIサービス

## 概要

このプロジェクトは、アップロードされた画像から顔を検出し、その鮮明度スコアを返すREST APIです。

単なるラプラシアン分散の計算にとどまらず、**商用グレードの「客観的かつカメラ・環境に依存しない」正規化鮮明度スコアリング**を実装しています。

### 正規化鮮明度スコアリング（商用グレード）

カメラの画素数、レンズの品質、逆光、手ブレ、被写体の模様の多さに左右されず、人間の視覚的な「ピントの合い具合」に合致した 0〜100点 のスコアを出力します。

1. **解像度の正規化**: 画像サイズによるエッジ強度の変化を防ぐため、検出領域を一定サイズ（128x128px）にリサイズして解析します。
2. **コントラストの正規化 (CLAHE)**: 逆光や暗所でのスコア低下を防ぐため、コントラストを局所的に最適化します。
3. **ノイズ除去 (Bilateral Filter)**: 高感度ノイズをエッジとして誤検知するのを防ぐため、エッジ情報を保ったままノイズを除去します。
4. **エッジ減衰率 (Edge Decay Ratio) 解析**: 独自のブラー適用（カーネルサイズ5のガウシアンフィルタ）前後のエッジエネルギー（Tenengrad法およびLaplacian法）の減少比率を測定します。元からぼやけている画像はブラーによる減衰が小さく、ピンボケしていない画像は大きく減衰します。この相対的な減衰比率を使用することで、被写体やカメラの違いに依存しない純粋なピント精度を抽出します。
5. **非線形スコアリング (Sigmoid関数)**: 減衰比率をシグモイド関数により 0〜100点 にマッピングし、人間の直感的な「ブレ」の許容レベルに最適化します。
   - **80点以上**: 非常に鮮明（ピントが合っている）
   - **50〜80点**: 許容範囲（わずかなボケや微細なブレ）
   - **50点未満**: ブレ・ボケ・ピンボケあり（不合格）

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

画像をアップロードして全体の鮮明度スコアを検知します。

**リクエスト:**
- Content-Type: multipart/form-data
- フィールド: `image` (画像ファイル)

**レスポンス (JSON):**
```json
{
  "normalized_score": 99.7,
  "raw_laplacian_variance": 542.12,
  "raw_tenengrad_variance": 1204.56,
  "edge_decay_ratio": 0.826,
  "mean_brightness": 128.5,
  "estimated_blur_level": 45.2,
  "original_width": 640,
  "original_height": 480,
  "analyzed_width": 128,
  "analyzed_height": 128
}
```

### POST /detect/face

画像をアップロードして、検出された顔領域（DNNまたはHaar Cascadeにより抽出された顔の中心60%領域）の鮮明度スコアを返します。

**リクエスト:**
- Content-Type: multipart/form-data
- フィールド: `image` (画像ファイル)

**レスポンス (JSON):**
```json
{
  "normalized_score": 99.9,
  "raw_laplacian_variance": 752.43,
  "raw_tenengrad_variance": 1824.11,
  "edge_decay_ratio": 0.862,
  "mean_brightness": 135.2,
  "estimated_blur_level": 58.1,
  "original_width": 640,
  "original_height": 480,
  "analyzed_width": 128,
  "analyzed_height": 128
}
```

### レスポンスフィールドの説明

| フィールド | 型 | 説明 |
| :--- | :--- | :--- |
| `normalized_score` | float64 | 0〜100点に正規化された鮮明度スコア。カメラや解像度に依存しない客観的指標。**80点以上が実用上「鮮明」**と判定されます。 |
| `raw_laplacian_variance` | float64 | ラプラシアン分散の生値。エッジのシャープさの目安になりますが、画像自体のコントラストや模様の複雑さに影響されます。 |
| `raw_tenengrad_variance` | float64 | Tenengrad法（Sobel勾配の平方和）によるエッジ分散値。ノイズに強い特徴があります。 |
| `edge_decay_ratio` | float64 | エッジ減衰率（0.0〜1.0）。意図的に加えたガウシアンブラー前後でのエッジ強度の減少割合。被写体に依存しない相対指標として最重要です。 |
| `mean_brightness` | float64 | 解析領域の平均輝度（0〜255）。逆光判定や暗所判定などの診断に利用可能です。 |
| `estimated_blur_level` | float64 | ラプラシアン分散に基づき推定されたブレの強さ。数値が低いほどブレが大きいことを示します。 |
| `original_width` | int | 入力画像の元の幅（ピクセル）。 |
| `original_height` | int | 入力画像の元の高さ（ピクセル）。 |
| `analyzed_width` | int | 鮮明度解析時に解像度の正規化としてリサイズされた幅（デフォルト: 128px）。 |
| `analyzed_height` | int | 鮮明度解析時に解像度の正規化としてリサイズされた高さ（デフォルト: 128px）。 |

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
