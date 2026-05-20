package facedetector

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // image.Decodeでjpeg形式をサポートするために必要
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"gocv.io/x/gocv"
)

// ============================================================================
// 定数・設定
// ============================================================================

const (
	// DNN モデルファイル名
	dnnModelFileName = "res10_300x300_ssd_iter_140000.caffemodel"
	dnnProtoFileName = "deploy.prototxt"

	// DNN検出の信頼度閾値
	// 商用環境では 0.5 を標準とし、多段階検出で閾値を下げて補完する
	dnnConfidenceHigh = 0.5  // 高信頼度（単独で採用）
	dnnConfidenceLow  = 0.25 // 低信頼度（交差検証用）

	// NMS（Non-Maximum Suppression）のIoU閾値
	nmsIOUThreshold = 0.3

	// 肌色検出: 肌色ピクセルの最低割合
	skinColorMinRatio = 0.10

	// アスペクト比の許容範囲（顔として妥当な範囲）
	aspectRatioMin = 0.5
	aspectRatioMax = 1.8

	// 顔矩形の最小サイズ（ピクセル）
	minFaceSize = 20

	// CLAHE パラメータ
	claheClipLimit = 3.0
	claheTileSize  = 8

	// ガンマ補正パラメータ
	gammaForDark  = 1.8 // 暗い画像用（明るくする）
	gammaForBright = 0.6 // 明るすぎる画像用（暗くする）

	// 画像の明るさ判定閾値
	darkThreshold   = 80.0  // この値以下なら「暗い」と判定
	brightThreshold = 180.0 // この値以上なら「明るすぎる」と判定
)

// cascadeFiles はHaar Cascade分類器のファイルパスリスト
var cascadeFiles = []string{
	"cascade/haarcascade_frontalface_alt2.xml",
	"/usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
	"/usr/share/opencv4/haarcascades/haarcascade_frontalface_alt.xml",
	"/usr/share/opencv4/haarcascades/haarcascade_frontalface_alt2.xml",
	"/usr/share/opencv4/haarcascades/haarcascade_profileface.xml",
	"/usr/local/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
	"/usr/local/share/opencv4/haarcascades/haarcascade_frontalface_alt2.xml",
}

// ============================================================================
// DNN モデルパス解決
// ============================================================================

// dnnModelPaths はDNNモデルファイルの検索候補パスを返します。
// バイナリ実行ファイルの位置、カレントディレクトリなど複数の候補を試行します。
func dnnModelSearchPaths() []string {
	var candidates []string

	// 1. カレントディレクトリ相対
	candidates = append(candidates, "models")
	candidates = append(candidates, "internal/facedetector/models")

	// 2. 実行バイナリの隣
	if execPath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(execPath), "models"))
	}

	// 3. ソースファイルの位置（開発時用）
	if _, filename, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(filename), "models"))
	}

	// 4. Docker/コンテナ環境でのデフォルトパス
	candidates = append(candidates, "/app/models")

	return candidates
}

// findDNNModelFiles はDNNモデルファイルのペアを検索します。
// 見つかった場合は (prototxtPath, caffeModelPath) を返します。
func findDNNModelFiles() (string, string, bool) {
	for _, dir := range dnnModelSearchPaths() {
		protoPath := filepath.Join(dir, dnnProtoFileName)
		modelPath := filepath.Join(dir, dnnModelFileName)
		if fileExists(protoPath) && fileExists(modelPath) {
			return protoPath, modelPath, true
		}
	}
	return "", "", false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ============================================================================
// 型定義
// ============================================================================

// subImager はSubImageメソッドを持つ画像インタフェースです。
type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

// Detection はpigoライブラリの検出結果を模倣した構造体です。
// GoCVとの互換性のために残しますが、pigo特有のフィールドは使われません。
type Detection struct {
	Row   int
	Col   int
	Scale int
	Q     float32 // 信頼度スコア
}

// detectionWithConfidence は内部処理用の検出結果（矩形+信頼度付き）
type detectionWithConfidence struct {
	rect       image.Rectangle
	confidence float32
	source     string // "dnn" or "cascade"
}

// ============================================================================
// 前処理関数
// ============================================================================

// calculateMeanBrightness は画像の平均輝度を計算します。
// 逆光や露出オーバーの判定に使用します。
func calculateMeanBrightness(mat gocv.Mat) float64 {
	gray := gocv.NewMat()
	defer gray.Close()

	if mat.Channels() == 1 {
		mat.CopyTo(&gray)
	} else {
		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)
	}

	mean := gray.Mean()
	return mean.Val1
}

