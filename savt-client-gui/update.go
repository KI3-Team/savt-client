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

package main

import (
	"image/color"
	url2 "net/url"
	"savt-client/savt-client-gui/globals"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func updateDialog() {
	aboutIcon := canvas.NewImageFromResource(resourceIconAbout128Png)
	infoIcon := container.New(layout.NewGridWrapLayout(fyne.NewSize(18, 18)), aboutIcon)
	infoIconForm := container.New(layout.NewVBoxLayout(), layout.NewSpacer(), infoIcon, layout.NewSpacer())
	infoLabel := canvas.NewText("New version available. Please update app to new version.", color.Black)
	infoLabel.TextSize = 18.0
	infoLabel.Alignment = fyne.TextAlignLeading
	infoMessage := container.New(layout.NewHBoxLayout(), infoIconForm, infoLabel)
	url, _ := url2.Parse("https://ki3.org.cn/#/sav?sub=clientDownload")
	updateButton := widget.NewHyperlink("UPDATE NOW", url)
	updateButton.Cursor()
	updateButton.TextStyle.Bold = true

	var updateButtonLayout = container.New(layout.NewHBoxLayout(), layout.NewSpacer(), updateButton, layout.NewSpacer())

	layout := container.New(layout.NewVBoxLayout(), infoMessage, updateButtonLayout)

	keysDialog := dialog.NewCustom("Information", "NO THANKS", layout, globals.GUIAPP.Win)
	keysDialog.Show()
}
