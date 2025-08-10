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

	"gocv.io/x/gocv"
)

// Detection はpigoライブラリの検出結果を模倣した構造体です。
// GoCVとの互換性のために残しますが、pigo特有のフィールドは使われません。
type Detection struct {
	Row   int
	Col   int
	Scale int
	Q     float32 // 信頼度スコア (GoCVでは直接提供されない)
}

var cascadeFile = "cascade/haarcascade_frontalface_alt2.xml"

// detectFaces は画像データから顔を検出し、検出結果と元画像を返します。
// pigoの代わりにGoCVを使用するように変更されました。
func detectFaces(imageData []byte) (image.Image, []Detection, error) {
	// バイトスライスから画像をデコード
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, nil, fmt.Errorf("画像のデコードに失敗しました: %v", err)
	}

	// Goのimage.Imageをgocv.Matに変換
	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return nil, nil, fmt.Errorf("image.Imageからgocv.Matへの変換に失敗しました: %v", err)
	}
	defer mat.Close()

	// カスケード分類器をロード
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load(cascadeFile) {
		return nil, nil, fmt.Errorf("カスケードファイルの読み込みに失敗しました: %s", cascadeFile)
	}

	// 顔検出を実行
	rects := classifier.DetectMultiScale(mat)
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
			Q:     10.0,                   // GoCVは信頼度スコアを直接返さないため、ダミー値を設定
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