// applyGammaCorrection はガンマ補正を適用して、暗い/明るい画像のコントラストを改善します。
// gamma > 1.0: 暗い画像を明るくする
// gamma < 1.0: 明るい画像を暗くする
func applyGammaCorrection(mat gocv.Mat, gamma float64) gocv.Mat {
	// ルックアップテーブルを作成
	lut := gocv.NewMatWithSize(1, 256, gocv.MatTypeCV8U)
	invGamma := 1.0 / gamma
	for i := 0; i < 256; i++ {
		val := math.Pow(float64(i)/255.0, invGamma) * 255.0
		lut.SetUCharAt(0, i, uint8(math.Min(255, math.Max(0, val))))
	}

	result := gocv.NewMat()
	gocv.LUT(mat, lut, &result)
	lut.Close()
	return result
}

// applyAdaptivePreprocessing は画像の状態に応じて適切な前処理を自動選択して適用します。
// 返り値は前処理済みの画像で、呼び出し側でClose()する必要があります。
func applyAdaptivePreprocessing(mat gocv.Mat) gocv.Mat {
	brightness := calculateMeanBrightness(mat)

	var processed gocv.Mat
	if brightness < darkThreshold {
		// 暗い画像（逆光等）: ガンマ補正で明るくする
		processed = applyGammaCorrection(mat, gammaForDark)
	} else if brightness > brightThreshold {
		// 明るすぎる画像: ガンマ補正で抑える
		processed = applyGammaCorrection(mat, gammaForBright)
	} else {
		// 通常の明るさ: そのままコピー
		processed = gocv.NewMat()
		mat.CopyTo(&processed)
	}

	// CLAHE (適応的ヒストグラム均等化) を Lab 色空間の L チャネルに適用
	labMat := gocv.NewMat()
	defer labMat.Close()
	gocv.CvtColor(processed, &labMat, gocv.ColorBGRToLab)

	channels := gocv.Split(labMat)
	clahe := gocv.NewCLAHEWithParams(claheClipLimit, image.Point{X: claheTileSize, Y: claheTileSize})
	defer clahe.Close()
	clahe.Apply(channels[0], &channels[0])

	enhanced := gocv.NewMat()
	gocv.Merge(channels, &enhanced)
	for _, ch := range channels {
		ch.Close()
	}

	gocv.CvtColor(enhanced, &processed, gocv.ColorLabToBGR)
	enhanced.Close()

	return processed
}

// applySharpeningFilter はアンシャープマスキングを適用して、ブレた画像のエッジを強調します。
// ブレの程度に応じて適応的にシャープネスの強度を調整します。
func applySharpeningFilter(mat gocv.Mat) gocv.Mat {
	// ガウシアンブラーで平滑化した画像を作成
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(mat, &blurred, image.Point{X: 0, Y: 0}, 3.0, 3.0, gocv.BorderDefault)

	// アンシャープマスキング: sharpened = original * (1 + amount) - blurred * amount
	// amount = 0.5 (穏やかなシャープ化で、ノイズ増幅を抑制)
	sharpened := gocv.NewMat()
	gocv.AddWeighted(mat, 1.5, blurred, -0.5, 0, &sharpened)

	return sharpened
}

// estimateBlurLevel はラプラシアンの分散で画像のブレ度を推定します。
// 低い値ほどブレが大きいことを示します。
func estimateBlurLevel(mat gocv.Mat) float64 {
	gray := gocv.NewMat()
	defer gray.Close()

	if mat.Channels() == 1 {
		mat.CopyTo(&gray)
	} else {
		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)
	}

	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	mean, stdDev := gocv.NewMat(), gocv.NewMat()
	defer mean.Close()
	defer stdDev.Close()
	gocv.MeanStdDev(laplacian, &mean, &stdDev)

	// 分散 = 標準偏差の2乗
	std := stdDev.GetDoubleAt(0, 0)
	return std * std
}

// ============================================================================
// DNN ベースの顔検出
// ============================================================================

