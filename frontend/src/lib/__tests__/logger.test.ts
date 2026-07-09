import { describe, it, expect, vi } from 'vitest'
import { logger, setLevel } from '../logger'

describe('logger', () => {
  it('should not log debug when level is info', () => {
    setLevel('info')
    const spy = vi.spyOn(console, 'debug')
    logger.debug('should not appear')
    expect(spy).not.toHaveBeenCalled()
    spy.mockRestore()
  })

  it('should log info when level is info', () => {
    setLevel('info')
    const spy = vi.spyOn(console, 'info')
    logger.info('test message')
    expect(spy).toHaveBeenCalled()
    spy.mockRestore()
  })

  it('should log error when level is info', () => {
    setLevel('info')
    const spy = vi.spyOn(console, 'error')
    logger.error('error message')
    expect(spy).toHaveBeenCalled()
    spy.mockRestore()
  })

  it('should log debug when level is debug', () => {
    setLevel('debug')
    const spy = vi.spyOn(console, 'debug')
    logger.debug('debug message')
    expect(spy).toHaveBeenCalled()
    spy.mockRestore()
  })
})