package sync

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// client handles WebDAV HTTP operations with Basic Auth.
type client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string

	// Rate limiting — minimum 200ms between requests to avoid 503 from 坚果云.
	lastReq time.Time
	reqMu   sync.Mutex

	// MKCOL cache — tracks already-created prefix directories.
	mkdirs map[string]bool
}

func newClient(cfg Config) *client {
	return &client{
		baseURL:    strings.TrimRight(cfg.URL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		username:   cfg.Username,
		password:   cfg.Password,
		mkdirs:     make(map[string]bool),
	}
}

// --- Public API ---

// testConnect verifies the WebDAV server is reachable and directories exist.
// Returns nil on success.
func (c *client) testConnect() error {
	// Ensure base directory exists (no trailing slash: some servers reject it).
	if err := c.mkcol("/jPaste"); err != nil {
		return fmt.Errorf("create jPaste dir: %w", err)
	}
	if err := c.mkcol("/jPaste/entries"); err != nil {
		return fmt.Errorf("create entries dir: %w", err)
	}
	return nil
}

// listEntries returns all remote entry files with their last-modified timestamps.
func (c *client) listEntries() ([]remoteEntry, error) {
	files, err := c.propfind("/jPaste/entries/")
	if err != nil {
		return nil, err
	}
	var entries []remoteEntry
	for _, f := range files {
		// Filter to only .json files, extract content_hash from filename.
		name := filepath.Base(f.href)
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		hash := strings.TrimSuffix(name, ".json")
		if len(hash) != 64 {
			continue
		}
		entries = append(entries, remoteEntry{
			hash:         hash,
			lastModified: f.lastModified,
		})
	}
	return entries, nil
}

// getEntry downloads the JSON content for a single entry hash.
func (c *client) getEntry(hash string) ([]byte, error) {
	path := fmt.Sprintf("/jPaste/entries/%s/%s.json", hash[:2], hash)
	return c.doRequest("GET", path, nil)
}

// putEntry uploads an entry JSON blob to the correct prefix directory.
func (c *client) putEntry(hash string, data []byte) error {
	prefix := "/jPaste/entries/" + hash[:2]
	if err := c.mkcol(prefix); err != nil {
		return fmt.Errorf("mkcol prefix: %w", err)
	}
	path := fmt.Sprintf("/jPaste/entries/%s/%s.json", hash[:2], hash)
	return c.doRequestNoBody("PUT", path, data)
}

// deleteEntry removes an entry file from WebDAV.
func (c *client) deleteEntry(hash string) error {
	path := fmt.Sprintf("/jPaste/entries/%s/%s.json", hash[:2], hash)
	return c.doRequestNoBody("DELETE", path, nil)
}

// putSettings uploads settings.json to WebDAV.
func (c *client) putSettings(data []byte) error {
	return c.doRequestNoBody("PUT", "/jPaste/settings.json", data)
}

// getSettings downloads settings.json from WebDAV.
func (c *client) getSettings() ([]byte, error) {
	return c.doRequest("GET", "/jPaste/settings.json", nil)
}

// --- Internal HTTP helpers ---

func (c *client) mkcol(path string) error {
	path = strings.TrimRight(path, "/")
	// Skip if we already created this directory (cache hit).
	c.reqMu.Lock()
	if c.mkdirs[path] {
		c.reqMu.Unlock()
		return nil
	}
	c.reqMu.Unlock()

	c.rateLimit()
	req, err := c.buildReq("MKCOL", path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mkcol %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 || resp.StatusCode == 405 || resp.StatusCode == 400 || resp.StatusCode == 403 {
		c.reqMu.Lock()
		c.mkdirs[path] = true
		c.reqMu.Unlock()
		return nil
	}
	return fmt.Errorf("mkcol %s: unexpected status %d", path, resp.StatusCode)
}

// rateLimit ensures at least 200ms between consecutive HTTP requests.
func (c *client) rateLimit() {
	c.reqMu.Lock()
	elapsed := time.Since(c.lastReq)
	if elapsed < 200*time.Millisecond {
		time.Sleep(200*time.Millisecond - elapsed)
	}
	c.lastReq = time.Now()
	c.reqMu.Unlock()
}

func (c *client) doRequest(method, path string, body []byte) ([]byte, error) {
	c.rateLimit()
	req, err := c.buildReq(method, path, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil // not found, no error
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	return io.ReadAll(resp.Body)
}

func (c *client) doRequestNoBody(method, path string, body []byte) error {
	c.rateLimit()
	req, err := c.buildReq(method, path, body)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if method == "DELETE" && resp.StatusCode == 404 {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *client) buildReq(method, path string, body []byte) (*http.Request, error) {
	url := c.baseURL + path
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method == "PROPFIND" {
		req.Header.Set("Depth", "1")
		req.Header.Set("Content-Type", "application/xml")
	}
	return req, nil
}

// --- PROPFIND XML structures ---

type propfindRequest struct {
	XMLName xml.Name `xml:"D:propfind"`
	D       string   `xml:"xmlns:D,attr"`
	Prop    struct {
		GetLastModified string `xml:"D:getlastmodified"`
	} `xml:"D:prop"`
}

type propfindResponse struct {
	XMLName  xml.Name            `xml:"multistatus"`
	Response []propfindResponseH `xml:"response"`
}

type propfindResponseH struct {
	Href     string `xml:"href"`
	PropStat struct {
		Prop struct {
			LastModified string `xml:"getlastmodified"`
		} `xml:"prop"`
		Status string `xml:"status"`
	} `xml:"propstat"`
}

type remoteEntry struct {
	href         string    // full path from PROPFIND (internal use)
	hash         string    // content_hash (64-char hex)
	lastModified time.Time
}

func (c *client) propfind(dir string) ([]remoteEntry, error) {
	body := propfindRequest{D: "DAV:"}
	body.Prop.GetLastModified = ""
	data, err := xml.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal propfind: %w", err)
	}

	resp, err := c.doRequest("PROPFIND", dir, data)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	var pr propfindResponse
	if err := xml.Unmarshal(resp, &pr); err != nil {
		return nil, fmt.Errorf("unmarshal propfind response: %w", err)
	}

	var entries []remoteEntry
	for _, r := range pr.Response {
		// Skip the directory itself (href matches the request path).
		if strings.TrimRight(r.Href, "/") == strings.TrimRight(dir, "/") {
			continue
		}
		if r.PropStat.Prop.LastModified == "" {
			continue
		}
		t, err := parseHTTPDate(r.PropStat.Prop.LastModified)
		if err != nil {
			continue
		}
		entries = append(entries, remoteEntry{
			href:         r.Href,
			lastModified: t,
		})
	}
	return entries, nil
}

// parseHTTPDate parses a date in RFC 1123 format (used by WebDAV getlastmodified).
func parseHTTPDate(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC1123, s)
	if err != nil {
		// Some servers use GMT suffix instead of standard timezone format.
		s = strings.Replace(s, "GMT", "", 1)
		return time.Parse("Mon, 02 Jan 2006 15:04:05", strings.TrimSpace(s))
	}
	return t, nil
}
