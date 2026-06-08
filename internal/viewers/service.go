// Package viewers provides service types for secondary Wails windows
// (JSON viewer, image viewer, curl debugger, WebSocket debugger).
package viewers

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	applog "jpaste/internal/log"
)

// CreateWindowFunc is a callback that opens a new Wails window at the given path.
type CreateWindowFunc func(path string)

// --- JSON Viewer -----------------------------------------------------------

// JSONViewerService manages JSON viewer windows.
type JSONViewerService struct {
	createWin CreateWindowFunc
}

// NewJSONViewerService creates a new JSON viewer service.
func NewJSONViewerService(createWin CreateWindowFunc) *JSONViewerService {
	return &JSONViewerService{createWin: createWin}
}

// OpenJsonViewer opens a new Wails window at /json-view?id=<entryID>.
func (s *JSONViewerService) OpenJsonViewer(id int64) {
	path := "/json-view?id=" + fmt.Sprint(id)
	applog.Info("jsonviewer: open", "id", id, "path", path)
	s.createWin(path)
}

// --- Image Viewer ----------------------------------------------------------

// ImageViewerService manages image viewer windows.
type ImageViewerService struct {
	createWin CreateWindowFunc
}

// NewImageViewerService creates a new image viewer service.
func NewImageViewerService(createWin CreateWindowFunc) *ImageViewerService {
	return &ImageViewerService{createWin: createWin}
}

// OpenImageViewer opens a new Wails window for viewing an image entry.
func (s *ImageViewerService) OpenImageViewer(id int64, tagMask int, search string) {
	path := fmt.Sprintf("/image-view?id=%d&tag=%d&search=%s", id, tagMask, search)
	applog.Info("imageviewer: open", "id", id, "tag", tagMask, "search", search)
	s.createWin(path)
}

// --- Curl Viewer -----------------------------------------------------------

// CurlViewerService manages curl viewer windows.
type CurlViewerService struct {
	createWin CreateWindowFunc
}

// NewCurlViewerService creates a new curl viewer service.
func NewCurlViewerService(createWin CreateWindowFunc) *CurlViewerService {
	return &CurlViewerService{createWin: createWin}
}

// OpenCurlViewer opens a new Wails window at /curl-view?id=<entryID>.
func (s *CurlViewerService) OpenCurlViewer(id int64) {
	path := "/curl-view?id=" + fmt.Sprint(id)
	applog.Info("curlviewer: open", "id", id, "path", path)
	s.createWin(path)
}

// CurlRequest represents a parsed HTTP request to be executed.
type CurlRequest struct {
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	FollowRedirects bool              `json:"followRedirects"`
	Timeout         int               `json:"timeout"`
}

// CurlResponse represents the HTTP response returned to the frontend.
type CurlResponse struct {
	StatusCode int               `json:"statusCode"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	DurationMs int64             `json:"durationMs"`
}

// SendCurlRequest executes the given HTTP request and returns the response.
func (s *CurlViewerService) SendCurlRequest(req CurlRequest) (CurlResponse, error) {
	applog.Info("curlviewer: send request", "method", req.Method, "url", req.URL,
		"followRedirect", req.FollowRedirects, "timeout", req.Timeout)

	bodyReader := io.NopCloser(strings.NewReader(req.Body))
	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		applog.Error("curlviewer: build request failed", "error", err)
		return CurlResponse{}, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if req.FollowRedirects {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			}
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		applog.Error("curlviewer: request failed", "error", err)
		return CurlResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("curlviewer: read body failed", "error", err)
		return CurlResponse{}, err
	}

	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	// resp.Status is like "400 Bad Request" — strip the numeric prefix
	// so the frontend can render statusCode and statusText separately.
	statusText := resp.Status
	if idx := strings.Index(resp.Status, " "); idx >= 0 {
		statusText = resp.Status[idx+1:]
	}

	result := CurlResponse{
		StatusCode: resp.StatusCode,
		StatusText: statusText,
		Headers:    respHeaders,
		Body:       string(bodyBytes),
		DurationMs: duration,
	}

	applog.Info("curlviewer: response received",
		"status", result.StatusCode, "bodyLen", len(result.Body), "durationMs", result.DurationMs)

	return result, nil
}

// --- WebSocket Viewer ------------------------------------------------------

// WsViewerService manages WebSocket viewer windows.
type WsViewerService struct {
	createWin CreateWindowFunc
}

// NewWsViewerService creates a new WebSocket viewer service.
func NewWsViewerService(createWin CreateWindowFunc) *WsViewerService {
	return &WsViewerService{createWin: createWin}
}

// OpenWsViewer opens a new Wails window at /ws-view?id=<entryID>.
func (s *WsViewerService) OpenWsViewer(id int64) {
	path := "/ws-view?id=" + fmt.Sprint(id)
	applog.Info("wssviewer: open", "id", id, "path", path)
	s.createWin(path)
}
