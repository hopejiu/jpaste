import { useCallback } from 'react'
import { Window } from '@wailsio/runtime'

export function useKeyboardNavigation({ entries, focusedIdx, settings, useEntry, setSearch, setFocusedIdx, inputRef, modal, closeModal, activeTag, tags, onTagChange, search, listRef }) {
  const handleKeyDown = useCallback((e) => {
    // Modal handling: only Escape and PageUp/Down pass through.
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

    // Tab switching: Left/Right arrow keys.
    if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      e.preventDefault()
      const currentIdx = tags.findIndex(t => t.id === activeTag)
      if (currentIdx === -1) return
      const delta = e.key === 'ArrowRight' ? 1 : -1
      const nextIdx = (currentIdx + delta + tags.length) % tags.length
      onTagChange(tags[nextIdx].id)
      return
    }

    // Entry navigation: Up/Down arrow keys.
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

    // Page Up/Down: scroll the entry list.
    if (e.key === 'PageUp' || e.key === 'PageDown') {
      e.preventDefault()
      const list = listRef?.current
      if (!list) return
      const delta = e.key === 'PageDown' ? list.clientHeight : -list.clientHeight
      list.scrollBy({ top: delta, behavior: 'smooth' })
      return
    }

    // Escape: close modal → clear search → hide window.
    if (e.key === 'Escape') {
      e.preventDefault()
      if (search) {
        setSearch('')
        setFocusedIdx(-1)
        inputRef.current?.blur()
      } else {
        // Hide window via Wails runtime.
        Window.Hide()
      }
      return
    }

    if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      inputRef.current?.focus()
    }
  }, [entries, focusedIdx, useEntry, settings, setSearch, setFocusedIdx, inputRef, modal, closeModal, activeTag, tags, onTagChange, search, listRef])

  return handleKeyDown
}
