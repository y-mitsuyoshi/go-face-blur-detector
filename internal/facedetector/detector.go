package facedetector

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"

	"gocv.io/x/gocv"
)

// dnnModelFilesExist はDNNモデルファイルが両方存在するか確認します。
func dnnModelFilesExist() bool {
	if _, err := os.Stat(dnnProtoFile); err != nil {
		return false
	}
	if _, err := os.Stat(dnnModelFile); err != nil {
		return false
	}
	return true
}

// Detection はpigoライブラリの検出結果を模倣した構造体です。
// GoCVとの互換性のために残しますが、pigo特有のフィールドは使われません。
type Detection struct {
	Row   int
	Col   int
	Scale int
	Q     float32 // 信頼度スコア (GoCVでは直接提供されない)
}

var cascadeFiles = []string{
	"cascade/haarcascade_frontalface_alt2.xml",
	"/usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
	"/usr/share/opencv4/haarcascades/haarcascade_frontalface_alt.xml",
	"/usr/share/opencv4/haarcascades/haarcascade_profileface.xml",
}

const (
	dnnModelFile  = "models/res10_300x300_ssd_iter_140000.caffemodel"
	dnnProtoFile  = "models/deploy.prototxt"
	minConfidence = 0.5
)

// isSkinColor は検出領域がHSV色空間で肌色範囲に入っているか検証します。
// 肌色ピクセルが一定割合以上ならtrueを返します。
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

	// 肌色のHSV範囲 (広めに設定して多様な肌色をカバー)
	lowerBound := gocv.NewMatFromScalar(gocv.NewScalar(0, 20, 70, 0), gocv.MatTypeCV8UC3)
	defer lowerBound.Close()
	upperBound := gocv.NewMatFromScalar(gocv.NewScalar(35, 255, 255, 0), gocv.MatTypeCV8UC3)
	defer upperBound.Close()

	mask := gocv.NewMat()
	defer mask.Close()
	gocv.InRange(hsvMat, lowerBound, upperBound, &mask)

	totalPixels := mask.Rows() * mask.Cols()
	if totalPixels == 0 {
		return false
	}
	skinPixels := gocv.CountNonZero(mask)
	ratio := float64(skinPixels) / float64(totalPixels)

	// 肌色ピクセルが15%以上であれば顔と判定
	return ratio >= 0.15
}

// hasValidAspectRatio は検出矩形のアスペクト比が顔として妥当かチェックします。
func hasValidAspectRatio(rect image.Rectangle) bool {
	w := float64(rect.Dx())
	h := float64(rect.Dy())
	if w == 0 || h == 0 {
		return false
	}
	ratio := w / h
	// 顔のアスペクト比は概ね0.6〜1.6の範囲
	return ratio >= 0.6 && ratio <= 1.6
}

