package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ap-andersson/tunnelvision/internal/config"
	"github.com/ap-andersson/tunnelvision/internal/tunnel"
)

// App holds all UI state and dependencies.
type App struct {
	FyneApp       fyne.App
	Window        fyne.Window
	Store         *config.Store
	Manager       *tunnel.Manager
	Status        *tunnel.Status
	TunnelList    *TunnelList
	Editor        *ConfigEditor
	Tray          *Tray
	connectBtn    *widget.Button
	disconnectBtn *widget.Button
	deleteBtn     *widget.Button
}

// NewApp creates the main application UI.
func NewApp(fyneApp fyne.App, store *config.Store, manager *tunnel.Manager, status *tunnel.Status) *App {
	a := &App{
		FyneApp: fyneApp,
		Store:   store,
		Manager: manager,
		Status:  status,
	}

	a.Window = fyneApp.NewWindow("TunnelVision")
	a.Window.Resize(fyne.NewSize(900, 650))

	a.Editor = NewConfigEditor(a)
	a.TunnelList = NewTunnelList(a)
	a.Tray = NewTray(a)

	toolbar := a.createToolbar()

	split := container.NewHSplit(
		a.TunnelList.Widget(),
		a.Editor.Widget(),
	)
	split.SetOffset(0.3)

	content := container.NewBorder(toolbar, nil, nil, nil, split)
	a.Window.SetContent(content)

	// Close to tray instead of quitting
	a.Window.SetCloseIntercept(func() {
		a.Window.Hide()
	})

	// Wire up state change callback
	a.Manager.SetOnStateChange(func() {
		fyne.Do(func() {
			a.TunnelList.Refresh()
			a.Tray.Update()
			a.updateConnectButtons()
		})
	})

	// Initial data load
	a.TunnelList.Refresh()
	a.Tray.Setup()
	a.updateConnectButtons()
	a.updateSelectionButtons()

	return a
}

func (a *App) createToolbar() fyne.CanvasObject {
	importFileBtn := widget.NewButtonWithIcon("Import File", theme.FolderOpenIcon(), func() {
		a.showImportDialog()
	})
	importFolderBtn := widget.NewButtonWithIcon("Import Folder", theme.DownloadIcon(), func() {
		a.ShowMultiImportDialog()
	})
	newConfigBtn := widget.NewButtonWithIcon("New Tunnel", theme.DocumentCreateIcon(), func() {
		a.showNewConfigDialog()
	})
	newFolderBtn := widget.NewButtonWithIcon("New Folder", theme.FolderNewIcon(), func() {
		a.showNewFolderDialog()
	})
	a.deleteBtn = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		a.deleteCurrent()
	})
	a.deleteBtn.Disable()
	a.connectBtn = widget.NewButtonWithIcon("Connect", theme.MediaPlayIcon(), func() {
		a.connectSelected()
	})
	a.connectBtn.Importance = widget.SuccessImportance
	a.disconnectBtn = widget.NewButtonWithIcon("Disconnect", theme.MediaStopIcon(), func() {
		a.disconnectTunnel()
	})
	a.disconnectBtn.Importance = widget.DangerImportance

	return container.NewHBox(
		importFileBtn,
		importFolderBtn,
		newConfigBtn,
		newFolderBtn,
		a.deleteBtn,
		layout.NewSpacer(),
		a.connectBtn,
		a.disconnectBtn,
	)
}

func (a *App) updateConnectButtons() {
	if a.Manager.IsConnected() {
		a.connectBtn.Hide()
		a.disconnectBtn.Show()
	} else {
		a.connectBtn.Show()
		a.disconnectBtn.Hide()
	}
}

// updateSelectionButtons enables or disables buttons that depend on having a selection.
func (a *App) updateSelectionButtons() {
	hasSelection := a.TunnelList.SelectedFilename() != "" || a.TunnelList.SelectedFolder() != ""
	if hasSelection {
		a.deleteBtn.Enable()
		a.connectBtn.Enable()
	} else {
		a.deleteBtn.Disable()
		a.connectBtn.Disable()
	}
}

// refreshEditorIfFolder re-shows folder info with updated count if a folder is selected.
func (a *App) refreshEditorIfFolder() {
	folder := a.TunnelList.SelectedFolder()
	if folder != "" {
		count := len(a.TunnelList.metadata.folders[folder])
		a.Editor.ShowFolderInfo(folder, count)
	}
}

func (a *App) showImportDialog() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		reader.Close()
		// Fyne's file dialog doesn't support multi-select natively.
		// We handle single file at a time, user can repeat.
		// Alternatively, we use our own multi-import approach.
		uri := reader.URI()
		imported, importErr := a.Store.ImportFiles([]string{uri.Path()})
		if importErr != nil {
			a.showErrorDialog(importErr)
			return
		}
		if len(imported) > 0 {
			a.TunnelList.Refresh()
			a.Tray.Update()
			a.refreshEditorIfFolder()
		}
	}, a.Window)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".conf"}))
	fd.Show()
}

// ShowMultiImportDialog opens a folder picker and imports all .conf files to root (ungrouped).
func (a *App) ShowMultiImportDialog() {
	a.importFolderInto("")
}

