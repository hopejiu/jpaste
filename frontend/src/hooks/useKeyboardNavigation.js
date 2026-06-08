import { useCallback } from 'react'
import { Window } from '@wailsio/runtime'
import { log } from '../logger'

export function useKeyboardNavigation({ entries, focusedIdx, settings, useEntry, setSearch, setFocusedIdx, inputRef, modal, closeModal, activeTag, tags, onTagChange, search, listRef, deleteEntry, toggleFavorite, onOpenEditor, selectedActionIdx, setSelectedActionIdx }) {
  const handleKeyDown = useCallback((e) => {
    // Log every keydown for debugging.
    log.debug('Keyboard', `key=${e.key} focusedIdx=${focusedIdx} selectedActionIdx=${selectedActionIdx} entriesLen=${entries.length} modal=${modal}`)

    // Modal handling: only Escape passes through.
    if (modal) {
      if (e.key === 'Escape') { e.preventDefault(); closeModal() }
      return
    }

    // Ctrl+L: focus search input.
    if (e.ctrlKey && e.key === 'l') {
      e.preventDefault()
      inputRef.current?.focus()
      inputRef.current?.select()
      return
    }

    // Ctrl+E: open focused entry in editor.
    if (e.ctrlKey && e.key === 'e') {
      e.preventDefault()
      if (focusedIdx >= 0 && entries[focusedIdx]) {
        onOpenEditor(entries[focusedIdx].id)
      }
      return
    }

    // Ctrl+C: copy focused entry.
    if (e.ctrlKey && e.key === 'c' && focusedIdx >= 0 && entries[focusedIdx]) {
      e.preventDefault()
      useEntry(entries[focusedIdx].id, 'copy')
      return
    }

    // Ctrl+1~9: execute default action on Nth entry.
    if (e.ctrlKey && e.key >= '1' && e.key <= '9') {
      e.preventDefault()
      const idx = parseInt(e.key) - 1
      if (idx < entries.length) {
        useEntry(entries[idx].id, settings.default_action)
      }
      return
    }

    // Tab / Shift+Tab: switch tag tabs (always works).
    if (e.key === 'Tab') {
      e.preventDefault()
      const currentIdx = tags.findIndex(t => t.id === activeTag)
      if (currentIdx === -1) return
      const delta = e.shiftKey ? -1 : 1
      const nextIdx = (currentIdx + delta + tags.length) % tags.length
      onTagChange(tags[nextIdx].id)
      return
    }

    // ── Action mode: user is navigating action buttons on the focused entry ──
    if (selectedActionIdx >= 0) {
      // Esc: exit action mode.
      if (e.key === 'Escape') {
        e.preventDefault()
        log.debug('Keyboard', '→ exit action mode via Esc')
        setSelectedActionIdx(-1)
        return
      }
      // → : next action button.
      if (e.key === 'ArrowRight') {
        e.preventDefault()
        const currentId = entries[focusedIdx]?.id
        if (currentId) {
          const sel = `[data-entry-id="${currentId}"] [data-action-btn="${selectedActionIdx + 1}"]`
          const nextBtn = document.querySelector(sel)
          log.debug('Keyboard', `→ next: sel="${sel}" found=${!!nextBtn}`)
          if (nextBtn) setSelectedActionIdx(prev => prev + 1)
        }
        return
      }
      // ← : previous action button (at first → exit mode).
      if (e.key === 'ArrowLeft') {
        e.preventDefault()
        if (selectedActionIdx === 0) {
          setSelectedActionIdx(-1)
        } else {
          setSelectedActionIdx(prev => prev - 1)
        }
        return
      }
      // Enter: activate the selected action button.
      if (e.key === 'Enter') {
        e.preventDefault()
        const currentId = entries[focusedIdx]?.id
        if (currentId) {
          const btn = document.querySelector(`[data-entry-id="${currentId}"] [data-action-btn="${selectedActionIdx}"]`)
          if (btn) { btn.click(); setSelectedActionIdx(-1) }
        }
        return
      }
      // Other keys fall through → still handled below (e.g. Ctrl+L, Ctrl+E…).
    }

    // ── Entry list navigation ──

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

    // → : enter action mode on focused entry (drill-in).
    if (e.key === 'ArrowRight' && focusedIdx >= 0 && entries[focusedIdx]) {
      const eid = entries[focusedIdx]?.id
      // Check what data-action-btn buttons exist on the entry.
      const entryEl = document.querySelector(`[data-entry-id="${eid}"]`)
      const btnCount = entryEl ? entryEl.querySelectorAll('[data-action-btn]').length : -1
      log.debug('Keyboard', `→ drill-in: entryId=${eid} entryEl=${!!entryEl} btnCount=${btnCount}`)
      e.preventDefault()
      setSelectedActionIdx(0)
      return
    }

    // Enter: execute default action on focused entry.
    if (e.key === 'Enter' && focusedIdx >= 0) {
      e.preventDefault()
      useEntry(entries[focusedIdx].id, settings.default_action)
      return
    }

    // Delete: remove focused entry.
    if (e.key === 'Delete' && focusedIdx >= 0) {
      e.preventDefault()
      const id = entries[focusedIdx].id
      const newLen = entries.length - 1
      setFocusedIdx(prev => prev >= newLen ? Math.max(newLen - 1, -1) : prev)
      deleteEntry(id)
      return
    }

    // Space: toggle favorite on focused entry.
    if (e.key === ' ' && focusedIdx >= 0) {
      e.preventDefault()
      const entry = entries[focusedIdx]
      toggleFavorite(entry.id, !entry.is_favorite)
      return
    }

    // Home / End: scroll to top / bottom of entry list.
    if (e.key === 'Home') {
      e.preventDefault()
      listRef.current?.scrollTo({ top: 0, behavior: 'smooth' })
      return
    }
    if (e.key === 'End') {
      e.preventDefault()
      const list = listRef.current
      if (list) list.scrollTo({ top: list.scrollHeight, behavior: 'smooth' })
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

    // Escape: clear search → hide window.
    if (e.key === 'Escape') {
      e.preventDefault()
      if (search) {
        setSearch('')
        setFocusedIdx(-1)
        inputRef.current?.blur()
      } else {
        Window.Hide()
      }
      return
    }

    // Any letter key focuses search input.
    if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      inputRef.current?.focus()
    }
  }, [entries, focusedIdx, useEntry, settings, setSearch, setFocusedIdx, inputRef, modal, closeModal, activeTag, tags, onTagChange, search, listRef, deleteEntry, toggleFavorite, onOpenEditor, selectedActionIdx, setSelectedActionIdx])

  return handleKeyDown
}
