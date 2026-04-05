# TunnelVision

**TunnelVision** is a graphical WireGuard VPN client for Linux, built with Go and [Wails v2](https://wails.io/). It aims to provide the same ease-of-use as the official WireGuard Windows client — something Linux desktop users have been missing.

This project was created with the assistance of **GitHub Copilot** (Claude Opus 4.6).

## Features

- **Import configs** — Import `.conf` files individually or an entire folder of configs at once
- **Create & edit configs** — Create new tunnel configurations from scratch, or edit existing ones with a raw text editor
- **Folder grouping** — Organize tunnels into folders (great for VPN providers with dozens of servers)
- **Connect / Disconnect** — One-click connect to any tunnel via NetworkManager (no password prompts needed)
- **Connect to random** — Select a folder and connect to a random tunnel within it (handy when you just want *any* server from a provider)
- **System tray** — Sits in your system tray with a status icon (green = connected, grey = disconnected). Right-click for quick connect/disconnect without opening the main window
- **Graceful shutdown** — Active tunnels are automatically disconnected when the app quits (or receives SIGINT/SIGTERM)
- **Close to tray** — Closing the window keeps the app running in the tray

## Screenshots

*(Coming soon)*

## Requirements

- **Linux** (tested on Fedora 43 KDE, should work on any distro with NetworkManager and WireGuard support)
- **Go 1.21+** — required to compile the application
- **Wails CLI** — the app uses [Wails v2](https://wails.io/) and must be built with the Wails CLI (not plain `go build`)
- **NetworkManager** — must be running (ships by default on Fedora, Ubuntu, and most desktop distros)
- **WireGuard tools** — `wireguard-tools` package must be installed

## Building

### Fedora (43+)

```bash
# Install build dependencies
sudo dnf install -y gcc-c++ pkgconf-pkg-config gtk3-devel \
    webkit2gtk4.1-devel wireguard-tools

# Install Node.js/npm (needed by the Wails CLI)
sudo dnf install -y nodejs-npm

# Install the Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Make sure ~/go/bin is in your PATH
export PATH="$HOME/go/bin:$PATH"

# Verify everything is in order
wails doctor

# Build (the webkit2_41 tag is required on Fedora 43+ which ships webkit2gtk 4.1)
wails build -tags webkit2_41
```

The binary will be at `build/bin/tunnelvision`.

### Ubuntu / Debian

```bash
# Install build dependencies
sudo apt install -y build-essential pkg-config libgtk-3-dev \
    libwebkit2gtk-4.0-dev wireguard-tools

# Install Node.js/npm (needed by the Wails CLI)
sudo apt install -y npm

# Install the Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Make sure ~/go/bin is in your PATH
export PATH="$HOME/go/bin:$PATH"

# Build
wails build
```

The binary will be at `build/bin/tunnelvision`.

### Run

```bash
./build/bin/tunnelvision
```

### Development mode

```bash
wails dev -tags webkit2_41   # Fedora 43+
wails dev                    # Ubuntu/Debian
```

The app stores its configuration in `~/.config/tunnelvision/`.

## Usage

1. **Import tunnels** — Click "Import File" for a single `.conf` file, or "Import Folder" to bulk-import all `.conf` files from a directory
2. **Create folders** — Click "New Folder" to create a folder, then select a tunnel and use "Move to Folder..." to organize
3. **Connect** — Select a tunnel and click "Connect" (or click a tunnel name in the system tray)
4. **Random connect** — Select a folder and click "Connect" to connect to a random tunnel from that folder
5. **Disconnect** — Click "Disconnect", or right-click the tray icon and select "Disconnect"
6. **Edit** — Click any tunnel to view/edit its raw `.conf` content. Click "Save" to persist changes.

## Architecture

```
tunnelvision/
├── main.go                     # Entry point, graceful shutdown, lifecycle
├── internal/
│   ├── config/
│   │   ├── parser.go           # WireGuard .conf file parser
│   │   └── store.go            # Config storage, folder metadata (metadata.json)
│   ├── tunnel/
│   │   ├── manager.go          # Connect/disconnect via nmcli + NetworkManager
│   │   └── status.go           # Active WireGuard interface detection
│   └── ui/
│       ├── mainwindow.go       # Main window layout, toolbar, dialogs
│       ├── tunnellist.go       # Tree widget for folders + tunnels
│       ├── configeditor.go     # Raw config text editor panel
│       ├── tray.go             # System tray icon and menu
│       └── resources.go        # Embedded SVG icon resources
└── assets/                     # Source SVG icons
```

**Key design decisions:**
- Uses `nmcli connection import/up/down/delete` for tunnel operations — NetworkManager handles privilege escalation via D-Bus
- Configs stored flat in `~/.config/tunnelvision/tunnels/`, folder grouping is virtual (tracked in `metadata.json`)
- One active tunnel at a time — connecting a new tunnel disconnects the current one first

## Tech Stack

- **Go** — Application language
- **[Wails v2](https://wails.io/)** — Desktop application framework (Go backend + web frontend via WebKit)
- **[fyne.io/systray](https://github.com/niceguyit/systray)** — System tray integration
- **[NetworkManager](https://networkmanager.dev/)** — WireGuard tunnel lifecycle via `nmcli` (no root/pkexec needed)

## Cross-Distribution Support

TunnelVision is designed to be distro-agnostic. It depends only on:
- `nmcli` / NetworkManager — available on all major desktop distros (Fedora, Ubuntu, Arch, openSUSE, etc.)
- `wireguard-tools` — available on all major distros
- OpenGL / X11 / Wayland libraries — standard on any desktop Linux

Only the build dependency package names differ between distros (see Building section above).

## License

MIT

## Attribution

This project was created with the assistance of **GitHub Copilot** (Claude Opus 4.6).
