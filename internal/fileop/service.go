package fileop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// contentLookup returns the CF_UNICODETEXT content for a given entry id.
type contentLookup func(id int64) (string, error)

// Service handles opening clipboard entries in external applications.
type Service struct {
	contentFn     contentLookup
	openFileMgrFn func(path string, selectFile bool) error
}

// Option configures the Service.
type Option func(*Service)

// WithOpenFileManager sets the function that opens a file or folder in Explorer.
func WithOpenFileManager(fn func(path string, selectFile bool) error) Option {
	return func(s *Service) { s.openFileMgrFn = fn }
}

// NewService creates a file operation service.
func NewService(lookup contentLookup, opts ...Option) *Service {
	s := &Service{contentFn: lookup}
	for _, o := range opts {
		o(s)
	}
	return s
}

// --- frontend-accessible methods ---

// OpenInEditor writes the entry's text content to a temp file and opens it
// in VS Code (if available) or the system default .txt handler.
func (s *Service) OpenInEditor(id int64) error {
	content, err := s.contentFn(id)
	if err != nil {
		return fmt.Errorf("get entry content: %w", err)
	}
	if content == "" {
		return fmt.Errorf("entry %d has no text content", id)
	}

	tmpDir := filepath.Join(os.Getenv("TEMP"), "jPaste")
	os.MkdirAll(tmpDir, 0700)

	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("jpaste_%d.txt", time.Now().UnixNano()))
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Try VS Code first.
	if vscode := findVSCode(); vscode != "" {
		cmd := exec.Command(vscode, tmpPath)
		if err := cmd.Start(); err == nil {
			// Clean up after VS Code closes.
			go func() {
				cmd.Wait()
				os.Remove(tmpPath)
			}()
			return nil
		}
	}
	// Fall back to system default .txt handler.
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", tmpPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	go func() {
		time.Sleep(30 * time.Second)
		os.Remove(tmpPath)
	}()
	return nil
}

// OpenInExplorer opens the folder containing the entry in Explorer.
// If the entry content is a valid path format, it opens that path directly.
func (s *Service) OpenInExplorer(id int64, selectFile bool) error {
	if s.openFileMgrFn == nil {
		return fmt.Errorf("file manager not wired")
	}
	content, err := s.contentFn(id)
	if err != nil {
		return fmt.Errorf("get entry content: %w", err)
	}
	if content == "" {
		return fmt.Errorf("entry %d has no content", id)
	}
	return s.openFileMgrFn(content, selectFile)
}

func findVSCode() string {
	paths := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Microsoft VS Code", "Code.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft VS Code", "Code.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft VS Code", "Code.exe"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
