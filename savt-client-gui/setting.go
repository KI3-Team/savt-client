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
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// showValidationErrorDialog displays a validation error dialog
func showValidationErrorDialog(title, message string, win fyne.Window) {
	dialog.ShowCustom(
		title,
		"Close",
		container.NewVBox(
			widget.NewLabel(message),
		),
		win,
	)
}

// createSettingPage creates the settings page
func createSettingPage(wm *worker.WorkerManager, win fyne.Window) (*fyne.Container, func()) {
	loadingLabel := widget.NewLabel("Loading...")
	contentContainer := container.NewVBox(loadingLabel)

	loadContent := func() {
		go func() {
			config, err := globals.GUIAPP.Proxy.GetConfig()
			if err != nil {
				contentContainer.Objects = []fyne.CanvasObject{
					widget.NewLabel("Failed to load configuration, please check if the external process is running"),
				}
				contentContainer.Refresh()
				return
			}

			measurementIntervalEntry := widget.NewEntry()
			measurementIntervalEntry.SetText(strconv.Itoa(int(config.CompleteInterval)))

			retryIntervalEntry := widget.NewEntry()
			retryIntervalEntry.SetText(strconv.Itoa(int(config.IncompleteRetryInterval)))

			networkCheckIntervalEntry := widget.NewEntry()
			networkCheckIntervalEntry.SetText(strconv.Itoa(int(config.CheckNetworkInterval)))

			waitAfterNetworkChangeEntry := widget.NewEntry()
			waitAfterNetworkChangeEntry.SetText(strconv.Itoa(int(config.WaitAfterChangeInterval)))

			maxRetryLimitEntry := widget.NewEntry()
			maxRetryLimitEntry.SetText(strconv.Itoa(int(config.RetryLimit)))

			publishResultsCheck := widget.NewCheck("", func(checked bool) {
				config.IsPublic = checked
			})
			publishResultsCheck.SetChecked(config.IsPublic)

			enableSchedulerrCheck := widget.NewCheck("", func(checked bool) {
				config.EnableScheduler = checked
			})
			enableSchedulerrCheck.SetChecked(config.EnableScheduler)

			saveButton := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
				if val, err := strconv.Atoi(measurementIntervalEntry.Text); err != nil || val <= 0 {
					showValidationErrorDialog(
						"Validation Error",
						"Invalid value for 'Complete Interval'. It must be a positive number greater than 0.",
						win,
					)
					return
				} else {
					config.CompleteInterval = int32(val)
				}

				if val, err := strconv.Atoi(retryIntervalEntry.Text); err != nil || val <= 0 {
					showValidationErrorDialog(
						"Validation Error",
						"Invalid value for 'Incomplete Retry Interval'. It must be a positive number greater than 0.",
						win,
					)
					return
				} else {
					config.IncompleteRetryInterval = int32(val)
				}

				if val, err := strconv.Atoi(networkCheckIntervalEntry.Text); err != nil || val <= 60 {
					showValidationErrorDialog(
						"Validation Error",
						"Invalid value for 'Network Check Interval'. It must be greater than 60.",
						win,
					)
					return
				} else {
					config.CheckNetworkInterval = int32(val)
				}

				if val, err := strconv.Atoi(waitAfterNetworkChangeEntry.Text); err != nil || val <= 0 {
					showValidationErrorDialog(
						"Validation Error",
						"Invalid value for 'Wait After Change Interval'. It must be a positive number greater than 0.",
						win,
					)
					return
				} else {
					config.WaitAfterChangeInterval = int32(val)
				}

				if val, err := strconv.Atoi(maxRetryLimitEntry.Text); err != nil || val <= 1 {
					showValidationErrorDialog(
						"Validation Error",
						"Invalid value for 'Retry Limit'. It must be greater than 1.",
						win,
					)
					return
				} else {
					config.RetryLimit = int32(val)
				}

				_, saveErr := globals.GUIAPP.Proxy.SetConfig(config)
				if saveErr != nil {
					dialog.ShowCustom(
						"Save Failed",
						"Close",
						container.NewVBox(
							widget.NewLabel("Failed to save the configuration. Please try again."),
							widget.NewLabel(fmt.Sprintf("Error: %v", saveErr)),
						),
						win,
					)
				} else {
					customDialog := dialog.NewCustom(
						"",
						"Close",
						widget.NewLabel("Configuration has been successfully saved!"),
						win,
					)
					customDialog.Show()
					time.AfterFunc(1*time.Second, func() {
						customDialog.Hide()
					})
				}
			})
			saveButton.Importance = widget.HighImportance
			fixedWidthSaveButton := container.NewGridWrap(
				fyne.NewSize(100, saveButton.MinSize().Height),
				saveButton,
			)
			rightAlignedSaveButton := container.NewHBox(
				layout.NewSpacer(),
				fixedWidthSaveButton,
			)
			form := container.New(layout.NewFormLayout(),
				widget.NewLabel("Publish Results"), publishResultsCheck,
				widget.NewLabel("Enable Scheduler"), enableSchedulerrCheck,
				widget.NewLabel("Wait to check for a network change(seconds)"), networkCheckIntervalEntry,
				widget.NewLabel("Wait to run a test after detecting a network change(seconds)"), waitAfterNetworkChangeEntry,
				widget.NewLabel("Wait to run a test after a successful run on the same network(seconds)"), measurementIntervalEntry,
				widget.NewLabel("Wait to retry after first incomplete run(seconds)(double each time)"), retryIntervalEntry,
				widget.NewLabel("Maximum number of retries after an incomplete run"), maxRetryLimitEntry,
			)
			finalContent := container.NewVBox(
				widget.NewLabelWithStyle("Settings:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewSeparator(),
				form, rightAlignedSaveButton,
				widget.NewSeparator(),
			)
			contentContainer.Objects = []fyne.CanvasObject{
				container.NewGridWithColumns(2,
					finalContent,
				),
			}
			contentContainer.Refresh()
		}()
	}
	return contentContainer, loadContent
}
