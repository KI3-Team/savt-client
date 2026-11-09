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
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"savt-client/savt-client-upgrader/internal/protocol"
)

// VerifyPackageEntries checks optional per-file SHA256 values from the manifest.
func VerifyPackageEntries(packageRoot string, entries []protocol.FileEntry) error {
	for _, entry := range entries {
		src := filepath.Join(packageRoot, filepath.Clean(entry.SourcePath()))
		if err := ensureInside(packageRoot, src); err != nil {
			return err
		}
		sum, err := FileSHA256(src)
		if err != nil {
			return fmt.Errorf("checksum %s: %w", entry.SourcePath(), err)
		}
		if entry.SHA256 != "" && !strings.EqualFold(sum, strings.TrimSpace(entry.SHA256)) {
			return fmt.Errorf("sha256 mismatch for %s: got %s want %s", entry.Path, sum, entry.SHA256)
		}
	}
	return nil
}

// VerifyDetachedSignature verifies an optional signature file next to the zip
// or a base64 signature string against publicKeyPEM.
func VerifyDetachedSignature(filePath string, sig []byte, publicKeyPEM string) error {
	if len(sig) == 0 {
		return nil
	}
	if strings.TrimSpace(publicKeyPEM) == "" {
		return fmt.Errorf("signature present but public_key_pem is empty")
	}
	pub, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	switch key := pub.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(key, data, sig) {
			return fmt.Errorf("ed25519 signature invalid")
		}
		return nil
	case *rsa.PublicKey:
		hash := sha256.Sum256(data)
		return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sig)
	default:
		return fmt.Errorf("unsupported public key type %T", pub)
	}
}

func parsePublicKey(pemData string) (any, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM public key")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("unable to parse public key")
}

// DecodeSignature parses a base64 signature string.
func DecodeSignature(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(s)
}

// VerifyInstalledCodeSignature runs platform code-signing checks when available.
func VerifyInstalledCodeSignature(path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	cmd := exec.Command("codesign", "--verify", "--verbose=2", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "code object is not signed") || strings.Contains(msg, "not signed at all") {
			return nil
		}
		return fmt.Errorf("codesign verify %s: %w (%s)", path, err, msg)
	}
	return nil
}
