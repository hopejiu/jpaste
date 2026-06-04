import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { Service as SettingsService } from '../../bindings/jpaste/internal/settings'
import { Service as SyncService } from '../../bindings/jpaste/internal/sync'
import { EVENTS } from '../events'
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
}

const DEFAULT_SYNC_STATUS = { status: 'none', error: '' }
const DEFAULT_WD_CONFIG = { url: '', username: '', password: '', enabled: false }

export function AppProvider({ children }) {
  const [settings, setSettings] = useState(DEFAULT_SETTINGS)
  const [syncStatus, setSyncStatus] = useState(DEFAULT_SYNC_STATUS)
  const [wdConfig, setWdConfig] = useState(DEFAULT_WD_CONFIG)

  // Load settings.
  useEffect(() => {
    SettingsService.GetSettings()
      .then(s => {
        const actionConfig = { ...defaultConfig(), ...(s.action_config || {}) }
        setSettings({ ...s, action_config: actionConfig })
      })
      .catch(console.error)
  }, [])

  // Load WebDAV config.
  useEffect(() => {
    SyncService.GetConfig()
      .then(c => {
        if (c && (c.url || c.username)) {
          setWdConfig({ url: c.url || '', username: c.username || '', password: c.password || '', enabled: c.enabled || false })
        }
      })
      .catch(e => console.error('GetConfig error:', e))
  }, [])

  const refreshWdConfig = useCallback(async () => {
    try {
      const c = await SyncService.GetConfig()
      if (c) {
        setWdConfig({ url: c.url || '', username: c.username || '', password: c.password || '', enabled: c.enabled || false })
      }
    } catch (e) {
      console.error('[wd:ctx] refresh config error:', e)
    }
  }, [])

  // Listen for sync status events.
  useEffect(() => {
    const unsub = Events.On(EVENTS.SYNC_STATUS, (evt) => {
      setSyncStatus(evt.data || DEFAULT_SYNC_STATUS)
    })
    return unsub
  }, [])

  const saveSettings = useCallback(async (newSettings) => {
    try {
      await SettingsService.SaveSettings(newSettings)
      setSettings(newSettings)
    } catch (err) {
      console.error('Failed to save settings:', err)
    }
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
      syncStatus,
      wdConfig, refreshWdConfig,
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
