import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('credential wrappers', () => {
  beforeEach(() => {
    // Mock @wailsio/runtime Call.ByName since the typed wrappers use it
    vi.mock('@wailsio/runtime', () => ({
      Call: {
        ByName: vi.fn().mockImplementation((method: string, ...args: any[]) => {
          if (method === 'main.App.SaveCredential') return Promise.resolve(undefined)
          if (method === 'main.App.GetCredential') return Promise.resolve({ name: 'openai', type: 'api_key', keys: { api_key: 'sk-test' } })
          if (method === 'main.App.ListCredentialNames') return Promise.resolve(['openai', 'anthropic'])
          if (method === 'main.App.DeleteCredential') return Promise.resolve(undefined)
          return Promise.resolve(undefined)
        }),
      },
      default: {},
      Events: { On: vi.fn(), Emit: vi.fn() },
    }))
  })

  it('SaveCredential calls Go IPC', async () => {
    const { SaveCredential } = await import('@/lib/wails')
    await SaveCredential('openai', 'api_key', { api_key: 'sk-test' })
    const { Call } = await import('@wailsio/runtime')
    expect(Call.ByName).toHaveBeenCalledWith('main.App.SaveCredential', 'openai', 'api_key', { api_key: 'sk-test' })
  })

  it('GetCredential returns credential', async () => {
    const { GetCredential } = await import('@/lib/wails')
    const cred = await GetCredential('openai')
    expect(cred).toBeDefined()
    expect(cred.keys.api_key).toBe('sk-test')
  })

  it('listCredentialNames returns names', async () => {
    const { ListCredentialNames } = await import('@/lib/wails')
    const names = await ListCredentialNames()
    expect(names).toEqual(['openai', 'anthropic'])
  })

  it('DeleteCredential calls Go IPC', async () => {
    const { DeleteCredential } = await import('@/lib/wails')
    await DeleteCredential('openai')
    const { Call } = await import('@wailsio/runtime')
    expect(Call.ByName).toHaveBeenCalledWith('main.App.DeleteCredential', 'openai')
  })
})