import { useEffect, useState, useRef, useCallback } from 'react'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'

/**
 * Hook for lazy-loading image thumbnails via IntersectionObserver.
 * @param {React.RefObject} listRef - ref to the scrollable list container
 * @param {Array} entries - clipboard entries array (triggers re-registration)
 * @returns {{ thumbnailsRef: React.MutableRefObject }}
 *   thumbnailsRef.current: { [entryId]: { url, loading, error } }
 */
export function useImageThumbnail(listRef, entries) {
  const thumbnailsRef = useRef({})
  const [, setTick] = useState(0)
  const thumbObserverRef = useRef(null)

  // Setup observer once on mount.
  useEffect(() => {
    const loadThumb = async (entryId) => {
      const cur = thumbnailsRef.current
      if (cur[entryId]?.url || cur[entryId]?.loading) return
      cur[entryId] = { url: '', loading: true, error: false }
      setTick(t => t + 1)
      try {
        const url = await HistoryService.GetImageDataURL(entryId)
        cur[entryId] = { url, loading: false, error: false }
      } catch {
        cur[entryId] = { url: '', loading: false, error: true }
      }
      setTick(t => t + 1)
    }

    thumbObserverRef.current = new IntersectionObserver((observed) => {
      for (const obs of observed) {
        if (obs.isIntersecting) {
          const id = parseInt(obs.target.dataset.thumbId, 10)
          if (id) loadThumb(id)
        }
      }
    }, { root: listRef.current, rootMargin: '200px' })

    return () => thumbObserverRef.current?.disconnect()
  }, [])

  // Re-register image entries with observer when entries change.
  useEffect(() => {
    const observer = thumbObserverRef.current
    if (!observer || !listRef.current) return
    const items = listRef.current.querySelectorAll('[data-thumb-id]')
    for (const item of items) observer.observe(item)
  }, [entries])

  return { thumbnailsRef }
}
