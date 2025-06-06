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
	"context"
	"io"
	"savt-client/savt-client-api/savt"
	"savt-client/savt-client-gui/globals"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LogWidget represents a custom log widget with auto-scroll and stream support
type LogWidget struct {
	widget.BaseWidget
	logEntry     *widget.Entry
	scroll       *container.Scroll
	buffer       []string
	mutex        sync.Mutex
	autoScroll   bool
	cancelFunc   context.CancelFunc
	streamActive bool
	maxLines     int
}

const maxLineLength = 200

// NewLogWidget creates a new LogWidget
func NewLogWidget(defaultText string) *LogWidget {
	logEntry := widget.NewMultiLineEntry()
	logEntry.SetText(defaultText)

	scroll := container.NewScroll(logEntry)
	scroll.SetMinSize(fyne.NewSize(fyne.CurrentApp().Settings().Scale()*400, 200))

	w := &LogWidget{
		logEntry:   logEntry,
		scroll:     scroll,
		autoScroll: true,
		buffer:     make([]string, 0),
		maxLines:   1000,
	}

	w.ExtendBaseWidget(w)
	return w
}

func (w *LogWidget) AppendLog(lines []string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for i, line := range lines {
		if len(line) > maxLineLength {
			lines[i] = line[:maxLineLength] + "..."
		}
	}

	w.buffer = append(w.buffer, lines...)

	if len(w.buffer) > w.maxLines {
		w.buffer = w.buffer[len(w.buffer)-w.maxLines:]
	}

	w.refreshLog()
}

// refreshLog refreshes the log content
func (w *LogWidget) refreshLog() {
	text := strings.Join(w.buffer, "\n")
	fyne.Do(func() {
		w.logEntry.SetText(text)
		w.logEntry.Refresh()
		if w.autoScroll {
			w.scroll.ScrollToBottom()
			w.scroll.Refresh()
		}
	})
}

// ClearLog clears the log
func (w *LogWidget) ClearLog() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.buffer = make([]string, 0)
	w.refreshLog()
}

func (w *LogWidget) EnableAutoScroll(enable bool) {
	w.autoScroll = enable
}

// BindLogStream binds the log stream to the widget
func (w *LogWidget) BindLogStream(jobID string) {
	w.StopLogStream()
	ctx, cancel := context.WithCancel(context.Background())
	w.cancelFunc = cancel
	w.streamActive = true
	go func() {
		defer func() {
			w.streamActive = false
		}()
		stream, err := globals.GUIAPP.Proxy.ReadLog(ctx, &savt.JobIdRequest{JobId: jobID})
		if err != nil {
			if err == io.EOF {
				return
			}
			globals.GUIAPP.Logger.Error("Error reading log stream: %v", err)
			return
		}
		for {
			entry, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				globals.GUIAPP.Logger.Error("Error reading log stream: %v", err)
				return
			}
			lines := entry.GetLine()
			w.AppendLog(lines)
		}
	}()
}

// StopLogStream stops the current log stream binding
func (w *LogWidget) StopLogStream() {
	if w.cancelFunc != nil {
		w.cancelFunc()
		w.cancelFunc = nil
		w.streamActive = false
	}
}

// CreateRenderer creates the renderer for the LogWidget
func (w *LogWidget) CreateRenderer() fyne.WidgetRenderer {
	content := container.NewStack(w.scroll)
	return widget.NewSimpleRenderer(content)
}

func (w *LogWidget) GetLogContent() string {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return strings.Join(w.buffer, "\n")
}
