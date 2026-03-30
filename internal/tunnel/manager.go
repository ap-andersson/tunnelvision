package tunnel

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Manager handles connecting and disconnecting WireGuard tunnels via NetworkManager (nmcli).
type Manager struct {
	mu             sync.Mutex
	activeConnName string // NM connection name for the active tunnel
	activeFilename string // Config filename (e.g. "myserver.conf")
	onStateChange  func()
}

// NewManager creates a new tunnel Manager.
func NewManager() *Manager {
	return &Manager{}
}

// SetOnStateChange sets a callback that fires when connection state changes.
func (m *Manager) SetOnStateChange(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onStateChange = fn
}

func (m *Manager) notifyStateChange() {
	if m.onStateChange != nil {
		m.onStateChange()
	}
}

// Connect imports a WireGuard config into NetworkManager and activates it.
// If another tunnel is active, it is disconnected first.
func (m *Manager) Connect(confPath string, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Disconnect existing tunnel first
	if m.activeConnName != "" {
		if err := m.disconnectLocked(); err != nil {
			return fmt.Errorf("disconnect existing tunnel: %w", err)
		}
	}

	connName := connectionNameFromFilename(filename)

	// Delete any stale NM connection with the same name
	_ = exec.Command("nmcli", "connection", "delete", connName).Run()

	// Import the config file into NetworkManager
	cmd := exec.Command("nmcli", "connection", "import", "type", "wireguard", "file", confPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nmcli import: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	// Activate the connection
	cmd = exec.Command("nmcli", "connection", "up", connName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Clean up the imported connection on failure
		_ = exec.Command("nmcli", "connection", "delete", connName).Run()
		return fmt.Errorf("nmcli up: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	m.activeConnName = connName
	m.activeFilename = filename

	go m.notifyStateChange()
	return nil
}

// Disconnect tears down the active WireGuard tunnel and removes it from NetworkManager.
func (m *Manager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.disconnectLocked()
}

func (m *Manager) disconnectLocked() error {
	if m.activeConnName == "" {
		return nil
	}

	// Deactivate (may already be down, ignore errors)
	cmd := exec.Command("nmcli", "connection", "down", m.activeConnName)
	_ = cmd.Run()

	// Remove from NetworkManager to keep things clean
	cmd = exec.Command("nmcli", "connection", "delete", m.activeConnName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nmcli delete: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	m.activeConnName = ""
	m.activeFilename = ""

	go m.notifyStateChange()
	return nil
}

// DisconnectGraceful disconnects the active tunnel during shutdown.
// No privilege escalation needed — nmcli runs unprivileged via NetworkManager D-Bus.
func (m *Manager) DisconnectGraceful(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeConnName == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, "nmcli", "connection", "down", m.activeConnName)
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "nmcli", "connection", "delete", m.activeConnName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("graceful disconnect failed: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	m.activeConnName = ""
	m.activeFilename = ""
	return nil
}

// IsConnected returns whether a tunnel is currently active.
func (m *Manager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeConnName != ""
}

// ActiveTunnel returns the filename of the active tunnel, or empty string.
func (m *Manager) ActiveTunnel() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeFilename
}

// ActiveInterface returns the NM connection name of the active tunnel.
func (m *Manager) ActiveInterface() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeConnName
}

// connectionNameFromFilename derives the NM connection name from the config filename.
// nmcli import uses the filename stem as the connection name.
func connectionNameFromFilename(filename string) string {
	return strings.TrimSuffix(filename, ".conf")
}
