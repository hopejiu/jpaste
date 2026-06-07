import { useState, useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { EVENTS } from '../events'

export default function ToastPage() {
  const [toastData, setToastData] = useState(null)
  const [themeReady, setThemeReady] = useState(false)

  // Load initial theme on mount. Subsequent updates come with each event.
  useEffect(() => {
    const unsub = Events.On(EVENTS.TOAST_NOTIFICATION, (raw) => {
      let title = 'jPaste'
      let message = ''
      if (raw != null) {
        const payload = raw.data || raw
        title = payload.title || title
        message = payload.message || message

        // Theme is injected by Go's toastEmit at notification time.
        if (payload.theme) {
          document.documentElement.className = `theme-${payload.theme}`
        }
      }
      setToastData({ title, message })
      setThemeReady(true)
    })
    return () => { unsub() }
  }, [])

  // Override body background for transparent window.
  useEffect(() => {
    const original = document.body.style.background
    document.body.style.background = 'transparent'
    return () => { document.body.style.background = original }
  }, [])

  // Keep the toast fully visible at all times. The Go-side 3s timer
  // is the only mechanism that makes it disappear (off-screen repositioning).
  // No front-end exit animation — any opacity transition would leave a gap
  // of white window before the off-screen move.

  return (
    <div style={{
      width: '100%', height: '100vh',
      display: 'flex', alignItems: 'center',
      fontFamily: 'inherit',
      background: 'transparent',
      opacity: themeReady ? 1 : 0,
      transition: 'opacity 150ms ease',
      userSelect: 'none', cursor: 'default',
    }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: '10px',
        margin: '0 12px', padding: '10px 16px', flex: 1,
        borderRadius: '10px',
        background: 'var(--toast-glass-bg)',
        backdropFilter: 'blur(16px)',
        WebkitBackdropFilter: 'blur(16px)',
      }}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-primary)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <rect x="8" y="2" width="8" height="4" rx="1" ry="1" />
          <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
          <path d="M12 11h4" /><path d="M12 16h4" /><path d="M8 11h.01" /><path d="M8 16h.01" />
        </svg>
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <div style={{
            fontSize: '13px', fontWeight: 600, color: 'var(--color-foreground)',
            whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
          }}>{toastData?.title}</div>
          <div style={{
            fontSize: '12px', color: 'var(--color-muted)',
            whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
            marginTop: '2px',
          }}>{toastData?.message}</div>
        </div>
      </div>
    </div>
  )
}
