package main

import (
	"context"
	"embed"
	"log"
	"time"

	"github.com/ap-andersson/tunnelvision/internal/config"
	logstreamer "github.com/ap-andersson/tunnelvision/internal/log"
	"github.com/ap-andersson/tunnelvision/internal/tray"
	"github.com/ap-andersson/tunnelvision/internal/tunnel"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend
var assets embed.FS

//go:embed appicon.png
var appIcon []byte

func main() {
	// Initialize config store
	configDir := config.DefaultConfigDir()
	store, err := config.NewStore(configDir)
	if err != nil {
		log.Fatalf("Failed to initialize config store: %v", err)
	}

	// Initialize tunnel manager and status checker
	manager := tunnel.NewManager()
	status := tunnel.NewStatus()

	// Initialize log streamer
	streamer := logstreamer.NewStreamer()
	if err := streamer.Start(); err != nil {
		log.Printf("Warning: failed to start log streamer: %v", err)
	}

	// Create service layer for Wails bindings
	tunnelService := NewTunnelService(store, manager, status)
	configService := NewConfigService(store)
	logService := NewLogService(streamer)

	// System tray (set up in OnStartup when we have the Wails context)
	var appTray *tray.Tray

	// Create and run Wails application
	err = wails.Run(&options.App{
		Title:             "TunnelVision",
		Width:             960,
		Height:            650,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Linux: &linux.Options{
			Icon:        appIcon,
			ProgramName: "tunnelvision",
		},
		OnStartup: func(ctx context.Context) {
			tunnelService.SetContext(ctx)
			configService.SetContext(ctx)
			logService.SetContext(ctx)

			// Set up system tray
			appTray = tray.New(store, manager, status,
				func() { wailsRuntime.WindowShow(ctx) },
				func() { wailsRuntime.Quit(ctx) },
			)
			appTray.Setup()

			// Single callback for state changes: update both tray and frontend
			manager.SetOnStateChange(func() {
				appTray.Update()
				tunnelService.EmitStateChanged()
			})

			// Update tray when config changes (import, delete, rename, etc.)
			configService.SetOnConfigChange(func() {
				appTray.Update()
			})

			// Start periodic status refresh
			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				for range ticker.C {
					if err := status.Refresh(); err != nil {
						log.Printf("Status refresh error: %v", err)
					}
				}
			}()
		},
		OnShutdown: func(ctx context.Context) {
			streamer.Stop()
			if appTray != nil {
				appTray.Stop()
			}
			gracefulShutdown(manager)
		},
		Bind: []interface{}{
			tunnelService,
			configService,
			logService,
		},
	})
	if err != nil {
		log.Fatalf("Wails error: %v", err)
	}
}

func gracefulShutdown(manager *tunnel.Manager) {
	if !manager.IsConnected() {
		return
	}
	log.Println("TunnelVision: disconnecting active tunnel before shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := manager.DisconnectGraceful(ctx); err != nil {
		log.Printf("Warning: failed to disconnect tunnel on shutdown: %v", err)
	} else {
		log.Println("TunnelVision: tunnel disconnected successfully")
	}
}
