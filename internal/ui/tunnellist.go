package ui

import (
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	treeRootUID       = ""
	treeFolderPrefix  = "folder:"
	treeTunnelPrefix  = "tunnel:"
)

// TunnelList displays the tree of folders and tunnels.
type TunnelList struct {
	app              *App
	tree             *widget.Tree
	selectedFilename string // Currently selected tunnel filename
	selectedFolder   string // Currently selected folder name
	metadata         *treeData
}

// treeData holds the snapshot of metadata used to render the tree.
type treeData struct {
	folders    map[string][]string // folder name -> tunnel filenames
	ungrouped  []string            // tunnel filenames not in any folder
	folderNames []string           // sorted folder names
}

// NewTunnelList creates the tunnel list widget.
func NewTunnelList(a *App) *TunnelList {
	tl := &TunnelList{
		app: a,
		metadata: &treeData{
			folders: make(map[string][]string),
		},
	}

	tl.tree = &widget.Tree{
		ChildUIDs: func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return tl.childUIDs(uid)
		},
		IsBranch: func(uid widget.TreeNodeID) bool {
			return tl.isBranch(uid)
		},
		CreateNode: func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		UpdateNode: func(uid widget.TreeNodeID, branch bool, node fyne.CanvasObject) {
			tl.updateNode(uid, branch, node)
		},
		OnSelected: func(uid widget.TreeNodeID) {
			tl.onSelected(uid)
		},
	}

	return tl
}

// Widget returns the underlying fyne widget.
func (tl *TunnelList) Widget() fyne.CanvasObject {
	return tl.tree
}

// Refresh reloads data from the store and updates the tree.
func (tl *TunnelList) Refresh() {
	meta, err := tl.app.Store.LoadMetadata()
	if err != nil {
		return
	}

	data := &treeData{
		folders:   make(map[string][]string),
		ungrouped: make([]string, len(meta.Ungrouped)),
	}
	copy(data.ungrouped, meta.Ungrouped)

	for name, tunnels := range meta.Folders {
		copied := make([]string, len(tunnels))
		copy(copied, tunnels)
		data.folders[name] = copied
		data.folderNames = append(data.folderNames, name)
	}
	sort.Strings(data.folderNames)

	tl.metadata = data
	tl.tree.Refresh()
}

// SelectedFilename returns the currently selected tunnel filename, or empty.
func (tl *TunnelList) SelectedFilename() string {
	return tl.selectedFilename
}

// SelectedFolder returns the currently selected folder name, or empty.
func (tl *TunnelList) SelectedFolder() string {
	return tl.selectedFolder
}

// FolderNames returns the sorted list of folder names.
func (tl *TunnelList) FolderNames() []string {
	return tl.metadata.folderNames
}

// ClearSelection deselects the current tree node and resets selection state.
func (tl *TunnelList) ClearSelection() {
	tl.selectedFilename = ""
	tl.selectedFolder = ""
	tl.tree.UnselectAll()
}

func (tl *TunnelList) childUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	if uid == treeRootUID {
		var children []widget.TreeNodeID
		for _, name := range tl.metadata.folderNames {
			children = append(children, treeFolderPrefix+name)
		}
		for _, filename := range tl.metadata.ungrouped {
			children = append(children, treeTunnelPrefix+filename)
		}
		return children
	}

	if len(uid) > len(treeFolderPrefix) && uid[:len(treeFolderPrefix)] == treeFolderPrefix {
		folderName := uid[len(treeFolderPrefix):]
		tunnels := tl.metadata.folders[folderName]
		var children []widget.TreeNodeID
		for _, filename := range tunnels {
			children = append(children, treeFolderPrefix+folderName+"/"+treeTunnelPrefix+filename)
		}
		return children
	}

	return nil
}

func (tl *TunnelList) isBranch(uid widget.TreeNodeID) bool {
	if uid == treeRootUID {
		return true
	}
	if len(uid) > len(treeFolderPrefix) && uid[:len(treeFolderPrefix)] == treeFolderPrefix {
		rest := uid[len(treeFolderPrefix):]
		if idx := indexOf(rest, "/"+treeTunnelPrefix); idx >= 0 {
			return false
		}
		return true
	}
	return false
}

func (tl *TunnelList) updateNode(uid widget.TreeNodeID, branch bool, node fyne.CanvasObject) {
	label := node.(*widget.Label)

	if branch {
		folderName := uid[len(treeFolderPrefix):]
		count := len(tl.metadata.folders[folderName])
		label.SetText(folderName + " (" + itoa(count) + ")")
		label.TextStyle.Bold = true
		return
	}

	filename := tl.extractFilename(uid)
	displayName := filename
	if len(displayName) > 5 && displayName[len(displayName)-5:] == ".conf" {
		displayName = displayName[:len(displayName)-5]
	}

	if tl.app.Manager.ActiveTunnel() == filename {
		label.SetText("● " + displayName)
		label.Importance = widget.SuccessImportance
	} else {
		label.SetText(displayName)
		label.Importance = widget.MediumImportance
	}
	label.TextStyle.Bold = false
}

func (tl *TunnelList) onSelected(uid widget.TreeNodeID) {
	tl.selectedFilename = ""
	tl.selectedFolder = ""

	if tl.isBranch(uid) {
		tl.selectedFolder = uid[len(treeFolderPrefix):]
		count := len(tl.metadata.folders[tl.selectedFolder])
		tl.app.Editor.ShowFolderInfo(tl.selectedFolder, count)
		tl.app.updateSelectionButtons()
		return
	}

	filename := tl.extractFilename(uid)
	tl.selectedFilename = filename

	content, err := tl.app.Store.ReadConfigRaw(filename)
	if err != nil {
		return
	}
	tl.app.Editor.ShowConfig(filename, content)
	tl.app.updateSelectionButtons()
}

func (tl *TunnelList) extractFilename(uid string) string {
	if len(uid) > len(treeFolderPrefix) && uid[:len(treeFolderPrefix)] == treeFolderPrefix {
		rest := uid[len(treeFolderPrefix):]
		if idx := indexOf(rest, "/"+treeTunnelPrefix); idx >= 0 {
			return rest[idx+1+len(treeTunnelPrefix):]
		}
	}

	if len(uid) > len(treeTunnelPrefix) && uid[:len(treeTunnelPrefix)] == treeTunnelPrefix {
		return uid[len(treeTunnelPrefix):]
	}

	return uid
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 4)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// ShowMoveToFolderMenu shows a context menu to move a single tunnel to a folder.
func (tl *TunnelList) ShowMoveToFolderMenu(filename string) {
	meta := tl.metadata
	var items []*fyne.MenuItem

	items = append(items, fyne.NewMenuItem("(no folder)", func() {
		if err := tl.app.Store.MoveToFolder(filename, ""); err != nil {
			return
		}
		tl.Refresh()
	}))

	for _, folder := range meta.folderNames {
		f := folder
		items = append(items, fyne.NewMenuItem(f, func() {
			if err := tl.app.Store.MoveToFolder(filename, f); err != nil {
				return
			}
			tl.Refresh()
		}))
	}

	menu := fyne.NewMenu("Move to folder", items...)
	popup := widget.NewPopUpMenu(menu, tl.app.Window.Canvas())
	popup.ShowAtPosition(fyne.NewPos(200, 200))
	_ = theme.FolderIcon()
}
