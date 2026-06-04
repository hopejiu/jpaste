// rasterize-logo — Pure Go SVG → PNG + multi-res ICO rasterizer.
//
//	go run ./scripts/rasterize-logo/
//
// Prereq: go get github.com/srwiley/oksvg github.com/srwiley/rasterx
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const (
	svgPath     = "jpaste-logo.svg"
	pngOut      = "jpaste-logo.png"
	pasteOut    = "paste.png"
	icoOut      = "build/windows/icon.ico"
	renderHiDPI = 1024
)

var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Read SVG bytes once
	svgData, err := os.ReadFile(svgPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", svgPath, err)
	}

	// 2. Render hi-res PNG
	fmt.Printf("=== Rasterize %d×%d ===\n", renderHiDPI, renderHiDPI)
	hiRes, err := renderSVG(svgData, renderHiDPI, renderHiDPI)
	if err != nil {
		return fmt.Errorf("render hi-res: %w", err)
	}

	if err := writePNG(pngOut, hiRes); err != nil {
		return err
	}
	fmt.Printf("  Saved: %s\n", pngOut)
	if err := writePNG(pasteOut, hiRes); err != nil {
		return err
	}
	fmt.Printf("  Saved: %s (tray)\n", pasteOut)

	// 3. Render each icon size & pack ICO (fresh parse per size)
	fmt.Printf("\n=== Generate ICO (%v) ===\n", icoSizes)
	var frames []framePNG
	for _, sz := range icoSizes {
		fmt.Printf("  Rendering %d×%d...\n", sz, sz)
		img, err := renderSVG(svgData, sz, sz)
		if err != nil {
			return fmt.Errorf("render %d: %w", sz, err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("encode %d PNG: %w", sz, err)
		}
		frames = append(frames, framePNG{size: sz, data: buf.Bytes()})
	}

	if err := os.MkdirAll(filepath.Dir(icoOut), 0755); err != nil {
		return err
	}
	if err := writeICO(icoOut, frames); err != nil {
		return err
	}
	fmt.Printf("  Saved: %s\n", icoOut)

	fmt.Println("\n=== Done ===")
	return nil
}

// ---------------------------------------------------------------------------
// SVG rendering
// ---------------------------------------------------------------------------

// gradientFallback maps gradient id → solid fallback color.
// oksvg has limited gradient support; this ensures structural shapes render.
var gradientFallback = map[string]string{
	"board":    "#7C3AED",
	"clip":     "#8B5CF6",
	"paper":    "#FFFFFF",
	"ribbon":   "#F472B6",
	"hairGrad": "#22D3EE",
}

// reGradientRef matches fill="url(#name)" or stroke="url(#name)" references.
var reGradientRef = regexp.MustCompile(`(fill|stroke)="url\(#([^)]+)\)"`)

func flattenGradients(svg []byte) []byte {
	return reGradientRef.ReplaceAllFunc(svg, func(m []byte) []byte {
		parts := reGradientRef.FindSubmatch(m)
		attr, name := string(parts[1]), string(parts[2])
		if solid, ok := gradientFallback[name]; ok {
			return []byte(fmt.Sprintf(`%s="%s"`, attr, solid))
		}
		return m
	})
}

func renderSVG(svgData []byte, w, h int) (*image.RGBA, error) {
	// Replace gradient refs with solid colors (oksvg gradient support is limited)
	flat := flattenGradients(svgData)

	icon, err := oksvg.ReadIconStream(bytes.NewReader(flat))
	if err != nil {
		return nil, fmt.Errorf("parse SVG: %w", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Transparent background — no fill needed, image.NewRGBA zero-initializes (all 0 = transparent black)

	// Map viewBox to full output dimensions
	icon.SetTarget(0, 0, float64(w), float64(h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	raster := rasterx.NewDasher(w, h, scanner)
	icon.Draw(raster, 1.0)
	return img, nil
}

// ---------------------------------------------------------------------------
// PNG output
// ---------------------------------------------------------------------------

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ---------------------------------------------------------------------------
// ICO file format
// ---------------------------------------------------------------------------

type framePNG struct {
	size int
	data []byte
}

// ICO header: 6 bytes
// Directory entry: 16 bytes per frame
func writeICO(path string, frames []framePNG) error {
	// Gather PNG data
	var dirEntries []bytes.Buffer
	var imageBlocks []bytes.Buffer
	dataOffset := 6 + 16*len(frames) // header + directory

	for _, f := range frames {
		// Directory entry
		var de bytes.Buffer
		b := make([]byte, 16)
		sz := byte(f.size)
		if f.size >= 256 {
			sz = 0
		}
		b[0] = sz // width
		b[1] = sz // height
		b[2] = 0  // palette
		b[3] = 0  // reserved
		binary.LittleEndian.PutUint16(b[4:], 1)              // color planes
		binary.LittleEndian.PutUint16(b[6:], 32)             // bpp
		binary.LittleEndian.PutUint32(b[8:], uint32(len(f.data))) // size
		binary.LittleEndian.PutUint32(b[12:], uint32(dataOffset)) // offset
		de.Write(b)
		dirEntries = append(dirEntries, de)

		// Image block
		var ib bytes.Buffer
		ib.Write(f.data)
		imageBlocks = append(imageBlocks, ib)

		dataOffset += len(f.data)
	}

	// Write file
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// ICO header
	header := make([]byte, 6)
	binary.LittleEndian.PutUint16(header[0:], 0)               // reserved
	binary.LittleEndian.PutUint16(header[2:], 1)               // type=ICO
	binary.LittleEndian.PutUint16(header[4:], uint16(len(frames))) // count
	out.Write(header)

	// Directory
	for _, d := range dirEntries {
		out.Write(d.Bytes())
	}
	// Images
	for _, ib := range imageBlocks {
		out.Write(ib.Bytes())
	}

	return nil
}
