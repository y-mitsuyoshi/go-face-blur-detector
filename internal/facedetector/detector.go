package facedetector

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"math"

	pigo "github.com/esimov/pigo/core"
)

var cascadeFile = "cascade/facefinder"

// CalculateFaceSharpness は、画像内の顔の鮮明度スコアを計算します。
// 複数の顔が検出された場合は、最も高いスコアを返します。
func CalculateFaceSharpness(imageData []byte) (float64, error) {
	// カスケードファイル（分類器）を読み込む
	cascade, err := ioutil.ReadFile(cascadeFile)
	if err != nil {
		return 0, fmt.Errorf("カスケードファイルの読み込みに失敗しました: %v", err)
	}

	// バイトスライスから画像をデコード
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return 0, fmt.Errorf("画像のデコードに失敗しました: %v", err)
	}

	// pigoが要求する画像形式に変換
	pixels := pigo.RgbToGrayscale(img)
	cols, rows := img.Bounds().Max.X, img.Bounds().Max.Y

	// pigo分類器を初期化
	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}
	pigo := pigo.NewPigo()

	classifier, err := pigo.Unpack(cascade)
	if err != nil {
		return 0, fmt.Errorf("カスケードのアンパックに失敗しました: %v", err)
	}

	// 顔検出を実行
	angle := 0.0 // 0.0は正面顔のみ
	dets := classifier.RunCascade(cParams, angle)
	dets = classifier.ClusterDetections(dets, 0.2)

	if len(dets) == 0 {
		return 0, fmt.Errorf("顔が検出されませんでした")
	}

	maxSharpness := 0.0
	// 検出された各顔に対して鮮明度を計算
	for _, det := range dets {
		if det.Scale > 50 {
			// 顔領域を切り抜く
			faceRect := image.Rect(
				det.Col-det.Scale/2,
				det.Row-det.Scale/2,
				det.Col+det.Scale/2,
				det.Row+det.Scale/2,
			)

			// Goのimage.Imageから顔部分をサブイメージとして切り出す
			// サブイメージをサポートする型(例: *image.RGBA)に変換が必要
			// ここでは簡単な実装のため、元の画像から直接ピクセルを読み出す
			// 注意：この実装は元の画像が*image.Grayでないと正しく動作しない可能性がある
			// より堅牢な実装では、型アサーションと変換が必要
			// ここではconvertToGrayscaleがimage.Imageを受け入れるので、サブイメージでなくとも良い
			bounds := img.Bounds()
			croppedImg := image.NewRGBA(faceRect)

			for y := faceRect.Min.Y; y < faceRect.Max.Y; y++ {
				for x := faceRect.Min.X; x < faceRect.Max.X; x++ {
					if x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
						croppedImg.Set(x, y, img.At(x, y))
					}
				}
			}
			// グレースケール画像に変換
			grayImg := convertToGrayscale(croppedImg)

			// ラプラシアンフィルタを適用して鮮明度を計算
			sharpness := calculateLaplacianVariance(grayImg)

			if sharpness > maxSharpness {
				maxSharpness = sharpness
			}
		}
	}

	return maxSharpness, nil
}

// CalculateSharpness は、画像データの鮮明度スコアを計算します。
// スコアが高いほど、画像が鮮明であることを示します。
// この実装では、ラプラシアンフィルタの分散を鮮明度スコアとして使用します。
func CalculateSharpness(imageData []byte) (float64, error) {
	if len(imageData) == 0 {
		return 0, fmt.Errorf("画像データが空です")
	}

	// バイトスライスから画像をデコード
	reader := bytes.NewReader(imageData)
	img, _, err := image.Decode(reader)
	if err != nil {
		// JPEGとPNGで再試行
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

	// グレースケール画像に変換
	grayImg := convertToGrayscale(img)

	// ラプラシアンフィルタを適用して鮮明度を計算
	sharpness := calculateLaplacianVariance(grayImg)

	return sharpness, nil
}

// convertToGrayscale は画像をグレースケールに変換します
func convertToGrayscale(img image.Image) [][]float64 {
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	gray := make([][]float64, height)
	for y := 0; y < height; y++ {
		gray[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// グレースケール変換 (ITU-R BT.601 formula)
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

	// ラプラシアンカーネル
	// [ 0  1  0]
	// [ 1 -4  1]
	// [ 0  1  0]
	laplacian := make([][]float64, height-2)
	for y := 0; y < height-2; y++ {
		laplacian[y] = make([]float64, width-2)
		for x := 0; x < width-2; x++ {
			// ラプラシアンフィルタを適用
			value := gray[y+1][x+1]*(-4) +
				gray[y][x+1] +
				gray[y+2][x+1] +
				gray[y+1][x] +
				gray[y+1][x+2]
			laplacian[y][x] = value
		}
	}

	// 分散を計算
	if len(laplacian) == 0 || len(laplacian[0]) == 0 {
		return 0
	}

	// 平均を計算
	sum := 0.0
	count := 0
	for y := 0; y < len(laplacian); y++ {
		for x := 0; x < len(laplacian[y]); x++ {
			sum += laplacian[y][x]
			count++
		}
	}

	if count == 0 {
		return 0
	}

	mean := sum / float64(count)

	// 分散を計算
	variance := 0.0
	for y := 0; y < len(laplacian); y++ {
		for x := 0; x < len(laplacian[y]); x++ {
			diff := laplacian[y][x] - mean
			variance += diff * diff
		}
	}
	variance /= float64(count)

	return math.Abs(variance)
}
