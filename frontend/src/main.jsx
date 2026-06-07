import React from 'react'
import ReactDOM from 'react-dom/client'
import { MemoryRouter } from 'react-router-dom'
import './index.css'
import App from './App'
import ToastPage from './pages/ToastPage'

// MemoryRouter ignores the browser URL, so we must detect the intended
// route from window.location (used by secondary windows like /json-view /image-view /curl-view /ws-view).
const isToastWindow = window.location.pathname === '/toast'
const isSecondaryWindow = !isToastWindow && window.location.pathname !== '/'

const root = ReactDOM.createRoot(document.getElementById('root'))

if (isToastWindow) {
  // Toast window: render ToastPage directly, no MemoryRouter, no App component.
  // Completely isolated from main window routing to avoid route contamination
  // that causes ToastPage to unmount (the root cause of the white screen).
  root.render(
    <React.StrictMode>
      <ToastPage />
    </React.StrictMode>,
  )
} else {
  const initialPath = isSecondaryWindow
    ? window.location.pathname + window.location.search
    : '/'

  root.render(
    <React.StrictMode>
      <MemoryRouter initialEntries={[initialPath]}>
        <App />
      </MemoryRouter>
    </React.StrictMode>,
  )
}
