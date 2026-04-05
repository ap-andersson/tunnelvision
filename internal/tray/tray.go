package tray

import (
	_ "embed"
	"sort"
	"sync"

	"fyne.io/systray"
	"github.com/ap-andersson/tunnelvision/internal/config"
	"github.com/ap-andersson/tunnelvision/internal/tunnel"
)

//go:embed icon_connected.png
var iconConnected []byte

//go:embed icon_disconnected.png
var iconDisconnected []byte

// Tray manages the system tray icon and menu.
type Tray struct {
	store      *config.Store
	manager    *tunnel.Manager
	status     *tunnel.Status
	showWindow func()
	quitApp    func()
	startFn    func() // systray start function
	endFn      func() // systray end function
	mu         sync.Mutex
}

// New creates a new Tray manager.
func New(store *config.Store, manager *tunnel.Manager, status *tunnel.Status, showWindow func(), quitApp func()) *Tray {
	return &Tray{
		store:      store,
		manager:    manager,
		status:     status,
		showWindow: showWindow,
		quitApp:    quitApp,
	}
}

// Setup initializes the system tray with RunWithExternalLoop so it coexists with Wails.
func (t *Tray) Setup() {
	t.startFn, t.endFn = systray.RunWithExternalLoop(t.onReady, t.onExit)
	t.startFn()
}

// Stop tears down the system tray.
func (t *Tray) Stop() {
	if t.endFn != nil {
		t.endFn()
	}
}

// Update rebuilds the tray menu to reflect current state.
// Safe to call from any goroutine — serialized with a mutex.
func (t *Tray) Update() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.updateIcon()
	systray.ResetMenu()
	t.buildMenu()
}

func (t *Tray) onReady() {
	t.updateIcon()
	systray.SetTooltip("TunnelVision")
	t.buildMenu()
}

func (t *Tray) onExit() {
	// nothing to clean up
}

func (t *Tray) updateIcon() {
	if t.manager.IsConnected() {
		systray.SetIcon(iconConnected)
	} else {
		systray.SetIcon(iconDisconnected)
	}
}

func (t *Tray) buildMenu() {
	// Status line
	if t.manager.IsConnected() {
		name := t.manager.ActiveTunnel()
		if len(name) > 5 && name[len(name)-5:] == ".conf" {
			name = name[:len(name)-5]
		}
		item := systray.AddMenuItem("Connected: "+name, "")
		item.Disable()
	} else {
		item := systray.AddMenuItem("Disconnected", "")
		item.Disable()
	}

	systray.AddSeparator()

	// Load metadata for tunnel/folder list
	meta, err := t.store.LoadMetadata()
	if err != nil {
		return
	}

	// Ungrouped tunnels — direct connect
	for _, filename := range meta.Ungrouped {
		name := filename
		if len(name) > 5 && name[len(name)-5:] == ".conf" {
			name = name[:len(name)-5]
		}
		fn := filename // capture
		item := systray.AddMenuItem(name, "Connect to "+name)
		go t.handleTunnelClick(item, fn)
	}

	// Folders — connect to random
	folders := make([]string, 0, len(meta.Folders))
	for f := range meta.Folders {
		folders = append(folders, f)
	}
	sort.Strings(folders)

	for _, folder := range folders {
		f := folder // capture
		item := systray.AddMenuItem(f+" (random)", "Connect to random tunnel in "+f)
		go t.handleFolderClick(item, f)
	}

	if len(meta.Ungrouped) > 0 || len(folders) > 0 {
		systray.AddSeparator()
	}

	// Disconnect (if connected)
	if t.manager.IsConnected() {
		mDisconnect := systray.AddMenuItem("Disconnect", "Disconnect active tunnel")
		go func() {
			for range mDisconnect.ClickedCh {
				t.manager.Disconnect()
			}
		}()
		systray.AddSeparator()
	}

	// Open window
	mOpen := systray.AddMenuItem("Open TunnelVision", "Show main window")
	go func() {
		for range mOpen.ClickedCh {
			if t.showWindow != nil {
				t.showWindow()
			}
		}
	}()

	// Quit
	mQuit := systray.AddMenuItem("Quit", "Quit TunnelVision")
	go func() {
		for range mQuit.ClickedCh {
			if t.quitApp != nil {
				t.quitApp()
			}
		}
	}()
}

func (t *Tray) handleTunnelClick(item *systray.MenuItem, filename string) {
	for range item.ClickedCh {
		confPath := t.store.TunnelPath(filename)
		t.manager.Connect(confPath, filename)
	}
}

func (t *Tray) handleFolderClick(item *systray.MenuItem, folder string) {
	for range item.ClickedCh {
		filename, err := t.store.RandomTunnelInFolder(folder)
		if err != nil {
			continue
		}
		confPath := t.store.TunnelPath(filename)
		t.manager.Connect(confPath, filename)
	}
}
