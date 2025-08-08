package facedetector

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

// CalculateSharpness は、画像データの鮮明度スコアを計算します。
// スコアが高いほど、画像が鮮明であることを示します。
// この実装では、ラプラシアンフィルタの分散を鮮明度スコアとして使用します。
func CalculateSharpness(imageData []byte) (float64, error) {
	if len(imageData) == 0 {
		return 0, fmt.Errorf("画像データが空です")
	}

	// バイトスライスから画像をデコード
	// image.Decodeは、登録されたフォーマット（jpeg, pngなど）を自動的に検出します。
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return 0, fmt.Errorf("画像のデコードに失敗しました: %v", err)
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
