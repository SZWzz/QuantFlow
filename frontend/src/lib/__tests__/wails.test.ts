import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('credential wrappers', () => {
  beforeEach(() => {
    ;(window as any).go = {
      main: {
        App: {
          SaveCredential: vi.fn().mockResolvedValue(undefined),
          GetCredential: vi.fn().mockResolvedValue({ Name: 'openai', Keys: { api_key: 'sk-test' } }),
          DeleteCredential: vi.fn().mockResolvedValue(undefined),
          ListCredentialNames: vi.fn().mockResolvedValue(['openai', 'anthropic']),
        }
      }
    }
  })

  it('saveCredential calls Go IPC', async () => {
    const { saveCredential } = await import('@/lib/wails')
    await saveCredential('openai', { api_key: 'sk-test' })
    expect((window as any).go.main.App.SaveCredential).toHaveBeenCalledWith('openai', 'api_key', { api_key: 'sk-test' })
  })

  it('getCredential returns credential', async () => {
    const { getCredential } = await import('@/lib/wails')
    const cred = await getCredential('openai')
    expect(cred).toBeDefined()
  })

  it('listCredentialNames returns names', async () => {
    const { listCredentialNames } = await import('@/lib/wails')
    const names = await listCredentialNames()
    expect(names).toEqual(['openai', 'anthropic'])
  })

  it('deleteCredential calls Go IPC', async () => {
    const { deleteCredential } = await import('@/lib/wails')
    await deleteCredential('openai')
    expect((window as any).go.main.App.DeleteCredential).toHaveBeenCalledWith('openai')
  })

  it('getCredential returns null when unavailable', async () => {
    delete (window as any).go
    const { getCredential } = await import('@/lib/wails')
    const cred = await getCredential('openai')
    expect(cred).toBeNull()
  })
})