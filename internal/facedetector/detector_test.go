package facedetector

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// createTestImage creates a simple test image with the specified pattern
func createTestImage(width, height int, pattern string) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	switch pattern {
	case "sharp":
		// Create a sharp checkerboard pattern
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if (x/10+y/10)%2 == 0 {
					img.Set(x, y, color.RGBA{255, 255, 255, 255}) // White
				} else {
					img.Set(x, y, color.RGBA{0, 0, 0, 255}) // Black
				}
			}
		}
	case "blurred":
		// Create a smoother pattern (less sharp)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				val := uint8((x + y) % 256)
				img.Set(x, y, color.RGBA{val, val, val, 255})
			}
		}
	default:
		// Solid color
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.Set(x, y, color.RGBA{128, 128, 128, 255})
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestCalculateSharpness(t *testing.T) {
	// Create test images programmatically
	sharpImgData := createTestImage(100, 100, "sharp")
	blurredImgData := createTestImage(100, 100, "blurred")

	// Define test cases
	testCases := []struct {
		name           string
		imageData      []byte
		expectErr      bool
		checkSharpness func(t *testing.T, sharpness float64)
	}{
		{
			name:      "Sharp Image",
			imageData: sharpImgData,
			expectErr: false,
			checkSharpness: func(t *testing.T, sharpness float64) {
				// We expect a non-zero sharpness for a sharp image.
				// The exact value can vary, so we check it's positive.
				if sharpness <= 0 {
					t.Errorf("expected sharpness score to be positive for a sharp image, got %f", sharpness)
				}
			},
		},
		{
			name:      "Blurred Image",
			imageData: blurredImgData,
			expectErr: false,
			checkSharpness: func(t *testing.T, sharpness float64) {
				// We expect a non-zero sharpness for a blurred image, but lower than the sharp one.
				if sharpness <= 0 {
					t.Errorf("expected sharpness score to be positive for a blurred image, got %f", sharpness)
				}
			},
		},
		{
			name:      "Empty Image Data",
			imageData: []byte{},
			expectErr: true,
		},
		{
			name:      "Invalid Image Data",
			imageData: []byte("invalid image data"),
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sharpness, err := CalculateSharpness(tc.imageData)

			if tc.expectErr {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error but got: %v", err)
				}
				if tc.checkSharpness != nil {
					tc.checkSharpness(t, sharpness)
				}
			}
		})
	}

	// Additional test to compare sharpness scores directly
	t.Run("Compare Sharpness", func(t *testing.T) {
		sharpScore, err := CalculateSharpness(sharpImgData)
		if err != nil {
			t.Fatalf("failed to calculate sharpness for sharp image: %v", err)
		}

		blurredScore, err := CalculateSharpness(blurredImgData)
		if err != nil {
			t.Fatalf("failed to calculate sharpness for blurred image: %v", err)
		}

		if sharpScore <= blurredScore {
			t.Errorf("expected sharpness of sharp image (%f) to be greater than blurred image (%f)", sharpScore, blurredScore)
		}
		t.Logf("Sharp score: %f, Blurred score: %f", sharpScore, blurredScore)
	})
}

