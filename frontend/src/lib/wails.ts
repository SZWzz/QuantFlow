/**
 * Wails v3 runtime wrapper.
 *
 * Provides typed wrappers around the Wails Go service calls via
 * @wailsio/runtime's Call.ByName API.
 *
 * Also creates a window.go shim (setupWailsBridge) so that existing
 * Wails v2-style (window as any).go.main.App.XXX calls work
 * transparently — no changes needed in individual panels.
 */

import { Call, Dialogs, Events } from '@wailsio/runtime'
import { logger } from '@/lib/logger'

// ---------------------------------------------------------------------------
// Market auto-detection (mirrors Go's MarketForSymbol in registry.go)
// ---------------------------------------------------------------------------

/** Detects the market type from a symbol format. */
export function detectMarket(symbol: string): string {
  const s = symbol.toUpperCase()
  // A-share suffixes
  if (s.endsWith('.SZ') || s.endsWith('.SH') || s.endsWith('.BJ')) return 'CN'
  // HK suffix
  if (s.endsWith('.HK')) return 'HK'
  // Crypto markers
  const cryptoMarkers = ['USDT', 'USDC', 'BTC', 'ETH', 'SOL', 'BNB']
  for (const m of cryptoMarkers) {
    if (s.endsWith(m)) return 'CRYPTO'
  }
  // Bare crypto bases
  if (['BTC', 'ETH', 'SOL', 'BNB'].includes(s)) return 'CRYPTO'
  // 6-digit numeric → CN A-share
  if (/^\d{6}$/.test(s)) return 'CN'
  // 1-5 digit numeric → HK (e.g. 00700, 5, 9988)
  if (/^\d{1,5}$/.test(s)) return 'HK'
  // Default → US
  return 'US'
}

/** Check if the given market is currently in trading hours (local time). */
export function isTradingHours(market: string): boolean {
  const now = new Date()
  const day = now.getDay()
  if (day === 0 || day === 6) return false
  if (market === 'CRYPTO') return true
  if (market === 'HK') {
    // HKEX: 09:30-12:00, 13:00-16:00 (Mon-Fri)
    const h = now.getHours()
    const m = now.getMinutes()
    const t = h * 60 + m
    return (t >= 9 * 60 + 30 && t <= 12 * 60) || (t >= 13 * 60 && t <= 16 * 60)
  }
  if (market === 'US') {
    // NYSE/Nasdaq: 09:30-16:00 ET ≈ 13:30-21:00 UTC
    const ut = now.getUTCHours() * 60 + now.getUTCMinutes()
    return ut >= 13 * 60 + 30 && ut <= 21 * 60
  }
  // CN default: 09:30-11:30, 13:00-15:00 (Mon-Fri)
  const h = now.getHours()
  const m = now.getMinutes()
  const t = h * 60 + m
  return (t >= 9 * 60 + 30 && t <= 11 * 60 + 30) || (t >= 13 * 60 && t <= 15 * 60)
}

// ---------------------------------------------------------------------------
// Internal call helper — delegates to Wails v3 Call.ByName
// ---------------------------------------------------------------------------

const FQN_PREFIX = 'main.App.'

async function wailsCall<T = any>(method: string, ...args: any[]): Promise<T> {
  return Call.ByName(`${FQN_PREFIX}${method}`, ...args) as Promise<T>
}

// ---------------------------------------------------------------------------
// window.go shim — Wails v2 API compat layer
// ---------------------------------------------------------------------------

/**
 * Creates a window.go.main.App Proxy that translates Wails v2-style calls
 * (e.g. window.go.main.App.GetQuote('CN', '600519'))
 * into Wails v3 Call.ByName('main.App.GetQuote', 'CN', '600519').
 *
 * Must be called early, before any Vue component mounts.
 */
