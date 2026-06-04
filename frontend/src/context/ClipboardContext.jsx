import { createContext, useContext, useReducer, useCallback, useEffect, useRef } from 'react'
import { Events } from '@wailsio/runtime'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { EVENTS } from '../events'

const ClipboardContext = createContext(null)

// Tag mask constants (must match Go clipboard package).
export const TAG_ALL = 0
export const TAG_TEXT = 1
export const TAG_IMAGE = 4
export const TAG_URL = 8
export const TAG_FILE = 16
export const TAG_FAVORITE = 32

export const TAGS = [
  { id: TAG_ALL, label: '全部' },
  { id: TAG_TEXT, label: '文本' },
  { id: TAG_IMAGE, label: '图片' },
  { id: TAG_URL, label: '网址' },
  { id: TAG_FILE, label: '文件' },
  { id: TAG_FAVORITE, label: '收藏' },
]

// --- Reducer ---

const initialState = {
  entries: [],
  search: '',
  activeTag: TAG_ALL,
  hasMore: false,
  loading: true,
  isRegex: false,
  cursor: { updatedAt: '', id: 0 },
}

function clipboardReducer(state, action) {
  switch (action.type) {
    case 'SET_SEARCH':
      return { ...state, search: action.payload }
    case 'SET_TAG':
      return { ...state, activeTag: action.payload, focusedIdx: -1 }
    case 'SET_REGEX':
      return { ...state, isRegex: action.payload }
    case 'LOAD_START':
      return { ...state, loading: true }
    case 'LOAD_FIRST_PAGE':
      return {
        ...state,
        loading: false,
        entries: action.payload.list,
        hasMore: action.payload.hasMore,
        cursor: action.payload.cursor,
      }
    case 'LOAD_MORE':
      return {
        ...state,
        loading: false,
        entries: [...state.entries, ...action.payload.list],
        hasMore: action.payload.hasMore,
        cursor: action.payload.cursor,
      }
    case 'LOAD_ERROR':
      return { ...state, loading: false }
    case 'DELETE_ENTRY':
      return {
        ...state,
        entries: state.entries.filter(e => e.id !== action.payload),
      }
    case 'TOGGLE_FAVORITE':
      return {
        ...state,
        entries: state.entries.map(e =>
          e.id === action.payload.id ? { ...e, is_favorite: action.payload.value } : e
        ),
      }
    // Regex results: replace all entries, no pagination.
    case 'LOAD_REGEX':
      return {
        ...state,
        loading: false,
        entries: action.payload,
        hasMore: false,
      }
    default:
      return state
  }
}

// --- Provider ---

