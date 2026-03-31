package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ap-andersson/tunnelvision/internal/config"
	"github.com/ap-andersson/tunnelvision/internal/tunnel"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// StatusInfo is the JSON-friendly tunnel status returned to the frontend.
type StatusInfo struct {
	Connected        bool     `json:"connected"`
	ActiveTunnel     string   `json:"activeTunnel"`
	ActiveInterfaces []string `json:"activeInterfaces"`
}

// ConfigDTO is the JSON-friendly config data returned to the frontend.
type ConfigDTO struct {
	Raw      string `json:"raw"`
	Address  string `json:"address"`
	Endpoint string `json:"endpoint"`
	DNS      string `json:"dns"`
}

// MetadataDTO is the JSON-friendly metadata returned to the frontend.
type MetadataDTO struct {
	Folders   map[string][]string `json:"folders"`
	Ungrouped []string            `json:"ungrouped"`
}

// TunnelInfoDTO is the JSON-friendly tunnel info returned to the frontend.
type TunnelInfoDTO struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Folder   string `json:"folder"`
	Address  string `json:"address"`
	Endpoint string `json:"endpoint"`
}

// TunnelService wraps the tunnel Manager and Status for Wails binding.
type TunnelService struct {
	ctx     context.Context
	manager *tunnel.Manager
	status  *tunnel.Status
	store   *config.Store
}

// NewTunnelService creates a new TunnelService.
func NewTunnelService(store *config.Store, manager *tunnel.Manager, status *tunnel.Status) *TunnelService {
	return &TunnelService{
		manager: manager,
		status:  status,
		store:   store,
	}
}

// SetContext stores the Wails runtime context (called from OnStartup).
func (t *TunnelService) SetContext(ctx context.Context) {
	t.ctx = ctx
}

// EmitStateChanged emits the tunnel state change event to the frontend.
func (t *TunnelService) EmitStateChanged() {
	if t.ctx != nil {
		runtime.EventsEmit(t.ctx, "tunnel:state-changed")
	}
}

// GetStatus returns the current tunnel connection status.
func (t *TunnelService) GetStatus() StatusInfo {
	return StatusInfo{
		Connected:        t.manager.IsConnected(),
		ActiveTunnel:     t.manager.ActiveTunnel(),
		ActiveInterfaces: t.status.ActiveInterfaces(),
	}
}

// Connect activates a tunnel by filename.
func (t *TunnelService) Connect(filename string) error {
	confPath := t.store.TunnelPath(filename)
	return t.manager.Connect(confPath, filename)
}

// ConnectRandom connects to a random tunnel in a folder.
func (t *TunnelService) ConnectRandom(folder string) error {
	filename, err := t.store.RandomTunnelInFolder(folder)
	if err != nil {
		return err
	}
	confPath := t.store.TunnelPath(filename)
	return t.manager.Connect(confPath, filename)
}

// Disconnect tears down the active tunnel.
func (t *TunnelService) Disconnect() error {
	return t.manager.Disconnect()
}

// RefreshStatus refreshes the WireGuard interface status.
func (t *TunnelService) RefreshStatus() error {
	return t.status.Refresh()
}

// ConfigService wraps the config Store for Wails binding.
type ConfigService struct {
	ctx            context.Context
	store          *config.Store
	onConfigChange func()
}

// NewConfigService creates a new ConfigService.
func NewConfigService(store *config.Store) *ConfigService {
	return &ConfigService{store: store}
}

// SetContext stores the Wails runtime context (called from OnStartup).
func (c *ConfigService) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// SetOnConfigChange sets a callback that fires when config/metadata changes.
func (c *ConfigService) SetOnConfigChange(fn func()) {
	c.onConfigChange = fn
}

func (c *ConfigService) notifyConfigChange() {
	if c.onConfigChange != nil {
		c.onConfigChange()
	}
}

// GetMetadata returns the folder/tunnel structure.
func (c *ConfigService) GetMetadata() (MetadataDTO, error) {
	meta, err := c.store.LoadMetadata()
	if err != nil {
		return MetadataDTO{}, err
	}
	dto := MetadataDTO{
		Folders:   meta.Folders,
		Ungrouped: meta.Ungrouped,
	}
	if dto.Folders == nil {
		dto.Folders = make(map[string][]string)
	}
	if dto.Ungrouped == nil {
		dto.Ungrouped = []string{}
	}
	return dto, nil
}

// GetTunnelInfos returns display info for all tunnels.
func (c *ConfigService) GetTunnelInfos() ([]TunnelInfoDTO, error) {
	infos, err := c.store.GetTunnelInfos()
	if err != nil {
		return nil, err
	}
	dtos := make([]TunnelInfoDTO, len(infos))
	for i, info := range infos {
		dtos[i] = TunnelInfoDTO{
			Name:     info.Name,
			Filename: info.Filename,
			Folder:   info.Folder,
			Address:  info.Address,
			Endpoint: info.Endpoint,
		}
	}
	return dtos, nil
}