export function setupWailsBridge(): void {
  if (typeof window === 'undefined') return

  // Guard: don't overwrite if already set (e.g. by Wails itself)
  const existingGo = (window as any).go
  if (existingGo) {
    logger.info('[WailsBridge] window.go already exists, skipping shim setup')
    return
  }

  ;(window as any).go = {
    main: {
      App: new Proxy(
        {},
        {
          get(_target, methodName: string) {
            if (typeof methodName !== 'string') return undefined
            return (...args: any[]) =>
              Call.ByName(`${FQN_PREFIX}${methodName}`, ...args)
          },
        }
      ),
    },
  }

  logger.info('[WailsBridge] window.go shim installed (Wails v3 → v2 compat)')

  installDialogPolyfill()
}

/**
 * Installs async polyfills for window.confirm / window.alert.
 *
 * Wails v3's webview disables the synchronous native dialog primitives:
 * `window.confirm()` returns `false` without ever showing a prompt, and
 * `window.alert()` is a no-op. Any code path gated by `if (!confirm(...))`
 * therefore silently aborts — e.g. every delete action in BacktestPanel.
 *
 * We override both with implementations backed by Wails' native MessageDialog
 * (Dialogs.Question / Dialogs.Info), which DO render in the desktop window.
 * The signatures are kept synchronous-friendly where possible, but because the
 * underlying dialog is async, callers that `await confirm(...)` get correct
 * behavior; bare `if (confirm(...))` callers still work because the returned
 * promise is thenable and truthy-gates via await at the call site only when
 * awaited. For this codebase every confirm() caller uses the
 * `if (!confirm(...)) return` pattern without await, so we additionally expose
 * the async helpers `confirmDialog` / `alertDialog` for new code.
 */
function installDialogPolyfill(): void {
  if (typeof window === 'undefined') return
  if ((window as any).__wailsDialogPolyfill) {
    logger.info('[WailsBridge] dialog polyfill already installed, skipping')
    return
  }
  ;(window as any).__wailsDialogPolyfill = true

  // window.confirm → Wails Question dialog.
  // Returns a Promise<boolean> so `await confirm(...)` works. Legacy callers
  // that do `if (confirm(...))` without await see a truthy Promise and proceed
  // — those callers must be migrated to await (see BacktestPanel). We keep the
  // override so that the *awaited* form works everywhere going forward.
  ;(window as any).confirm = async (message: string): Promise<boolean> => {
    try {
      const chosen = await Dialogs.Question({
        Title: '确认',
        Message: String(message ?? ''),
        Buttons: [
          { Label: '确定', IsDefault: true },
          { Label: '取消', IsCancel: true },
        ],
      })
      // "取消" or "" (window closed) → not confirmed
      return chosen === '确定'
    } catch (e) {
      logger.error('[WailsBridge] confirm dialog failed:', e)
      return false
    }
  }

  // window.alert → Wails Info dialog. Resolves when dismissed.
  ;(window as any).alert = async (message: string): Promise<void> => {
    try {
      await Dialogs.Info({
        Title: '提示',
        Message: String(message ?? ''),
        Buttons: [{ Label: '确定', IsDefault: true }],
      })
    } catch (e) {
      logger.error('[WailsBridge] alert dialog failed:', e)
    }
  }

  logger.info('[WailsBridge] window.confirm / window.alert polyfilled via MessageDialog')
}

// ---------------------------------------------------------------------------
// Async dialog helpers — preferred for new code
// ---------------------------------------------------------------------------

/** Async confirm dialog backed by Wails MessageDialog. Resolves true on accept. */
export async function confirmDialog(message: string, title = '确认'): Promise<boolean> {
  try {
    const chosen = await Dialogs.Question({
      Title: title,
      Message: String(message ?? ''),
      Buttons: [{ Label: '确定', IsDefault: true }, { Label: '取消', IsCancel: true }],
    })
    return chosen === '确定'
  } catch (e) {
    logger.error('[WailsBridge] confirmDialog failed:', e)
    return false
  }
}

/** Async alert dialog backed by Wails MessageDialog. Resolves when dismissed. */
export async function alertDialog(message: string, title = '提示'): Promise<void> {
  try {
    await Dialogs.Info({ Title: title, Message: String(message ?? ''), Buttons: [{ Label: '确定', IsDefault: true }] })
  } catch (e) {
    logger.error('[WailsBridge] alertDialog failed:', e)
  }
}

