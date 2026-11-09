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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFetchZipRangeResume(t *testing.T) {
	payload := []byte("hello-range-resume-payload-0123456789")
	sum := sha256.Sum256(payload)
	expect := hex.EncodeToString(sum[:])

	var ranged atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHdr := r.Header.Get("Range")
		if rangeHdr == "" {
			w.Header().Set("Content-Length", "10")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload[:10])
			return
		}
		ranged.Store(true)
		if !strings.HasPrefix(rangeHdr, "bytes=") {
			t.Errorf("unexpected Range header %q", rangeHdr)
		}
		start := 10
		w.Header().Set("Content-Range", "bytes 10-35/"+strconv.Itoa(len(payload)))
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)-start))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start:])
	}))
	defer srv.Close()

	staging := t.TempDir()
	d := NewDownloader(staging, srv.URL, 0, true)

	// Seed a partial .part file as if a previous attempt wrote the first 10 bytes.
	name := "pkg.zip"
	part := filepath.Join(staging, name+".part")
	if err := os.WriteFile(part, payload[:10], 0600); err != nil {
		t.Fatal(err)
	}

	var lastDown, lastTotal int64
	zipPath, gotSum, err := d.FetchZip(context.Background(), "1.0.0", srv.URL+"/"+name, expect, func(downloaded, total int64) {
		lastDown, lastTotal = downloaded, total
	})
	if err != nil {
		t.Fatalf("FetchZip: %v", err)
	}
	if !ranged.Load() {
		t.Fatal("expected Range request for existing .part")
	}
	if gotSum != expect {
		t.Fatalf("sha256 got %s want %s", gotSum, expect)
	}
	if _, err := os.Stat(part); !os.IsNotExist(err) {
		t.Fatalf(".part should be renamed away, stat err=%v", err)
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Fatalf("final content mismatch")
	}
	if lastTotal != int64(len(payload)) {
		t.Fatalf("progress total=%d want %d", lastTotal, len(payload))
	}
	if lastDown != int64(len(payload)) {
		t.Fatalf("progress downloaded=%d want %d", lastDown, len(payload))
	}
}

func TestFetchZipFullRewriteOn200(t *testing.T) {
	payload := []byte("full-rewrite-body")
	sum := sha256.Sum256(payload)
	expect := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ignore Range and always return 200 with full body.
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, string(payload))
	}))
	defer srv.Close()

	staging := t.TempDir()
	d := NewDownloader(staging, srv.URL, 0, true)
	part := filepath.Join(staging, "pkg.zip.part")
	if err := os.WriteFile(part, []byte("stale-partial"), 0600); err != nil {
		t.Fatal(err)
	}

	zipPath, gotSum, err := d.FetchZip(context.Background(), "1.0.0", srv.URL+"/pkg.zip", expect, nil)
	if err != nil {
		t.Fatalf("FetchZip: %v", err)
	}
	if gotSum != expect {
		t.Fatalf("sha256 got %s want %s", gotSum, expect)
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Fatalf("content=%q want %q", data, payload)
	}
}
