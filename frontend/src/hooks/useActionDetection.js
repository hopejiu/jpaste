import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { getAll } from '../actions'

// Shared detection cache — persists across re-renders.
const detectionCache = new Map() // entryId → string[]

/**
 * IntersectionObserver-based lazy detection hook.
 * Only detects entries that scroll into the viewport.
 *
 * Returns: { detectedMap, observeItem }
 *   detectedMap: { [entryId]: actionId[] }
 *   observeItem: callback ref to register a list item DOM node
 */
export function useActionDetection(entries, actionConfig, containerRef) {
  const [detectedMap, setDetectedMap] = useState({})
  const observerRef = useRef(null)
  const actionConfigRef = useRef(actionConfig)
  actionConfigRef.current = actionConfig

  // Build sorted list of enabled actions (stable when config doesn't change).
  const sortedActions = useMemo(() => {
    const cfg = actionConfig || {}
    return getAll()
      .filter(a => cfg[a.id]?.enabled !== false)
      .sort((a, b) => (cfg[b.id]?.priority ?? b.priority) - (cfg[a.id]?.priority ?? a.priority))
  }, [actionConfig])

  // Create/recreate IntersectionObserver.
  useEffect(() => {
    if (!containerRef.current) return

    const observer = new IntersectionObserver(
      (observedEntries) => {
        let updates = null
        for (const obs of observedEntries) {
          if (!obs.isIntersecting) continue
          const el = obs.target
          const entryId = Number(el.dataset.entryId)
          if (!entryId || detectionCache.has(entryId)) continue

          const content = el.dataset.entryContent || ''
          const matched = sortedActions
            .filter(a => {
              try { return a.detect(content) }
              catch { return false }
            })
            .map(a => a.id)
            .slice(0, 3)

          detectionCache.set(entryId, matched)
          if (!updates) updates = {}
          updates[entryId] = matched

          // Unobserve after detection.
          observer.unobserve(el)
        }
        if (updates) {
          setDetectedMap(prev => ({ ...prev, ...updates }))
        }
      },
      { root: containerRef.current, rootMargin: '120px' }
    )

    observerRef.current = observer
    return () => observer.disconnect()
  }, [containerRef, sortedActions])

  // Clean stale cache entries when entries list changes.
  useEffect(() => {
    const currentIds = new Set(entries.map(e => e.id))
    let cleared = false
    for (const id of detectionCache.keys()) {
      if (!currentIds.has(id)) {
        detectionCache.delete(id)
        cleared = true
      }
    }
    if (cleared) {
      setDetectedMap(prev => {
        const next = {}
        for (const [k, v] of Object.entries(prev)) {
          if (currentIds.has(Number(k))) next[k] = v
        }
        return next
      })
    }
  }, [entries])

  // observeItem: callback ref to register a DOM element for observation.
  // Returns a ref callback suitable for use on each list item element.
  const observeItem = useCallback((el, entryId, content) => {
    if (!el) return
    el.dataset.entryId = String(entryId)
    el.dataset.entryContent = content

    // Check cache first.
    const cached = detectionCache.get(entryId)
    if (cached !== undefined) {
      setDetectedMap(prev => {
        if (prev[entryId] === cached) return prev // avoid re-render
        return { ...prev, [entryId]: cached }
      })
      return
    }

    // Observe for first detection.
    if (observerRef.current) {
      observerRef.current.observe(el)
    }
  }, [])

  return { detectedMap, observeItem }
}
