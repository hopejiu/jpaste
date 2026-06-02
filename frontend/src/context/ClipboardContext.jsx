import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { Service as SettingsService } from '../../bindings/jpaste/internal/settings'
import { Service as SyncService } from '../../bindings/jpaste/internal/sync'
import { EVENTS } from '../events'
import { defaultConfig } from '../actions'

const ClipboardContext = createContext(null)

const DEFAULT_SETTINGS = {
  hotkey: 'Alt+V',
  retain_days: 30,
  default_action: 'copy',
  auto_start: false,
  start_minimized: false,
  notify_enabled: true,
  action_config: {},
}

const DEFAULT_SYNC_STATUS = { status: 'none', error: '' }
const DEFAULT_WD_CONFIG = { url: '', username: '', password: '', enabled: false }

export function ClipboardProvider({ children }) {
  const [entries, setEntries] = useState([])
  const [search, setSearch] = useState('')
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

  // Load WebDAV config (once on start, persists across page navigations).
  useEffect(() => {
    SyncService.GetConfig()
      .then(c => {
        if (c && (c.url || c.username)) {
          setWdConfig({ url: c.url || '', username: c.username || '', password: c.password || '', enabled: c.enabled || false })
        }
      })
      .catch(e => console.error('GetConfig error:', e))
  }, [])

  // Reload config when sync status indicates a change.
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

  // Load initial history.
  const refreshHistory = useCallback(async (searchTerm = '') => {
    try {
      const result = await HistoryService.GetHistory(searchTerm)
      setEntries(result || [])
    } catch (err) {
      console.error('Failed to load history:', err)
    }
  }, [])

  // Refresh on mount and when search changes.
  useEffect(() => {
    refreshHistory(search)
  }, [search, refreshHistory])

  // Listen for clipboard updates from Go.
  useEffect(() => {
    const unsub = Events.On(EVENTS.CLIPBOARD_UPDATED, () => {
      refreshHistory(search)
    })
    return unsub
  }, [search, refreshHistory])

  const useEntry = useCallback(async (id, action) => {
    try {
      await HistoryService.UseEntry(id, action)
      refreshHistory(search)
    } catch (err) {
      console.error('Failed to use entry:', err)
    }
  }, [search, refreshHistory])

  const deleteEntry = useCallback(async (id) => {
    try {
      await HistoryService.DeleteEntry(id)
      refreshHistory(search)
    } catch (err) {
      console.error('Failed to delete entry:', err)
    }
  }, [search, refreshHistory])

  const saveSettings = useCallback(async (newSettings) => {
    try {
      await SettingsService.SaveSettings(newSettings)
      setSettings(newSettings)
    } catch (err) {
      console.error('Failed to save settings:', err)
    }
  }, [])

  return (
    <ClipboardContext.Provider value={{
      entries, search, setSearch,
      refreshHistory, useEntry, deleteEntry,
      settings, saveSettings,
      syncStatus,
      wdConfig, refreshWdConfig,
    }}>
      {children}
    </ClipboardContext.Provider>
  )
}

export function useClipboard() {
  const ctx = useContext(ClipboardContext)
  if (!ctx) throw new Error('useClipboard must be used within ClipboardProvider')
  return ctx
}
