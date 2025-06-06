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
	"savt-client/savt-client-api/savt"
	"savt-client/savt-client-api/utils"
	"savt-client/savt-client-gui/globals"
	"savt-client/savt-client-gui/widgets"
	"savt-client/savt-client-gui/worker"
	"strings"
	"time"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/golang/protobuf/ptypes/timestamp"
)

var stopChan chan bool
var jobId *string
var progressBar *widgets.ProgressBarWidget
var taskStatusGroup *widgets.TaskGroupWidget
var jobWidget *widgets.JobWidget
var logPanel *widgets.LogWidget
var runButton *widget.Button
var disableRunLabel *widget.Label
var logger = globals.GUIAPP.Logger

// startRunProcess starts the measurement task and handles related status updates
func startRunProcess() {
	if stopChan != nil {
		close(stopChan)
	}
	stopChan = make(chan bool)

	proxy := globals.GUIAPP.Proxy
	job, err := proxy.Start()
	if err != nil {
		globals.GUIAPP.Logger.Error("Task failed to start: %v", err)
		select {
		case stopChan <- true:
		default:
		}
		return
	}
	jobId = &job.JobId
	if err := saveJobId(*jobId); err != nil {
		globals.GUIAPP.Logger.Error("Failed to save jobId: %v", err)
	}
	runButton.Hide()
	disableRunLabel.Show()
	logPanel.BindLogStream(*jobId)
	status := updateJobStatus(*jobId)
	if status == savt.Status_RUNNING {
		go periodicUpdate(stopChan)
	} else {
		select {
		case stopChan <- true:
		default:
		}
	}
}

// formatTimestamp converts Timestamp to readable format
func formatTimestamp(ts *timestamp.Timestamp) string {
	if ts == nil {
		return "N/A"
	}
	t := time.Unix(ts.Seconds, int64(ts.Nanos))
	return t.Format("2006-01-02 15:04:05")
}

// printJobStatus outputs job status information
func printJobStatus(job *savt.Job) {
	logger = globals.GUIAPP.Logger

	logger.Debug("\n===== Job Status =====")
	logger.Debug("Job ID: %s", job.GetJobId())
	logger.Debug("Start Time: %s", formatTimestamp(job.GetStartTime()))
	logger.Debug("Duration: %d seconds", job.GetDurationSeconds())
	logger.Debug("Job Status: %s", job.GetStatus().String())
	logger.Debug("Progress: %d%%", job.GetProgressBar())
	logger.Debug("Token: %s", job.GetToken())
	logger.Debug("=======================\n")

	logger.Debug("===== Task Status =====")
	for _, task := range job.GetTasks() {
		logger.Debug("Step: %-2d | Status: %-15s | Description: %-30s",
			task.GetStep(),
			task.GetStatus().String(),
			task.GetStatusDesc())
	}
	logger.Debug("=======================\n")

	logger.Debug("===== IPv4 Measurement Results =====")
	printMeasurementResult(job.GetIpv4(), logger)
	logger.Debug("=======================\n")

	logger.Debug("===== IPv6 Measurement Results =====")
	printMeasurementResult(job.GetIpv6(), logger)
	logger.Debug("=======================\n")

	logger.Debug("============================================================================================")
}

// printMeasurementResult outputs measurement result details
func printMeasurementResult(result *savt.MeasurementResult, logger utils.Logger) {
	if result == nil {
		logger.Debug("Measurement result is not available.")
		return
	}

	logger.Debug("Client Address: %s", result.GetClientAddress())
	logger.Debug("ASN: %d", result.GetAsn())
	logger.Debug("Outbound Private: %s", result.GetOutboundPrivate().String())
	logger.Debug("Outbound Routable: %s", result.GetOutboundRoutable().String())
	logger.Debug("Spoofable Prefix Length: %d", result.GetSpoofablePrefixLength())
	logger.Debug("Inbound Private: %s", result.GetInboundPrivate().String())
	logger.Debug("Inbound Internal: %s", result.GetInboundInternal().String())
}

// updateJobStatus updates job status and UI display
func updateJobStatus(jobId string) savt.Status {
	proxy := globals.GUIAPP.Proxy
	jobIdRequest := &savt.JobIdRequest{JobId: jobId}
	job, err := proxy.GetStatus(jobIdRequest)
	if err != nil {
		globals.GUIAPP.Logger.Error("Failed to retrieve job status for jobId %s: %v", jobId, err)
		return savt.Status_API_UNKNOWN
	}
	progressBar.SetProgress(int(job.GetProgressBar()), job.Status)
	taskStatusGroup.SetList(job.GetTasks())
	printJobStatus(job)
	switch job.Status {
	case savt.Status_RUNNING:
		fyne.Do(func() {
			runButton.Hide()
			disableRunLabel.Show()
		})
	default:
		fyne.Do(func() {
			runButton.Show()
			disableRunLabel.Hide()
		})
	}
	jobWidget.SetJob(job)
	return job.Status
}

