package main

import (
	"fmt"
	"image/color"
	"log"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Vladroon22/2FA/internal/core"
	"github.com/Vladroon22/2FA/internal/update"
)

var (
	appVersion string
)

func GetVersion() string {
	return appVersion
}

func isMobile() bool {
	if fyne.CurrentDevice().IsMobile() {
		return true
	}

	return runtime.GOOS == "android" || runtime.GOOS == "ios"
}

// Структура для хранения данных приложения
type AppItem struct {
	Name     string
	Code     string
	Progress float32
	Color    color.NRGBA
	Secret   string
	ID       int
}

func main() {
	v := GetVersion()
	myApp := app.New()
	myApp.Settings().SetTheme(theme.DefaultTheme())

	window := myApp.NewWindow("Custom 2FA")

	titleLabel := canvas.NewText("Your connected apps", color.NRGBA{R: 0, G: 120, B: 255, A: 255})
	titleLabel.TextSize = 28
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	// switcher of theme (Dark/Light)
	themeSwitch := widget.NewRadioGroup([]string{"Dark", "Light"}, func(selected string) {
		switch selected {
		case "Dark":
			myApp.Settings().SetTheme(theme.DarkTheme())
		case "Light":
			myApp.Settings().SetTheme(theme.LightTheme())
		}
		window.Content().Refresh()
	})
	themeSwitch.SetSelected("Dark")
	themeSwitch.Horizontal = true

	var appItems []AppItem
	colors := []color.NRGBA{
		{R: 0, G: 120, B: 255, A: 255},  // Blue
		{R: 255, G: 60, B: 60, A: 255},  // Red
		{R: 0, G: 200, B: 80, A: 255},   // Green
		{R: 160, G: 32, B: 240, A: 255}, // Purple
		{R: 255, G: 140, B: 0, A: 255},  // Orange
	}

	var nextID, selectedID = 1, -1

	appList := widget.NewList(
		func() int {
			return len(appItems)
		},
		func() fyne.CanvasObject {
			colorRect := canvas.NewRectangle(color.White)
			colorRect.SetMinSize(fyne.NewSize(20, 20))

			appLabel := widget.NewLabel("App Name")
			appLabel.TextStyle = fyne.TextStyle{Bold: true}

			codeLabel := widget.NewLabel("••••••")
			codeLabel.TextStyle = fyne.TextStyle{Monospace: true}

			progressBar := widget.NewProgressBar()
			progressBar.Max = 30

			return container.NewHBox(
				colorRect,
				container.NewVBox(
					appLabel,
					codeLabel,
					progressBar,
				),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= len(appItems) {
				return
			}

			cont := o.(*fyne.Container)
			colorRect := cont.Objects[0].(*canvas.Rectangle)
			innerCont := cont.Objects[1].(*fyne.Container)

			appLabel := innerCont.Objects[0].(*widget.Label)
			codeLabel := innerCont.Objects[1].(*widget.Label)
			progressBar := innerCont.Objects[2].(*widget.ProgressBar)

			item := appItems[i]
			colorRect.FillColor = item.Color
			appLabel.SetText(item.Name)
			codeLabel.SetText(item.Code)
			progressBar.SetValue(float64(item.Progress))
			colorRect.Refresh()
		},
	)

	appList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(appItems) {
			selectedID = appItems[id].ID
			log.Printf("Selected app: %s (ID: %d)", appItems[id].Name, selectedID)
		}
	}

	go func() {
		for {
			time.Sleep(time.Second)

			fyne.DoAndWait(func() {
				now := time.Now()

				for i := range appItems {
					remainingSeconds := 30 - (now.Unix() % 30)

					progress := 30 - float32(remainingSeconds)
					appItems[i].Progress = progress

					if remainingSeconds == 30 {
						code, err := core.GenerateOTP(now, appItems[i].Secret)
						if err == nil {
							appItems[i].Code = code
						} else {
							appItems[i].Code = "ERROR"
							log.Printf("Error generating OTP for %s: %v", appItems[i].Name, err)
						}
					}
				}

				appList.Refresh()
			})
		}
	}()

	removeButton := widget.NewButton("Remove App", func() {
		if selectedID == -1 {
			dialog.ShowInformation("No Selection", "Please select an app to remove", window)
			return
		}

		var indexToRemove = -1
		for i, item := range appItems {
			if item.ID == selectedID {
				indexToRemove = i
				break
			}
		}

		if indexToRemove != -1 {
			dialog.ShowConfirm("Remove App", "Are you sure you want to remove '"+appItems[indexToRemove].Name+"'?",
				func(confirmed bool) {
					if confirmed {
						appItems = append(appItems[:indexToRemove], appItems[indexToRemove+1:]...)
						selectedID = -1
						appList.Refresh()
					}
				}, window)
		}
	})

	addButton := widget.NewButtonWithIcon("Add App", theme.ContentAddIcon(), func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Enter app name")
		nameEntry.Validator = func(s string) error {
			if len(s) == 0 {
				return fmt.Errorf("app name cannot be empty")
			}
			return nil
		}

		secretEntry := widget.NewEntry()
		secretEntry.SetPlaceHolder("Enter secret key (Base32)")
		secretEntry.Validator = func(s string) error {
			if len(s) == 0 {
				return fmt.Errorf("secret key cannot be empty")
			}

			return nil
		}

		items := []*widget.FormItem{
			widget.NewFormItem("App Name", nameEntry),
			widget.NewFormItem("Secret Key", secretEntry),
		}

		dialog.ShowForm("Add New App", "Add", "Cancel", items, func(confirmed bool) {
			if !confirmed {
				return
			}

			if nameEntry.Validate() != nil {
				dialog.ShowError(fmt.Errorf("invalid app name"), window)
				return
			}

			if secretEntry.Validate() != nil {
				dialog.ShowError(fmt.Errorf("invalid secret"), window)
				return
			}

			now := time.Now()
			code, err := core.GenerateOTP(now, secretEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid secret key: %v", err), window)
				return
			}

			newItem := AppItem{
				Name:     nameEntry.Text,
				Code:     code,
				Progress: 0.0,
				Color:    colors[len(appItems)%len(colors)],
				Secret:   secretEntry.Text,
				ID:       nextID,
			}
			nextID++

			appItems = append(appItems, newItem)
			appList.Refresh()

			dialog.ShowInformation("Success", "App added successfully!", window)
		}, window)
	})

	copyButton := widget.NewButtonWithIcon("Copy Code", theme.ContentCopyIcon(), func() {
		if selectedID == -1 {
			dialog.ShowInformation("No Selection", "Please select an app to copy code", window)
			return
		}

		var selectedItem *AppItem
		for i, item := range appItems {
			if item.ID == selectedID {
				selectedItem = &appItems[i]
				break
			}
		}

		if selectedItem != nil {
			myApp.Clipboard().SetContent(selectedItem.Code)
			dialog.ShowInformation("Copied", "Code copied to clipboard", window)
		}
	})

	IsUpToDate := widget.NewButton("Check for Updates", func() {
		if err := update.Fetch(v); err != nil {
			dialog.ShowInformation("Result of checking", err.Error(), window)
		} else {
			myApp.Quit()
		}
	})

	settingsContainer := container.NewVBox(
		widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		themeSwitch,
	)

	actionsContainer := container.NewHBox(
		addButton,
		removeButton,
		copyButton,
		IsUpToDate,
	)

	mainContainer := container.NewBorder(
		container.NewVBox( // top
			container.NewCenter(titleLabel),
			widget.NewSeparator(),
			container.NewCenter(actionsContainer),
		),
		container.NewVBox( // bottom
			widget.NewSeparator(),
			container.NewCenter(settingsContainer),
		),
		nil,     // left
		nil,     // right
		appList, // center
	)

	window.SetContent(mainContainer)

	if !isMobile() {
		window.Resize(fyne.NewSize(800, 600))
		window.CenterOnScreen()
		window.SetFixedSize(false)
	} else {
		window.SetFullScreen(true)
	}

	window.ShowAndRun()
}
