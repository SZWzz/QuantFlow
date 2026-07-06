import { describe, it, expect } from 'vitest'
import type { LogEntry } from '@/lib/wails'

describe('useLogger', () => {
  it('LogEntry type has required fields', () => {
    const entry: LogEntry = {
      id: 1,
      time: '2026-07-06T10:00:00Z',
      level: 'info',
      message: 'test',
      attrs: { key: 'val' },
    }
    expect(entry.id).toBe(1)
    expect(entry.level).toBe('info')
    expect(entry.message).toBe('test')
  })

  it('LogEntry works without attrs', () => {
    const entry: LogEntry = {
      id: 2,
      time: '2026-07-06T10:00:01Z',
      level: 'error',
      message: 'no attrs',
    }
    expect(entry.attrs).toBeUndefined()
  })
})