// ---------------------------------------------------------------------------
// Typed service methods — use these for new code that wants type safety
// ---------------------------------------------------------------------------

export async function GetVersion(): Promise<string> {
  return wailsCall<string>('GetVersion')
}

export async function ListNodes(): Promise<
  Array<{ node_type: string; category: string }>
> {
  return wailsCall('ListNodes')
}

export async function GetNodePorts(nodeType: string): Promise<{ inputs: Array<{ name: string; type: string }>; outputs: Array<{ name: string; type: string }> }> {
  return wailsCall('GetNodePorts', nodeType)
}

export async function ValidateWorkflow(jsonDef: string): Promise<string> {
  return wailsCall('ValidateWorkflow', jsonDef)
}

export async function RunWorkflow(jsonDef: string): Promise<any> {
  return wailsCall('RunWorkflow', jsonDef)
}

export async function LoadWorkflow(id: string): Promise<any> {
  return wailsCall('LoadWorkflow', id)
}

export async function SaveWorkflow(jsonDef: string): Promise<string> {
  return wailsCall('SaveWorkflow', jsonDef)
}

export async function ListWorkflows(): Promise<
  Array<{ id: string; name: string; description: string; updated_at: string }>
> {
  return wailsCall('ListWorkflows')
}

// --- Trading / Portfolio APIs ---

export async function GetPortfolioSummary(): Promise<Record<string, any>> {
  return wailsCall('GetPortfolioSummary')
}

export async function GetTrades(): Promise<
  Array<{
    ID: string
    Symbol: string
    Side: string
    Quantity: number
    Price: number
    Timestamp: string
    PnL: number
  }>
> {
  return wailsCall('GetTrades')
}

export async function GetOrders(): Promise<
  Array<{
    ID: string
    Symbol: string
    Side: string
    Quantity: number
    Price: number
    Status: string
    PlacedAt: string
  }>
> {
  return wailsCall('GetOrders')
}

export async function GetPositions(): Promise<
  Array<{
    Symbol: string
    Quantity: number
    AvgPrice: number
    MarketPrice: number
    PnL: number
    PnLPct: number
  }>
> {
  return wailsCall('GetPositions')
}

// ── Color scheme helpers ─────────────────────────────────────────────────

/**
 * Returns CSS color for a price change percentage based on the selected color scheme.
 * Default: CN convention (涨红跌绿). US convention: 涨绿跌红.
 */
export function pctColor(pct: number, scheme: string = 'cn'): string {
  const up = scheme === 'us' ? '#22c55e' : '#ef4444'  // US: green up, CN: red up
  const down = scheme === 'us' ? '#ef4444' : '#22c55e'
  return pct >= 0 ? up : down
}

/**
 * Returns change class suffix for use with CSS class bindings.
 */
export function changeColorClass(pct: number, scheme: string = 'cn'): string {
  if (scheme === 'us') return pct >= 0 ? 'up' : 'down' // CSS handles mapping
  return pct >= 0 ? 'up' : 'down'
}

// ── Log Panel ────────────────────────────────────────────────────────

export interface LogEntry {
  id: number
  time: string
  level: string
  message: string
  attrs?: Record<string, any>
}

export async function GetLogs(afterID: number): Promise<LogEntry[]> {
  return wailsCall<LogEntry[]>('GetLogs', afterID)
}

// ── Update Management ─────────────────────────────────────────────────

export interface UpdateInfo {
  has_update: boolean
  current_version: string
  latest_version: string
  asset_url: string
  asset_size: number
  checksum: string
  changelog: string
}

export async function CheckUpdate(): Promise<UpdateInfo> {
  return wailsCall<UpdateInfo>('CheckUpdate')
}

export async function ApplyUpdate(assetURL: string, checksum: string): Promise<void> {
  return wailsCall<void>('ApplyUpdate', assetURL, checksum)
}

export async function GetUpdateInterval(): Promise<string> {
  return wailsCall<string>('GetUpdateInterval')
}

