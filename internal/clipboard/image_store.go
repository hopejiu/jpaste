package clipboard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	_ "golang.org/x/image/bmp"
)

// ImageStore saves clipboard image payloads to disk and cleans them up.
type ImageStore struct {
	basePath string
}

// NewImageStore creates an ImageStore rooted at %APPDATA%/jPaste/images.
func NewImageStore(appData string) *ImageStore {
	return &ImageStore{basePath: filepath.Join(appData, "images")}
}

// Save converts DIB/DIBV5 raw bytes to PNG and saves to images/YYYY-MM-DD/uuid.png.
// Returns the relative file path (e.g. "images/2026-06-02/abc123.png").
func (s *ImageStore) Save(raw []byte, today string) (string, error) {
	if len(raw) < 40 {
		return "", fmt.Errorf("DIB too small: %d bytes", len(raw))
	}

	// DIB = BITMAPINFOHEADER + [color table] + pixel data
	// Prepend BITMAPFILEHEADER to make a valid BMP.
	bmpData := prependBMPHeader(raw)

	img, _, err := image.Decode(bytes.NewReader(bmpData))
	if err != nil {
		return "", fmt.Errorf("decode BMP: %w", err)
	}

	dir := filepath.Join(s.basePath, today)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	filename := uuid.New().String() + ".png"
	fullPath := filepath.Join(dir, filename)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("encode PNG: %w", err)
	}

	// Return path relative to %APPDATA%/jPaste
	rel := filepath.Join("images", today, filename)
	return rel, nil
}

// DeleteByEntry removes all image files referenced by an entry's formats.
func (s *ImageStore) DeleteByEntry(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		full := filepath.Join(filepath.Dir(s.basePath), p)
		os.Remove(full)
	}
}

// DeleteByDate removes all images under the given date folder.
func (s *ImageStore) DeleteByDate(dateFolder string) {
	dir := filepath.Join(s.basePath, dateFolder)
	os.RemoveAll(dir)
}

// AppDataPath returns the parent of the images directory (i.e. %APPDATA%/jPaste).
func (s *ImageStore) AppDataPath() string {
	return filepath.Dir(s.basePath)
}

// ReadImage reads an image file given its relative path (e.g. "images/2026-06-02/abc.png").
func (s *ImageStore) ReadImage(relPath string) ([]byte, error) {
	full := filepath.Join(filepath.Dir(s.basePath), relPath)
	return os.ReadFile(full)
}

// CleanEmptyDirs removes empty date folders.
func (s *ImageStore) CleanEmptyDirs() {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(s.basePath, e.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		if len(files) == 0 {
			os.Remove(dir)
		}
	}
}

// prependBMPHeader builds a valid BMP from a DIB by prefixing BITMAPFILEHEADER.
func prependBMPHeader(dib []byte) []byte {
	// BITMAPFILEHEADER: 14 bytes
	//   bfType      uint16 = 'BM' (0x4D42)
	//   bfSize      uint32 = total file size
	//   bfReserved1 uint16 = 0
	//   bfReserved2 uint16 = 0
	//   bfOffBits   uint32 = 14 + 40 + colorTableSize
	headerSize := binary.LittleEndian.Uint32(dib[0:4]) // biSize
	bitCount := binary.LittleEndian.Uint16(dib[14:16])
	clrUsed := binary.LittleEndian.Uint32(dib[32:36])

	var colorTableSize uint32
	if bitCount <= 8 {
		if clrUsed == 0 {
			colorTableSize = uint32(1<<bitCount) * 4
		} else {
			colorTableSize = clrUsed * 4
		}
	}

	offset := uint32(14 + headerSize + colorTableSize)
	fileSize := uint32(14 + len(dib))

	buf := make([]byte, 14+len(dib))
	binary.LittleEndian.PutUint16(buf[0:2], 0x4D42) // 'BM'
	binary.LittleEndian.PutUint32(buf[2:6], fileSize)
	// Reserved: [6:10] zero
	binary.LittleEndian.PutUint32(buf[10:14], offset)
	copy(buf[14:], dib)

	return buf
}


