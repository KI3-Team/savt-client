// Copyright 2025 KI3
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProgressFunc reports download progress. total may be 0 if unknown.
type ProgressFunc func(downloaded, total int64)

// Downloader fetches version zip packages over HTTPS.
type Downloader struct {
	Client         *http.Client
	AllowHTTP      bool
	Staging        string
	PackageBaseURL string // e.g. https://ki3.org.cn/public/sav/upgrader
	OnProgress     ProgressFunc
}

// NewDownloader creates a Downloader with TLS-oriented defaults.
// Request lifetime is controlled by context passed to FetchZip; the HTTP client
// has no overall Timeout so long / resumed downloads are not cut off early.
func NewDownloader(staging, packageBaseURL string, _ time.Duration, allowHTTP bool) *Downloader {
	return &Downloader{
		Client:         &http.Client{},
		AllowHTTP:      allowHTTP,
		Staging:        staging,
		PackageBaseURL: strings.TrimRight(packageBaseURL, "/"),
	}
}

// PackageURL builds the default zip URL for a version.
func (d *Downloader) PackageURL(version string) string {
	return fmt.Sprintf("%s/savt-client-%s.zip", d.PackageBaseURL, version)
}

// FetchZip downloads the version package with HTTP Range resume support.
// A .part file is retained across failures / cancellation for resume.
// After a complete download, sha256 of the full file is verified against expectSHA256.
func (d *Downloader) FetchZip(ctx context.Context, version, overrideURL, expectSHA256 string, onProgress ProgressFunc) (zipPath string, sum string, err error) {
	if onProgress == nil {
		onProgress = d.OnProgress
	}

	downloadURL := overrideURL
	if downloadURL == "" {
		if version == "" {
			return "", "", fmt.Errorf("version is required")
		}
		downloadURL = d.PackageURL(version)
	}
	if err := validateURL(downloadURL, d.AllowHTTP); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(d.Staging, 0755); err != nil {
		return "", "", fmt.Errorf("create staging dir: %w", err)
	}

	name := filepath.Base(downloadURL)
	if name == "." || name == "/" || name == "" {
		name = "savt-client-" + version + ".zip"
	}
	dest := filepath.Join(d.Staging, name+".part")
	final := filepath.Join(d.Staging, name)

	var existingSize int64
	if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
		existingSize = st.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", "", err
	}
	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	var offset int64
	var openFlags int
	switch resp.StatusCode {
	case http.StatusPartialContent:
		openFlags = os.O_CREATE | os.O_APPEND | os.O_WRONLY
		offset = existingSize
	case http.StatusOK:
		openFlags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
		offset = 0
	default:
		return "", "", fmt.Errorf("download %s: unexpected status %s", downloadURL, resp.Status)
	}

	total := contentTotal(resp, offset)

	f, err := os.OpenFile(dest, openFlags, 0600)
	if err != nil {
		return "", "", err
	}

	downloaded := offset
	if onProgress != nil {
		onProgress(downloaded, total)
	}

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				_ = f.Close()
				// Keep .part for resume.
				return "", "", fmt.Errorf("write download: %w", werr)
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = f.Close()
			// Keep .part for resume (including context cancel).
			return "", "", fmt.Errorf("write download: %w", readErr)
		}
	}
	if err := f.Close(); err != nil {
		return "", "", err
	}

	sum, err = FileSHA256(dest)
	if err != nil {
		return "", "", err
	}
	if expectSHA256 != "" && !strings.EqualFold(sum, strings.TrimSpace(expectSHA256)) {
		_ = os.Remove(dest)
		return "", "", fmt.Errorf("zip sha256 mismatch: got %s want %s", sum, expectSHA256)
	}
	_ = os.Remove(final)
	if err := os.Rename(dest, final); err != nil {
		return "", "", err
	}
	return final, sum, nil
}

func contentTotal(resp *http.Response, offset int64) int64 {
	if resp.StatusCode == http.StatusPartialContent {
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			// bytes start-end/total
			if i := strings.LastIndex(cr, "/"); i >= 0 {
				totalStr := strings.TrimSpace(cr[i+1:])
				if totalStr != "*" {
					if n, err := strconv.ParseInt(totalStr, 10, 64); err == nil && n > 0 {
						return n
					}
				}
			}
		}
		if resp.ContentLength > 0 {
			return offset + resp.ContentLength
		}
		return 0
	}
	if resp.ContentLength > 0 {
		return resp.ContentLength
	}
	return 0
}

func validateURL(raw string, allowHTTP bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if allowHTTP {
			return nil
		}
		return fmt.Errorf("refusing non-HTTPS url: %s", raw)
	default:
		return fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}
}

// FileSHA256 returns the hex SHA256 of a local file.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