export async function SetUpdateInterval(interval: string): Promise<void> {
  return wailsCall<void>('SetUpdateInterval', interval)
}

// ── Data Lifecycle Management ──────────────────────────────────────────

export interface TableStat {
  table: string
  rows: number
  size_bytes: number
  oldest: string
  newest: string
}

export interface ArchiveResult {
  id: number
  source: string
  symbol: string
  interval: string
  date_from: string
  date_to: string
  row_count: number
  compressed_bytes: number
}

export interface CleanupResult {
  affected_rows: number
  preview: Record<string, unknown>[]
  table: string
  dry_run: boolean
}

export async function GetStorageStats(): Promise<TableStat[]> {
  return wailsCall<TableStat[]>('GetStorageStats')
}

export async function ArchiveData(source: string, symbol: string, before: string): Promise<ArchiveResult> {
  return wailsCall<ArchiveResult>('ArchiveData', source, symbol, before)
}

export async function ExportData(table: string, symbol: string, interval: string, format: string, dateFrom: string, dateTo: string): Promise<string> {
  return wailsCall<string>('ExportData', table, symbol, interval, format, dateFrom, dateTo)
}

export async function ImportData(filePath: string, table: string): Promise<number> {
  return wailsCall<number>('ImportData', filePath, table)
}

export async function CleanupData(table: string, symbol: string, before: string, dryRun: boolean): Promise<CleanupResult> {
  return wailsCall<CleanupResult>('CleanupData', table, symbol, before, dryRun)
}

// ── Layout Template Management ─────────────────────────────────────────

export async function SaveLayout(name: string, layoutJSON: string): Promise<void> {
  return wailsCall<void>('SaveLayout', name, layoutJSON)
}

export async function LoadLayout(name: string): Promise<string> {
  return wailsCall<string>('LoadLayout', name)
}

export async function ListLayouts(): Promise<string[]> {
  return wailsCall<string[]>('ListLayouts')
}

export async function DeleteLayout(name: string): Promise<void> {
  return wailsCall<void>('DeleteLayout', name)
}

// ── Crash Reports ──────────────────────────────────────────────────────

/** Non-PII application state captured at crash time (mirrors Go crash.AppState). */
export interface CrashAppState {
  trading_mode: string
  active_brokers: string[]
  panel_count: number
  workflow_count: number
  uptime_seconds: number
}

/** A saved crash report (mirrors Go crash.CrashReport JSON). */
export interface CrashReport {
  id: string
  timestamp: string
  version: string
  go_version: string
  os: string
  arch: string
  build_mode: string
  panic: string
  stack: string
  logs: string[] | null
  app_state: CrashAppState
}

export async function ListCrashReports(): Promise<CrashReport[]> {
  return wailsCall<CrashReport[]>('ListCrashReports')
}

export async function DeleteCrashReport(id: string): Promise<void> {
  return wailsCall<void>('DeleteCrashReport', id)
}

/** Opt-in upload of a crash report to the configured endpoint. */
export async function UploadCrashReport(id: string): Promise<void> {
  return wailsCall<void>('UploadCrashReport', id)
}

/** Returns the platform-specific directory where crash reports are saved. */
export async function GetCrashDir(): Promise<string> {
  return wailsCall<string>('GetCrashDir')
}

// ── Connection Status ───────────────────────────────────────────────────

export interface ConnectionStatus {
  markets: Record<string, string>
  brokers: Record<string, string>
  python: string
}

/** Returns live connection status for markets, brokers, and Python sidecar. */
export async function GetConnectionStatus(): Promise<ConnectionStatus> {
  return wailsCall<ConnectionStatus>('GetConnectionStatus')
}

// ── Daily Reports ───────────────────────────────────────────────────────

export interface DailyReport {
  date: string
  market_value: number
  day_pnl: number
  day_pnl_percent: number
  total_pnl: number
  total_pnl_percent: number
  trades: number
  commission: number
  tax: number
  max_drawdown: number
  best_trade: { symbol: string; side: string; quantity: number; price: number; pnl: number } | null
  worst_trade: { symbol: string; side: string; quantity: number; price: number; pnl: number } | null
  positions: Array<{ symbol: string; quantity: number; market_val: number; pnl: number; pnl_pct: number }>
  notes: string
}