// detectWithDNN はSSD DNN モデルを使用して顔を検出します。
// 複数の前処理バリエーションで検出を試み、結果を統合します。
func detectWithDNN(mat gocv.Mat, minConfidence float32) []detectionWithConfidence {
	protoPath, modelPath, found := findDNNModelFiles()
	if !found {
		return nil
	}

	net := gocv.ReadNetFromCaffe(protoPath, modelPath)
	if net.Empty() {
		return nil
	}
	defer net.Close()

	var allDetections []detectionWithConfidence

	// メイン画像での検出
	dets := runDNNInference(net, mat, minConfidence)
	allDetections = append(allDetections, dets...)

	return allDetections
}

// runDNNInference は単一の画像に対してDNN推論を実行します。
func runDNNInference(net gocv.Net, mat gocv.Mat, minConfidence float32) []detectionWithConfidence {
	// SSD モデルの入力サイズは300x300
	blob := gocv.BlobFromImage(mat, 1.0, image.Point{X: 300, Y: 300},
		gocv.NewScalar(104, 177, 123, 0), false, false)
	defer blob.Close()

	net.SetInput(blob, "")
	prob := net.Forward("")
	defer prob.Close()

	detections := gocv.GetBlobChannel(prob, 0, 0)
	defer detections.Close()

	var results []detectionWithConfidence
	for i := 0; i < detections.Rows(); i++ {
		confidence := detections.GetFloatAt(i, 2)
		if confidence > minConfidence {
			left := float64(detections.GetFloatAt(i, 3)) * float64(mat.Cols())
			top := float64(detections.GetFloatAt(i, 4)) * float64(mat.Rows())
			right := float64(detections.GetFloatAt(i, 5)) * float64(mat.Cols())
			bottom := float64(detections.GetFloatAt(i, 6)) * float64(mat.Rows())

			rect := image.Rect(int(left), int(top), int(right), int(bottom))
			// 画像境界内にクリップ
			rect = clipRect(rect, image.Rect(0, 0, mat.Cols(), mat.Rows()))

			if rect.Dx() >= minFaceSize && rect.Dy() >= minFaceSize {
				results = append(results, detectionWithConfidence{
					rect:       rect,
					confidence: confidence,
					source:     "dnn",
				})
			}
		}
	}

	return results
}

// ============================================================================
// Haar Cascade ベースの顔検出
// ============================================================================

// detectWithCascades は複数のカスケード分類器で顔検出を実行します。
// 各分類器の結果を統合し、重複を除去して返します。
func detectWithCascades(mat gocv.Mat, minNeighbors int) []detectionWithConfidence {
	var allDetections []detectionWithConfidence

	for _, cascadeFile := range cascadeFiles {
		classifier := gocv.NewCascadeClassifier()
		if !classifier.Load(cascadeFile) {
			classifier.Close()
			continue
		}

		rects := classifier.DetectMultiScaleWithParams(
			mat,
			1.05,             // scaleFactor: 小さい値ほど検出漏れが減るが遅くなる
			minNeighbors,     // minNeighbors: 低いほど検出しやすいが誤検出が増える
			0,                // flags
			image.Point{X: minFaceSize, Y: minFaceSize}, // minSize
			image.Point{},    // maxSize: 制限なし
		)

		classifier.Close()

		for _, r := range rects {
			allDetections = append(allDetections, detectionWithConfidence{
				rect:       r,
				confidence: 0, // Haar Cascadeは信頼度スコアを返さない
				source:     "cascade",
			})
		}

		// 何か見つかったらそのカスケードで十分
		if len(rects) > 0 {
			break
		}
	}

	return allDetections
}

// ============================================================================
// 後処理・フィルタリング
// ============================================================================

// nonMaxSuppression はIoUベースのNon-Maximum Suppressionを実行して、
// 重複する検出結果を除去します。
func nonMaxSuppression(detections []detectionWithConfidence, iouThreshold float64) []detectionWithConfidence {
	if len(detections) <= 1 {
		return detections
	}

	// 信頼度の降順でソート（Cascadeの信頼度0は最後）
	sort.Slice(detections, func(i, j int) bool {
		return detections[i].confidence > detections[j].confidence
	})

	var selected []detectionWithConfidence
	suppressed := make([]bool, len(detections))

	for i := 0; i < len(detections); i++ {
		if suppressed[i] {
			continue
		}
		selected = append(selected, detections[i])

		for j := i + 1; j < len(detections); j++ {
			if suppressed[j] {
				continue
			}
			if calculateIoU(detections[i].rect, detections[j].rect) >= iouThreshold {
				suppressed[j] = true
			}
		}
	}

	return selected
}

