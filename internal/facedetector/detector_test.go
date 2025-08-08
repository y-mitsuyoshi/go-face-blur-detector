package facedetector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateSharpness(t *testing.T) {
	// Load test images
	sharpImgData, err := os.ReadFile(filepath.Join("testdata", "test.png"))
	if err != nil {
		t.Fatalf("failed to read sharp image: %v", err)
	}

	blurredImgData, err := os.ReadFile(filepath.Join("testdata", "test_blurred.png"))
	if err != nil {
		t.Fatalf("failed to read blurred image: %v", err)
	}

	// Define test cases
	testCases := []struct {
		name          string
		imageData     []byte
		expectErr     bool
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