func (a *App) importFolderInto(targetFolder string) {
	fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		items, listErr := uri.List()
		if listErr != nil {
			a.showErrorDialog(listErr)
			return
		}
		var paths []string
		for _, item := range items {
			if item.Extension() == ".conf" {
				paths = append(paths, item.Path())
			}
		}
		if len(paths) == 0 {
			dialog.ShowInformation("Import", "No .conf files found in the selected folder.", a.Window)
			return
		}
		imported, importErr := a.Store.ImportFilesToFolder(paths, targetFolder)
		if importErr != nil {
			a.showErrorDialog(importErr)
			return
		}
		msg := fmt.Sprintf("Imported %d tunnel(s).", len(imported))
		if targetFolder != "" {
			msg = fmt.Sprintf("Imported %d tunnel(s) into folder %q.", len(imported), targetFolder)
		}
		dialog.ShowInformation("Import", msg, a.Window)
		a.TunnelList.Refresh()
		a.Tray.Update()
		a.refreshEditorIfFolder()
	}, a.Window)
	fd.Show()
}

func (a *App) showNewConfigDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("tunnel-name")

	dialog.ShowForm("New Tunnel Config", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
		},
		func(ok bool) {
			if !ok || nameEntry.Text == "" {
				return
			}
			defaultContent := "[Interface]\nPrivateKey = \nAddress = \nDNS = \n\n[Peer]\nPublicKey = \nEndpoint = \nAllowedIPs = 0.0.0.0/0\n"
			if err := a.Store.AddConfig(nameEntry.Text, defaultContent); err != nil {
				a.showErrorDialog(err)
				return
			}
			a.TunnelList.Refresh()
			a.Tray.Update()
		},
		a.Window,
	)
}

func (a *App) showNewFolderDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Folder name")

	dialog.ShowForm("New Folder", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
		},
		func(ok bool) {
			if !ok || nameEntry.Text == "" {
				return
			}
			if err := a.Store.CreateFolder(nameEntry.Text); err != nil {
				a.showErrorDialog(err)
				return
			}
			a.TunnelList.Refresh()
			a.Tray.Update()
		},
		a.Window,
	)
}

func (a *App) deleteCurrent() {
	selected := a.TunnelList.SelectedFilename()
	if selected == "" {
		selectedFolder := a.TunnelList.SelectedFolder()
		if selectedFolder == "" {
			return
		}
		dialog.ShowConfirm("Delete Folder",
			fmt.Sprintf("Delete folder %q and all its tunnels?", selectedFolder),
			func(ok bool) {
				if !ok {
					return
				}
				if err := a.Store.DeleteFolder(selectedFolder); err != nil {
					a.showErrorDialog(err)
					return
				}
				a.Editor.Clear()
				a.TunnelList.ClearSelection()
				a.TunnelList.Refresh()
				a.Tray.Update()
				a.updateSelectionButtons()
			},
			a.Window,
		)
		return
	}

	dialog.ShowConfirm("Delete Tunnel",
		fmt.Sprintf("Delete tunnel %q?", selected),
		func(ok bool) {
			if !ok {
				return
			}
			if err := a.Store.DeleteTunnel(selected); err != nil {
				a.showErrorDialog(err)
				return
			}
			a.Editor.Clear()
			a.TunnelList.ClearSelection()
			a.TunnelList.Refresh()
			a.Tray.Update()
			a.updateSelectionButtons()
		},
		a.Window,
	)
}

func (a *App) connectSelected() {
	filename := a.TunnelList.SelectedFilename()
	if filename == "" {
		// Check if a folder is selected — connect to random
		folder := a.TunnelList.SelectedFolder()
		if folder != "" {
			a.ConnectRandomFromFolder(folder)
		}
		return
	}
	a.ConnectTunnel(filename)
}

// ConnectTunnel connects to a specific tunnel by filename.
func (a *App) ConnectTunnel(filename string) {
	confPath := a.Store.TunnelPath(filename)
	go func() {
		if err := a.Manager.Connect(confPath, filename); err != nil {
			fyne.Do(func() {
				a.showErrorDialog(err)
			})
		}
	}()
}

// ConnectRandomFromFolder picks a random tunnel from a folder and connects.
func (a *App) ConnectRandomFromFolder(folder string) {
	filename, err := a.Store.RandomTunnelInFolder(folder)
	if err != nil {
		a.showErrorDialog(err)
		return
	}
	a.ConnectTunnel(filename)
}

func (a *App) disconnectTunnel() {
	go func() {
		if err := a.Manager.Disconnect(); err != nil {
			fyne.Do(func() {
				a.showErrorDialog(err)
			})
		}
	}()
}
// showErrorDialog displays an error with selectable, copyable text.
func (a *App) showErrorDialog(err error) {
	errText := widget.NewEntry()
	errText.SetText(err.Error())
	errText.MultiLine = true
	errText.Wrapping = fyne.TextWrapWord
	errText.Disable() // read-only but still selectable/copyable

	errText.Resize(fyne.NewSize(500, 150))

	d := dialog.NewCustom("Error", "OK", errText, a.Window)
	d.Resize(fyne.NewSize(550, 250))
	d.Show()
}
// Show displays the main window.
func (a *App) Show() {
	a.Window.Show()
}
