import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react'
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

// Tag mask constants (must match Go clipboard package).
export const TAG_ALL = 0
export const TAG_TEXT = 1
export const TAG_RICH_TEXT = 2
export const TAG_IMAGE = 4
export const TAG_URL = 8
export const TAG_FILE = 16

export const TAGS = [
  { id: TAG_ALL, label: '全部' },
  { id: TAG_TEXT, label: '文本' },
  { id: TAG_IMAGE, label: '图片' },
  { id: TAG_URL, label: '网址' },
  { id: TAG_FILE, label: '文件' },
]

export function ClipboardProvider({ children }) {
  const [entries, setEntries] = useState([])
  const [search, setSearch] = useState('')
  const [activeTag, setActiveTag] = useState(TAG_ALL)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(false)

  // Cursor state — refs to avoid dependency loops in useEffect.
  const cursorRef = useRef({ updatedAt: '', id: 0 })
  const searchRef = useRef('')
  const tagRef = useRef(TAG_ALL)

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

  // refreshHistory: first page (resets cursor).
  const refreshHistory = useCallback(async (searchTerm = '', tagMask = TAG_ALL) => {
    setLoading(true)
    searchRef.current = searchTerm
    tagRef.current = tagMask
    try {
      const result = await HistoryService.GetHistory(searchTerm, tagMask, '', 0)
      // Wails passes Go return values as array: [entries, hasMore]
      if (Array.isArray(result)) {
        const [list, more] = result
        setEntries(list || [])
        setHasMore(!!more)
        if (list && list.length > 0) {
          const last = list[list.length - 1]
          cursorRef.current = { updatedAt: last.updated_at, id: last.id }
        } else {
          cursorRef.current = { updatedAt: '', id: 0 }
        }
      } else {
        setEntries([])
        setHasMore(false)
      }
    } catch (err) {
      console.error('Failed to load history:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  // loadMore: next page (keeps cursor).
  const loadMore = useCallback(async () => {
    if (!hasMore || loading) return
    setLoading(true)
    const { updatedAt, id } = cursorRef.current
    try {
      const result = await HistoryService.GetHistory(searchRef.current, tagRef.current, updatedAt, id)
      if (Array.isArray(result)) {
        const [list, more] = result
        if (list && list.length > 0) {
          setEntries(prev => [...prev, ...list])
          const last = list[list.length - 1]
          cursorRef.current = { updatedAt: last.updated_at, id: last.id }
        }
        setHasMore(!!more)
      }
    } catch (err) {
      console.error('Failed to load more:', err)
    } finally {
      setLoading(false)
    }
  }, [hasMore, loading])

  // Refresh on mount.
  useEffect(() => {
    refreshHistory('', TAG_ALL)
  }, [refreshHistory])

  // When search or tag changes, reset and reload.
  const handleSetSearch = useCallback((term) => {
    setSearch(term)
    refreshHistory(term, activeTag)
  }, [activeTag, refreshHistory])

  const handleSetTag = useCallback((tag) => {
    setActiveTag(tag)
    refreshHistory(search, tag)
  }, [search, refreshHistory])

  // Listen for clipboard updates from Go.
  useEffect(() => {
    const unsub = Events.On(EVENTS.CLIPBOARD_UPDATED, () => {
      refreshHistory(searchRef.current, tagRef.current)
    })
    return unsub
  }, [refreshHistory])

  const useEntry = useCallback(async (id, action) => {
    try {
      await HistoryService.UseEntry(id, action)
      refreshHistory(searchRef.current, tagRef.current)
    } catch (err) {
      console.error('Failed to use entry:', err)
    }
  }, [refreshHistory])

  const deleteEntry = useCallback(async (id) => {
    try {
      await HistoryService.DeleteEntry(id)
      refreshHistory(searchRef.current, tagRef.current)
    } catch (err) {
      console.error('Failed to delete entry:', err)
    }
  }, [refreshHistory])

  const saveSettings = useCallback(async (newSettings) => {
    try {
      await SettingsService.SaveSettings(newSettings)
      setSettings(newSettings)
    } catch (err) {
      console.error('Failed to save settings:', err)
    }
  }, [])

  const clearAll = useCallback(async () => {
    try {
      await HistoryService.ClearAll()
      refreshHistory(searchRef.current, tagRef.current)
    } catch (err) {
      console.error('Failed to clear all:', err)
    }
  }, [refreshHistory])

  return (
    <ClipboardContext.Provider value={{
      entries, search, setSearch: handleSetSearch,
      activeTag, setActiveTag: handleSetTag,
      hasMore, loading, loadMore,
      refreshHistory, useEntry, deleteEntry, clearAll,
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
