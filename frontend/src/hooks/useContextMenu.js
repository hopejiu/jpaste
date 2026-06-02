import { useState, useEffect } from 'react'

export function useContextMenu() {
  const [ctxMenu, setCtxMenu] = useState(null)

  useEffect(() => {
    if (!ctxMenu) return
    const close = () => setCtxMenu(null)
    document.addEventListener('click', close)
    return () => document.removeEventListener('click', close)
  }, [ctxMenu])

  const showCtxMenu = (e, entry) => {
    e.preventDefault()
    setCtxMenu({ x: e.clientX, y: e.clientY, entry })
  }

  const hideCtxMenu = () => setCtxMenu(null)

  return { ctxMenu, showCtxMenu, hideCtxMenu }
}
