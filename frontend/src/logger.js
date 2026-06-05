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
  }
}

export const log = {
  debug(...args) { sendLog('debug', ...args) },
  info(...args)  { sendLog('info', ...args) },
  warn(...args)  { sendLog('warn', ...args) },
  error(...args) { sendLog('error', ...args) },
}