func TestCalculateFaceSharpness(t *testing.T) {
	// テスト用の鮮明な顔写真を読み込む
	sharpFaceImgData, err := os.ReadFile("testdata/face.jpg")
	if err != nil {
		t.Fatalf("鮮明な顔画像の読み込みに失敗しました: %v", err)
	}

	// 鮮明な顔画像の鮮明度を計算
	sharpScore, err := CalculateFaceSharpness(sharpFaceImgData)
	if err != nil {
		t.Fatalf("鮮明な顔画像の鮮明度計算に失敗しました: %v", err)
	}

	// スコアが0より大きいことを確認
	if sharpScore <= 0 {
		t.Errorf("鮮明な顔画像のスコア (%f) が0より大きくありません", sharpScore)
	}
	t.Logf("鮮明な顔画像のスコア: %f", sharpScore)

	// テスト用のぼやけた顔写真を読み込む
	blurredFaceImgData, err := os.ReadFile("testdata/face_blurred.jpg")
	if err != nil {
		t.Fatalf("ぼやけた顔画像の読み込みに失敗しました: %v", err)
	}

	// ぼやけた顔画像の鮮明度を計算
	blurredScore, err := CalculateFaceSharpness(blurredFaceImgData)
	if err != nil {
		t.Fatalf("ぼやけた顔画像の鮮明度計算に失敗しました: %v", err)
	}

	if blurredScore <= 0 {
		t.Errorf("ぼやけた顔画像のスコア (%f) が0より大きくありません", blurredScore)
	}
	t.Logf("ぼやけた顔画像のスコア: %f", blurredScore)

	// 鮮明な画像のスコアがぼやけた画像のスコアより高いことを確認
	if sharpScore <= blurredScore {
		t.Errorf("鮮明な画像のスコア (%f) がぼやけた画像のスコア (%f) より高くありません", sharpScore, blurredScore)
	}

	// 顔が検出されない画像をテスト
	noFaceImgData := createTestImage(100, 100, "sharp") // 顔ではない画像
	_, err = CalculateFaceSharpness(noFaceImgData)
	if err == nil {
		t.Errorf("顔のない画像でエラーが返されませんでした")
	} else {
		t.Logf("顔のない画像で期待どおりエラーを検出: %v", err)
	}
}

func TestCropFace(t *testing.T) {
	// 顔が検出される画像をテスト
	faceImgData, err := os.ReadFile("testdata/face.jpg")
	if err != nil {
		t.Fatalf("顔画像の読み込みに失敗しました: %v", err)
	}

	croppedData, err := CropFace(faceImgData)
	if err != nil {
		t.Fatalf("CropFaceでエラーが発生しました: %v", err)
	}

	if len(croppedData) == 0 {
		t.Errorf("出力された画像データが空です")
	}

	// 出力画像がデコード可能か確認
	_, _, err = image.Decode(bytes.NewReader(croppedData))
	if err != nil {
		t.Errorf("出力画像のデコードに失敗しました: %v", err)
	}

	// 元の画像よりサイズが小さいことを確認
	if len(croppedData) >= len(faceImgData) {
		t.Errorf("切り抜かれた画像が元の画像より小さくありません")
	}

	// 顔が検出されない画像をテスト
	noFaceImgData := createTestImage(100, 100, "solid") // 顔ではない画像
	_, err = CropFace(noFaceImgData)
	if err == nil {
		t.Errorf("顔のない画像でエラーが返されませんでした")
	} else {
		t.Logf("顔のない画像で期待どおりエラーを検出: %v", err)
	}
}

func TestDrawFaceRects(t *testing.T) {
	// 顔が検出される画像をテスト
	faceImgData, err := os.ReadFile("testdata/face.jpg")
	if err != nil {
		t.Fatalf("顔画像の読み込みに失敗しました: %v", err)
	}

	visualizedData, err := DrawFaceRects(faceImgData)
	if err != nil {
		t.Fatalf("DrawFaceRectsでエラーが発生しました: %v", err)
	}

	if len(visualizedData) == 0 {
		t.Errorf("出力された画像データが空です")
	}

	// 出力画像がデコード可能か確認
	_, _, err = image.Decode(bytes.NewReader(visualizedData))
	if err != nil {
		t.Errorf("出力画像のデコードに失敗しました: %v", err)
	}

	// 顔が検出されない画像をテスト
	noFaceImgData := createTestImage(100, 100, "solid") // 顔ではない画像
	_, err = DrawFaceRects(noFaceImgData)
	if err == nil {
		t.Errorf("顔のない画像でエラーが返されませんでした")
	} else {
		t.Logf("顔のない画像で期待どおりエラーを検出: %v", err)
	}
}