export async function GetDailyReport(date: string): Promise<DailyReport | null> {
  return wailsCall<DailyReport | null>('GetDailyReport', date)
}

export async function ListDailyReports(limit: number): Promise<DailyReport[]> {
  return wailsCall<DailyReport[]>('ListDailyReports', limit)
}

export async function GenerateDailyReport(date: string): Promise<DailyReport> {
  return wailsCall<DailyReport>('GenerateDailyReport', date)
}

export async function ExportReportCSV(date: string): Promise<void> {
  return wailsCall<void>('ExportReportCSV', date)
}

// ── Trading Mode Management ──────────────────────────────────────────────

export interface SafetyReport {
  checks: Array<{ name: string; ok: boolean; message: string; blocking: boolean }>
  all_clear: boolean
}

export async function GetTradingMode(): Promise<string> {
  return wailsCall<string>('GetTradingMode')
}

export async function SwitchToLive(skipChecks: boolean): Promise<SafetyReport> {
  return wailsCall<SafetyReport>('SwitchToLive', skipChecks)
}

export async function SwitchToPaper(): Promise<void> {
  return wailsCall<void>('SwitchToPaper')
}

export async function EmergencyClose(): Promise<Record<string, any>> {
  return wailsCall<Record<string, any>>('EmergencyClose')
}

// ── Position Reconciliation ──────────────────────────────────────────────

export interface ReconciliationDiff {
  symbol: string
  oms_quantity: number
  broker_quantity: number
  oms_avg_price: number
  broker_avg_price: number
}

export interface ReconciliationReport {
  id: number
  created_at: string
  broker_name: string
  match_count: number
  diff_count: number
  dirt: string
  diffs: ReconciliationDiff[]
  oms_only: string[]
  broker_only: string[]
}

export async function ReconcileAll(): Promise<ReconciliationReport[]> {
  return wailsCall<ReconciliationReport[]>('ReconcileAll')
}

// ── Sector Dashboard ──────────────────────────────────────────────────────

export async function GetSectorHeatmap(market: string): Promise<Array<{name:string;change_pct:number;volume:number;pe:number;pe_pct:number}>> {
  return wailsCall('GetSectorHeatmap', market)
}
export async function GetSectorValuation(market: string): Promise<Array<{name:string;pe:number;pe_pct:number;pb:number;pb_pct:number;roe:number}>> {
  return wailsCall('GetSectorValuation', market)
}

// ── Valuation & Dupont ────────────────────────────────────────────────────

export interface BandResult {
  symbol: string; metric: string; current: number; mean: number; stddev: number; percentile: number
  points: Array<{date:string;close:number;band_1:number;band_2:number;band_3:number;band_4:number;band_5:number}>
}
export interface DupontBreakdown { symbol:string;roe:number;net_margin:number;asset_turnover:number;equity_multiplier:number;gross_margin:number;eps:number }
export interface PeerRadar { symbol:string;name:string;metrics:Record<string,number> }

export async function GetPriceBand(symbol: string, market: string, interval: string, lookbackDays: number): Promise<BandResult> {
  return wailsCall('GetPriceBand', symbol, market, interval, lookbackDays)
}
export async function GetDupontAnalysis(symbol: string): Promise<DupontBreakdown> {
  return wailsCall('GetDupontAnalysis', symbol)
}
export async function GetPeerRadar(symbol: string): Promise<PeerRadar[]> {
  return wailsCall('GetPeerRadar', symbol)
}

// ── P1 Analysis ───────────────────────────────────────────────────────────

