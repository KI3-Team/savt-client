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
	savt "savt-client/savt-client-api/savt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// taskStatusRow defines a single task row component
type taskStatusRow struct {
	widget.BaseWidget
	icons     map[savt.Status]*canvas.Image
	label     *widget.Label
	container *fyne.Container
	current   savt.Status
}

// newTaskStatusRow creates a single task status row
func newTaskStatusRow(task *savt.Task) *taskStatusRow {
	icons := map[savt.Status]*canvas.Image{
		savt.Status_INITIAL:        canvas.NewImageFromResource(resourceIconTime256Png),
		savt.Status_RUNNING:        canvas.NewImageFromResource(resourceIconRunning256Png),
		savt.Status_SUCCEEDED:      canvas.NewImageFromResource(resourceIconCheck256Png),
		savt.Status_FAILED:         canvas.NewImageFromResource(resourceIconError256Png),
		savt.Status_CANCELLED:      canvas.NewImageFromResource(resourceIconError256Png),
		savt.Status_SEMI_SUCCEEDED: canvas.NewImageFromResource(resourceIconCheck256Png),
	}

	for _, icon := range icons {
		icon.SetMinSize(fyne.NewSize(24, 24))
		icon.FillMode = canvas.ImageFillContain
		icon.Hide()
	}
	icons[task.Status].Show()

	label := widget.NewLabel(task.StatusDesc)
	label.Alignment = fyne.TextAlignLeading

	iconContainer := container.NewStack(
		icons[savt.Status_INITIAL],
		icons[savt.Status_RUNNING],
		icons[savt.Status_SUCCEEDED],
		icons[savt.Status_FAILED],
		icons[savt.Status_CANCELLED],
		icons[savt.Status_SEMI_SUCCEEDED],
	)

	rowContainer := container.New(
		layout.NewHBoxLayout(),
		iconContainer,
		label,
	)

	row := &taskStatusRow{
		icons:     icons,
		label:     label,
		container: rowContainer,
		current:   task.Status,
	}
	row.ExtendBaseWidget(row)

	return row
}

// CreateRenderer implements custom renderer
func (row *taskStatusRow) CreateRenderer() fyne.WidgetRenderer {
	return &taskStatusRowRenderer{
		container: row.container,
		objects:   []fyne.CanvasObject{row.container},
	}
}

// taskStatusRowRenderer renderer
type taskStatusRowRenderer struct {
	container *fyne.Container
	objects   []fyne.CanvasObject
}

func (r *taskStatusRowRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

func (r *taskStatusRowRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

func (r *taskStatusRowRenderer) Refresh() {
	r.container.Refresh()
}

func (r *taskStatusRowRenderer) Destroy() {}

func (r *taskStatusRowRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

// TaskGroupWidget represents a task group component
type TaskGroupWidget struct {
	widget.BaseWidget
	tasks     []*savt.Task
	taskRows  []*taskStatusRow
	container *fyne.Container
}

// NewTaskGroupWidget creates a new task group component
func NewTaskGroupWidget() *TaskGroupWidget {
	w := &TaskGroupWidget{
		tasks:     []*savt.Task{},
		taskRows:  []*taskStatusRow{},
		container: container.NewVBox(),
	}
	w.ExtendBaseWidget(w)
	return w
}

// SetList sets the task list in the task group
func (w *TaskGroupWidget) SetList(tasks []*savt.Task) {
	w.tasks = tasks
	w.taskRows = make([]*taskStatusRow, 0, len(tasks))

	newObjects := make([]fyne.CanvasObject, 0, len(tasks))
	for _, task := range tasks {
		row := newTaskStatusRow(task)
		w.taskRows = append(w.taskRows, row)
		newObjects = append(newObjects, row)
	}

	w.container.Objects = newObjects
	w.container.Refresh()
}

// Clear clears all tasks in the task group
func (w *TaskGroupWidget) Clear() {
	w.tasks = []*savt.Task{}
	w.taskRows = []*taskStatusRow{}
	w.container.Objects = nil
	w.Refresh()
}

// CreateRenderer implements custom renderer
func (w *TaskGroupWidget) CreateRenderer() fyne.WidgetRenderer {
	return &taskGroupRenderer{
		container: w.container,
		objects:   []fyne.CanvasObject{w.container},
	}
}

// taskGroupRenderer renderer
type taskGroupRenderer struct {
	container *fyne.Container
	objects   []fyne.CanvasObject
}

func (r *taskGroupRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

func (r *taskGroupRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

func (r *taskGroupRenderer) Refresh() {
	r.container.Refresh()
}

func (r *taskGroupRenderer) Destroy() {}

func (r *taskGroupRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
