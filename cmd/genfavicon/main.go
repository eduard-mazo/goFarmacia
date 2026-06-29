// Command genfavicon generates favicon PNG files from the store logo.
//
// It extracts the left portion of the logo (the crescent moon + pharmacy
// cross icon) and produces square PNGs at multiple sizes suitable for
// browser favicons and app icons.
//
// Usage: go run ./cmd/genfavicon
package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func main() {
	src, err := loadLogo("backend/assets/logo.jpg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading logo: %v\n", err)
		os.Exit(1)
	}

	// The logo is 598×249. The crescent+cross icon occupies roughly the
	// left 190px. Crop that region and center it in a square canvas.
	bounds := src.Bounds()
	h := bounds.Dy()
	iconW := h * 78 / 100 // ~194px — captures the symbol, not the text
	iconCrop := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+iconW, bounds.Min.Y+h)
	cropped := centerInSquare(src, iconCrop)

	outDir := filepath.Join("frontend", "public")

	sizes := []int{16, 32, 180, 192, 512}
	for _, sz := range sizes {
		name := fmt.Sprintf("favicon-%d.png", sz)
		if sz == 180 {
			name = "apple-touch-icon.png"
		}
		if sz == 192 {
			name = "icon-192.png"
		}
		if sz == 512 {
			name = "icon-512.png"
		}
		path := filepath.Join(outDir, name)
		if err := saveScaled(cropped, sz, path); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("  ✓ %s (%dx%d)\n", path, sz, sz)
	}

	fmt.Println("Done. Update frontend/index.html with favicon links.")
}

func loadLogo(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return jpeg.Decode(f)
}

func centerInSquare(src image.Image, rect image.Rectangle) image.Image {
	side := rect.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	for i := 0; i < len(dst.Pix); i += 4 {
		dst.Pix[i] = 0xFF
		dst.Pix[i+1] = 0xFF
		dst.Pix[i+2] = 0xFF
		dst.Pix[i+3] = 0xFF
	}
	offsetX := (side - rect.Dx()) / 2
	draw.Copy(dst, image.Pt(offsetX, 0), src, rect, draw.Over, nil)
	return dst
}

func saveScaled(src image.Image, size int, path string) error {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	// Fill white background.
	for i := 0; i < len(dst.Pix); i += 4 {
		dst.Pix[i] = 0xFF
		dst.Pix[i+1] = 0xFF
		dst.Pix[i+2] = 0xFF
		dst.Pix[i+3] = 0xFF
	}
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, dst)
}
