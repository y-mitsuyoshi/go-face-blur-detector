package facedetector

import (
	"fmt"
	"math"

	"gocv.io/x/gocv"
)

// CalculateSharpness は、画像データの鮮明度スコアを計算します。
// スコアが高いほど、画像が鮮明であることを示します。
// この実装では、ラプラシアンフィルタの分散を鮮明度スコアとして使用します。
func CalculateSharpness(imageData []byte) (float64, error) {
	// バイトスライスから画像を gocv.Mat に直接デコード
	// gocv.IMReadColor は BGR フォーマットで読み込む
	mat, err := gocv.IMDecode(imageData, gocv.IMReadColor)
	if err != nil {
		return 0, fmt.Errorf("画像のデコードに失敗しました: %v", err)
	}
	if mat.Empty() {
		return 0, fmt.Errorf("画像データが空か、サポートされていない形式です")
	}
	defer mat.Close()

	// グレースケールに変換
	grayMat := gocv.NewMat()
	defer grayMat.Close()
	gocv.CvtColor(mat, &grayMat, gocv.ColorBGRToGray)

	// ラプラシアンフィルタを適用
	laplacianMat := gocv.NewMat()
	defer laplacianMat.Close()
	gocv.Laplacian(grayMat, &laplacianMat, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	// 平均と標準偏差を計算
	mean, stdDev := gocv.MeanStdDev(laplacianMat)
	defer mean.Close()
	defer stdDev.Close()

	// 鮮明度スコア（分散）を計算
	// 標準偏差は1x1のMatで返されるため、その値を取得します。
	sharpness := math.Pow(stdDev.GetDoubleAt(0, 0), 2)

	return sharpness, nil
}
