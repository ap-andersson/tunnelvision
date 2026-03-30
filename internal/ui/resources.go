package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icons/icon_connected.svg
var iconConnectedSVG []byte

//go:embed icons/icon_disconnected.svg
var iconDisconnectedSVG []byte

//go:embed icons/appicon.svg
var appIconSVG []byte

var resourceIconConnectedPng = &fyne.StaticResource{
	StaticName:    "icon_connected.svg",
	StaticContent: iconConnectedSVG,
}

var resourceIconDisconnectedPng = &fyne.StaticResource{
	StaticName:    "icon_disconnected.svg",
	StaticContent: iconDisconnectedSVG,
}

var ResourceAppIcon = &fyne.StaticResource{
	StaticName:    "appicon.svg",
	StaticContent: appIconSVG,
}
