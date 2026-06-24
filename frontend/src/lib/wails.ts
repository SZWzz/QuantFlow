/**
 * Wails v3 runtime wrapper.
 *
 * Provides typed wrappers around the Wails Go function calls.
 * In development, these call the Go backend via the Wails IPC bridge.
 * The generated bindings (wailsjs/) will eventually replace this manual wrapper.
 */

// Placeholder for the Wails runtime. In Wails v3, the runtime is injected
// via the `@wailsapp/runtime` package or the generated bindings.
// For now, we use a fallback that works when running in the browser without Wails.

interface WailsRuntime {
  Call: (serviceName: string, methodName: string, ...args: any[]) => Promise<any>
}

function getRuntime(): WailsRuntime | null {
  if (typeof window !== 'undefined' && (window as any).wails) {
    return (window as any).wails as WailsRuntime
  }
  return null
}

async function call<T = any>(method: string, ...args: any[]): Promise<T> {
  const rt = getRuntime()
  if (rt) {
    return rt.Call('main.App', method, ...args) as Promise<T>
  }
  // Fallback: log and return mock data for development without Wails
  console.warn(`[Wails] No runtime available. Mock call: App.${method}`, args)
  throw new Error(`Wails runtime not available for App.${method}`)
}

// --- App service methods ---

export async function GetVersion(): Promise<string> {
  return call<string>('GetVersion')
}

export async function ListNodes(): Promise<Array<{ node_type: string; category: string }>> {
  return call('ListNodes')
}

export async function ValidateWorkflow(jsonDef: string): Promise<string> {
  return call('ValidateWorkflow', jsonDef)
}

export async function RunWorkflow(jsonDef: string): Promise<any> {
  return call('RunWorkflow', jsonDef)
}

export async function LoadWorkflow(id: string): Promise<any> {
  return call('LoadWorkflow', id)
}

export async function SaveWorkflow(jsonDef: string): Promise<string> {
  return call('SaveWorkflow', jsonDef)
}

export async function ListWorkflows(): Promise<Array<{ id: string; name: string; description: string; updated_at: string }>> {
  return call('ListWorkflows')
}

// --- Trading / Portfolio APIs ---

export async function GetPortfolioSummary(): Promise<Record<string, any>> {
  return call('GetPortfolioSummary')
}

export async function GetTrades(): Promise<Array<{ ID: string; Symbol: string; Side: string; Quantity: number; Price: number; Timestamp: string; PnL: number }>> {
  return call('GetTrades')
}

export async function GetOrders(): Promise<Array<{ ID: string; Symbol: string; Side: string; Quantity: number; Price: number; Status: string; PlacedAt: string }>> {
  return call('GetOrders')
}

export async function GetPositions(): Promise<Array<{ Symbol: string; Quantity: number; AvgPrice: number; MarketPrice: number; PnL: number; PnLPct: number }>> {
  return call('GetPositions')
}