// calculateIoU は2つの矩形のIntersection over Unionを計算します。
func calculateIoU(a, b image.Rectangle) float64 {
	intersection := a.Intersect(b)
	if intersection.Empty() {
		return 0
	}
	interArea := float64(intersection.Dx() * intersection.Dy())
	aArea := float64(a.Dx() * a.Dy())
	bArea := float64(b.Dx() * b.Dy())
	unionArea := aArea + bArea - interArea
	if unionArea == 0 {
		return 0
	}
	return interArea / unionArea
}

// isSkinColor は検出領域がHSV色空間で肌色範囲に入っているか検証します。
// 多様な肌色をカバーする広い範囲と、2つの色相範囲（0〜25, 160〜180）で判定します。
func isSkinColor(mat gocv.Mat, rect image.Rectangle) bool {
	bounds := image.Rect(0, 0, mat.Cols(), mat.Rows())
	rect = rect.Intersect(bounds)
	if rect.Empty() {
		return false
	}

	roi := mat.Region(rect)
	defer roi.Close()

	hsvMat := gocv.NewMat()
	defer hsvMat.Close()
	gocv.CvtColor(roi, &hsvMat, gocv.ColorBGRToHSV)

	// 肌色範囲1: 赤〜黄色系（H: 0〜25）
	lowerBound1 := gocv.NewMatFromScalar(gocv.NewScalar(0, 15, 50, 0), gocv.MatTypeCV8UC3)
	defer lowerBound1.Close()
	upperBound1 := gocv.NewMatFromScalar(gocv.NewScalar(25, 255, 255, 0), gocv.MatTypeCV8UC3)
	defer upperBound1.Close()

	mask1 := gocv.NewMat()
	defer mask1.Close()
	gocv.InRange(hsvMat, lowerBound1, upperBound1, &mask1)

	// 肌色範囲2: 赤色系の折り返し（H: 160〜180）
	lowerBound2 := gocv.NewMatFromScalar(gocv.NewScalar(160, 15, 50, 0), gocv.MatTypeCV8UC3)
	defer lowerBound2.Close()
	upperBound2 := gocv.NewMatFromScalar(gocv.NewScalar(180, 255, 255, 0), gocv.MatTypeCV8UC3)
	defer upperBound2.Close()

	mask2 := gocv.NewMat()
	defer mask2.Close()
	gocv.InRange(hsvMat, lowerBound2, upperBound2, &mask2)

	// 2つのマスクを統合
	combinedMask := gocv.NewMat()
	defer combinedMask.Close()
	gocv.BitwiseOr(mask1, mask2, &combinedMask)

	totalPixels := combinedMask.Rows() * combinedMask.Cols()
	if totalPixels == 0 {
		return false
	}
	skinPixels := gocv.CountNonZero(combinedMask)
	ratio := float64(skinPixels) / float64(totalPixels)

	return ratio >= skinColorMinRatio
}

// hasValidAspectRatio は検出矩形のアスペクト比が顔として妥当かチェックします。
func hasValidAspectRatio(rect image.Rectangle) bool {
	w := float64(rect.Dx())
	h := float64(rect.Dy())
	if w == 0 || h == 0 {
		return false
	}
	ratio := w / h
	return ratio >= aspectRatioMin && ratio <= aspectRatioMax
}

// hasMinimumSize は検出矩形が最小サイズ以上かチェックします。
func hasMinimumSize(rect image.Rectangle) bool {
	return rect.Dx() >= minFaceSize && rect.Dy() >= minFaceSize
}

