//go:build gtk3

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/srnoob2570/better-rpt/internal/bridge"
)

func init() {
	application.RegisterEvent[bridge.Catalog](bridge.EventCatalogUpdated)
	application.RegisterEvent[bridge.SessionState](bridge.EventSessionUpdated)
}
