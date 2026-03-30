package tunnel

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Status tracks the state of WireGuard tunnels on the system.
type Status struct {
	mu         sync.Mutex
	interfaces []string // List of active WireGuard interface names
	lastCheck  time.Time
}

// NewStatus creates a new Status checker.
func NewStatus() *Status {
	return &Status{}
}

// Refresh updates the list of active WireGuard interfaces.
// Uses `ip link show type wireguard` which doesn't require root.
func (s *Status) Refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.Command("ip", "link", "show", "type", "wireguard")
	output, err := cmd.Output()
	if err != nil {
		// No wireguard interfaces or command failed
		s.interfaces = nil
		s.lastCheck = time.Now()
		return nil
	}

	var ifaces []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		// Lines like: "4: wg0: <POINTOPOINT,NOARP,UP,LOWER_UP> ..."
		if strings.Contains(line, ": ") && strings.Contains(line, "<") {
			parts := strings.SplitN(line, ": ", 3)
			if len(parts) >= 2 {
				ifaceName := strings.TrimSuffix(parts[1], "@NONE")
				ifaces = append(ifaces, ifaceName)
			}
		}
	}

	s.interfaces = ifaces
	s.lastCheck = time.Now()
	return nil
}

// ActiveInterfaces returns the list of active WireGuard interface names.
func (s *Status) ActiveInterfaces() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.interfaces))
	copy(result, s.interfaces)
	return result
}

// IsInterfaceActive checks if a specific interface is currently active.
func (s *Status) IsInterfaceActive(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, iface := range s.interfaces {
		if iface == name {
			return true
		}
	}
	return false
}