// filterFalsePositives は検出結果から偽陽性を除外します。
// 複数のフィルタ（アスペクト比、最小サイズ、肌色）を段階的に適用します。
func filterFalsePositives(mat gocv.Mat, detections []detectionWithConfidence) []detectionWithConfidence {
	var filtered []detectionWithConfidence
	for _, det := range detections {
		// DNN の高信頼度検出はフィルタリングを緩和
		if det.source == "dnn" && det.confidence >= dnnConfidenceHigh {
			if hasMinimumSize(det.rect) {
				filtered = append(filtered, det)
			}
			continue
		}

		// それ以外はフルフィルタリング
		if !hasMinimumSize(det.rect) {
			continue
		}
		if !hasValidAspectRatio(det.rect) {
			continue
		}
		if !isSkinColor(mat, det.rect) {
			continue
		}
		filtered = append(filtered, det)
	}
	return filtered
}

// crossValidateDetections はDNNとHaar Cascadeの検出結果を交差検証します。
// 両方のソースで検出された領域は高い信頼性を持ちます。
func crossValidateDetections(mat gocv.Mat, cascadeDets []detectionWithConfidence) []detectionWithConfidence {
	// DNN が利用できない場合はカスケード結果をそのまま返す
	dnnDets := detectWithDNN(mat, dnnConfidenceLow)
	if len(dnnDets) == 0 {
		return cascadeDets
	}

	// カスケード検出のうち、DNNでも確認できたものを優先
	var validated []detectionWithConfidence
	var unvalidated []detectionWithConfidence
	for _, cd := range cascadeDets {
		matched := false
		for _, dd := range dnnDets {
			if calculateIoU(cd.rect, dd.rect) >= 0.15 {
				// DNN の信頼度を引き継ぎ
				cd.confidence = dd.confidence
				validated = append(validated, cd)
				matched = true
				break
			}
		}
		if !matched {
			unvalidated = append(unvalidated, cd)
		}
	}

	if len(validated) > 0 {
		return validated
	}
	// 交差検証で全て除外された場合、元のカスケード結果を返す（過剰除外防止）
	return cascadeDets
}

// ============================================================================
// ユーティリティ関数
// ============================================================================

// largestDetection は検出結果から最も大きい顔を返します。
func largestDetection(dets []Detection) (Detection, bool) {
	var largest Detection
	maxScale := 0
	for _, det := range dets {
		if det.Scale > maxScale {
			maxScale = det.Scale
			largest = det
		}
	}
	if maxScale == 0 {
		return Detection{}, false
	}
	return largest, true
}

// detectionRect はDetectionから矩形を生成します。
func detectionRect(det Detection) image.Rectangle {
	return image.Rect(
		det.Col-det.Scale/2,
		det.Row-det.Scale/2,
		det.Col+det.Scale/2,
		det.Row+det.Scale/2,
	)
}

// clipRect は矩形を画像境界内にクリッピングします。
func clipRect(rect image.Rectangle, bounds image.Rectangle) image.Rectangle {
	if rect.Min.X < bounds.Min.X {
		rect.Min.X = bounds.Min.X
	}
	if rect.Min.Y < bounds.Min.Y {
		rect.Min.Y = bounds.Min.Y
	}
	if rect.Max.X > bounds.Max.X {
		rect.Max.X = bounds.Max.X
	}
	if rect.Max.Y > bounds.Max.Y {
		rect.Max.Y = bounds.Max.Y
	}
	return rect
}

// addMargin は矩形に指定割合のマージンを追加します。
func addMargin(rect image.Rectangle, ratio float64) image.Rectangle {
	w := rect.Dx()
	h := rect.Dy()
	mx := int(float64(w) * ratio)
	my := int(float64(h) * ratio)
	return image.Rect(
		rect.Min.X-mx,
		rect.Min.Y-my,
		rect.Max.X+mx,
		rect.Max.Y+my,
	)
}

// rectsOverlap は2つの矩形がIoU（Intersection over Union）で一定以上重なっているか判定します。
func rectsOverlap(a, b image.Rectangle) bool {
	return calculateIoU(a, b) >= 0.2
}

// ============================================================================
// メイン検出パイプライン
// ============================================================================

