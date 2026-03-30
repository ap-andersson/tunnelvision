package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Section represents a named section in a WireGuard config file.
type Section struct {
	Name  string   // "Interface" or "Peer"
	Lines []string // Raw lines within this section (key = value lines and comments)
}

// WgConfig represents a parsed WireGuard configuration file.
type WgConfig struct {
	Sections []Section
	RawText  string // Original full text, preserved for editing
}

// ParseFile reads and parses a WireGuard .conf file from disk.
func ParseFile(path string) (*WgConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads a WireGuard config from a reader.
func Parse(r io.Reader) (*WgConfig, error) {
	var buf strings.Builder
	tee := io.TeeReader(r, &buf)

	scanner := bufio.NewScanner(tee)
	cfg := &WgConfig{}

	var current *Section
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			name := trimmed[1 : len(trimmed)-1]
			cfg.Sections = append(cfg.Sections, Section{Name: name})
			current = &cfg.Sections[len(cfg.Sections)-1]
			continue
		}

		if current != nil {
			current.Lines = append(current.Lines, line)
		}
		// Lines before any section header are ignored (or could be comments)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan config: %w", err)
	}

	cfg.RawText = buf.String()
	return cfg, nil
}

// Serialize writes the config back to its text representation.
func Serialize(cfg *WgConfig) string {
	var sb strings.Builder
	for i, sec := range cfg.Sections {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[")
		sb.WriteString(sec.Name)
		sb.WriteString("]\n")
		for _, line := range sec.Lines {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// GetValue returns the first value for a key in a given section, or empty string if not found.
func GetValue(sec *Section, key string) string {
	prefix := key + " "
	prefixEq := key + "="
	for _, line := range sec.Lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(trimmed, prefixEq) {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// InterfaceSection returns the first [Interface] section, or nil.
func (c *WgConfig) InterfaceSection() *Section {
	for i := range c.Sections {
		if c.Sections[i].Name == "Interface" {
			return &c.Sections[i]
		}
	}
	return nil
}

// PeerSections returns all [Peer] sections.
func (c *WgConfig) PeerSections() []*Section {
	var peers []*Section
	for i := range c.Sections {
		if c.Sections[i].Name == "Peer" {
			peers = append(peers, &c.Sections[i])
		}
	}
	return peers
}

// Address returns the Address field from the [Interface] section.
func (c *WgConfig) Address() string {
	if iface := c.InterfaceSection(); iface != nil {
		return GetValue(iface, "Address")
	}
	return ""
}

// Endpoint returns the Endpoint from the first [Peer] section.
func (c *WgConfig) Endpoint() string {
	peers := c.PeerSections()
	if len(peers) > 0 {
		return GetValue(peers[0], "Endpoint")
	}
	return ""
}

// DNS returns the DNS field from the [Interface] section.
func (c *WgConfig) DNS() string {
	if iface := c.InterfaceSection(); iface != nil {
		return GetValue(iface, "DNS")
	}
	return ""
}