export interface MacroSnapshot {
  growth: Array<{country:string;name:string;value:number;unit:string;date:string;change:number}>
  inflation: Array<{country:string;name:string;value:number;unit:string;date:string;change:number}>
  monetary: Array<{country:string;name:string;value:number;unit:string;date:string;change:number}>
  policy: Array<{country:string;name:string;value:number;unit:string;date:string;change:number}>
  updated_at: string
}
export interface StyleQuadrant { index:string;size:number;style:number;return_1m:number }
export interface MarketSentimentGauge { limit_up:number;limit_down:number;turnover:number;northbound_cum:number }
export interface EventStudyResult { symbol:string;event_date:string;event_type:string;window:number;car:number;daily_ar:Array<{date:string;day:number;ar:number;car:number;stock_r:number;bench_r:number}> }

export async function GetMacroSnapshot(country: string): Promise<MacroSnapshot> {
  return wailsCall('GetMacroSnapshot', country)
}
export async function GetStyleQuadrant(market: string): Promise<StyleQuadrant[]> {
  return wailsCall('GetStyleQuadrant', market)
}
export async function GetMarketSentiment(market: string): Promise<MarketSentimentGauge> {
  return wailsCall('GetMarketSentiment', market)
}
export async function ComputeEventStudy(symbol: string, market: string, interval: string, eventDate: string, window: number): Promise<EventStudyResult> {
  return wailsCall('ComputeEventStudy', symbol, market, interval, eventDate, window)
}

// ── P2 Analysis ───────────────────────────────────────────────────────────

export interface ShareholderRecord { name:string;type:string;shares:number;pct:number;change:number;market_value:number;report_date:string }
export interface UnlockEvent { symbol:string;name:string;unlock_date:string;unlock_shares:number;unlock_pct:number;float_ratio:number;market_value:number }
export interface FactorAttribution { total_return:number;market_beta:number;style_factors:Record<string,number>;industry_factors:Record<string,number>;alpha:number }

export async function GetTop10Holders(symbol: string): Promise<ShareholderRecord[]> {
  return wailsCall('GetTop10Holders', symbol)
}
export async function GetUnlockCalendar(days: number): Promise<UnlockEvent[]> {
  return wailsCall('GetUnlockCalendar', days)
}
export async function GetFactorAttribution(totalReturn: number): Promise<FactorAttribution> {
  return wailsCall('GetFactorAttribution', totalReturn)
}

export async function GetReconciliationReports(limit: number): Promise<ReconciliationReport[]> {
  return wailsCall<ReconciliationReport[]>('GetReconciliationReports', limit)
}

// ── Market Data Wrappers (for data store migration) ─────────────────────

export async function fetchMarketOverview(market: string): Promise<Record<string, any>> {
  return wailsCall('GetMarketOverview', market)
}

export async function fetchMinuteLine(symbol: string, sinceTimestamp: number): Promise<any> {
  return wailsCall('GetMinuteLine', symbol, sinceTimestamp)
}

export async function GetIndustryRanks(mkt: string, topN: number): Promise<Array<{ name: string; change_pct?: number; changePct?: number }>> {
  return wailsCall('GetIndustryRanks', mkt, topN)
}

// ── Credential Management ────────────────────────────────────────────────

export async function GetCredential(name: string): Promise<{ name: string; type: string; keys: Record<string, string> }> {
  return wailsCall('GetCredential', name)
}

export async function SaveCredential(name: string, type: string, keys: Record<string, string>): Promise<void> {
  return wailsCall('SaveCredential', name, type, keys)
}

export async function DeleteCredential(name: string): Promise<void> {
  return wailsCall('DeleteCredential', name)
}

export async function ListCredentialNames(): Promise<string[]> {
  return wailsCall<string[]>('ListCredentialNames')
}

/**
 * Subscribes to crash reports pushed from the Go backend via Wails events.
 * Returns an unsubscribe function.
 *
 * Reserved for future use — the Go side does not yet emit this event; crash
 * detection currently uses the localStorage watermark in the crash store.
 */
export function onCrashReport(cb: (report: CrashReport) => void): () => void {
  return Events.On('crash:detected', (ev: { data: any }) => {
    // Wails v3 delivers event payloads in ev.data (arrays for multi-arg emits).
    const payload = Array.isArray(ev.data) ? ev.data[0] : ev.data
    if (payload) cb(payload as CrashReport)
  })
}

