import { useEffect, useRef, useCallback } from 'react'
import { Routes, Route, useNavigate } from 'react-router-dom'
import { Events } from '@wailsio/runtime'
import { ClipboardProvider, useClipboard } from './context/ClipboardContext'
import { AppProvider, useApp } from './context/AppContext'
import { EVENTS } from './events'
import MainPage from './pages/MainPage'
import SettingsPage from './pages/SettingsPage'
import JsonViewPage from './pages/JsonViewPage'
import ImageViewPage from './pages/ImageViewPage'
import ToastPage from './pages/ToastPage'

function AppContent() {
  const navigate = useNavigate()
  const { settings } = useApp()
  const lastHideTimeRef = useRef(Date.now())
  const { setSearch } = useClipboard()

  // Apply theme class based on settings
  const themeClass = `theme-${settings.theme || 'a'}`

  // Sync theme class to <html> so CSS variables cascade to <body> too.
  useEffect(() => {
    document.documentElement.className = themeClass
  }, [themeClass])

  // Handle navigate event from Go (system tray).
  useEffect(() => {
    const unsub = Events.On(EVENTS.NAVIGATE, (evt) => {
      const path = evt.data || '/'
      navigate(path)
    })
    return unsub
  }, [navigate])

  // Window show: clear search after 30s absence.
  useEffect(() => {
    const unsubShow = Events.On(EVENTS.WINDOW_SHOWN, () => {
      if (Date.now() - lastHideTimeRef.current >= 30000) {
        setSearch('')
      }
    })
    const unsubHide = Events.On(EVENTS.WINDOW_HIDING, () => {
      lastHideTimeRef.current = Date.now()
    })
    return () => { unsubShow(); unsubHide() }
  }, [setSearch])

  return (
    <div className={themeClass} style={{
      width: '100%', height: '100vh', display: 'flex', flexDirection: 'column',
      background: 'var(--color-surface)',
    }}>
      <Routes>
        <Route path="/" element={<MainPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/json-view" element={<JsonViewPage />} />
        <Route path="/image-view" element={<ImageViewPage />} />
        <Route path="/toast" element={<ToastPage />} />
      </Routes>
    </div>
  )
}

function AppContentWithClipboard() {
  const { settings, saveSettings } = useApp()

  const onSortChange = useCallback((field, order) => {
    saveSettings({ ...settings, sort_field: field, sort_order: order })
  }, [settings, saveSettings])

  return (
    <ClipboardProvider
      initialSortField={settings.sort_field || 'updated_at'}
      initialSortOrder={settings.sort_order || 'desc'}
      onSortChange={onSortChange}
    >
      <AppContent />
    </ClipboardProvider>
  )
}

export default function App() {
  return (
    <AppProvider>
      <AppContentWithClipboard />
    </AppProvider>
  )
}
