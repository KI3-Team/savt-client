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
	"image/color"
	url2 "net/url"
	"savt-client/savt-client-api/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CustomTable represents a custom table widget with loading overlay and column width control
type CustomTable struct {
	widget.BaseWidget
	headers      []string
	data         []MeasurementResultWrapper
	columnWidths map[int]int
	loading      bool
	content      *fyne.Container
	table        *widget.Table
	overlay      *fyne.Container
	mountedFunc  func()
}

// NewCustomTable creates a new CustomTable
func NewCustomTable(headers []string, columnWidths map[int]int) *CustomTable {
	t := &CustomTable{
		headers:      headers,
		data:         nil,
		columnWidths: columnWidths,
		loading:      true,
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *CustomTable) SetMounted(callback func()) {
	t.mountedFunc = callback
}

// SetData sets the table data
func (t *CustomTable) SetData(results []MeasurementResultWrapper) {
	t.data = results
	t.loading = false
	t.content.Objects = []fyne.CanvasObject{t.table}
	t.content.Refresh()
}

func (t *CustomTable) createTable() *widget.Table {
	return widget.NewTable(
		func() (int, int) {
			if t.loading {
				return 1, len(t.headers)
			}
			return len(t.data) + 1, len(t.headers)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(widget.NewLabel(""), layout.NewSpacer())
		},
		func(cell widget.TableCellID, obj fyne.CanvasObject) {
			hbox := obj.(*fyne.Container)
			hbox.Objects = nil
			if cell.Row == 0 {
				if cell.Col < len(t.headers) {
					text := canvas.NewText(t.headers[cell.Col], theme.Color(theme.ColorNameForeground))
					text.TextStyle = fyne.TextStyle{
						Bold: true,
					}
					text.TextSize = 12
					text.Alignment = fyne.TextAlignCenter
					hbox.Add(text)
				}
				return
			}
			rowIndex := cell.Row - 1
			if rowIndex < len(t.data) {
				result := t.data[rowIndex]
				switch cell.Col {
				case 0:
					localTime := result.StartTime.Local()
					hbox.Add(widget.NewLabel(localTime.Format("2006-01-02 15:04:05")))
				case 1:
					hbox.Add(widget.NewLabel(result.JobType.String()))
				case 2:
					ipType := "4"
					if result.IPVersion == IPv6 {
						ipType = "6"
					}
					hbox.Add(widget.NewLabel(ipType))
				case 3:
					hbox.Add(widget.NewLabel(result.ClientAddress))
				case 4:
					hbox.Add(widget.NewLabel(fmt.Sprintf("%d", result.Asn)))
				case 5:
					hbox.Add(widget.NewLabel(result.OutboundPrivate.String()))
				case 6:
					hbox.Add(widget.NewLabel(result.OutboundRoutable.String()))
				case 7:
					hbox.Add(widget.NewLabel(result.InboundPrivate.String()))
				case 8:
					hbox.Add(widget.NewLabel(result.InboundInternal.String()))
				case 9:
					ins, _ := utils.GetServiceManagerIns()
					env := ins.GetEnv()
					var url *url2.URL
					if result.Token != "" {
						switch env {
						case utils.Test:
							url, _ = url2.Parse(fmt.Sprintf("https://test.kdp.ki3.org.cn/#/sav-t-private?session=%s", result.Token))
						case utils.Production:
							url, _ = url2.Parse(fmt.Sprintf("https://ki3.org.cn/#/sav-t-private?session=%s", result.Token))
						}
						reportButton := widget.NewButtonWithIcon("Report", theme.DocumentIcon(), func() {
							_ = fyne.CurrentApp().OpenURL(url)
						})
						reportButton.Resize(fyne.NewSize(4, 4))
						reportButton.Importance = widget.LowImportance

						copyButton := widget.NewButtonWithIcon("Token", theme.ContentCopyIcon(), func() {
							clipboard := fyne.CurrentApp().Driver().AllWindows()[0].Clipboard()
							clipboard.SetContent(result.Token)
						})
						copyButton.Resize(fyne.NewSize(4, 4))
						copyButton.Importance = widget.LowImportance

						grid := container.NewGridWithColumns(2, reportButton, copyButton)
						hbox.Add(grid)
					} else {
						label := widget.NewLabel("-")
						hbox.Add(label)
					}
				default:
					hbox.Add(widget.NewLabel(""))
				}
			}
		})
}

// CreateRenderer creates the renderer for the CustomTable
func (t *CustomTable) CreateRenderer() fyne.WidgetRenderer {
	t.table = t.createTable()
	for col, width := range t.columnWidths {
		t.table.SetColumnWidth(col, float32(width))
	}
	border := container.NewStack(
		canvas.NewRectangle(color.RGBA{R: 169, G: 169, B: 169, A: 255}),
		t.table,
	)
	t.overlay = container.NewStack(
		canvas.NewRectangle(&color.RGBA{128, 128, 128, 128}),
		container.NewCenter(widget.NewLabelWithStyle(
			"Loading...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true},
		)),
	)
	t.content = container.NewStack(border, t.overlay)
	if t.mountedFunc != nil {
		t.mountedFunc()
	}
	return widget.NewSimpleRenderer(t.content)
}
