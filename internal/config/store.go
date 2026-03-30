package config

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
)

// Metadata holds the folder structure and tunnel grouping.
type Metadata struct {
	Folders    map[string][]string `json:"folders"`
	Ungrouped  []string            `json:"ungrouped"`
	configDir  string
}

// TunnelInfo provides display information about a tunnel.
type TunnelInfo struct {
	Name     string // Filename without .conf extension
	Filename string // Full filename (e.g. "my-vpn.conf")
	Folder   string // Folder name, empty if ungrouped
	Address  string // From [Interface] Address
	Endpoint string // From first [Peer] Endpoint
}

// Store manages WireGuard config files on disk.
type Store struct {
	configDir  string
	tunnelsDir string
	metaPath   string
}

// NewStore creates a Store rooted at the given config directory.
// It creates the directory structure if it doesn't exist.
func NewStore(configDir string) (*Store, error) {
	tunnelsDir := filepath.Join(configDir, "tunnels")
	if err := os.MkdirAll(tunnelsDir, 0700); err != nil {
		return nil, fmt.Errorf("create tunnels dir: %w", err)
	}
	return &Store{
		configDir:  configDir,
		tunnelsDir: tunnelsDir,
		metaPath:   filepath.Join(configDir, "metadata.json"),
	}, nil
}

// DefaultConfigDir returns the default config directory following XDG.
func DefaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "tunnelvision")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tunnelvision")
}

// LoadMetadata reads the metadata.json file.
func (s *Store) LoadMetadata() (*Metadata, error) {
	meta := &Metadata{
		Folders:   make(map[string][]string),
		configDir: s.configDir,
	}

	data, err := os.ReadFile(s.metaPath)
	if os.IsNotExist(err) {
		return meta, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	if err := json.Unmarshal(data, meta); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	if meta.Folders == nil {
		meta.Folders = make(map[string][]string)
	}
	meta.configDir = s.configDir
	return meta, nil
}

// SaveMetadata writes the metadata.json file.
func (s *Store) SaveMetadata(meta *Metadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(s.metaPath, data, 0600); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

// ImportFiles copies one or more .conf files into the store.
// Returns the list of imported filenames.
func (s *Store) ImportFiles(paths []string) ([]string, error) {
	meta, err := s.LoadMetadata()
	if err != nil {
		return nil, err
	}

	var imported []string
	for _, srcPath := range paths {
		filename := filepath.Base(srcPath)
		if !strings.HasSuffix(strings.ToLower(filename), ".conf") {
			continue
		}

		// Handle name collisions
		destPath := filepath.Join(s.tunnelsDir, filename)
		filename = s.resolveCollision(filename)
		destPath = filepath.Join(s.tunnelsDir, filename)

		src, err := os.Open(srcPath)
		if err != nil {
			return imported, fmt.Errorf("open %s: %w", srcPath, err)
		}

		dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if err != nil {
			src.Close()
			return imported, fmt.Errorf("create %s: %w", destPath, err)
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return imported, fmt.Errorf("copy %s: %w", filename, copyErr)
		}

		meta.Ungrouped = append(meta.Ungrouped, filename)
		imported = append(imported, filename)
	}

	if err := s.SaveMetadata(meta); err != nil {
		return imported, err
	}
	return imported, nil
}

// ImportFilesToFolder copies .conf files into the store and places them directly into a folder.
// If folder is empty, they go to ungrouped.
func (s *Store) ImportFilesToFolder(paths []string, folder string) ([]string, error) {
	meta, err := s.LoadMetadata()
	if err != nil {
		return nil, err
	}

	// Ensure folder exists
	if folder != "" {
		if _, exists := meta.Folders[folder]; !exists {
			meta.Folders[folder] = []string{}
		}
	}

	var imported []string
	for _, srcPath := range paths {
		filename := filepath.Base(srcPath)
		if !strings.HasSuffix(strings.ToLower(filename), ".conf") {
			continue
		}

		filename = s.resolveCollision(filename)
		destPath := filepath.Join(s.tunnelsDir, filename)

		src, err := os.Open(srcPath)
		if err != nil {
			return imported, fmt.Errorf("open %s: %w", srcPath, err)
		}

		dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if err != nil {
			src.Close()
			return imported, fmt.Errorf("create %s: %w", destPath, err)
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return imported, fmt.Errorf("copy %s: %w", filename, copyErr)
		}

		if folder == "" {
			meta.Ungrouped = append(meta.Ungrouped, filename)
		} else {
			meta.Folders[folder] = append(meta.Folders[folder], filename)
		}
		imported = append(imported, filename)
	}

	if err := s.SaveMetadata(meta); err != nil {
		return imported, err
	}
	return imported, nil
}

// resolveCollision appends a number suffix if a file already exists.
func (s *Store) resolveCollision(filename string) string {
	destPath := filepath.Join(s.tunnelsDir, filename)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return filename
	}

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s_%d%s", base, i, ext)
		candidatePath := filepath.Join(s.tunnelsDir, candidate)
		if _, err := os.Stat(candidatePath); os.IsNotExist(err) {
			return candidate
		}
	}
	return filename // give up
}