// filterFalsePositives は検出結果から偽陽性を除外します。
func filterFalsePositives(mat gocv.Mat, rects []image.Rectangle) []image.Rectangle {
	var filtered []image.Rectangle
	for _, r := range rects {
		if !hasValidAspectRatio(r) {
			continue
		}
		if !isSkinColor(mat, r) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// crossValidateWithDNN はHaar Cascadeの検出結果をDNNで再確認します。
// DNNモデルが利用できない場合は元の結果をそのまま返します。
func crossValidateWithDNN(mat gocv.Mat, cascadeRects []image.Rectangle) []image.Rectangle {
	if !dnnModelFilesExist() {
		return cascadeRects
	}
	net := gocv.ReadNetFromCaffe(dnnProtoFile, dnnModelFile)
	if net.Empty() {
		// DNNモデルが利用できない場合はフィルタリングのみで返す
		return cascadeRects
	}
	defer net.Close()

	blob := gocv.BlobFromImage(mat, 1.0, image.Point{X: 300, Y: 300}, gocv.NewScalar(104, 177, 123, 0), false, false)
	defer blob.Close()

	net.SetInput(blob, "")
	prob := net.Forward("")
	defer prob.Close()

	detections := gocv.GetBlobChannel(prob, 0, 0)
	defer detections.Close()

	var dnnRects []image.Rectangle
	for i := 0; i < detections.Rows(); i++ {
		confidence := detections.GetFloatAt(i, 2)
		// 交差検証では低めの閾値を使用（0.3）
		if confidence > 0.3 {
			left := float64(detections.GetFloatAt(i, 3)) * float64(mat.Cols())
			top := float64(detections.GetFloatAt(i, 4)) * float64(mat.Rows())
			right := float64(detections.GetFloatAt(i, 5)) * float64(mat.Cols())
			bottom := float64(detections.GetFloatAt(i, 6)) * float64(mat.Rows())
			dnnRects = append(dnnRects, image.Rect(int(left), int(top), int(right), int(bottom)))
		}
	}

	if len(dnnRects) == 0 {
		// DNNが何も検出しなかった場合、Cascade結果をそのまま返す（DNNが苦手なケース）
		return cascadeRects
	}

	// Cascade検出結果のうち、DNNの検出結果と重なるものだけを残す
	var validated []image.Rectangle
	for _, cr := range cascadeRects {
		for _, dr := range dnnRects {
			if rectsOverlap(cr, dr) {
				validated = append(validated, cr)
				break
			}
		}
	}

	if len(validated) == 0 {
		// 交差検証で全て除外された場合、Cascade結果を返す
		return cascadeRects
	}
	return validated
}

// rectsOverlap は2つの矩形がIoU（Intersection over Union）で一定以上重なっているか判定します。
func rectsOverlap(a, b image.Rectangle) bool {
	intersection := a.Intersect(b)
	if intersection.Empty() {
		return false
	}
	interArea := float64(intersection.Dx() * intersection.Dy())
	aArea := float64(a.Dx() * a.Dy())
	bArea := float64(b.Dx() * b.Dy())
	unionArea := aArea + bArea - interArea
	if unionArea == 0 {
		return false
	}
	iou := interArea / unionArea
	return iou >= 0.2
}

// tryDetectWithDNN はOpenCVのDNNモジュールを使用して顔検出を実行します。
func tryDetectWithDNN(mat gocv.Mat) []image.Rectangle {
	// モデルファイルが存在するか確認
	if !dnnModelFilesExist() {
		return nil
	}
	net := gocv.ReadNetFromCaffe(dnnProtoFile, dnnModelFile)
	if net.Empty() {
		return nil
	}
	defer net.Close()

	// 画像をブロブに変換 (300x300, mean subtraction, RGB swap false)
	blob := gocv.BlobFromImage(mat, 1.0, image.Point{X: 300, Y: 300}, gocv.NewScalar(104, 177, 123, 0), false, false)
	defer blob.Close()

	net.SetInput(blob, "")
	prob := net.Forward("")
	defer prob.Close()

	// 推論結果の解析
	// 結果の形状は [1, 1, N, 7]
	// 7つの要素: [batchId, classId, confidence, left, top, right, bottom]
	rects := []image.Rectangle{}
	detections := gocv.GetBlobChannel(prob, 0, 0)
	defer detections.Close()

	for i := 0; i < detections.Rows(); i++ {
		confidence := detections.GetFloatAt(i, 2)
		if confidence > minConfidence {
			left := float64(detections.GetFloatAt(i, 3)) * float64(mat.Cols())
			top := float64(detections.GetFloatAt(i, 4)) * float64(mat.Rows())
			right := float64(detections.GetFloatAt(i, 5)) * float64(mat.Cols())
			bottom := float64(detections.GetFloatAt(i, 6)) * float64(mat.Rows())

			rect := image.Rect(int(left), int(top), int(right), int(bottom))
			rects = append(rects, rect)
		}
	}

	return rects
}

// tryDetectWithCascades は複数のカスケード分類器を順に試して顔検出を実行します。
func tryDetectWithCascades(mat gocv.Mat, minNeighbors int) []image.Rectangle {
	for _, cascadeFile := range cascadeFiles {
		classifier := gocv.NewCascadeClassifier()
		if !classifier.Load(cascadeFile) {
			classifier.Close()
			continue
		}
		rects := classifier.DetectMultiScaleWithParams(mat, 1.05, minNeighbors,
			0, image.Point{X: 20, Y: 20}, image.Point{})
		classifier.Close()
		if len(rects) > 0 {
			return rects
		}
	}
	return nil
}

// detectFaces は画像データから顔を検出し、検出結果と元画像を返します。
// DNN（SSD）を優先し、利用できない場合はHaar Cascadeを使用します。
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

	// --- DNN用にCLAHE前処理を適用 ---
	enhancedMat := gocv.NewMat()
	defer enhancedMat.Close()
	{
		labMat := gocv.NewMat()
		defer labMat.Close()
		gocv.CvtColor(mat, &labMat, gocv.ColorBGRToLab)

		// LABのLチャネルにCLAHEを適用
		channels := gocv.Split(labMat)
		dnnClahe := gocv.NewCLAHEWithParams(2.0, image.Point{X: 8, Y: 8})
		defer dnnClahe.Close()
		dnnClahe.Apply(channels[0], &channels[0])

		gocv.Merge(channels, &enhancedMat)
		for _, ch := range channels {
			ch.Close()
		}
		gocv.CvtColor(enhancedMat, &enhancedMat, gocv.ColorLabToBGR)
	}

	// --- DNNによる検出を試行（CLAHE前処理済み画像で） ---
	var rects []image.Rectangle
	rects = tryDetectWithDNN(enhancedMat)

	// CLAHE前処理で見つからなければ元画像でも試行
	if len(rects) == 0 {
		rects = tryDetectWithDNN(mat)
	}

	// DNNで見つからなかった場合、従来のHaar Cascadeで試行
	if len(rects) == 0 {
		// グレースケールに変換（Haar Cascade用）
		grayMat := gocv.NewMat()
		defer grayMat.Close()
		gocv.CvtColor(mat, &grayMat, gocv.ColorBGRToGray)

		// CLAHE（適応的ヒストグラム均等化）で逆光・低コントラストを改善
		clahe := gocv.NewCLAHEWithParams(2.0, image.Point{X: 8, Y: 8})
		defer clahe.Close()
		equalizedMat := gocv.NewMat()
		defer equalizedMat.Close()
		clahe.Apply(grayMat, &equalizedMat)

		// ガウシアンブラーを適用して背景の照明などのノイズ（エッジ）を滑らかにする
		blurredMat := gocv.NewMat()
		defer blurredMat.Close()
		gocv.GaussianBlur(equalizedMat, &blurredMat, image.Point{X: 5, Y: 5}, 0, 0, gocv.BorderDefault)

		// 複数のカスケード分類器を順に試して顔検出を実行
		rects = tryDetectWithCascades(blurredMat, 4)

		// 顔が検出されなかった場合、画像を段階的に拡大して再試行
		for _, scale := range []int{2, 4} {
			if len(rects) > 0 {
				break
			}
			upscaled := gocv.NewMat()
			gocv.Resize(blurredMat, &upscaled, image.Point{
				X: blurredMat.Cols() * scale,
				Y: blurredMat.Rows() * scale,
			}, 0, 0, gocv.InterpolationLinear)
			rects = tryDetectWithCascades(upscaled, 3)
			upscaled.Close()
			// 座標を元のスケールに戻す
			for i := range rects {
				rects[i].Min.X /= scale
				rects[i].Min.Y /= scale
				rects[i].Max.X /= scale
				rects[i].Max.Y /= scale
			}
		}
	}

	// Haar Cascade経路の結果に対して偽陽性フィルタリングを適用
	if len(rects) > 0 {
		// 肌色・アスペクト比フィルタ
		filtered := filterFalsePositives(mat, rects)
		if len(filtered) > 0 {
			// DNN交差検証
			rects = crossValidateWithDNN(mat, filtered)
		}
		// フィルタで全て除外された場合は元の検出結果を維持（過剰除外を防止）
	}

	if len(rects) == 0 {
		// 顔が検出されなかった場合、空のスライスを返す
		return img, []Detection{}, nil
	}

	// gocv.Rectをpigo互換のDetection構造体に変換
	var dets []Detection
	for _, r := range rects {
		dets = append(dets, Detection{
			Row:   r.Min.Y + r.Dy()/2,
			Col:   r.Min.X + r.Dx()/2,
			Scale: (r.Dx() + r.Dy()) / 2, // スケールを平均サイズとして近似
			Q:     10.0,                  // GoCVは信頼度スコアを直接返さないため、ダミー値を設定
		})
	}

	return img, dets, nil
}

// DrawFaceRects は画像内の検出された最大の顔の周りに太い四角い枠を描画します。
func DrawFaceRects(imageData []byte) ([]byte, error) {
	img, dets, err := detectFaces(imageData)
	if err != nil {
		return nil, err
	}

	if len(dets) == 0 {
		// 顔が検出されなかった場合でもエラーではなく、元の画像を返すか、特定のメッセージを返すか選択
		// ここではエラーとして扱う
		return nil, fmt.Errorf("顔が検出されませんでした")
	}

	// 最も大きい顔を見つける
	var largestDet Detection
	maxScale := 0
	for _, det := range dets {
		if det.Scale > maxScale {
			maxScale = det.Scale
			largestDet = det
		}
	}

	if maxScale == 0 {
		return nil, fmt.Errorf("適切なサイズの顔が検出されませんでした")
	}

	// 描画用の新しいRGBA画像を作成
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, img, image.Point{0, 0}, draw.Src)

	// 最も大きい顔の周りに赤い四角を描画
	rect := image.Rect(
		largestDet.Col-largestDet.Scale/2,
		largestDet.Row-largestDet.Scale/2,
		largestDet.Col+largestDet.Scale/2,
		largestDet.Row+largestDet.Scale/2,
	)

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

	// 最も大きい顔を見つける
	var largestDet Detection
	maxScale := 0
	for _, det := range dets {
		if det.Scale > maxScale {
			maxScale = det.Scale
			largestDet = det
		}
	}

	if maxScale == 0 {
		return nil, fmt.Errorf("適切なサイズの顔が検出されませんでした")
	}

	// 顔領域を切り抜く
	faceRect := image.Rect(
		largestDet.Col-largestDet.Scale/2,
		largestDet.Row-largestDet.Scale/2,
		largestDet.Col+largestDet.Scale/2,
		largestDet.Row+largestDet.Scale/2,
	)

	// 新しい画像に切り抜いた部分をコピー
	// `image.Image`インタフェースはSubImageをサポートしている
	type SubImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	var croppedImg image.Image
	if sub, ok := img.(SubImager); ok {
		croppedImg = sub.SubImage(faceRect)
	} else {
		// SubImageをサポートしていない場合は手動でコピー
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
		// 顔領域を切り抜く
		faceRect := image.Rect(
			det.Col-det.Scale/2,
			det.Row-det.Scale/2,
			det.Col+det.Scale/2,
			det.Row+det.Scale/2,
		)

		// Goのimage.Imageから顔部分をサブイメージとして切り出す
		type SubImager interface {
			SubImage(r image.Rectangle) image.Image
		}
		var faceImg image.Image
		if sub, ok := img.(SubImager); ok {
			faceImg = sub.SubImage(faceRect)
		} else {
			// fallback
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

	reader := bytes.NewReader(imageData)
	img, _, err := image.Decode(reader)
	if err != nil {
		reader.Seek(0, 0)
		img, err = jpeg.Decode(reader)
		if err != nil {
			reader.Seek(0, 0)
			img, err = png.Decode(reader)
			if err != nil {
				return 0, fmt.Errorf("画像のデコードに失敗しました: %v", err)
			}
		}
	}

	grayImg := convertToGrayscale(img)
	sharpness := calculateLaplacianVariance(grayImg)
	return sharpness, nil
}

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
