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
	"savt-client/savt-client-gui/globals"
	"savt-client/savt-client-gui/widgets"
	"savt-client/savt-client-gui/worker"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// createHistoryPage creates the history page
func createHistoryPage(wm *worker.WorkerManager, win fyne.Window) (*fyne.Container, func()) {
	headers := []string{"Date", "Type", "IPv", "Client Address", "ASN", "Outbound Private", "Outbound Routable", "Inbound Private", "Inbound Internal", "Menu"}
	columnWidths := map[int]int{
		0: 140,
		1: 87,
		2: 20,
		3: 270,
		4: 45,
		5: 100,
		6: 120,
		7: 100,
		8: 100,
		9: 138,
	}
	table := widgets.NewCustomTable(headers, columnWidths)
	loadHistoryData := func() {
		logger = globals.GUIAPP.Logger
		proxy := globals.GUIAPP.Proxy
		historyData, err := proxy.GetHistory()
		if err != nil {
			logger.Error("Failed to fetch table data: %v", err)
			return
		}
		jobs := historyData.Jobs
		var measurementResultWrappers []widgets.MeasurementResultWrapper
		for _, job := range jobs {
			if job.Ipv4.ClientAddress != "" {
				measurementResultWrappers = append(measurementResultWrappers,
					widgets.MeasurementResultWrapper{
						MeasurementResult: job.Ipv4,
						IPVersion:         widgets.IPv4,
						StartTime:         job.StartTime.AsTime(),
						Token:             job.Token,
						JobType:           job.JobType,
					},
				)
			}
			if job.Ipv6.ClientAddress != "" {
				measurementResultWrappers = append(measurementResultWrappers,
					widgets.MeasurementResultWrapper{
						MeasurementResult: job.Ipv6,
						IPVersion:         widgets.IPv6,
						StartTime:         job.StartTime.AsTime(),
						Token:             job.Token,
						JobType:           job.JobType,
					},
				)
			}
		}
		table.SetData(measurementResultWrappers)
	}

	ct_header := container.NewVBox(
		widget.NewLabelWithStyle("Running history", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
	)
	ct := container.NewBorder(
		ct_header,
		nil, nil, layout.NewSpacer(), table,
	)
	return ct, loadHistoryData
}
