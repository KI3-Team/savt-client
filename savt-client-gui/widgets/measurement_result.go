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
	savt "savt-client/savt-client-api/savt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var defaultMeasurementResult = &savt.MeasurementResult{
	ClientAddress:         "-",
	Asn:                   0,
	OutboundPrivate:       savt.MeasurementStatus_NOT_TESTED,
	OutboundRoutable:      savt.MeasurementStatus_NOT_TESTED,
	SpoofablePrefixLength: -1,
	InboundPrivate:        savt.MeasurementStatus_NOT_TESTED,
	InboundInternal:       savt.MeasurementStatus_NOT_TESTED,
}

type MeasurementResultWidget struct {
	widget.BaseWidget
	ipVersion             IPVersion
	clientAddressLabel    *widget.Label
	clientAddressValue    *widget.Label
	asnLabel              *widget.Label
	asnValue              *widget.Label
	outboundPrivateLabel  *widget.Label
	outboundPrivateIcon   *canvas.Image
	outboundRoutableLabel *widget.Label
	outboundRoutableIcon  *canvas.Image

	SpoofablePrefixLengthLabel *widget.Label
	SpoofablePrefixLengthValue *widget.Label

	inboundPrivateLabel  *widget.Label
	inboundPrivateIcon   *canvas.Image
	inboundInternalLabel *widget.Label
	inboundInternalIcon  *canvas.Image
	container            *fyne.Container
	result               *savt.MeasurementResult
}

func NewMeasurementResultWidget(ipVersion IPVersion) *MeasurementResultWidget {
	result := defaultMeasurementResult
	w := &MeasurementResultWidget{
		ipVersion:                  ipVersion,
		clientAddressLabel:         newBoldLabel("Client address: "),
		clientAddressValue:         newWrappedLabel(formatAddress(result.ClientAddress)),
		asnLabel:                   newBoldLabel("ASN: "),
		asnValue:                   widget.NewLabel(formatASN(result.Asn)),
		outboundPrivateLabel:       newBoldLabel("Outbound private:"),
		outboundPrivateIcon:        createStatusIcon(ipVersion, result.OutboundPrivate),
		outboundRoutableLabel:      newBoldLabel("Outbound routable:"),
		outboundRoutableIcon:       createStatusIcon(ipVersion, result.OutboundRoutable),
		SpoofablePrefixLengthLabel: newBoldLabel("Spoofable prefix length:"),
		SpoofablePrefixLengthValue: widget.NewLabel(formatSpoofablePrefixLength(result.SpoofablePrefixLength)),
		inboundPrivateLabel:        newBoldLabel("Inbound private:"),
		inboundPrivateIcon:         createStatusIcon(ipVersion, result.InboundPrivate),
		inboundInternalLabel:       newBoldLabel("Inbound internal:"),
		inboundInternalIcon:        createStatusIcon(ipVersion, result.InboundInternal),
		result:                     result,
	}
	w.clientAddressValue.Alignment = fyne.TextAlignLeading
	w.asnValue.Alignment = fyne.TextAlignLeading
	w.SpoofablePrefixLengthValue.Alignment = fyne.TextAlignLeading
	common_label_size := getCommonLabelSize()
	common_value_size := getCommonValueSize()
	clientAddressRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.clientAddressLabel),
		container.New(layout.NewGridWrapLayout(common_value_size), w.clientAddressValue),
	)
	asnRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.asnLabel),
		container.New(layout.NewGridWrapLayout(common_value_size), w.asnValue),
	)
	spoofablePrefixRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.SpoofablePrefixLengthLabel),
		container.New(layout.NewGridWrapLayout(common_value_size), w.SpoofablePrefixLengthValue),
	)
	outboundPrivateRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.outboundPrivateLabel),
		w.outboundPrivateIcon,
	)
	outboundRoutableRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.outboundRoutableLabel),
		w.outboundRoutableIcon,
	)
	inboundPrivateRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.inboundPrivateLabel),
		w.inboundPrivateIcon,
	)
	inboundInternalRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.inboundInternalLabel),
		w.inboundInternalIcon,
	)
	w.container = container.NewVBox(
		clientAddressRow,
		asnRow,
		spoofablePrefixRow,
		outboundPrivateRow,
		outboundRoutableRow,
		inboundPrivateRow,
		inboundInternalRow,
	)

	w.ExtendBaseWidget(w)
	return w
}

func (w *MeasurementResultWidget) applyResult(result *savt.MeasurementResult) {
	fyne.Do(func() {
		w.clientAddressValue.SetText(formatAddress(result.ClientAddress))
		w.asnValue.SetText(formatASN(result.Asn))
		w.outboundPrivateIcon.Resource = getImageStatus(w.ipVersion, result.OutboundPrivate).Resource
		w.outboundRoutableIcon.Resource = getImageStatus(w.ipVersion, result.OutboundRoutable).Resource
		w.SpoofablePrefixLengthValue.SetText(formatSpoofablePrefixLength(result.SpoofablePrefixLength))
		w.inboundPrivateIcon.Resource = getImageStatus(w.ipVersion, result.InboundPrivate).Resource
		w.inboundInternalIcon.Resource = getImageStatus(w.ipVersion, result.InboundInternal).Resource
	})
}

func (w *MeasurementResultWidget) SetResult(result *savt.MeasurementResult) {
	if result == nil {
		result = defaultMeasurementResult
	}
	w.result = result
	w.applyResult(result)
	fyne.Do(func() {
		w.Refresh()
	})
}

func (w *MeasurementResultWidget) Clear() {
	w.applyResult(defaultMeasurementResult)
	fyne.Do(func() {
		w.Refresh()
	})
}

// CreateRenderer
func (w *MeasurementResultWidget) CreateRenderer() fyne.WidgetRenderer {
	return &measurementResultRenderer{
		container: w.container,
		objects:   []fyne.CanvasObject{w.container},
	}
}

// measurementResultRenderer
type measurementResultRenderer struct {
	container *fyne.Container
	objects   []fyne.CanvasObject
}

func (r *measurementResultRenderer) Layout(size fyne.Size) {
	fyne.Do(func() {
		r.container.Resize(size)
	})
}

func (r *measurementResultRenderer) MinSize() fyne.Size {
	var min fyne.Size
	fyne.Do(func() {
		min = r.container.MinSize()
	})
	return min
}

func (r *measurementResultRenderer) Refresh() {
	fyne.Do(func() {
		r.container.Refresh()
	})
}

func (r *measurementResultRenderer) Destroy() {}

func (r *measurementResultRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func createStatusIcon(ipVersion IPVersion, status savt.MeasurementStatus) *canvas.Image {
	var icon *canvas.Image
	fyne.Do(func() {
		icon = getImageStatus(ipVersion, status)
		icon.SetMinSize(fyne.NewSize(72, 36))
		icon.FillMode = canvas.ImageFillContain
	})
	return icon
}

func newBoldLabel(text string) *widget.Label {
	var label *widget.Label
	fyne.Do(func() {
		label = widget.NewLabel(text)
		label.TextStyle = fyne.TextStyle{Bold: true}
	})
	return label
}

// formatAddress formats the address string
func formatAddress(address string) string {
	if address == "" {
		return "-"
	}
	return address
}

func formatASN(asn int32) string {
	if asn == 0 {
		return "-"
	}
	return strconv.Itoa(int(asn))
}

func formatSpoofablePrefixLength(spoofable_prefix_length int32) string {
	if spoofable_prefix_length < 0 {
		return "-"
	}
	return strconv.Itoa(int(spoofable_prefix_length))
}
