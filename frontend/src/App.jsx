import { useState, useEffect, useRef } from 'react'
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom'
import { Events } from '@wailsio/runtime'
import { ClipboardProvider } from './context/ClipboardContext'
import { AppProvider } from './context/AppContext'
import { EVENTS } from './events'
import MainPage from './pages/MainPage'
import SettingsPage from './pages/SettingsPage'
import JsonViewPage from './pages/JsonViewPage'
import ImageViewPage from './pages/ImageViewPage'

export default function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const [animState, setAnimState] = useState('enter')
  const appRef = useRef(null)

  // Handle navigate event from Go (system tray).
  useEffect(() => {
    const unsub = Events.On(EVENTS.NAVIGATE, (evt) => {
      const path = evt.data || '/'
      navigate(path)
    })
    return unsub
  }, [navigate])

  // Window show/hide animation.
  useEffect(() => {
    const unsubShow = Events.On(EVENTS.WINDOW_SHOWN, () => setAnimState('enter'))
    const unsubHide = Events.On(EVENTS.WINDOW_HIDING, () => setAnimState('exit'))
    return () => { unsubShow(); unsubHide() }
  }, [])

  // Listen for route changes to trigger enter animation.
  useEffect(() => {
    setAnimState('enter')
  }, [location.pathname])

  const animClass = animState === 'enter' ? 'app-enter' : 'app-exit'

  return (
    <AppProvider>
    <ClipboardProvider>
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
    </ClipboardProvider>
    </AppProvider>
  )
}
