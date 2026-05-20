package facedetector

import (
	"os"
	"testing"
)

// ============================================================================
// 基本的な顔検出テスト
// ============================================================================

func TestDrawFaceRects_Selfie(t *testing.T) {
	imageData, err := os.ReadFile("testdata/selfie1.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	resultImage, err := DrawFaceRects(imageData)
	if err != nil {
		t.Fatalf("DrawFaceRects failed: %v", err)
	}

	if len(resultImage) == 0 {
		t.Fatal("The resulting image is empty")
	}
}

func TestCropFace_Selfie(t *testing.T) {
	imageData, err := os.ReadFile("testdata/selfie2.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	resultImage, err := CropFace(imageData)
	if err != nil {
		t.Fatalf("CropFace failed: %v", err)
	}

	if len(resultImage) == 0 {
		t.Fatal("The resulting image is empty")
	}
}

func TestDrawFaceRects_LegacyFace(t *testing.T) {
	imageData, err := os.ReadFile("testdata/face.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	_, err = DrawFaceRects(imageData)
	if err != nil {
		t.Fatalf("DrawFaceRects failed on legacy image: %v", err)
	}
}

// ============================================================================
// 異常系テスト
// ============================================================================

func TestDrawFaceRects_NoFace(t *testing.T) {
	imageData, err := os.ReadFile("../../LICENSE")
	if err != nil {
		t.Fatalf("Failed to read non-image file: %v", err)
	}

	_, err = DrawFaceRects(imageData)
	if err == nil {
		t.Fatal("Expected an error when processing a file that is not a valid image, but got none")
	}
}

func TestDrawFaceRects_EmptyData(t *testing.T) {
	_, err := DrawFaceRects([]byte{})
	if err == nil {
		t.Fatal("Expected an error for empty data, but got none")
	}
}

func TestCalculateSharpness_EmptyData(t *testing.T) {
	_, err := CalculateSharpness([]byte{})
	if err == nil {
		t.Fatal("Expected an error for empty data, but got none")
	}
}

// ============================================================================
// 鮮明度計算テスト
// ============================================================================

func TestCalculateSharpness_SharpImage(t *testing.T) {
	imageData, err := os.ReadFile("testdata/selfie1.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	sharpness, err := CalculateSharpness(imageData)
	if err != nil {
		t.Fatalf("CalculateSharpness failed: %v", err)
	}

	if sharpness <= 0 {
		t.Fatalf("Expected positive sharpness score, got %f", sharpness)
	}
	t.Logf("Sharpness score for sharp image: %f", sharpness)
}

func TestCalculateFaceSharpness_SharpFace(t *testing.T) {
	imageData, err := os.ReadFile("testdata/face.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	sharpness, err := CalculateFaceSharpness(imageData)
	if err != nil {
		t.Fatalf("CalculateFaceSharpness failed: %v", err)
	}

	if sharpness <= 0 {
		t.Fatalf("Expected positive sharpness score, got %f", sharpness)
	}
	t.Logf("Face sharpness score for sharp image: %f", sharpness)
}

// ============================================================================
// 複数画像テスト（検出の安定性確認）
// ============================================================================

func TestDrawFaceRects_MultipleSelfies(t *testing.T) {
	testFiles := []string{
		"testdata/selfie1.jpg",
		"testdata/selfie2.jpg",
		"testdata/selfie3.jpg",
		"testdata/selfie4.jpg",
		"testdata/selfie5.jpg",
	}

	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			imageData, err := os.ReadFile(file)
			if err != nil {
				t.Skipf("Test image not available: %s", file)
				return
			}

			resultImage, err := DrawFaceRects(imageData)
			if err != nil {
				t.Errorf("DrawFaceRects failed for %s: %v", file, err)
				return
			}

			if len(resultImage) == 0 {
				t.Errorf("The resulting image is empty for %s", file)
			}
		})
	}
}

// ============================================================================
// ブレ画像テスト（検出の堅牢性確認）
// ============================================================================

func TestDrawFaceRects_BlurryImages(t *testing.T) {
	testFiles := []string{
		"testdata/selfie_blurry_1.jpg",
		"testdata/selfie_blurry_2.jpg",
		"testdata/selfie_blurry_3.jpg",
	}

	detected := 0
	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			imageData, err := os.ReadFile(file)
			if err != nil {
				t.Skipf("Test image not available: %s", file)
				return
			}

			_, err = DrawFaceRects(imageData)
			if err == nil {
				detected++
				t.Logf("Face detected in blurry image: %s", file)
			} else {
				t.Logf("No face detected in blurry image (may be expected): %s: %v", file, err)
			}
		})
	}
	t.Logf("Detected faces in %d/%d blurry images", detected, len(testFiles))
}

// ============================================================================
// 鮮明度比較テスト
// ============================================================================

