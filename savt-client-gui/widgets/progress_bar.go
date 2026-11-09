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
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"savt-client/savt-client-api/savt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ProgressBarWidget represents a custom progress bar widget
type ProgressBarWidget struct {
	widget.BaseWidget
	progress       int
	status         savt.Status
	imageContainer *fyne.Container
	progressText   *canvas.Text
}

// NewProgressBarWidget creates a new ProgressBarWidget
func NewProgressBarWidget() *ProgressBarWidget {
	p := &ProgressBarWidget{
		progress: 0,
		status:   savt.Status_INITIAL,
		progressText: &canvas.Text{
			Text:      "0%",
			Alignment: fyne.TextAlignCenter,
			Color:     color.Gray{Y: 128},
		},
	}
	svgContent := p.generateSVG(p.progress, p.status)
	img := p.createSVGImage(svgContent)
	p.imageContainer = container.NewStack(img)
	p.ExtendBaseWidget(p)
	return p
}

// SetProgress sets the progress value and status
func (p *ProgressBarWidget) SetProgress(progress int, status savt.Status) {
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}
	p.progress = progress
	p.status = status

	p.updateProgressText()

	svgContent := p.generateSVG(progress, status)
	img := p.createSVGImage(svgContent)

	p.imageContainer.Objects = []fyne.CanvasObject{img}
	fyne.Do(func() {
		p.imageContainer.Refresh()
		p.Refresh()
	})
}

// updateProgressText
func (p *ProgressBarWidget) updateProgressText() {
	p.progressText.Text = fmt.Sprintf("%d%%", p.progress)
	p.progressText.Color = p.getColorByStatus(p.status)
	fyne.Do(func() {
		p.progressText.Refresh()
	})
}

func (p *ProgressBarWidget) generateSVG(progress int, status savt.Status) string {
	colorHex := p.getHexColorByStatus(status)

	switch progress {
	case 0:
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="300" height="300">
  <circle cx="50" cy="50" r="45" fill="none" stroke="#D3D3D3" stroke-width="7"/>
</svg>`
	case 100:
		return fmt.Sprintf(`
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="300" height="300">
  <circle cx="50" cy="50" r="45" fill="none" stroke="%s" stroke-width="7"/>
</svg>
`, colorHex)
	default:
		angle := float64(progress) / 100.0 * 360.0
		radian := angle * (math.Pi / 180.0)
		x := 50 + 45*math.Sin(radian)
		y := 50 - 45*math.Cos(radian)

		svgTemplate := fmt.Sprintf(`
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="300" height="300">
  <circle cx="50" cy="50" r="45" fill="none" stroke="#D3D3D3" stroke-width="7"/>
  <path fill="none" stroke="%s" stroke-width="7" d="
        M 50,5
        A 45,45 0 %d,1 %.2f,%.2f"/>
</svg>
`, colorHex, boolToArcFlag(progress > 50), x, y)
		return svgTemplate
	}
}

// getColorByStatus
func (p *ProgressBarWidget) getColorByStatus(status savt.Status) color.Color {
	switch status {
	case savt.Status_INITIAL:
		return color.Gray{Y: 128}
	case savt.Status_RUNNING:
		return color.RGBA{R: 3, G: 255, B: 0, A: 255}
	case savt.Status_SUCCEEDED:
		return color.RGBA{R: 0, G: 128, B: 0, A: 255}
	case savt.Status_SEMI_SUCCEEDED:
		return color.RGBA{R: 255, G: 165, B: 0, A: 255}
	case savt.Status_FAILED:
		return color.RGBA{R: 255, G: 0, B: 0, A: 255}
	case savt.Status_CANCELLED:
		return color.RGBA{R: 255, G: 215, B: 0, A: 255}
	default:
		return color.Gray{Y: 128}
	}
}

func (p *ProgressBarWidget) getHexColorByStatus(status savt.Status) string {
	switch status {
	case savt.Status_INITIAL:
		return "#D3D3D3"
	case savt.Status_RUNNING:
		return "#03FF00"
	case savt.Status_SUCCEEDED:
		return "#008001"
	case savt.Status_SEMI_SUCCEEDED:
		return "#FFA500"
	case savt.Status_FAILED:
		return "#FF0000"
	case savt.Status_CANCELLED:
		return "#FFD700"
	default:
		return "#D3D3D3"
	}
}

func boolToArcFlag(isLargeArc bool) int {
	if isLargeArc {
		return 1
	}
	return 0
}

func (p *ProgressBarWidget) createSVGImage(svgContent string) *canvas.Image {
	fileName := fmt.Sprintf("progress-%d-%s.svg", p.progress, p.status)
	reader := bytes.NewReader([]byte(svgContent))
	img := canvas.NewImageFromReader(reader, fileName)
	if img == nil {
		log.Fatalf("Failed to create SVG image\nGenerated SVG:\n%s", svgContent)
	}
	img.FillMode = canvas.ImageFillContain
	return img
}

func (p *ProgressBarWidget) CreateRenderer() fyne.WidgetRenderer {
	return &progressBarRenderer{
		widget:         p,
		imageContainer: p.imageContainer,
		progressText:   p.progressText,
		objects:        []fyne.CanvasObject{p.imageContainer, p.progressText},
	}
}

// progressBarRenderer is the renderer for the ProgressBarWidget
type progressBarRenderer struct {
	widget         *ProgressBarWidget
	imageContainer *fyne.Container
	progressText   *canvas.Text
	objects        []fyne.CanvasObject
}

// Layout arranges the widget components
func (r *progressBarRenderer) Layout(size fyne.Size) {
	r.imageContainer.Resize(size)

	r.progressText.TextSize = size.Width / 10
	r.progressText.Refresh()

	textSize := r.progressText.MinSize()
	textWidth := textSize.Width
	textHeight := textSize.Height

	textX := float32(math.Round(float64((size.Width - textWidth) / 2)))
	textY := float32(math.Round(float64((size.Height - textHeight) / 2)))
	textX += 30
	r.progressText.Move(fyne.NewPos(textX, textY))
}

// MinSize returns the minimum size of the widget
func (r *progressBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 200)
}

// Refresh updates the widget appearance
func (r *progressBarRenderer) Refresh() {
	fyne.Do(func() {
		canvas.Refresh(r.imageContainer)
		canvas.Refresh(r.progressText)
	})
}

// Destroy cleans up the renderer
func (r *progressBarRenderer) Destroy() {}

// Objects returns the objects that make up the widget
func (r *progressBarRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