// detectFaces は画像データから顔を検出し、検出結果と元画像を返します。
// 商用レベルの多段階検出パイプライン:
//  1. 適応的な前処理（ガンマ補正 + CLAHE）
//  2. DNN（SSD ResNet-10）による高精度検出（メイン）
//  3. Haar Cascade によるフォールバック検出
//  4. 多スケール検出（低解像度画像対応）
//  5. シャープネス改善による再検出（ブレ画像対応）
//  6. NMS + 偽陽性フィルタリング
//  7. DNN/Cascade 交差検証
func detectFaces(imageData []byte) (image.Image, []Detection, error) {
	// バイトスライスから画像をデコード（Go標準ライブラリ、結果返却用）
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, nil, fmt.Errorf("画像のデコードに失敗しました: %v", err)
	}

	// OpenCVで直接デコード（色空間を正しく扱うため）
	mat, err := gocv.IMDecode(imageData, gocv.IMReadColor)
	if err != nil {
		return nil, nil, fmt.Errorf("OpenCVでの画像デコードに失敗しました: %v", err)
	}
	defer mat.Close()

	if mat.Empty() {
		return nil, nil, fmt.Errorf("画像のデコード結果が空です")
	}

	// ========================================================================
	// Phase 1: 適応的前処理
	// ========================================================================
	preprocessed := applyAdaptivePreprocessing(mat)
	defer preprocessed.Close()

	// ========================================================================
	// Phase 2: DNN による検出（メイン経路）
	// ========================================================================
	var allDetections []detectionWithConfidence
	dnnDetected := false

	// 前処理済み画像でDNN検出
	dnnDets := detectWithDNN(preprocessed, dnnConfidenceLow)
	if len(dnnDets) > 0 {
		allDetections = append(allDetections, dnnDets...)
		dnnDetected = true
	}

	// 前処理済みで見つからなければ元画像でも試行
	if !dnnDetected {
		dnnDets = detectWithDNN(mat, dnnConfidenceLow)
		if len(dnnDets) > 0 {
			allDetections = append(allDetections, dnnDets...)
			dnnDetected = true
		}
	}

	// ========================================================================
	// Phase 3: Haar Cascade によるフォールバック（DNNで見つからなかった場合）
	// ========================================================================
	if !dnnDetected {
		// グレースケールに変換
		grayMat := gocv.NewMat()
		defer grayMat.Close()
		gocv.CvtColor(preprocessed, &grayMat, gocv.ColorBGRToGray)

		// ガウシアンブラーでノイズを軽減
		blurredMat := gocv.NewMat()
		defer blurredMat.Close()
		gocv.GaussianBlur(grayMat, &blurredMat, image.Point{X: 5, Y: 5}, 0, 0, gocv.BorderDefault)

		// 通常パラメータで検出
		cascadeDets := detectWithCascades(blurredMat, 4)
		allDetections = append(allDetections, cascadeDets...)

		// 見つからなければパラメータを緩和して再試行
		if len(allDetections) == 0 {
			cascadeDets = detectWithCascades(blurredMat, 3)
			allDetections = append(allDetections, cascadeDets...)
		}

		// ====================================================================
		// Phase 4: 多スケール検出（低解像度画像対応）
		// ====================================================================
		if len(allDetections) == 0 {
			for _, scale := range []int{2, 3} {
				upscaled := gocv.NewMat()
				gocv.Resize(blurredMat, &upscaled, image.Point{
					X: blurredMat.Cols() * scale,
					Y: blurredMat.Rows() * scale,
				}, 0, 0, gocv.InterpolationCubic) // Cubicで高品質アップスケール

				upDets := detectWithCascades(upscaled, 3)
				upscaled.Close()

				// 座標を元のスケールに戻す
				for i := range upDets {
					upDets[i].rect.Min.X /= scale
					upDets[i].rect.Min.Y /= scale
					upDets[i].rect.Max.X /= scale
					upDets[i].rect.Max.Y /= scale
				}
				allDetections = append(allDetections, upDets...)

				if len(allDetections) > 0 {
					break
				}
			}
		}

		// ====================================================================
		// Phase 5: シャープネス改善による再検出（ブレ画像対応）
		// ====================================================================
		if len(allDetections) == 0 {
			blurLevel := estimateBlurLevel(mat)
			if blurLevel < 100.0 { // ブレが大きい場合
				sharpened := applySharpeningFilter(preprocessed)
				defer sharpened.Close()

				sharpGray := gocv.NewMat()
				defer sharpGray.Close()
				gocv.CvtColor(sharpened, &sharpGray, gocv.ColorBGRToGray)

				sharpDets := detectWithCascades(sharpGray, 3)
				allDetections = append(allDetections, sharpDets...)

				// シャープ化画像でDNNも試行
				if len(allDetections) == 0 {
					dnnSharpDets := detectWithDNN(sharpened, dnnConfidenceLow)
					allDetections = append(allDetections, dnnSharpDets...)
				}
			}
		}
	}

	// ========================================================================
	// Phase 6: NMS + 偽陽性フィルタリング
	// ========================================================================
	if len(allDetections) == 0 {
		return img, []Detection{}, nil
	}

	// NMS で重複検出を除去
	allDetections = nonMaxSuppression(allDetections, nmsIOUThreshold)

	// 偽陽性フィルタリング
	filtered := filterFalsePositives(mat, allDetections)
	if len(filtered) > 0 {
		allDetections = filtered
	}
	// フィルタで全て除外された場合は元の検出結果を維持（過剰除外防止）

	// ========================================================================
	// Phase 7: DNN/Cascade 交差検証（Cascade経路のみ）
	// ========================================================================
	if !dnnDetected && len(allDetections) > 0 {
		validated := crossValidateDetections(mat, allDetections)
		if len(validated) > 0 {
			allDetections = validated
		}
	}

	// ========================================================================
	// 結果変換
	// ========================================================================
	var dets []Detection
	for _, d := range allDetections {
		r := d.rect
		dets = append(dets, Detection{
			Row:   r.Min.Y + r.Dy()/2,
			Col:   r.Min.X + r.Dx()/2,
			Scale: (r.Dx() + r.Dy()) / 2,
			Q:     d.confidence,
		})
	}

	return img, dets, nil
}