func TestSharpnessComparison_FaceImages(t *testing.T) {
	sharpData, err := os.ReadFile("testdata/face.jpg")
	if err != nil {
		t.Fatalf("Failed to read sharp face image: %v", err)
	}

	blurredData, err := os.ReadFile("testdata/face_blurred.jpg")
	if err != nil {
		t.Fatalf("Failed to read blurred face image: %v", err)
	}

	sharpScore, err := CalculateSharpness(sharpData)
	if err != nil {
		t.Fatalf("CalculateSharpness failed for sharp image: %v", err)
	}

	blurredScore, err := CalculateSharpness(blurredData)
	if err != nil {
		t.Fatalf("CalculateSharpness failed for blurred image: %v", err)
	}

	t.Logf("Sharp image score: %f, Blurred image score: %f", sharpScore, blurredScore)

	if sharpScore <= blurredScore {
		t.Errorf("Expected sharp image (%f) to have higher sharpness than blurred image (%f)",
			sharpScore, blurredScore)
	}
}

// ============================================================================
// 内部関数のユニットテスト
// ============================================================================

func TestHasValidAspectRatio(t *testing.T) {
	tests := []struct {
		name   string
		rect   image.Rectangle
		expect bool
	}{
		{"square face", image.Rect(0, 0, 100, 100), true},
		{"tall face", image.Rect(0, 0, 70, 100), true},
		{"wide face", image.Rect(0, 0, 100, 70), true},
		{"too narrow", image.Rect(0, 0, 30, 100), false},
		{"too wide", image.Rect(0, 0, 200, 50), false},
		{"zero width", image.Rect(0, 0, 0, 100), false},
		{"zero height", image.Rect(0, 0, 100, 0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasValidAspectRatio(tt.rect)
			if result != tt.expect {
				t.Errorf("hasValidAspectRatio(%v) = %v, want %v", tt.rect, result, tt.expect)
			}
		})
	}
}

func TestHasMinimumSize(t *testing.T) {
	tests := []struct {
		name   string
		rect   image.Rectangle
		expect bool
	}{
		{"valid size", image.Rect(0, 0, 50, 50), true},
		{"minimum size", image.Rect(0, 0, 20, 20), true},
		{"too small", image.Rect(0, 0, 10, 10), false},
		{"too small width", image.Rect(0, 0, 10, 50), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasMinimumSize(tt.rect)
			if result != tt.expect {
				t.Errorf("hasMinimumSize(%v) = %v, want %v", tt.rect, result, tt.expect)
			}
		})
	}
}

func TestCalculateIoU(t *testing.T) {
	tests := []struct {
		name string
		a, b image.Rectangle
		min  float64
		max  float64
	}{
		{"identical", image.Rect(0, 0, 100, 100), image.Rect(0, 0, 100, 100), 1.0, 1.0},
		{"no overlap", image.Rect(0, 0, 50, 50), image.Rect(100, 100, 150, 150), 0.0, 0.0},
		{"partial overlap", image.Rect(0, 0, 100, 100), image.Rect(50, 50, 150, 150), 0.1, 0.5},
		{"contained", image.Rect(0, 0, 100, 100), image.Rect(25, 25, 75, 75), 0.2, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iou := calculateIoU(tt.a, tt.b)
			if iou < tt.min || iou > tt.max {
				t.Errorf("calculateIoU(%v, %v) = %f, want in range [%f, %f]",
					tt.a, tt.b, iou, tt.min, tt.max)
			}
		})
	}
}

func TestNonMaxSuppression(t *testing.T) {
	// 重複する検出を渡して、NMSで1つに絞られることを確認
	detections := []detectionWithConfidence{
		{rect: image.Rect(10, 10, 110, 110), confidence: 0.9, source: "dnn"},
		{rect: image.Rect(15, 15, 115, 115), confidence: 0.8, source: "dnn"},
		{rect: image.Rect(200, 200, 300, 300), confidence: 0.7, source: "dnn"},
	}

	result := nonMaxSuppression(detections, 0.3)

	// 最初の2つは重複しているので1つに、3つ目は独立なので残る → 合計2つ
	if len(result) != 2 {
		t.Errorf("Expected 2 detections after NMS, got %d", len(result))
	}

	// 最高信頼度の検出が残っていることを確認
	if result[0].confidence != 0.9 {
		t.Errorf("Expected highest confidence detection (0.9), got %f", result[0].confidence)
	}
}

func TestClipRect(t *testing.T) {
	bounds := image.Rect(0, 0, 640, 480)

	tests := []struct {
		name   string
		rect   image.Rectangle
		expect image.Rectangle
	}{
		{"within bounds", image.Rect(10, 10, 100, 100), image.Rect(10, 10, 100, 100)},
		{"overflow right", image.Rect(600, 10, 700, 100), image.Rect(600, 10, 640, 100)},
		{"overflow bottom", image.Rect(10, 400, 100, 500), image.Rect(10, 400, 100, 480)},
		{"negative coords", image.Rect(-10, -10, 100, 100), image.Rect(0, 0, 100, 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clipRect(tt.rect, bounds)
			if result != tt.expect {
				t.Errorf("clipRect(%v, %v) = %v, want %v", tt.rect, bounds, result, tt.expect)
			}
		})
	}
}

func TestAddMargin(t *testing.T) {
	rect := image.Rect(100, 100, 200, 200)
	result := addMargin(rect, 0.1)

	// 100x100の矩形に10%マージン → 10px拡張
	expected := image.Rect(90, 90, 210, 210)
	if result != expected {
		t.Errorf("addMargin(%v, 0.1) = %v, want %v", rect, result, expected)
	}
}
