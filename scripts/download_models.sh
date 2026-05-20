#!/bin/bash
# DNN顔検出モデルのダウンロードスクリプト
#
# OpenCV の SSD ResNet-10 モデルをダウンロードします。
# このモデルは Haar Cascade よりも大幅に高精度な顔検出が可能です。
#
# 使用方法:
#   ./scripts/download_models.sh
#   make download-models

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
MODELS_DIR="${PROJECT_ROOT}/internal/facedetector/models"

# モデルファイルのURL（OpenCV公式リポジトリ）
CAFFEMODEL_URL="https://raw.githubusercontent.com/opencv/opencv_3rdparty/dnn_samples_face_detector_20170830/res10_300x300_ssd_iter_140000.caffemodel"
PROTOTXT_URL="https://raw.githubusercontent.com/opencv/opencv/4.x/samples/dnn/face_detector/deploy.prototxt"

CAFFEMODEL_FILE="${MODELS_DIR}/res10_300x300_ssd_iter_140000.caffemodel"
PROTOTXT_FILE="${MODELS_DIR}/deploy.prototxt"

# チェックサム（整合性検証用）
CAFFEMODEL_SHA256="2a56a11a57a4a295956b0660b4a3d76bbdca2206c4961cea8efe7d95c7cb2f2d"

echo "=== DNN顔検出モデルのダウンロード ==="
echo ""

# ダウンロードディレクトリの作成
mkdir -p "${MODELS_DIR}"

# Caffemodel のダウンロード
if [ -f "${CAFFEMODEL_FILE}" ]; then
    echo "✓ caffemodel は既にダウンロード済みです: ${CAFFEMODEL_FILE}"
else
    echo "→ caffemodel をダウンロード中..."
    if command -v curl &> /dev/null; then
        curl -L --progress-bar -o "${CAFFEMODEL_FILE}" "${CAFFEMODEL_URL}"
    elif command -v wget &> /dev/null; then
        wget --show-progress -O "${CAFFEMODEL_FILE}" "${CAFFEMODEL_URL}"
    else
        echo "エラー: curl または wget が必要です"
        exit 1
    fi
    echo "✓ caffemodel のダウンロードが完了しました"
fi

# Prototxt のダウンロード
if [ -f "${PROTOTXT_FILE}" ]; then
    echo "✓ prototxt は既にダウンロード済みです: ${PROTOTXT_FILE}"
else
    echo "→ prototxt をダウンロード中..."
    if command -v curl &> /dev/null; then
        curl -L --progress-bar -o "${PROTOTXT_FILE}" "${PROTOTXT_URL}"
    elif command -v wget &> /dev/null; then
        wget --show-progress -O "${PROTOTXT_FILE}" "${PROTOTXT_URL}"
    else
        echo "エラー: curl または wget が必要です"
        exit 1
    fi
    echo "✓ prototxt のダウンロードが完了しました"
fi

# チェックサム検証
echo ""
echo "→ チェックサムの検証中..."
if command -v sha256sum &> /dev/null; then
    ACTUAL_SHA256=$(sha256sum "${CAFFEMODEL_FILE}" | awk '{print $1}')
    if [ "${ACTUAL_SHA256}" = "${CAFFEMODEL_SHA256}" ]; then
        echo "✓ チェックサムが一致しました"
    else
        echo "⚠ チェックサムが一致しません（ファイルが破損している可能性があります）"
        echo "  期待値: ${CAFFEMODEL_SHA256}"
        echo "  実際値: ${ACTUAL_SHA256}"
        echo "  モデルは使用可能ですが、再ダウンロードを推奨します"
    fi
elif command -v shasum &> /dev/null; then
    ACTUAL_SHA256=$(shasum -a 256 "${CAFFEMODEL_FILE}" | awk '{print $1}')
    if [ "${ACTUAL_SHA256}" = "${CAFFEMODEL_SHA256}" ]; then
        echo "✓ チェックサムが一致しました"
    else
        echo "⚠ チェックサムが一致しません"
    fi
else
    echo "⚠ sha256sum/shasum が見つかりません。チェックサム検証をスキップします"
fi

# ファイルサイズの確認
echo ""
echo "=== ダウンロード結果 ==="
echo ""
ls -lh "${MODELS_DIR}/"
echo ""
echo "✓ モデルのセットアップが完了しました"
echo "  アプリケーション起動時に自動的にDNNモデルが使用されます"