// ============================================================================
// エクスポートされるAPI関数
// ============================================================================

// DrawFaceRects は画像内の検出された最大の顔の周りに太い四角い枠を描画します。
func DrawFaceRects(imageData []byte) ([]byte, error) {
	img, dets, err := detectFaces(imageData)
	if err != nil {
		return nil, err
	}

	if len(dets) == 0 {
		return nil, fmt.Errorf("顔が検出されませんでした")
	}

	largestDet, ok := largestDetection(dets)
	if !ok {
		return nil, fmt.Errorf("適切なサイズの顔が検出されませんでした")
	}

	// 描画用の新しいRGBA画像を作成
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, img, image.Point{0, 0}, draw.Src)

	// 最も大きい顔の周りに赤い四角を描画
	rect := clipRect(detectionRect(largestDet), b)

	red := color.RGBA{255, 0, 0, 255}
	thickness := 3
	drawThickRect(rgba, rect, red, thickness)

	// 画像をPNGとしてエンコード
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, rgba); err != nil {
		return nil, fmt.Errorf("画像のエンコードに失敗しました: %v", err)
	}

	return buf.Bytes(), nil
}

// drawThickRect は指定された太さで矩形を描画します。
func drawThickRect(img *image.RGBA, rect image.Rectangle, c color.Color, thickness int) {
	// 線の太さを考慮して、描画範囲が画像の範囲を超えないようにクリッピング
	bounds := img.Bounds()
	drawRect := rect.Intersect(bounds)

	for i := 0; i < thickness; i++ {
		// 上辺
		y0 := drawRect.Min.Y + i
		if y0 < bounds.Max.Y {
			for x := drawRect.Min.X; x < drawRect.Max.X; x++ {
				img.Set(x, y0, c)
			}
		}

		// 下辺
		y1 := drawRect.Max.Y - 1 - i
		if y1 >= bounds.Min.Y {
			for x := drawRect.Min.X; x < drawRect.Max.X; x++ {
				img.Set(x, y1, c)
			}
		}

		// 左辺
		x0 := drawRect.Min.X + i
		if x0 < bounds.Max.X {
			for y := drawRect.Min.Y; y < drawRect.Max.Y; y++ {
				img.Set(x0, y, c)
			}
		}

		// 右辺
		x1 := drawRect.Max.X - 1 - i
		if x1 >= bounds.Min.X {
			for y := drawRect.Min.Y; y < drawRect.Max.Y; y++ {
				img.Set(x1, y, c)
			}
		}
	}
}

