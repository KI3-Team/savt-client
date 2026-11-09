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
	"savt-client/savt-client-api/utils"
	"savt-client/savt-client-gui/globals"
	"savt-client/savt-client-gui/worker"
	"strings"

	"fyne.io/fyne/v2/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

// main is the entry point of the application, responsible for initializing the app and launching the main window
func main() {
	// 1. Initialize logger
	logger := utils.LoggerFactoryInstance.GetGUILogger()
	//logger.SetLevel(utils.Debug)
	logger.SetConsole(false)
	globals.GUIAPP.Logger = logger
	// 2. Initialize gRPC proxy
	serverAddr := "127.0.0.1:40001"
	proxy, _ := worker.NewGRPCProxy(serverAddr)
	globals.GUIAPP.Proxy = proxy
	defer globals.GUIAPP.Proxy.Close()
	// 3. Initialize app
	app := app.New()
	globals.GUIAPP.App = app
	app.Settings().SetTheme(theme.LightTheme()) //nolint
	app.SetIcon(resourceIconApp1024Png)
	vm, err := utils.GetServiceManagerIns()
	if err != nil {
		logger.Error("Error getting VersionManager instance: %v", err)
		return
	}
	win := app.NewWindow(strings.Join([]string{"SAV-T Client V", vm.GetVersion()}, ""))
	win.Resize(fyne.NewSize(1280, 720))
	globals.GUIAPP.Win = win
	showtabs(win)
	//needUpdate, _ := vm.CheckForUpdate()
	//if needUpdate {
	//	updateDialog()
	//}
	app.Run()
}
