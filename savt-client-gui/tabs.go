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
	"fmt"
	"savt-client/savt-client-gui/globals"
	"savt-client/savt-client-gui/worker"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// showtabs creates and displays the main interface tabs
func showtabs(win fyne.Window) {
	logger := globals.GUIAPP.Logger
	wm := worker.NewWorkerManager(*globals.GUIAPP.Proxy)

	historyPage, lodingHistoryData := createHistoryPage(wm, win)
	settingPage, loadSettingContent := createSettingPage(wm, win)

	// Create Tabs
	homeTab := container.NewTabItemWithIcon("       Home       ", resourceIconHome128Png, createHomePage(wm, win))
	historyTab := container.NewTabItemWithIcon("      History     ", resourceIconHistory128Png, historyPage)
	settingTab := container.NewTabItemWithIcon("      Setting     ", resourceIconSetting128Png, settingPage)
	aboutTab := container.NewTabItemWithIcon("       About      ", resourceIconAbout128Png, createAboutPage())

	tabs := container.NewAppTabs(homeTab, historyTab, settingTab, aboutTab)
	currentTab := homeTab
	tabs.Select(currentTab)
	tabs.OnSelected = func(tab *container.TabItem) {
		switch tab {
		case settingTab:
			if !wm.IsRunning() {
				dialog.ShowInformation("Notice", "The external process is not running, unable to access the settings page!", win)
				tabs.Select(currentTab)
				return
			}
			loadSettingContent()
		case historyTab:
			if !wm.IsRunning() {
				dialog.ShowInformation("Notice", "The external process is not running, unable to access the history page!", win)
				tabs.Select(currentTab)
				return
			}
			lodingHistoryData()
		}
		currentTab = tab
	}
	tabs.SetTabLocation(container.TabLocationLeading)

	workerStatus := widget.NewLabel("Worker status: Unknown")
	maxRetries := 3
	retryCount := 0
	go func() {
		for {
			if !wm.IsRunning() {
				if retryCount < maxRetries {
					err := wm.Start()
					if err != nil {
						logger.Error("Error starting worker:", err)
						retryCount++
						fyne.Do(func() {
							workerStatus.SetText(fmt.Sprintf("Worker Status: failed to start (%d/%d)", retryCount, maxRetries))
						})
					} else {
						fyne.Do(func() {
							workerStatus.SetText("Worker status: Running")
						})
						retryCount = 0
					}
				} else {
					fyne.Do(func() {
						workerStatus.SetText("Worker status: failed to start (max retries reached)")
					})
				}
			} else {
				fyne.Do(func() {
					workerStatus.SetText("Worker status: Running")
				})
			}
			time.Sleep(5 * time.Second)
		}
	}()

	footer := container.NewHBox(
		workerStatus,
	)
	footerWrapper := container.NewStack(
		container.NewVBox(
			footer,
		),
	)
	footerWrapper.Resize(fyne.NewSize(800, 30))
	content := container.NewBorder(
		nil,
		footerWrapper,
		nil,
		nil,
		tabs,
	)

	win.SetContent(content)
	win.Show()
}
