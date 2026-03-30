package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fyne.io/fyne/v2/app"

	"github.com/ap-andersson/tunnelvision/internal/config"
	"github.com/ap-andersson/tunnelvision/internal/tunnel"
	"github.com/ap-andersson/tunnelvision/internal/ui"
)

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

	// Create Fyne app
	fyneApp := app.NewWithID("com.github.ap-andersson.tunnelvision")
	fyneApp.SetIcon(ui.ResourceAppIcon)

	// Build UI
	application := ui.NewApp(fyneApp, store, manager, status)

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

	// Handle graceful shutdown (SIGINT, SIGTERM)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		gracefulShutdown(manager)
		os.Exit(0)
	}()

	// Hook into Fyne's lifecycle for clean shutdown
	fyneApp.Lifecycle().SetOnStopped(func() {
		gracefulShutdown(manager)
	})

	// Show and run
	application.Show()
	fyneApp.Run()
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
