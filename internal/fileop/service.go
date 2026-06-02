package fileop

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Service handles opening clipboard entries in external applications.
type Service struct {
	getContent     func(id int64) (string, error)
	openFileManager func(path string, selectFile bool) error
}

// ContentProvider is a function that retrieves clipboard content by ID.
type ContentProvider func(id int64) (string, error)

// OpenFileManagerFunc opens a path in the file manager.
// If selectFile is true, the file is selected/highlighted in its parent folder.
type OpenFileManagerFunc func(path string, selectFile bool) error

// Option configures the FileService.
type Option func(*Service)

// WithOpenFileManager sets the platform-specific file-manager-open function.
func WithOpenFileManager(fn OpenFileManagerFunc) Option {
	return func(s *Service) { s.openFileManager = fn }
}

// NewService creates a FileService.
func NewService(provider ContentProvider, opts ...Option) *Service {
	s := &Service{getContent: provider}
	for _, o := range opts {
		o(s)
	}
	return s
}

// OpenInExplorer opens the folder path contained in a clipboard entry in Windows Explorer.
func (s *Service) OpenInExplorer(id int64) error {
	if s.openFileManager == nil {
		return fmt.Errorf("open folder: not configured")
	}
	content, err := s.getContent(id)
	if err != nil {
		return fmt.Errorf("get content for id %d: %w", id, err)
	}
	return s.openFileManager(content, false)
}

// OpenFileLocation opens the folder containing the file path and selects the file.
func (s *Service) OpenFileLocation(id int64) error {
	if s.openFileManager == nil {
		return fmt.Errorf("open file location: not configured")
	}
	content, err := s.getContent(id)
	if err != nil {
		return fmt.Errorf("get content for id %d: %w", id, err)
	}
	return s.openFileManager(content, true)
}

// OpenInEditor creates a temp .txt file and opens it in the preferred editor.
// Tries VS Code first, then falls back to the system default .txt handler.
func (s *Service) OpenInEditor(id int64) error {
	content, err := s.getContent(id)
	if err != nil {
		return fmt.Errorf("get content for id %d: %w", id, err)
	}

	tmpFile, err := os.CreateTemp("", "jpaste-*.txt")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	// Try VS Code / Cursor first.
	editors := []string{"code", "cursor", "code-insiders"}
	var opened bool
	for _, ed := range editors {
		if p, err := exec.LookPath(ed); err == nil {
			cmd := exec.Command(p, "--wait", tmpPath)
			cmd.Stdout = nil
			cmd.Stderr = nil
			if err := cmd.Start(); err == nil {
				opened = true
				// Clean up temp file after editor closes.
				go func() {
					cmd.Wait()
					os.Remove(tmpPath)
				}()
				break
			}
		}
	}

	if !opened {
		// Fall back to system default handler.
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", tmpPath)
		if err := cmd.Start(); err != nil {
			// Last resort: try explorer.
			cmd = exec.Command("explorer", tmpPath)
			if err := cmd.Start(); err != nil {
				os.Remove(tmpPath)
				return fmt.Errorf("open file: %w", err)
			}
		}
		// Clean up after a delay (system handler might not block).
		go func() {
			time.Sleep(30 * time.Second)
			os.Remove(tmpPath)
		}()
	}

	return nil
}
