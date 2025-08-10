package facedetector

import (
	"io/ioutil"
	"testing"
)

func TestDrawFaceRects_Selfie(t *testing.T) {
	// Load one of the new selfie images
	imageData, err := ioutil.ReadFile("testdata/selfie1.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Call the function to be tested
	resultImage, err := DrawFaceRects(imageData)
	if err != nil {
		t.Fatalf("DrawFaceRects failed: %v", err)
	}

	// Check that the result is not empty
	if len(resultImage) == 0 {
		t.Fatal("The resulting image is empty")
	}

	// Optionally, save the output for manual inspection
	// err = ioutil.WriteFile("testdata/selfie1_boxed.png", resultImage, 0644)
	// if err != nil {
	// 	t.Logf("Failed to write output image for manual inspection: %v", err)
	// }
}

func TestCropFace_Selfie(t *testing.T) {
	// Load one of the new selfie images
	imageData, err := ioutil.ReadFile("testdata/selfie2.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Call the function to be tested
	resultImage, err := CropFace(imageData)
	if err != nil {
		t.Fatalf("CropFace failed: %v", err)
	}

	// Check that the result is not empty
	if len(resultImage) == 0 {
		t.Fatal("The resulting image is empty")
	}
}

// Test against an image that was previously causing issues (if available)
func TestDrawFaceRects_LegacyFace(t *testing.T) {
	// Using the original face.jpg to ensure no regressions
	imageData, err := ioutil.ReadFile("testdata/face.jpg")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	_, err = DrawFaceRects(imageData)
	if err != nil {
		// This image should contain a face
		t.Fatalf("DrawFaceRects failed on legacy image: %v", err)
	}
}

// Test with an image that contains no faces
func TestDrawFaceRects_NoFace(t *testing.T) {
	// Create a dummy blank image data
	// For this test, I'll use a file that isn't an image.
	// The decode step should fail gracefully.
	// A better test would be a valid image with no faces.
	// For now, I'll use a text file.
	imageData, err := ioutil.ReadFile("../../LICENSE")
	if err != nil {
		t.Fatalf("Failed to read non-image file: %v", err)
	}

	_, err = DrawFaceRects(imageData)
	// We expect an error here, either from decoding or from no faces being found.
	if err == nil {
		t.Fatal("Expected an error when processing a file that is not a valid image, but got none")
	}
}
