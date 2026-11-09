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

package device

import (
	"strings"
	"testing"
)

func TestGenerateDeviceIDLengthAndHex(t *testing.T) {
	id, err := generateDeviceID("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != deviceIDHexLen {
		t.Fatalf("len=%d want %d (%q)", len(id), deviceIDHexLen, id)
	}
	for _, r := range id {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("non-hex char in id: %q", id)
		}
	}
}

func TestGenerateDeviceIDUnique(t *testing.T) {
	a, err := generateDeviceID("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	b, err := generateDeviceID("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("expected different ids for successive generations, got %q", a)
	}
}
