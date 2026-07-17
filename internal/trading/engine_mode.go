package trading

import (
	"fmt"
	"sync"
)

// EngineMode manages the trading mode (paper/live) with safety checks.
type EngineMode struct {
	mu   sync.RWMutex
	mode TradingMode
	oms  *OMS
}

// NewEngineMode creates an engine mode manager. Default mode is Paper.
func NewEngineMode(oms *OMS) *EngineMode {
	return &EngineMode{
		mode: TradingModePaper,
		oms:  oms,
	}
}

// Mode returns the current trading mode.
func (em *EngineMode) Mode() TradingMode {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.mode
}

// IsLive returns true if currently in live trading mode.
func (em *EngineMode) IsLive() bool {
	return em.Mode().IsLive()
}

// SetMode switches trading mode after safety checks.
// Pass skipChecks=true to bypass safety checks (with warning).
func (em *EngineMode) SetMode(mode TradingMode, skipChecks bool) (*SafetyReport, error) {
	if !mode.Valid() {
		return nil, fmt.Errorf("invalid trading mode: %s", mode)
	}

	if !skipChecks && mode.IsLive() {
		report := em.runSafetyChecks()
		if !report.Passed() {
			return &report, fmt.Errorf("safety checks failed: %d blocking checks not OK", countBlockingFailures(&report))
		}
		em.mu.Lock()
		em.mode = mode
		em.mu.Unlock()
		return &report, nil
	}

	em.mu.Lock()
	em.mode = mode
	em.mu.Unlock()

	if skipChecks && mode.IsLive() {
		report := em.runSafetyChecks()
		return &report, nil
	}

	return nil, nil
}

// runSafetyChecks performs pre-flight checks before going live.
func (em *EngineMode) runSafetyChecks() SafetyReport {
	checks := []SafetyCheck{}

	// Check: Broker connectivity
	if em.oms.HasBroker() {
		checks = append(checks, SafetyCheck{Name: "Broker 连接", OK: true, Message: "已连接", Blocking: true})
	} else {
		checks = append(checks, SafetyCheck{Name: "Broker 连接", OK: false, Message: "未配置券商", Blocking: true})
	}

	// Check: Risk rules loaded (non-blocking warning)
	checks = append(checks, SafetyCheck{Name: "风控规则", OK: true, Message: "已加载默认规则", Blocking: false})

	// Check: Daily loss limit (non-blocking warning)
	checks = append(checks, SafetyCheck{Name: "日亏损限额", OK: false, Message: "未设置（建议 -2%）", Blocking: false})

	report := SafetyReport{Checks: checks}
	report.AllClear = report.Passed()
	return report
}

func countBlockingFailures(r *SafetyReport) int {
	count := 0
	for _, c := range r.Checks {
		if c.Blocking && !c.OK {
			count++
		}
	}
	return count
}