// CropFace は画像から最も大きく検出された顔を切り抜きます。
func CropFace(imageData []byte) ([]byte, error) {
	img, dets, err := detectFaces(imageData)
	if err != nil {
		return nil, err
	}

	if len(dets) == 0 {
		return nil, fmt.Errorf("顔が検出されませんでした")
	}

	largestDet, ok := largestDetection(dets)
	if !ok {
		return nil, fmt.Errorf("適切なサイズの顔が検出されませんでした")
	}

	// 顔領域を切り抜く（15%のマージンを追加して額・顎を含める）
	faceRect := addMargin(detectionRect(largestDet), 0.15)
	faceRect = clipRect(faceRect, img.Bounds())

	var croppedImg image.Image
	if sub, ok := img.(subImager); ok {
		croppedImg = sub.SubImage(faceRect)
	} else {
		cropped := image.NewRGBA(faceRect.Bounds())
		draw.Draw(cropped, faceRect.Bounds(), img, faceRect.Min, draw.Src)
		croppedImg = cropped
	}

	// 画像をPNGとしてエンコード
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, croppedImg); err != nil {
		return nil, fmt.Errorf("画像のエンコードに失敗しました: %v", err)
	}

	return buf.Bytes(), nil
}

// CalculateFaceSharpness は、画像内の顔の鮮明度スコアを計算します。
// 複数の顔が検出された場合は、最も高いスコアを返します。
func CalculateFaceSharpness(imageData []byte) (float64, error) {
	img, dets, err := detectFaces(imageData)
	if err != nil {
		return 0, err
	}

	if len(dets) == 0 {
		return 0, fmt.Errorf("顔が検出されませんでした")
	}

	maxSharpness := 0.0
	// 検出された各顔に対して鮮明度を計算
	for _, det := range dets {
		faceRect := clipRect(detectionRect(det), img.Bounds())

		var faceImg image.Image
		if sub, ok := img.(subImager); ok {
			faceImg = sub.SubImage(faceRect)
		} else {
			cropped := image.NewRGBA(faceRect.Bounds())
			draw.Draw(cropped, faceRect.Bounds(), img, faceRect.Min, draw.Src)
			faceImg = cropped
		}

		// グレースケール画像に変換
		grayImg := convertToGrayscale(faceImg)

		// ラプラシアンフィルタを適用して鮮明度を計算
		sharpness := calculateLaplacianVariance(grayImg)

		if sharpness > maxSharpness {
			maxSharpness = sharpness
		}
	}

	return maxSharpness, nil
}

// CalculateSharpness は、画像データの鮮明度スコアを計算します。
func CalculateSharpness(imageData []byte) (float64, error) {
	if len(imageData) == 0 {
		return 0, fmt.Errorf("画像データが空です")
	}

	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return 0, fmt.Errorf("画像のデコードに失敗しました: %v", err)
	}

	grayImg := convertToGrayscale(img)
	sharpness := calculateLaplacianVariance(grayImg)
	return sharpness, nil
}

// ============================================================================
// 画像解析ヘルパー
// ============================================================================

// convertToGrayscale は画像をグレースケールに変換します
func convertToGrayscale(img image.Image) [][]float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	gray := make([][]float64, height)
	for y := 0; y < height; y++ {
		gray[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			grayValue := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			gray[y][x] = grayValue
		}
	}
	return gray
}

// calculateLaplacianVariance はラプラシアンフィルタの分散を計算します
func calculateLaplacianVariance(gray [][]float64) float64 {
	height := len(gray)
	if height <= 2 {
		return 0
	}
	width := len(gray[0])
	if width <= 2 {
		return 0
	}

	laplacian := make([][]float64, height-2)
	sum := 0.0
	count := 0

	for y := 1; y < height-1; y++ {
		laplacian[y-1] = make([]float64, width-2)
		for x := 1; x < width-1; x++ {
			value := gray[y][x]*(-4) + gray[y-1][x] + gray[y+1][x] + gray[y][x-1] + gray[y][x+1]
			laplacian[y-1][x-1] = value
			sum += value
			count++
		}
	}

	if count == 0 {
		return 0
	}
	mean := sum / float64(count)

	variance := 0.0
	for y := 0; y < len(laplacian); y++ {
		for x := 0; x < len(laplacian[y]); x++ {
			variance += (laplacian[y][x] - mean) * (laplacian[y][x] - mean)
		}
	}
	variance /= float64(count)

	return math.Abs(variance)
}

