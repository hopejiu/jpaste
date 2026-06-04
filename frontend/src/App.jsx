import { useState, useEffect, useRef, useCallback } from 'react'
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom'
import { Events } from '@wailsio/runtime'
import { ClipboardProvider, useClipboard } from './context/ClipboardContext'
import { AppProvider, useApp } from './context/AppContext'
import { EVENTS } from './events'
import MainPage from './pages/MainPage'
import SettingsPage from './pages/SettingsPage'
import JsonViewPage from './pages/JsonViewPage'
import ImageViewPage from './pages/ImageViewPage'

function AppContent() {
  const navigate = useNavigate()
  const location = useLocation()
  const [animState, setAnimState] = useState('enter')
  const appRef = useRef(null)
  const lastHideTimeRef = useRef(Date.now())
  const { setSearch } = useClipboard()

  // Handle navigate event from Go (system tray).
  useEffect(() => {
    const unsub = Events.On(EVENTS.NAVIGATE, (evt) => {
      const path = evt.data || '/'
      navigate(path)
    })
    return unsub
  }, [navigate])

  // Window show/hide: clear search after 30s absence.
  useEffect(() => {
    const unsubShow = Events.On(EVENTS.WINDOW_SHOWN, () => {
      setAnimState('enter')
      if (Date.now() - lastHideTimeRef.current >= 30000) {
        setSearch('')
      }
    })
    const unsubHide = Events.On(EVENTS.WINDOW_HIDING, () => {
      setAnimState('exit')
      lastHideTimeRef.current = Date.now()
    })
    return () => { unsubShow(); unsubHide() }
  }, [setSearch])

  // Listen for route changes to trigger enter animation.
  useEffect(() => {
    setAnimState('enter')
  }, [location.pathname])

  const animClass = animState === 'enter' ? 'app-enter' : 'app-exit'

  return (
    <div ref={appRef} className={animClass} style={{
      width: '100%', height: '100vh', display: 'flex', flexDirection: 'column',
    }}>
      <Routes>
        <Route path="/" element={<MainPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/json-view" element={<JsonViewPage />} />
        <Route path="/image-view" element={<ImageViewPage />} />
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
