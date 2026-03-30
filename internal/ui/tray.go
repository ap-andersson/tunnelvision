package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// Tray manages the system tray icon and menu.
type Tray struct {
	app *App
}

// NewTray creates a new Tray manager.
func NewTray(a *App) *Tray {
	return &Tray{app: a}
}

// Setup initializes the system tray icon and menu.
func (t *Tray) Setup() {
	desk, ok := t.app.FyneApp.(desktop.App)
	if !ok {
		return
	}

	t.updateMenu(desk)
	t.updateIcon(desk)
}

// Update refreshes the tray menu and icon to reflect current state.
func (t *Tray) Update() {
	desk, ok := t.app.FyneApp.(desktop.App)
	if !ok {
		return
	}

	t.updateMenu(desk)
	t.updateIcon(desk)
}

func (t *Tray) updateMenu(desk desktop.App) {
	var items []*fyne.MenuItem

	// Status item
	if t.app.Manager.IsConnected() {
		active := t.app.Manager.ActiveTunnel()
		name := active
		if len(name) > 5 && name[len(name)-5:] == ".conf" {
			name = name[:len(name)-5]
		}
		statusItem := fyne.NewMenuItem("Connected: "+name, nil)
		statusItem.Disabled = true
		items = append(items, statusItem)
		items = append(items, fyne.NewMenuItemSeparator())
	} else {
		statusItem := fyne.NewMenuItem("Disconnected", nil)
		statusItem.Disabled = true
		items = append(items, statusItem)
		items = append(items, fyne.NewMenuItemSeparator())
	}

	// Load tunnels
	meta, err := t.app.Store.LoadMetadata()
	if err == nil {
		// Root-level tunnels
		for _, filename := range meta.Ungrouped {
			fn := filename
			name := fn
			if len(name) > 5 && name[len(name)-5:] == ".conf" {
				name = name[:len(name)-5]
			}
			items = append(items, fyne.NewMenuItem(name, func() {
				t.app.ConnectTunnel(fn)
			}))
		}

		// Folders as sub-menus
		for folderName, tunnels := range meta.Folders {
			if len(tunnels) == 0 {
				continue
			}
			folder := folderName
			subItem := fyne.NewMenuItem(folder+" (random)", func() {
				t.app.ConnectRandomFromFolder(folder)
			})
			items = append(items, subItem)
		}
	}

	items = append(items, fyne.NewMenuItemSeparator())

	// Disconnect
	if t.app.Manager.IsConnected() {
		items = append(items, fyne.NewMenuItem("Disconnect", func() {
			go func() {
				_ = t.app.Manager.Disconnect()
			}()
		}))
	}

	// Open window
	items = append(items, fyne.NewMenuItem("Open TunnelVision", func() {
		t.app.Window.Show()
		t.app.Window.RequestFocus()
	}))

	// Quit
	items = append(items, fyne.NewMenuItem("Quit", func() {
		t.app.FyneApp.Quit()
	}))

	menu := fyne.NewMenu("TunnelVision", items...)
	desk.SetSystemTrayMenu(menu)
}

func (t *Tray) updateIcon(desk desktop.App) {
	if t.app.Manager.IsConnected() {
		desk.SetSystemTrayIcon(resourceIconConnectedPng)
	} else {
		desk.SetSystemTrayIcon(resourceIconDisconnectedPng)
	}
}
