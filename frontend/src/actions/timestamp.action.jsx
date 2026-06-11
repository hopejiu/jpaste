import { useState, useCallback, useEffect, useRef } from 'react'
import { formBody, label as lbl, textarea, output, copyBtn, copyBtnSuccess } from './actionStyles'

// Reasonable timestamp range: 2000-01-01 to 2100-01-01 (seconds)
const MIN_SEC = 946684800
const MAX_SEC = 4102444800

function isTimestamp(s) {
  const trimmed = s.trim()
  if (!/^\d+$/.test(trimmed)) return false
  const len = trimmed.length
  if (len === 10) {
    const v = Number(trimmed)
    return v >= MIN_SEC && v <= MAX_SEC
  }
  if (len === 13) {
    const v = Number(trimmed)
    return v >= MIN_SEC * 1000 && v <= MAX_SEC * 1000
  }
  return false
}

function toDatetimeLocal(ms) {
  const d = new Date(ms)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function TimestampModal({ content }) {
  const trimmed = content.trim()
  const isMs = trimmed.length === 13

  const initialMs = Number(trimmed) * (isMs ? 1 : 1000)

  const [timestampInput, setTimestampInput] = useState(trimmed)
  const [mode, setMode] = useState(isMs ? 'ms' : 's') // 's' | 'ms'
  const [datetimeInput, setDatetimeInput] = useState(toDatetimeLocal(initialMs))
  const [error, setError] = useState(null)
  const [copiedTs, setCopiedTs] = useState(false)
  const [copiedDt, setCopiedDt] = useState(false)
  const syncing = useRef(false)

  // When timestampInput or mode changes → update datetime
  const syncFromTimestamp = useCallback(() => {
    if (syncing.current) return
    syncing.current = true
    const ts = timestampInput.trim()
    if (!/^\d+$/.test(ts)) {
      setError('请输入纯数字时间戳')
      syncing.current = false
      return
    }
    const ms = Number(ts) * (mode === 's' ? 1000 : 1)
    const year = new Date(ms).getFullYear()
    if (year < 1970 || year > 2100) {
      setError('时间戳超出有效范围')
      syncing.current = false
      return
    }
    setError(null)
    setDatetimeInput(toDatetimeLocal(ms))
    syncing.current = false
  }, [timestampInput, mode])

  // When datetimeInput changes → update timestamp
  const syncFromDatetime = useCallback(() => {
    if (syncing.current) return
    syncing.current = true
    const dt = new Date(datetimeInput)
    if (isNaN(dt.getTime())) {
      setError('无效的日期时间')
      syncing.current = false
      return
    }
    setError(null)
    const ms = dt.getTime()
    setTimestampInput(mode === 's' ? String(Math.floor(ms / 1000)) : String(ms))
    syncing.current = false
  }, [datetimeInput, mode])

  // Auto-sync on timestamp change (debounced feel via useEffect)
  useEffect(() => {
    syncFromTimestamp()
  }, [timestampInput, mode]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleCopy = async (text, setCopied) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {}
  }

  const handleModeChange = (newMode) => {
    if (newMode === mode) return
    const ts = timestampInput.trim()
    if (/^\d+$/.test(ts)) {
      if (newMode === 'ms' && mode === 's' && ts.length === 10) {
        // 秒 → 毫秒：合法秒级时间戳，末尾补 000
        setTimestampInput(ts + '000')
      } else if (newMode === 's' && mode === 'ms' && ts.length === 13) {
        // 毫秒 → 秒：合法毫秒级时间戳，舍弃最后 3 位
        setTimestampInput(ts.slice(0, 10))
      }
    }
    setMode(newMode)
  }

  const modeBtn = (value, label) => (
    <button
      onClick={() => handleModeChange(value)}
      style={{
        padding: '4px 12px', borderRadius: 'var(--radius-sm)',
        border: mode === value ? '1px solid var(--color-primary)' : '1px solid var(--color-border)',
        background: mode === value ? 'var(--color-primary)' : 'transparent',
        color: mode === value ? '#fff' : 'var(--color-foreground)',
        fontSize: 'var(--font-size-xs)', cursor: 'pointer',
        transition: 'all var(--transition-fast)',
      }}
    >
      {label}
    </button>
  )

  return (
    <div style={formBody}>
      {/* Timestamp section */}
      <div style={lbl}>
        时间戳
        <span style={{ marginLeft: 12, display: 'inline-flex', gap: 4 }}>
          {modeBtn('s', '秒')}
          {modeBtn('ms', '毫秒')}
        </span>
      </div>
      <div style={{ position: 'relative' }}>
        <input
          type="text"
          value={timestampInput}
          onChange={e => setTimestampInput(e.target.value.replace(/[^\d]/g, ''))}
          style={{
            ...textarea, minHeight: 'auto', maxHeight: 'none',
            fontFamily: 'monospace', resize: 'none',
          }}
          spellCheck={false}
          autoFocus
        />
        <button
          style={copiedTs ? copyBtnSuccess : copyBtn}
          onClick={() => handleCopy(timestampInput, setCopiedTs)}
          onMouseEnter={e => { if (!copiedTs) e.target.style.opacity = '1' }}
          onMouseLeave={e => { if (!copiedTs) e.target.style.opacity = '0.8' }}
        >
          {copiedTs ? '已复制' : '复制'}
        </button>
      </div>

      {/* Datetime section */}
      <div style={lbl}>日期时间（系统时区）</div>
      <div style={{ position: 'relative' }}>
        <input
          type="datetime-local"
          value={datetimeInput}
          onChange={e => { setDatetimeInput(e.target.value); }}
          onBlur={syncFromDatetime}
          onKeyDown={e => { if (e.key === 'Enter') syncFromDatetime() }}
          style={{
            ...textarea, minHeight: 'auto', maxHeight: 'none',
            fontFamily: 'monospace', resize: 'none',
          }}
          step="1"
        />
        <button
          style={copiedDt ? copyBtnSuccess : copyBtn}
          onClick={() => handleCopy(datetimeInput.replace('T', ' '), setCopiedDt)}
          onMouseEnter={e => { if (!copiedDt) e.target.style.opacity = '1' }}
          onMouseLeave={e => { if (!copiedDt) e.target.style.opacity = '0.8' }}
        >
          {copiedDt ? '已复制' : '复制'}
        </button>
      </div>

      {error && <div style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-destructive)' }}>{error}</div>}
    </div>
  )
}

export default {
  id: 'timestamp',
  label: '时间戳转换',
  icon: 'Clock',
  priority: 25,
  trigger: '10 位（秒）或 13 位（毫秒）纯数字时间戳',
  desc: '在弹窗中双向转换 Unix 时间戳与可读日期时间（系统时区）',
  detect(content) {
    return isTimestamp(content)
  },
  Component: TimestampModal,
}
