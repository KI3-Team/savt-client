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
	"savt-client/savt-client-api/utils"
	"savt-client/savt-client-gui/globals"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// createAboutPage creates the about page
func createAboutPage() fyne.CanvasObject {
	appResourceIcon := canvas.NewImageFromResource(resourceIconApp1024Png)
	appIcon := container.New(layout.NewGridWrapLayout(fyne.NewSize(64, 64)), appResourceIcon)
	appIconForm := container.New(layout.NewVBoxLayout(), layout.NewSpacer(), appIcon, layout.NewSpacer())

	appLabel := canvas.NewText("SAV-T Runner", color.RGBA{10, 50, 78, 255})
	appLabel.TextSize = 64.0
	appLabel.TextStyle.Bold = true
	appLabel.Alignment = fyne.TextAlignLeading
	appHeader := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), appIconForm, appLabel, layout.NewSpacer())

	vm, err := utils.GetServiceManagerIns()
	if err != nil {
		globals.GUIAPP.Logger.Error("Error getting VersionManager instance: %v", err)
		return widget.NewLabel("Error: Failed to get version information")
	}

	appVersionForm := canvas.NewText("KI3 IP Spoofing Testing Client, version V"+vm.GetVersion(), color.RGBA{0, 0, 0, 255})

	officailUrl, _ := url2.Parse("https://ki3.org.cn")
	officailLabel := canvas.NewText("Official website: ", color.RGBA{0, 0, 0, 255})
	officailValue := widget.NewHyperlink("https://ki3.org.cn", officailUrl)
	officailForm := container.New(layout.NewHBoxLayout(), officailLabel, officailValue)

	emailUrl, _ := url2.Parse("mailto:ki3contact@163.com")
	emailLabel := canvas.NewText("Email: ", color.RGBA{0, 0, 0, 255})
	emailValue := widget.NewHyperlink("ki3contact@163.com", emailUrl)
	emailForm := container.New(layout.NewHBoxLayout(), emailLabel, emailValue)

	contact := container.New(layout.NewVBoxLayout(), appVersionForm, officailForm, emailForm)
	contactForm := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), contact, layout.NewSpacer())

	updateButton := widget.NewButton("Check for update", func() {
		needUpdate, _, _ := vm.CheckForUpdate()
		if needUpdate {
			updateDialog()
		} else {
			aboutIcon := canvas.NewImageFromResource(resourceIconAbout128Png)
			infoIcon := container.New(layout.NewGridWrapLayout(fyne.NewSize(18, 18)), aboutIcon)
			infoIconForm := container.New(layout.NewVBoxLayout(), layout.NewSpacer(), infoIcon, layout.NewSpacer())
			infoLabel := canvas.NewText("No need to update. You are using the latest version: V"+vm.GetVersion(), color.Black)
			infoLabel.TextSize = 18.0
			infoLabel.Alignment = fyne.TextAlignLeading
			infoMessage := container.New(layout.NewHBoxLayout(), infoIconForm, infoLabel)
			layout := container.New(layout.NewVBoxLayout(), infoMessage)

			keysDialog := dialog.NewCustom("Information", "OK", layout, globals.GUIAPP.Win)
			keysDialog.Show()
		}
	})
	updateButton.Importance = widget.SuccessImportance
	updateCheckForm := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), updateButton, layout.NewSpacer())

	layout := container.New(layout.NewVBoxLayout(), layout.NewSpacer(), appHeader, contactForm, updateCheckForm, layout.NewSpacer())

	return layout
}