// ReadConfig reads a tunnel config and returns parsed data + raw text.
func (c *ConfigService) ReadConfig(filename string) (ConfigDTO, error) {
	raw, err := c.store.ReadConfigRaw(filename)
	if err != nil {
		return ConfigDTO{}, err
	}
	cfg, err := c.store.ReadConfig(filename)
	if err != nil {
		// Return raw text even if parsing fails
		return ConfigDTO{Raw: raw}, nil
	}
	return ConfigDTO{
		Raw:      raw,
		Address:  cfg.Address(),
		Endpoint: cfg.Endpoint(),
		DNS:      cfg.DNS(),
	}, nil
}

// SaveConfig writes config text to a tunnel file.
func (c *ConfigService) SaveConfig(filename string, content string) error {
	return c.store.SaveConfig(filename, content)
}

// AddConfig creates a new tunnel config.
func (c *ConfigService) AddConfig(name string, content string) error {
	err := c.store.AddConfig(name, content)
	if err == nil {
		c.notifyConfigChange()
	}
	return err
}

// DeleteTunnel removes a tunnel config and its metadata.
func (c *ConfigService) DeleteTunnel(filename string) error {
	err := c.store.DeleteTunnel(filename)
	if err == nil {
		c.notifyConfigChange()
	}
	return err
}

// DeleteFolder removes a folder and all its tunnel configs.
func (c *ConfigService) DeleteFolder(name string) error {
	err := c.store.DeleteFolder(name)
	if err == nil {
		c.notifyConfigChange()
	}
	return err
}

// CreateFolder adds a new empty folder.
func (c *ConfigService) CreateFolder(name string) error {
	err := c.store.CreateFolder(name)
	if err == nil {
		c.notifyConfigChange()
	}
	return err
}

// MoveToFolder moves a tunnel to a folder (empty string = ungrouped).
func (c *ConfigService) MoveToFolder(filename string, folder string) error {
	err := c.store.MoveToFolder(filename, folder)
	if err == nil {
		c.notifyConfigChange()
	}
	return err
}

// ImportFiles imports config files via native file dialog.
func (c *ConfigService) ImportFiles() ([]string, error) {
	paths, err := runtime.OpenMultipleFilesDialog(c.ctx, runtime.OpenDialogOptions{
		Title: "Import WireGuard Configs",
		Filters: []runtime.FileFilter{
			{DisplayName: "WireGuard Configs (*.conf)", Pattern: "*.conf"},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	result, err := c.store.ImportFiles(paths)
	if err == nil && len(result) > 0 {
		c.notifyConfigChange()
	}
	return result, err
}

// ImportFilesToFolder imports config files to a specific folder via dialog.
func (c *ConfigService) ImportFilesToFolder(folder string) ([]string, error) {
	paths, err := runtime.OpenMultipleFilesDialog(c.ctx, runtime.OpenDialogOptions{
		Title: fmt.Sprintf("Import WireGuard Configs to %q", folder),
		Filters: []runtime.FileFilter{
			{DisplayName: "WireGuard Configs (*.conf)", Pattern: "*.conf"},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	result, err := c.store.ImportFilesToFolder(paths, folder)
	if err == nil && len(result) > 0 {
		c.notifyConfigChange()
	}
	return result, err
}

// ImportFolder imports all .conf files from a selected directory.
func (c *ConfigService) ImportFolder() ([]string, error) {
	dir, err := runtime.OpenDirectoryDialog(c.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder to Import",
	})
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.conf"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .conf files found in %s", dir)
	}
	result, err := c.store.ImportFiles(matches)
	if err == nil && len(result) > 0 {
		c.notifyConfigChange()
	}
	return result, err
}

// ImportFolderToFolder imports all .conf files from a directory into a specific folder.
func (c *ConfigService) ImportFolderToFolder(folder string) ([]string, error) {
	dir, err := runtime.OpenDirectoryDialog(c.ctx, runtime.OpenDialogOptions{
		Title: fmt.Sprintf("Import Folder to %q", folder),
	})
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.conf"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .conf files found in %s", dir)
	}
	result, err := c.store.ImportFilesToFolder(matches, folder)
	if err == nil && len(result) > 0 {
		c.notifyConfigChange()
	}
	return result, err
}

// GetFolderNames returns just the folder names for the "Move to" dropdown.
func (c *ConfigService) GetFolderNames() ([]string, error) {
	meta, err := c.store.LoadMetadata()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(meta.Folders))
	for name := range meta.Folders {
		names = append(names, name)
	}
	return names, nil
}

// RandomTunnelInFolder returns a random tunnel filename from a folder.
func (c *ConfigService) RandomTunnelInFolder(folder string) (string, error) {
	return c.store.RandomTunnelInFolder(folder)
}

// GetDefaultTemplate returns a blank WireGuard config template.
func (c *ConfigService) GetDefaultTemplate() string {
	return strings.TrimSpace(`
[Interface]
PrivateKey = 
Address = 
DNS = 

[Peer]
PublicKey = 
Endpoint = 
AllowedIPs = 0.0.0.0/0
`)
}

// RenameTunnel renames a tunnel and returns the new filename.
func (c *ConfigService) RenameTunnel(oldFilename string, newName string) (string, error) {
	result, err := c.store.RenameTunnel(oldFilename, newName)
	if err == nil {
		c.notifyConfigChange()
	}
	return result, err
}
