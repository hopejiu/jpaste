import React from 'react'
import ReactDOM from 'react-dom/client'
import { MemoryRouter } from 'react-router-dom'
import App from './App'

// MemoryRouter ignores the browser URL, so we must detect the intended
// route from window.location (used by secondary windows like /json-view /image-view).
const isSecondaryWindow = window.location.pathname === '/json-view'
  || window.location.pathname === '/image-view'
  || window.location.pathname === '/toast'
const initialPath = isSecondaryWindow
  ? window.location.pathname + window.location.search
  : '/'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <MemoryRouter initialEntries={[initialPath]}>
      <App />
    </MemoryRouter>
  </React.StrictMode>,
)
