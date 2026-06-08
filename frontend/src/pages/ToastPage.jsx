import { useState, useEffect, useMemo } from 'react'
import { Events } from '@wailsio/runtime'
import { EVENTS } from '../events'

export default function ToastPage() {
  const [toastData, setToastData] = useState(null)
  const [themeReady, setThemeReady] = useState(false)
  const [toastOpacity, setToastOpacity] = useState(1)

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

        // Opacity from settings (0.0-1.0), always transmitted by Go.
        if (payload.opacity !== undefined && payload.opacity !== null) {
          setToastOpacity(payload.opacity)
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

  // Smoothly blend glass background toward fully opaque as opacity approaches 1.0.
  // Below 0.7: pure glass effect. Between 0.7–1.0: gradual transition.
  const glassCardStyle = useMemo(() => {
    if (!themeReady) return undefined

    if (toastOpacity >= 1) {
      return {
        background: 'var(--color-surface)',
        backdropFilter: 'none',
        WebkitBackdropFilter: 'none',
      }
    }

    const blendStart = 0.7
    if (toastOpacity >= blendStart) {
      const t = (toastOpacity - blendStart) / (1 - blendStart) // 0→1
      const pct = (t * 100).toFixed(0)
      const blurPx = 20 * (1 - t)
      return {
        background: `color-mix(in srgb, var(--color-surface) ${pct}%, var(--toast-glass-bg))`,
        backdropFilter: `blur(${blurPx.toFixed(0)}px)`,
        WebkitBackdropFilter: `blur(${blurPx.toFixed(0)}px)`,
      }
    }

    return undefined // default frosted-glass CSS
  }, [themeReady, toastOpacity])

  return (
    <div
      className="w-screen h-screen flex items-center font-inherit bg-transparent select-none cursor-default"
      style={{ opacity: themeReady ? toastOpacity : 0, transition: 'opacity 150ms ease' }}
    >
      <div className="glass-toast flex items-center gap-2.5 mx-3 px-4 py-2.5 flex-1 rounded-[10px]"
        style={glassCardStyle}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-primary)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <rect x="8" y="2" width="8" height="4" rx="1" ry="1" />
          <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
          <path d="M12 11h4" /><path d="M12 16h4" /><path d="M8 11h.01" /><path d="M8 16h.01" />
        </svg>
        <div className="flex-1 overflow-hidden">
          <div className="text-[13px] font-semibold text-foreground whitespace-nowrap overflow-hidden text-ellipsis">
            {toastData?.title}
          </div>
          <div className="text-xs text-muted whitespace-nowrap overflow-hidden text-ellipsis mt-0.5">
            {toastData?.message}
          </div>
        </div>
      </div>
    </div>
  )
}
