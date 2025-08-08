package facedetector

import (
	"os"
	"testing"
)

func TestCalculateSharpness(t *testing.T) {
	// Read the image files
	sharpImgData, err := os.ReadFile("../../test_large.png")
	if err != nil {
		t.Fatalf("failed to read sharp image: %v", err)
	}

	blurredImgData, err := os.ReadFile("../../test_blurred.png")
	if err != nil {
		t.Fatalf("failed to read blurred image: %v", err)
	}

	// Calculate sharpness for both images first to establish a baseline
	sharpness, err := CalculateSharpness(sharpImgData)
	if err != nil {
		t.Fatalf("CalculateSharpness for sharp image failed: %v", err)
	}

	blurredSharpness, err := CalculateSharpness(blurredImgData)
	if err != nil {
		t.Fatalf("CalculateSharpness for blurred image failed: %v", err)
	}

	// Basic assertion that sharp is sharper than blurred
	if sharpness <= blurredSharpness {
		t.Errorf("expected sharp image to have higher sharpness than blurred image, got sharp: %f, blurred: %f", sharpness, blurredSharpness)
	}

	// Now, create table-driven tests for more specific checks
	testCases := []struct {
		name          string
		input         []byte
		expectErr     bool
		checkSharpness func(t *testing.T, s float64)
	}{
		{
			name:      "sharp image",
			input:     sharpImgData,
			expectErr: false,
			checkSharpness: func(t *testing.T, s float64) {
				if s <= blurredSharpness {
					t.Errorf("sharpness of sharp image (%f) should be greater than blurred image (%f)", s, blurredSharpness)
				}
			},
		},
		{
			name:      "blurred image",
			input:     blurredImgData,
			expectErr: false,
			checkSharpness: func(t *testing.T, s float64) {
				if s >= sharpness {
					t.Errorf("sharpness of blurred image (%f) should be less than sharp image (%f)", s, sharpness)
				}
			},
		},
		{
			name:      "empty data",
			input:     []byte{},
			expectErr: true,
			checkSharpness: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sharpness, err := CalculateSharpness(tc.input)

			if tc.expectErr {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
				if tc.checkSharpness != nil {
					tc.checkSharpness(t, sharpness)
				}
			}
		})
	}
}
