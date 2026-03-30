package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ConfigEditor shows the contents of a selected WireGuard config.
type ConfigEditor struct {
	app           *App
	container     *fyne.Container
	editor        *widget.Entry
	nameLabel     *widget.Label
	infoContainer *fyne.Container
	saveBtn       *widget.Button
	moveBtn       *widget.Button
	importBtn     *widget.Button
	tunnelPanel   *fyne.Container // editor + tunnel buttons
	folderPanel   *fyne.Container // folder info + import button
	filename      string
	folderName    string
}

// NewConfigEditor creates the config editor panel.
func NewConfigEditor(a *App) *ConfigEditor {
	ce := &ConfigEditor{
		app: a,
	}

	ce.nameLabel = widget.NewLabel("Select a tunnel to view its configuration")
	ce.nameLabel.TextStyle.Bold = true

	ce.infoContainer = container.NewVBox()

	ce.editor = widget.NewMultiLineEntry()
	ce.editor.SetPlaceHolder("WireGuard configuration will appear here...")
	ce.editor.TextStyle.Monospace = true

	ce.saveBtn = widget.NewButton("Save", func() {
		ce.save()
	})
	ce.saveBtn.Disable()

	ce.moveBtn = widget.NewButton("Move to Folder...", func() {
		if ce.filename != "" {
			ce.app.TunnelList.ShowMoveToFolderMenu(ce.filename)
		}
	})
	ce.moveBtn.Disable()

	ce.importBtn = widget.NewButtonWithIcon("Import to This Folder", theme.DownloadIcon(), func() {
		if ce.folderName != "" {
			ce.app.importFolderInto(ce.folderName)
		}
	})

	tunnelButtons := container.NewHBox(ce.saveBtn, ce.moveBtn, layout.NewSpacer())
	ce.tunnelPanel = container.NewBorder(tunnelButtons, nil, nil, nil, ce.editor)

	ce.folderPanel = container.NewVBox(ce.importBtn)
	ce.folderPanel.Hide()
	ce.tunnelPanel.Hide()

	body := container.NewStack(ce.tunnelPanel, ce.folderPanel)
	header := container.NewVBox(ce.nameLabel, ce.infoContainer)

	ce.container = container.NewBorder(header, nil, nil, nil, body)
	return ce
}

// Widget returns the underlying fyne container.
func (ce *ConfigEditor) Widget() fyne.CanvasObject {
	return ce.container
}

// ShowConfig displays a tunnel config in the editor.
func (ce *ConfigEditor) ShowConfig(filename string, content string) {
	ce.filename = filename
	ce.folderName = ""
	name := filename
	if len(name) > 5 && name[len(name)-5:] == ".conf" {
		name = name[:len(name)-5]
	}
	ce.nameLabel.SetText(name)

	// Parse for info display
	cfg, err := ce.app.Store.ReadConfig(filename)
	if err == nil {
		ce.infoContainer.RemoveAll()
		if addr := cfg.Address(); addr != "" {
			ce.infoContainer.Add(widget.NewRichTextFromMarkdown("**Address:** " + addr))
		}
		if ep := cfg.Endpoint(); ep != "" {
			ce.infoContainer.Add(widget.NewRichTextFromMarkdown("**Endpoint:** " + ep))
		}
		if dns := cfg.DNS(); dns != "" {
			ce.infoContainer.Add(widget.NewRichTextFromMarkdown("**DNS:** " + dns))
		}
	} else {
		ce.infoContainer.RemoveAll()
	}

	ce.editor.SetText(content)
	ce.saveBtn.Enable()
	ce.moveBtn.Enable()

	ce.folderPanel.Hide()
	ce.tunnelPanel.Show()
}

// ShowFolderInfo displays information about a selected folder.
func (ce *ConfigEditor) ShowFolderInfo(folderName string, tunnelCount int) {
	ce.filename = ""
	ce.folderName = folderName
	ce.nameLabel.SetText("Folder: " + folderName)
	ce.infoContainer.RemoveAll()
	ce.infoContainer.Add(widget.NewRichTextFromMarkdown(fmt.Sprintf("**Tunnels:** %d", tunnelCount)))
	ce.infoContainer.Add(widget.NewLabel("Clicking Connect while a folder is selected will connect to a random tunnel inside the folder."))

	ce.tunnelPanel.Hide()
	ce.folderPanel.Show()
}

// Clear resets the editor to its empty state.
func (ce *ConfigEditor) Clear() {
	ce.filename = ""
	ce.folderName = ""
	ce.nameLabel.SetText("Select a tunnel to view its configuration")
	ce.infoContainer.RemoveAll()
	ce.editor.SetText("")
	ce.saveBtn.Disable()
	ce.moveBtn.Disable()

	ce.folderPanel.Hide()
	ce.tunnelPanel.Hide()
}

func (ce *ConfigEditor) save() {
	if ce.filename == "" {
		return
	}
	content := ce.editor.Text
	if err := ce.app.Store.SaveConfig(ce.filename, content); err != nil {
		ce.app.showErrorDialog(err)
		return
	}
	// Re-parse to update info labels
	ce.ShowConfig(ce.filename, content)
}
