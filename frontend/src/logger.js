// Logger forwards front-end log calls to the Go backend via Wails events.
// All logs appear in the same structured log file. Use instead of console.log / console.error.
//
// This approach avoids relying on generated Wails bindings; instead it uses
// Events.Emit (from @wailsio/runtime) which works without binding files.
//
// Usage:
//   import { log } from '../logger'
//   log.info('ComponentName', 'doing something', { key: value })
//   log.error('ComponentName', 'something failed', error)

import { Events } from '@wailsio/runtime'

function safeStringify(obj) {
  try {
    return JSON.stringify(obj)
  } catch {
    return String(obj)
  }
}

function formatArgs(...args) {
  return args.map(a => (typeof a === 'object' ? safeStringify(a) : String(a))).join(' ')
}

function sendLog(level, ...args) {
  const component = args[0] || 'frontend'
  const msg = formatArgs(...args.slice(1))
  try {
    Events.Emit('frontend-log', { level, component, msg })
  } catch {
    // Silently ignore if runtime not ready.
  }
}

export const log = {
  debug(...args) { sendLog('debug', ...args) },
  info(...args)  { sendLog('info', ...args) },
  warn(...args)  { sendLog('warn', ...args) },
  error(...args) { sendLog('error', ...args) },
}
