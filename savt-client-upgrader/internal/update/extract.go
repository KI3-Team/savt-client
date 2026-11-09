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
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZip unpacks zipPath into destDir (created if missing) and returns destDir.
func ExtractZip(zipPath, destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("clean extract dir: %w", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create extract dir: %w", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if err := extractZipFile(destDir, f); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(destDir string, f *zip.File) error {
	target, err := safeZipPath(destDir, f.Name)
	if err != nil {
		return err
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	mode := f.Mode()
	if mode == 0 {
		mode = 0644
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func safeZipPath(destDir, name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("illegal zip path: %s", name)
	}
	target := filepath.Join(destDir, clean)
	rel, err := filepath.Rel(destDir, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("illegal zip path: %s", name)
	}
	return target, nil
}
