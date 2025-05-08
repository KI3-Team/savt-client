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

package net

// Raw represents raw payload data
type Raw struct {
	Data []byte
}

// NewRaw creates a new Raw layer from bytes
func NewRaw(b []byte) (*Raw, error) {
	var r Raw
	if err := r.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return &r, nil
}

// Next returns the next layer, which is always nil for Raw
func (r Raw) Next() Layer {
	return nil
}

// SetNext sets the next layer, which is a no-op for Raw
func (r Raw) SetNext(Layer) {}

// MarshalBinary serializes the Raw data
func (r Raw) MarshalBinary() ([]byte, error) {
	return r.Data, nil
}

// UnmarshalBinary deserializes bytes into Raw data
func (r *Raw) UnmarshalBinary(b []byte) error {
	r.Data = b
	return nil
}
