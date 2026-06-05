import { useState, useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { EVENTS } from '../events'

export default function ToastPage() {
  const [toastData, setToastData] = useState(null)

  useEffect(() => {
    const unsub = Events.On(EVENTS.TOAST_NOTIFICATION, (raw) => {
      let title = 'jPaste'
      let message = ''
      if (raw != null) {
        const payload = raw.data || raw
        title = payload.title || title
        message = payload.message || message
      }
      setToastData({ title, message })
    })
    return () => { unsub() }
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
      background: 'var(--color-surface)',
      userSelect: 'none', cursor: 'default',
    }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: '10px',
        padding: '0 16px', width: '100%',
      }}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-muted)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
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
