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
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Paginator represents a pagination widget
type Paginator struct {
	TotalRecords int
	PerPage      int
	CurrentPage  int
	OnPageChange func(page int) // page number change callback
}

// NewPaginator creates a new Paginator instance
func NewPaginator(totalRecords, perPage int, onPageChange func(page int)) *Paginator {
	return &Paginator{
		TotalRecords: totalRecords,
		PerPage:      perPage,
		CurrentPage:  1,
		OnPageChange: onPageChange,
	}
}

// CreateUI generates the pagination UI
func (p *Paginator) CreateUI() *fyne.Container {
	totalPages := (p.TotalRecords + p.PerPage - 1) / p.PerPage
	totalLabel := widget.NewLabel(fmt.Sprintf("Total %d", p.TotalRecords))
	perPageSelect := widget.NewSelect([]string{"10", "20", "50", "100"}, func(selected string) {
		p.PerPage, _ = strconv.Atoi(selected)
		p.CurrentPage = 1
		if p.OnPageChange != nil {
			p.OnPageChange(p.CurrentPage)
		}
	})
	perPageSelect.SetSelected(fmt.Sprintf("%d", p.PerPage))
	pageButtons := container.NewHBox()
	for i := 1; i <= totalPages && i <= 5; i++ {
		page := i
		btn := widget.NewButton(fmt.Sprintf("%d", i), func() {
			p.CurrentPage = page
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		})
		pageButtons.Add(btn)
	}
	prevButton := widget.NewButton("<", func() {
		if p.CurrentPage > 1 {
			p.CurrentPage--
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		}
	})
	if p.CurrentPage == 1 {
		prevButton.Disable()
	}

	// Next button
	nextButton := widget.NewButton(">", func() {
		if p.CurrentPage < totalPages {
			p.CurrentPage++
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		}
	})
	if p.CurrentPage == totalPages {
		nextButton.Disable()
	}
	goToLabel := widget.NewLabel("Go to")
	goToInput := widget.NewEntry()
	goToInput.SetPlaceHolder("Page")
	goToButton := widget.NewButton("Go", func() {
		page, err := strconv.Atoi(goToInput.Text)
		if err == nil && page > 0 && page <= totalPages {
			p.CurrentPage = page
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		}
	})
	bottomContainer := container.NewHBox(
		totalLabel,
		perPageSelect,
		prevButton,
		pageButtons,
		nextButton,
		goToLabel,
		goToInput,
		goToButton,
	)

	return bottomContainer
}
