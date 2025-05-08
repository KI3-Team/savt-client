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

// Layer is an interface for network protocol layers that can be
// marshalled to and unmarshalled from binary representation.
// It provides methods to navigate and manipulate a chain of protocol layers.
type Layer interface {
	// MarshalBinary converts the layer to its binary representation
	MarshalBinary() ([]byte, error)

	// UnmarshalBinary parses the binary data into the layer
	UnmarshalBinary(data []byte) error

	// Next returns the next layer in the protocol chain
	Next() Layer

	// SetNext sets the next layer in the protocol chain
	SetNext(Layer)
}