export function ClipboardProvider({ children }) {
  const [state, dispatch] = useReducer(clipboardReducer, initialState)

  // Refs for latest state consumed by async callbacks / event listeners.
  const stateRef = useRef(state)
  stateRef.current = state
  const debounceRef = useRef(null)

  // Build cursor from last entry.
  const cursorFromList = (list) => {
    if (list && list.length > 0) {
      const last = list[list.length - 1]
      return { updatedAt: last.updated_at, id: last.id }
    }
    return { updatedAt: '', id: 0 }
  }

  // refreshHistory: first page (resets cursor).
  const refreshHistory = useCallback(async (searchTerm = '', tagMask = TAG_ALL) => {
    dispatch({ type: 'LOAD_START' })
    try {
      const result = await HistoryService.GetHistory(searchTerm, tagMask, '', 0)
      if (Array.isArray(result)) {
        const [list, more] = result
        dispatch({
          type: 'LOAD_FIRST_PAGE',
          payload: { list: list || [], hasMore: !!more, cursor: cursorFromList(list) },
        })
      } else {
        dispatch({ type: 'LOAD_FIRST_PAGE', payload: { list: [], hasMore: false, cursor: { updatedAt: '', id: 0 } } })
      }
    } catch (err) {
      console.error('Failed to load history:', err)
      dispatch({ type: 'LOAD_ERROR' })
    }
  }, [])

  // loadMore: next page (keeps cursor).
  const loadMore = useCallback(async () => {
    const s = stateRef.current
    if (!s.hasMore || s.loading || s.isRegex) return
    dispatch({ type: 'LOAD_START' })
    const { updatedAt, id } = s.cursor
    try {
      const result = await HistoryService.GetHistory(s.search, s.activeTag, updatedAt, id)
      if (Array.isArray(result)) {
        const [list, more] = result
        if (list && list.length > 0) {
          dispatch({
            type: 'LOAD_MORE',
            payload: { list, hasMore: !!more, cursor: cursorFromList(list) },
          })
        } else {
          dispatch({ type: 'LOAD_ERROR' })
        }
      }
    } catch (err) {
      console.error('Failed to load more:', err)
      dispatch({ type: 'LOAD_ERROR' })
    }
  }, [])

  // regexSearch: server-side regex, loads all matching entries at once.
  const regexSearch = useCallback(async (pattern, tagMask) => {
    dispatch({ type: 'LOAD_START' })
    try {
      const list = await HistoryService.GetHistoryRegex(pattern, tagMask)
      dispatch({ type: 'LOAD_REGEX', payload: list || [] })
    } catch (err) {
      console.error('Failed to regex search:', err)
      dispatch({ type: 'LOAD_REGEX', payload: [] })
    }
  }, [])

  // Refresh on mount.
  useEffect(() => {
    refreshHistory('', TAG_ALL)
  }, [refreshHistory])

  // When search or tag changes, reset and reload.
  const handleSetSearch = useCallback((term) => {
    dispatch({ type: 'SET_SEARCH', payload: term })
    clearTimeout(debounceRef.current)
    const s = stateRef.current
    debounceRef.current = setTimeout(() => {
      if (s.isRegex && term) {
        regexSearch(term, s.activeTag)
      } else {
        refreshHistory(term, s.activeTag)
      }
    }, 300)
  }, [refreshHistory, regexSearch])

  const handleSetTag = useCallback((tag) => {
    dispatch({ type: 'SET_TAG', payload: tag })
    const s = stateRef.current
    if (s.isRegex && s.search) {
      regexSearch(s.search, tag)
    } else {
      refreshHistory(s.search, tag)
    }
  }, [refreshHistory, regexSearch])

  // Toggle regex mode. Re-runs current search with the new mode.
  const handleToggleRegex = useCallback((enabled) => {
    dispatch({ type: 'SET_REGEX', payload: enabled })
    const s = stateRef.current
    if (enabled && s.search) {
      regexSearch(s.search, s.activeTag)
    } else if (!enabled) {
      refreshHistory(s.search, s.activeTag)
    }
  }, [refreshHistory, regexSearch])

  // Listen for clipboard updates from Go.
  useEffect(() => {
    const unsub = Events.On(EVENTS.CLIPBOARD_UPDATED, () => {
      const s = stateRef.current
      if (s.isRegex && s.search) {
        regexSearch(s.search, s.activeTag)
      } else {
        refreshHistory(s.search, s.activeTag)
      }
    })
    return unsub
  }, [refreshHistory, regexSearch])

  const useEntry = useCallback(async (id, action) => {
    try {
      await HistoryService.UseEntry(id, action)
      const s = stateRef.current
      refreshHistory(s.search, s.activeTag)
    } catch (err) {
      console.error('Failed to use entry:', err)
    }
  }, [refreshHistory])

  const deleteEntry = useCallback(async (id) => {
    // Optimistic delete: remove from UI immediately.
    dispatch({ type: 'DELETE_ENTRY', payload: id })
    try {
      await HistoryService.DeleteEntry(id)
    } catch (err) {
      console.error('Failed to delete entry:', err)
      // Reload on failure to recover.
      const s = stateRef.current
      refreshHistory(s.search, s.activeTag)
    }
  }, [refreshHistory])

  const clearAll = useCallback(async () => {
    try {
      await HistoryService.ClearAll()
      const s = stateRef.current
      refreshHistory(s.search, s.activeTag)
    } catch (err) {
      console.error('Failed to clear all:', err)
    }
  }, [refreshHistory])

  const toggleFavorite = useCallback(async (id, value) => {
    dispatch({ type: 'TOGGLE_FAVORITE', payload: { id, value } })
    try {
      await HistoryService.ToggleFavorite(id, value)
    } catch (err) {
      console.error('Failed to toggle favorite:', err)
      // Revert on failure.
      dispatch({ type: 'TOGGLE_FAVORITE', payload: { id, value: !value } })
    }
  }, [])

  return (
    <ClipboardContext.Provider value={{
      entries: state.entries,
      search: state.search,
      setSearch: handleSetSearch,
      activeTag: state.activeTag,
      setActiveTag: handleSetTag,
      hasMore: state.hasMore,
      loading: state.loading,
      isRegex: state.isRegex,
      toggleRegex: handleToggleRegex,
      loadMore,
      refreshHistory, useEntry, deleteEntry, clearAll, toggleFavorite,
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