// SaveConfig writes raw config text to a tunnel file.
func (s *Store) SaveConfig(filename string, content string) error {
	destPath := filepath.Join(s.tunnelsDir, filename)
	return os.WriteFile(destPath, []byte(content), 0600)
}

// ReadConfig reads a tunnel config file and returns the parsed config.
func (s *Store) ReadConfig(filename string) (*WgConfig, error) {
	return ParseFile(filepath.Join(s.tunnelsDir, filename))
}

// ReadConfigRaw reads a tunnel config file and returns raw text.
func (s *Store) ReadConfigRaw(filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(s.tunnelsDir, filename))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TunnelPath returns the full path to a tunnel config file.
func (s *Store) TunnelPath(filename string) string {
	return filepath.Join(s.tunnelsDir, filename)
}

// DeleteTunnel removes a tunnel config file and its metadata entry.
func (s *Store) DeleteTunnel(filename string) error {
	meta, err := s.LoadMetadata()
	if err != nil {
		return err
	}

	// Remove from ungrouped
	meta.Ungrouped = removeFromSlice(meta.Ungrouped, filename)

	// Remove from all folders
	for folder, tunnels := range meta.Folders {
		meta.Folders[folder] = removeFromSlice(tunnels, filename)
	}

	if err := s.SaveMetadata(meta); err != nil {
		return err
	}

	path := filepath.Join(s.tunnelsDir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CreateFolder adds a new empty folder.
func (s *Store) CreateFolder(name string) error {
	meta, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	if _, exists := meta.Folders[name]; exists {
		return fmt.Errorf("folder %q already exists", name)
	}
	meta.Folders[name] = []string{}
	return s.SaveMetadata(meta)
}

// DeleteFolder removes a folder and all its tunnel configs.
func (s *Store) DeleteFolder(name string) error {
	meta, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	tunnels, exists := meta.Folders[name]
	if !exists {
		return fmt.Errorf("folder %q does not exist", name)
	}
	// Delete all tunnel config files in the folder
	for _, filename := range tunnels {
		path := filepath.Join(s.tunnelsDir, filename)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", filename, err)
		}
	}
	delete(meta.Folders, name)
	return s.SaveMetadata(meta)
}

// MoveToFolder moves a tunnel from its current location to a folder.
func (s *Store) MoveToFolder(filename string, folder string) error {
	meta, err := s.LoadMetadata()
	if err != nil {
		return err
	}

	// Remove from ungrouped
	meta.Ungrouped = removeFromSlice(meta.Ungrouped, filename)

	// Remove from all folders
	for f, tunnels := range meta.Folders {
		meta.Folders[f] = removeFromSlice(tunnels, filename)
	}

	// Add to target folder
	if folder == "" {
		meta.Ungrouped = append(meta.Ungrouped, filename)
	} else {
		meta.Folders[folder] = append(meta.Folders[folder], filename)
	}

	return s.SaveMetadata(meta)
}

// GetTunnelInfos returns display info for all tunnels.
func (s *Store) GetTunnelInfos() ([]TunnelInfo, error) {
	meta, err := s.LoadMetadata()
	if err != nil {
		return nil, err
	}

	var infos []TunnelInfo

	addInfo := func(filename, folder string) {
		info := TunnelInfo{
			Name:     strings.TrimSuffix(filename, ".conf"),
			Filename: filename,
			Folder:   folder,
		}
		if cfg, err := s.ReadConfig(filename); err == nil {
			info.Address = cfg.Address()
			info.Endpoint = cfg.Endpoint()
		}
		infos = append(infos, info)
	}

	for folder, tunnels := range meta.Folders {
		for _, filename := range tunnels {
			addInfo(filename, folder)
		}
	}
	for _, filename := range meta.Ungrouped {
		addInfo(filename, "")
	}

	return infos, nil
}

// RandomTunnelInFolder returns a random tunnel filename from a folder.
func (s *Store) RandomTunnelInFolder(folder string) (string, error) {
	meta, err := s.LoadMetadata()
	if err != nil {
		return "", err
	}
	tunnels, exists := meta.Folders[folder]
	if !exists || len(tunnels) == 0 {
		return "", fmt.Errorf("folder %q is empty or does not exist", folder)
	}
	return tunnels[rand.IntN(len(tunnels))], nil
}

// AddConfig creates a new empty config file.
func (s *Store) AddConfig(filename string, content string) error {
	if !strings.HasSuffix(filename, ".conf") {
		filename += ".conf"
	}
	filename = s.resolveCollision(filename)

	destPath := filepath.Join(s.tunnelsDir, filename)
	if err := os.WriteFile(destPath, []byte(content), 0600); err != nil {
		return err
	}

	meta, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	meta.Ungrouped = append(meta.Ungrouped, filename)
	return s.SaveMetadata(meta)
}

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
