import { useCallback } from 'react'

export function useKeyboardNavigation({ entries, focusedIdx, settings, useEntry, setSearch, setFocusedIdx, inputRef, modal, closeModal }) {
  const handleKeyDown = useCallback((e) => {
    if (modal) {
      if (e.key === 'Escape') { e.preventDefault(); closeModal() }
      return
    }
    if (e.ctrlKey && e.key >= '1' && e.key <= '9') {
      e.preventDefault()
      const idx = parseInt(e.key) - 1
      console.log('[kb] Ctrl+' + e.key + ' pressed, hasFocus=' + document.hasFocus() +
        ', idx=' + idx + ', entries=' + entries.length + ', action=' + settings.default_action)
      if (idx < entries.length) {
        const entry = entries[idx]
        console.log('[kb] calling useEntry id=' + entry.id + ' action=' + settings.default_action)
        useEntry(entry.id, settings.default_action)
      }
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setFocusedIdx(prev => Math.min(prev + 1, entries.length - 1))
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setFocusedIdx(prev => Math.max(prev - 1, -1))
      return
    }
    if (e.key === 'Enter' && focusedIdx >= 0) {
      e.preventDefault()
      useEntry(entries[focusedIdx].id, settings.default_action)
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      setSearch('')
      setFocusedIdx(-1)
      inputRef.current?.blur()
      return
    }
    if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      inputRef.current?.focus()
    }
  }, [entries, focusedIdx, useEntry, settings, setSearch, setFocusedIdx, inputRef, modal, closeModal])

  return handleKeyDown
}
