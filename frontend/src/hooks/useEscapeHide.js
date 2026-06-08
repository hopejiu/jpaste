import { useEffect } from 'react'
import { Window } from '@wailsio/runtime'

/**
 * Hides the window when Escape is pressed, unless focus is on an input element.
 */
export function useEscapeHide() {
  useEffect(() => {
    const handler = (e) => {
      if (e.key !== 'Escape') return
      const tag = document.activeElement?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || document.activeElement?.isContentEditable) return
      e.preventDefault()
      Window.Close()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])
}
