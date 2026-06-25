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

import { Call } from '@wailsio/runtime'

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
  // Default → US
  return 'US'
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
    console.log('[WailsBridge] window.go already exists, skipping shim setup')
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

  console.log('[WailsBridge] window.go shim installed (Wails v3 → v2 compat)')
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