// periodicUpdate periodically updates job status
func periodicUpdate(stopChan chan bool) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if len(strings.TrimSpace(*jobId)) > 0 {
				status := updateJobStatus(*jobId)
				if status != savt.Status_RUNNING {
					logger.Info("Job finished. Stopping periodic updates.")
					select {
					case stopChan <- true:
					default:
					}
					return
				}
			}
		case <-stopChan:
			logger.Info("Stopping periodic updates.")
			return
		}
	}
}

func createTitle(text string) *canvas.Text {
	title := canvas.NewText(text, color.RGBA{0, 0, 0, 255})
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16
	return title
}

// createHomePage creates the main page layout
func createHomePage(_ *worker.WorkerManager, win fyne.Window) fyne.CanvasObject {
	taskStatusGroup = widgets.NewTaskGroupWidget()
	stepStatusLayout := container.NewVBox(
		createTitle("Measurement Process"),
		widget.NewSeparator(),
		taskStatusGroup,
	)
	jobWidget = widgets.NewJobWidget(nil)
	resultSummaryLayout := container.NewVBox(
		createTitle("Measurement Results"),
		widget.NewSeparator(),
		jobWidget,
	)
	progressBar = widgets.NewProgressBarWidget()
	runButton = widget.NewButton(" RUN ", func() {
		jobId = nil
		progressBar.SetProgress(0, savt.Status_INITIAL)
		taskStatusGroup.Clear()
		jobWidget.Clear()
		startRunProcess()
	})
	runButton.Importance = widget.SuccessImportance

	disableRunLabel = widget.NewLabel("Running...")
	disableRunLabel.TextStyle = fyne.TextStyle{Bold: true}
	disableRunLabel.Alignment = fyne.TextAlignCenter
	disableRunLabel.Hide()
	actionButtonLayout := container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(runButton, disableRunLabel),
		layout.NewSpacer(),
	)
	progressActionButtonLayout_0 := container.NewVBox(
		progressBar,
		actionButtonLayout,
	)
	progressActionButtonLayout := container.NewVBox(
		createTitle("Console"),
		widget.NewSeparator(),
		progressActionButtonLayout_0,
	)
	// Main container, using horizontal layout + Spacer to simulate proportions
	size := float32(1280 - 150)
	topLayout := container.New(layout.NewHBoxLayout(),
		container.New(layout.NewGridWrapLayout(fyne.NewSize(0.25*size, 0)), progressActionButtonLayout),
		layout.NewSpacer(),
		container.New(layout.NewGridWrapLayout(fyne.NewSize(0.25*size, 0)), stepStatusLayout),
		layout.NewSpacer(),
		container.New(layout.NewGridWrapLayout(fyne.NewSize(0.5*size, 0)), resultSummaryLayout),
	)
	logPanel = widgets.NewLogWidget("")
	consoleLogTitle := createTitle("Log")
	downloadLink := widget.NewHyperlink("Download", nil)
	downloadLink.TextStyle = fyne.TextStyle{}
	downloadLink.OnTapped = func() {
		if logPanel != nil {
			logContent := logPanel.GetLogContent()
			if len(logContent) > 0 {
				fileDialog := dialog.NewFileSave(
					func(uc fyne.URIWriteCloser, err error) {
						if err == nil && uc != nil {
							uc.Write([]byte(logContent))
							uc.Close()
						}
					},
					win,
				)
				now := time.Now()
				filename := now.Format("savt_client_log_2006-01-02_15:04.txt")
				fileDialog.SetFileName(filename)
				fileDialog.Show()
			}
		}
	}

	logPanelSection := container.NewVBox(
		container.NewHBox(
			consoleLogTitle,
			layout.NewSpacer(),
			downloadLink,
		),
		container.NewVBox(logPanel),
	)
	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(0, 24))
	mainContent := container.NewVBox(
		topLayout,
		layout.NewSpacer(),
		gap,
		layout.NewSpacer(),
		logPanelSection,
	)
	loadedJobId, err := loadJobId()
	if err != nil {
		globals.GUIAPP.Logger.Error("Failed to load jobId: %v", err)
	}
	if len(loadedJobId) > 0 {
		jobId = &loadedJobId
		logPanel.BindLogStream(*jobId)
		status := updateJobStatus(*jobId)
		if status == savt.Status_RUNNING {
			go periodicUpdate(stopChan)
		} else {
			select {
			case stopChan <- true:
			default:
			}
		}
	}
	return mainContent
}
