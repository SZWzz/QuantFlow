type LogLevel = 'debug' | 'info' | 'warn' | 'error'

const LEVEL_RANK: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
}

let currentLevel: LogLevel =
  (import.meta.env.VITE_LOG_LEVEL as LogLevel) || 'info'

export function setLevel(level: LogLevel) {
  currentLevel = level
}

function shouldLog(level: LogLevel): boolean {
  return LEVEL_RANK[level] >= LEVEL_RANK[currentLevel]
}

function formatArgs(args: unknown[]): unknown[] {
  return args.map((a) => (typeof a === 'object' ? JSON.stringify(a, null, 2) : a))
}

export const logger = {
  debug(...args: unknown[]) {
    if (shouldLog('debug')) {
      console.debug('[DEBUG]', ...formatArgs(args))
    }
  },

  info(...args: unknown[]) {
    if (shouldLog('info')) {
      console.info('[INFO]', ...formatArgs(args))
    }
  },

  warn(...args: unknown[]) {
    if (shouldLog('warn')) {
      console.warn('[WARN]', ...formatArgs(args))
    }
  },

  error(...args: unknown[]) {
    if (shouldLog('error')) {
      console.error('[ERROR]', ...formatArgs(args))
    }
  },
}