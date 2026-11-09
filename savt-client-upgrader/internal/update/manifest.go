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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"savt-client/savt-client-upgrader/internal/protocol"
)

// ManifestFileNames are candidate config filenames inside the zip.
var ManifestFileNames = []string{
	"upgrade.json",
	"update.json",
	"manifest.json",
}

// LoadManifest reads the package upgrade config from an extracted directory.
func LoadManifest(extractDir string) (*protocol.PackageManifest, string, error) {
	for _, name := range ManifestFileNames {
		path := filepath.Join(extractDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m protocol.PackageManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, "", fmt.Errorf("parse %s: %w", name, err)
		}
		return &m, path, nil
	}

	// Also support a single top-level directory wrapper: savt-client-x.y.z/upgrade.json
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return nil, "", fmt.Errorf("read extract dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip macOS resource-fork junk and hidden dirs from Finder zips.
		if name == "__MACOSX" || strings.HasPrefix(name, ".") {
			continue
		}
		sub := filepath.Join(extractDir, name)
		for _, cfgName := range ManifestFileNames {
			path := filepath.Join(sub, cfgName)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var m protocol.PackageManifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, "", fmt.Errorf("parse %s: %w", path, err)
			}
			return &m, path, nil
		}
	}
	listing, _ := os.ReadDir(extractDir)
	names := make([]string, 0, len(listing))
	for _, e := range listing {
		names = append(names, e.Name())
	}
	return nil, "", fmt.Errorf("upgrade config not found in package (need one of %v at zip root); extracted entries: %v", ManifestFileNames, names)
}

// ManifestRoot returns the directory that contains the manifest (payload root).
func ManifestRoot(manifestPath string) string {
	return filepath.Dir(manifestPath)
}
