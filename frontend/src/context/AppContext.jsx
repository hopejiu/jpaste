import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { Service as SettingsService } from '../../bindings/jpaste/internal/settings'
import { EVENTS } from '../events'
import { log } from '../logger'
import { defaultConfig } from '../actions'

const AppContext = createContext(null)

const DEFAULT_SETTINGS = {
  hotkey: 'Alt+V',
  retain_days: 30,
  default_action: 'copy',
  auto_start: false,
  start_minimized: false,
  notify_enabled: false,
  paste_order: 'normal',
  action_config: {},
  theme: 'a',
}

export function AppProvider({ children }) {
  const [settings, setSettings] = useState(DEFAULT_SETTINGS)

  // Load settings.
  useEffect(() => {
    SettingsService.GetSettings()
      .then(s => {
        const actionConfig = { ...defaultConfig(), ...(s.action_config || {}) }
        setSettings({ ...s, theme: s.theme || 'a', action_config: actionConfig })
      })
      .catch(e => log.error('AppContext', e))
  }, [])

  const saveSettings = useCallback(async (newSettings) => {
    await SettingsService.SaveSettings(newSettings)
    setSettings(newSettings)
  }, [])

  // Set paste order — wraps saveSettings.
  const setPasteOrder = useCallback(async (order) => {
    await saveSettings({ ...settings, paste_order: order })
  }, [settings, saveSettings])

  // Listen for paste order changes from Go.
  useEffect(() => {
    const unsub = Events.On(EVENTS.PASTE_ORDER_CHANGED, (evt) => {
      setSettings(prev => ({ ...prev, paste_order: evt.data || 'normal' }))
    })
    return unsub
  }, [])

  return (
    <AppContext.Provider value={{
      settings, saveSettings,
      setPasteOrder,
    }}>
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp must be used within AppProvider')
  return ctx
}
