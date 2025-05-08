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

package widgets

import (
	"savt-client/savt-client-api/savt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// IPVersion enum type for distinguishing between IPv4 and IPv6
type IPVersion int

const (
	IPv4 IPVersion = iota // IPv4 enum value
	IPv6                  // IPv6 enum value
)

type MeasurementResultWrapper struct {
	*savt.MeasurementResult
	IPVersion IPVersion
	StartTime time.Time
	Token     string
	JobType   savt.Jobtype
}

// createImage creates an image object
func createImage(resource fyne.Resource) *canvas.Image {
	image := canvas.NewImageFromResource(resource)
	return image
}

// getImageStatus gets the image corresponding to the status (without using cache)
func getImageStatus(ipVersion IPVersion, status savt.MeasurementStatus) *canvas.Image {
	var resource fyne.Resource
	switch ipVersion {
	case IPv4:
		switch status {
		case savt.MeasurementStatus_BLOCKED:
			resource = resourceIconBlocked256Png
		case savt.MeasurementStatus_RECEIVED:
			resource = resourceIconReceived256Png
		case savt.MeasurementStatus_REWRITTEN:
			resource = resourceIconRewritten256Png
		case savt.MeasurementStatus_NOT_TESTED:
			resource = resourceIconNotTested256Png
		default:
			resource = resourceIconEmpty256Png
		}
	case IPv6:
		switch status {
		case savt.MeasurementStatus_BLOCKED:
			resource = resourceIconBlocked256Png
		case savt.MeasurementStatus_RECEIVED:
			resource = resourceIconReceived256Png
		case savt.MeasurementStatus_REWRITTEN:
			resource = resourceIconRewritten256Png
		case savt.MeasurementStatus_NOT_TESTED:
			resource = resourceIconNotTested256Png
		default:
			resource = resourceIconEmpty256Png
		}
	}

	return createImage(resource)
}
