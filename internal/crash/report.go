// internal/crash/report.go
package crash

import (
	"crypto/rand"
	"fmt"
	"runtime"
	"time"
)

type CrashReport struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	GoVersion string    `json:"go_version"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	BuildMode string    `json:"build_mode"`

	Panic string   `json:"panic"`
	Stack string   `json:"stack"`
	Logs  []string `json:"logs"`

	AppState AppState `json:"app_state"`
}

type AppState struct {
	TradingMode   string   `json:"trading_mode"`
	ActiveBrokers []string `json:"active_brokers"`
	PanelCount    int      `json:"panel_count"`
	WorkflowCount int      `json:"workflow_count"`
	UptimeSeconds int64    `json:"uptime_seconds"`
}

var (
	appVersion  = "0.0.1"
	buildMode   = "dev"
	uptimeFn    = func() int64 { return 0 }
	panelCount  = func() int { return 0 }
	workflowCnt = func() int { return 0 }
	brokersFn   = func() []string { return nil }
	tradingMode = func() string { return "unknown" }
)

func SetAppInfo(version, mode string) {
	appVersion = version
	buildMode = mode
}

func SetStateGetters(
	uptime func() int64,
	panels func() int,
	workflows func() int,
	brokers func() []string,
	mode func() string,
) {
	uptimeFn = uptime
	panelCount = panels
	workflowCnt = workflows
	brokersFn = brokers
	tradingMode = mode
}

func newCrashID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func NewCrashReport(panicVal, stack string, logs []string, state AppState) *CrashReport {
	if state.TradingMode == "" {
		state.TradingMode = tradingMode()
	}
	if state.UptimeSeconds == 0 {
		state.UptimeSeconds = uptimeFn()
	}
	if state.PanelCount == 0 {
		state.PanelCount = panelCount()
	}
	if state.WorkflowCount == 0 {
		state.WorkflowCount = workflowCnt()
	}
	if len(state.ActiveBrokers) == 0 {
		state.ActiveBrokers = brokersFn()
	}

	return &CrashReport{
		ID:        newCrashID(),
		Timestamp: time.Now(),
		Version:   appVersion,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		BuildMode: buildMode,
		Panic:     panicVal,
		Stack:     stack,
		Logs:      logs,
		AppState:  state,
	}
}

func (r *CrashReport) FileName() string {
	return fmt.Sprintf("%s.json", r.Timestamp.Format("2006-01-02T15:04:05"))
}
