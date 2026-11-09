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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"savt-client/savt-client-upgrader/internal/paths"
	"savt-client/savt-client-upgrader/internal/protocol"
)

// Applier applies delete / add / replace steps from a package manifest.
type Applier struct{}

// ApplyEntries copies files from the extracted package into the install root.
func (a *Applier) ApplyEntries(entries []protocol.FileEntry, packageRoot string) error {
	for _, entry := range entries {
		if strings.TrimSpace(entry.Path) == "" {
			return fmt.Errorf("file entry missing path")
		}
		src := filepath.Join(packageRoot, filepath.Clean(entry.SourcePath()))
		if err := ensureInside(packageRoot, src); err != nil {
			return err
		}
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("package file %s: %w", entry.SourcePath(), err)
		}
		dest := paths.ResolvePath(entry.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", dest, err)
		}
		mode := parseMode(entry.Mode, 0755)
		if err := installFile(src, dest, mode); err != nil {
			return fmt.Errorf("install %s: %w", dest, err)
		}
		_ = VerifyInstalledCodeSignature(dest)
	}
	return nil
}

// DeleteFiles removes files listed in the update plan.
func (a *Applier) DeleteFiles(files []string) error {
	for _, f := range files {
		path := paths.ResolvePath(f)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete %s: %w", path, err)
		}
	}
	return nil
}

func ensureInside(root, target string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path escapes package root: %s", target)
	}
	return nil
}

func parseMode(mode string, fallback os.FileMode) os.FileMode {
	if mode == "" {
		return fallback
	}
	v, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return fallback
	}
	return os.FileMode(v)
}

func installFile(src, dest string, mode os.FileMode) error {
	tmp := dest + ".new"
	_ = os.Remove(tmp)
	if err := copyFile(src, tmp, mode); err != nil {
		return err
	}
	bak := dest + ".bak"
	if _, err := os.Stat(dest); err == nil {
		_ = os.Remove(bak)
		if err := os.Rename(dest, bak); err != nil {
			if err := copyFile(tmp, dest, mode); err != nil {
				_ = os.Remove(tmp)
				return err
			}
			_ = os.Remove(tmp)
			return nil
		}
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Rename(bak, dest)
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(bak)
	return nil
}

func copyFile(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dest)
		return err
	}
	if err := out.Chmod(mode); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// IsSelfTarget reports whether dest refers to the currently running upgrader binary.
func IsSelfTarget(dest string) (bool, error) {
	self, err := os.Executable()
	if err != nil {
		return false, err
	}
	selfEval, err := filepath.EvalSymlinks(self)
	if err != nil {
		selfEval = self
	}
	destEval, err := filepath.EvalSymlinks(dest)
	if err != nil {
		destEval = dest
	}
	return filepath.Clean(selfEval) == filepath.Clean(destEval), nil
}
