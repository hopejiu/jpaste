package curlviewer

import (
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"time"

	applog "jpaste/internal/log"
)

// CreateWindowFunc is a callback that opens a new Wails window at a given URL path.
type CreateWindowFunc func(path string)

// Service manages curl viewer windows.
type Service struct {
	createWin CreateWindowFunc
}

// NewService creates a new curl viewer service.
func NewService(createWin CreateWindowFunc) *Service {
	return &Service{createWin: createWin}
}

// OpenCurlViewer opens a new Wails window at /curl-view?id=<entryID>.
func (s *Service) OpenCurlViewer(id int64) {
	path := "/curl-view?id=" + formatInt(id)
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
	Timeout         int               `json:"timeout"` // seconds
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
func (s *Service) SendCurlRequest(req CurlRequest) (CurlResponse, error) {
	applog.Info("curlviewer: send request", "method", req.Method, "url", req.URL,
		"followRedirect", req.FollowRedirects, "timeout", req.Timeout)

	// Build HTTP request
	bodyReader := io.NopCloser(strings.NewReader(req.Body))
	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		applog.Error("curlviewer: build request failed", "error", err)
		return CurlResponse{}, err
	}

	// Set headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Configure client
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		// Do NOT follow redirects unless explicitly asked
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

	// Execute
	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		applog.Error("curlviewer: request failed", "error", err)
		return CurlResponse{}, err
	}
	defer resp.Body.Close()

	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("curlviewer: read body failed", "error", err)
		return CurlResponse{}, err
	}

	// Build response headers map
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	result := CurlResponse{
		StatusCode: resp.StatusCode,
		StatusText: resp.Status,
		Headers:    respHeaders,
		Body:       string(bodyBytes),
		DurationMs: duration,
	}

	applog.Info("curlviewer: response received",
		"status", result.StatusCode, "bodyLen", len(result.Body), "durationMs", result.DurationMs)

	return result, nil
}

// formatInt converts int64 to string without importing fmt.
func formatInt(i int64) string {
	if i == 0 {
		return "0"
	}
	negative := false
	if i < 0 {
		negative = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}


