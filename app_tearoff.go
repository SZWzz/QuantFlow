package main

import (
	"fmt"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// tearOffEntry holds information about a tear-off panel window.
type tearOffEntry struct {
	Win        *application.WebviewWindow
	PanelID    string
	InstanceID string
	Label      string
	Params     string // JSON
}

// TearOffPanel creates a new OS window containing the specified panel.
func (a *App) TearOffPanel(panelId, instanceId, label, paramsJson string) error {
	if a.wailsApp == nil {
		return fmt.Errorf("tearoff: wails app not initialized")
	}
	win := a.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      fmt.Sprintf("tearoff-%s", instanceId),
		Title:     label,
		Width:     800,
		Height:    600,
		MinWidth:  400,
		MinHeight: 300,
		URL:       fmt.Sprintf("/#/tearoff/%s", instanceId),
	})
	entry := &tearOffEntry{
		Win: win, PanelID: panelId,
		InstanceID: instanceId, Label: label, Params: paramsJson,
	}
	a.tearOffWindowsMu.Lock()
	defer a.tearOffWindowsMu.Unlock()
	a.tearOffWindows[instanceId] = entry

	win.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		slog.Debug("tear-off window closing", "instanceId", instanceId, "panelId", panelId)
		a.tearOffWindowsMu.Lock()
		delete(a.tearOffWindows, instanceId)
		a.tearOffWindowsMu.Unlock()
	})
	slog.Info("tear-off panel opened", "instanceId", instanceId, "panelId", panelId, "label", label)
	return nil
}

// CloseTearOffWindow closes a specific tear-off window by instance ID.
func (a *App) CloseTearOffWindow(instanceId string) error {
	a.tearOffWindowsMu.RLock()
	entry, ok := a.tearOffWindows[instanceId]
	a.tearOffWindowsMu.RUnlock()
	if !ok {
		return fmt.Errorf("tear-off window not found: %s", instanceId)
	}
	entry.Win.Close()
	return nil
}

// GetTearOffPanelInfo returns panelId, label, and params JSON for a tear-off panel.
func (a *App) GetTearOffPanelInfo(instanceId string) (string, string, string, error) {
	a.tearOffWindowsMu.RLock()
	entry, ok := a.tearOffWindows[instanceId]
	a.tearOffWindowsMu.RUnlock()
	if !ok {
		return "", "", "", fmt.Errorf("tear-off panel not found: %s", instanceId)
	}
	return entry.PanelID, entry.Label, entry.Params, nil
}

// ListTearOffWindows returns instance IDs of all open tear-off windows.
func (a *App) ListTearOffWindows() []string {
	a.tearOffWindowsMu.RLock()
	defer a.tearOffWindowsMu.RUnlock()
	ids := make([]string, 0, len(a.tearOffWindows))
	for id := range a.tearOffWindows {
		ids = append(ids, id)
	}
	return ids
}
