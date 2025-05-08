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
	"fmt"
	savt "savt-client/savt-client-api/savt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// define the default job data
var defaultJob = &savt.Job{
	StartTime:       nil,
	DurationSeconds: 0,
	Status:          savt.Status_API_UNKNOWN,
	Token:           "-",
	Ipv4:            defaultMeasurementResult,
	Ipv6:            defaultMeasurementResult,
}

type JobWidget struct {
	widget.BaseWidget
	startTimeLabel *widget.Label
	startTimeValue *widget.Label

	durationLabel *widget.Label
	durationValue *widget.Label

	statusLabel *widget.Label
	statusValue *widget.Label

	tokenLabel *widget.Label
	tokenValue *widget.Label
	copyButton *widget.Button
	tabs       *container.AppTabs
	ipv4Widget *MeasurementResultWidget
	ipv6Widget *MeasurementResultWidget
	container  *fyne.Container
}

func NewJobWidget(job *savt.Job) *JobWidget {
	if job == nil {
		job = defaultJob
	}
	w := &JobWidget{
		startTimeLabel: newBoldLabel("Start Time:"),
		startTimeValue: widget.NewLabel(formatStartTime(job.StartTime)),
		durationLabel:  newBoldLabel("Duration:"),
		durationValue:  widget.NewLabel(formatDuration(job.DurationSeconds)),
		statusLabel:    newBoldLabel("Status:"),
		statusValue:    widget.NewLabel(formatStatus(job.Status)),
		tokenLabel:     newBoldLabel("Token:"),
		tokenValue:     widget.NewLabel(formatToken(job.Token)),
		ipv4Widget:     NewMeasurementResultWidget(IPv4),
		ipv6Widget:     NewMeasurementResultWidget(IPv6),
	}
	w.ipv4Widget.SetResult(job.Ipv4)
	w.ipv6Widget.SetResult(job.Ipv6)
	w.copyButton = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		token := w.tokenValue.Text
		w.copyToClipboard(token)
	})
	w.copyButton.Resize(fyne.NewSize(8, 8))
	w.copyButton.Importance = widget.LowImportance
	w.copyButton.Hide()
	w.tabs = container.NewAppTabs(
		container.NewTabItem("IPv4 Results", w.ipv4Widget),
		container.NewTabItem("IPv6 Results", w.ipv6Widget),
	)
	w.startTimeValue.Alignment = fyne.TextAlignLeading
	w.durationValue.Alignment = fyne.TextAlignLeading
	w.tokenValue.Alignment = fyne.TextAlignLeading

	common_label_size := getCommonLabelSize()
	common_value_size := getCommonValueSize()
	startTimeRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.startTimeLabel),
		container.New(layout.NewGridWrapLayout(common_value_size), w.startTimeValue),
	)
	durationRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.durationLabel),
		container.New(layout.NewGridWrapLayout(common_value_size), w.durationValue),
	)
	statusRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.statusLabel),
		container.New(layout.NewGridWrapLayout(common_value_size), w.statusValue),
	)
	tokenRow := container.New(
		layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(common_label_size), w.tokenLabel),
		container.New(layout.NewHBoxLayout(), w.tokenValue, w.copyButton),
	)

	w.container = container.NewVBox(
		startTimeRow,
		durationRow,
		statusRow,
		tokenRow,
		w.tabs,
		layout.NewSpacer(),
	)
	w.ExtendBaseWidget(w)
	return w
}

func (w *JobWidget) SetJob(job *savt.Job) {
	if job == nil {
		job = defaultJob
	}
	w.startTimeValue.SetText(formatStartTime(job.StartTime))
	w.durationValue.SetText(formatDuration(job.DurationSeconds))
	w.statusValue.SetText(formatStatus(job.Status))
	w.tokenValue.SetText(formatToken(job.Token))

	switch job.Status {
	case savt.Status_RUNNING, savt.Status_INITIAL:
		w.copyButton.Hide()
	default:
		w.copyButton.Show()
	}
	w.ipv4Widget.SetResult(job.Ipv4)
	w.ipv6Widget.SetResult(job.Ipv6)
	w.Refresh()
}

// Clear clears the job information
func (w *JobWidget) Clear() {
	w.SetJob(defaultJob)
}

// CreateRenderer implements the Fyne WidgetRenderer interface
func (w *JobWidget) CreateRenderer() fyne.WidgetRenderer {
	return &jobRenderer{
		container: w.container,
		objects:   []fyne.CanvasObject{w.container},
	}
}

type jobRenderer struct {
	container *fyne.Container
	objects   []fyne.CanvasObject
}

func (r *jobRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

func (r *jobRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

func (r *jobRenderer) Refresh() {
	r.container.Refresh()
}

func (r *jobRenderer) Destroy() {}

func (r *jobRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
func (w *JobWidget) copyToClipboard(text string) {
	if text == "" {
		return
	}
	win := fyne.CurrentApp().Driver().AllWindows()[0]
	win.Clipboard().SetContent(text)
}
func formatStartTime(startTime *timestamppb.Timestamp) string {
	if startTime == nil {
		return "-"
	}
	utcTime := time.Unix(startTime.Seconds, 0).UTC()
	localLocation := time.Now().Location()
	localTime := utcTime.In(localLocation)
	return localTime.Format("2006-01-02 15:04:05 MST")
}

func formatDuration(seconds int32) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	var parts []string
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 || h > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	parts = append(parts, fmt.Sprintf("%ds", s))

	return strings.Join(parts, " ")
}
func formatStatus(status savt.Status) string {
	switch status {
	case savt.Status_INITIAL:
		return "NotStarted"
	case savt.Status_RUNNING:
		return "Running"
	case savt.Status_SUCCEEDED:
		return "Success"
	case savt.Status_SEMI_SUCCEEDED:
		return "Semi-Success" // deprected
	case savt.Status_FAILED:
		return "Failure"
	case savt.Status_CANCELLED: // deprected
		return "Cancelled"
	case savt.Status_API_UNKNOWN:
		return "Unknown"
	default:
		return "Unknown"
	}
}
func formatToken(token string) string {
	if token == "" {
		return "-"
	}
	return token
}
