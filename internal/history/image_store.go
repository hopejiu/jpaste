package history

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

// ImageStorer abstracts image persistence for clipboard entries.
// The production adapter is ImageStore (filesystem); tests use an in-memory fake.
type ImageStorer interface {
	Save(raw []byte, today string) (string, error)
	ReadDIB(pngRelPath string) ([]byte, error)
	ReadImage(relPath string) ([]byte, error)
	TotalImageBytes() (int64, error)
	DeleteByEntry(paths []string)
	DeleteByDate(dateFolder string)
	CleanEmptyDirs()
	AppDataPath() string
}

// ImageStore saves clipboard image payloads to disk and cleans them up.
type ImageStore struct {
	basePath string
}

// Ensure ImageStore implements ImageStorer.
var _ ImageStorer = (*ImageStore)(nil)

// NewImageStore creates an ImageStore rooted at %APPDATA%/jPaste/images.
func NewImageStore(appData string) *ImageStore {
	return &ImageStore{basePath: filepath.Join(appData, "images")}
}

// Save converts DIB/DIBV5 raw bytes to PNG and saves to images/YYYY-MM-DD/uuid.png.
// Also saves the raw DIB bytes as uuid.dib for clipboard restoration (paste).
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

	id := uuid.New().String()

	// Save PNG for display/preview.
	pngPath := filepath.Join(dir, id+".png")
	f, err := os.Create(pngPath)
	if err != nil {
		return "", fmt.Errorf("create PNG: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		os.Remove(pngPath)
		return "", fmt.Errorf("encode PNG: %w", err)
	}

	// Save raw DIB for clipboard restoration (paste).
	dibPath := filepath.Join(dir, id+".dib")
	if err := os.WriteFile(dibPath, raw, 0600); err != nil {
		// Non-fatal: paste won't work for this image but preview will.
		// Still clean up on failure.
		os.Remove(pngPath)
		return "", fmt.Errorf("write DIB: %w", err)
	}

	// Return path relative to %APPDATA%/jPaste (using PNG path as canonical ref).
	rel := filepath.Join("images", today, id+".png")
	return rel, nil
}

// ReadDIB reads the raw DIB bytes stored for clipboard restoration.
// Given a PNG relative path, returns the corresponding DIB data.
func (s *ImageStore) ReadDIB(pngRelPath string) ([]byte, error) {
	// Convert "images/YYYY-MM-DD/uuid.png" → "images/YYYY-MM-DD/uuid.dib"
	dibRel := pngRelPath[:len(pngRelPath)-4] + ".dib"
	full := filepath.Join(filepath.Dir(s.basePath), dibRel)
	return os.ReadFile(full)
}

// TotalImageBytes returns the total disk usage of all saved image files.
func (s *ImageStore) TotalImageBytes() (int64, error) {
	var total int64
	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// DeleteByEntry removes all image files referenced by an entry's formats.
func (s *ImageStore) DeleteByEntry(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		base := filepath.Dir(s.basePath)
		full := filepath.Join(base, p)
		os.Remove(full)
		// Also remove the corresponding .dib file.
		dibPath := full[:len(full)-4] + ".dib"
		os.Remove(dibPath)
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
