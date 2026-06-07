import { X } from 'lucide-react'
import { useEffect, useRef } from 'react'

/**
 * Reusable modal overlay for action content.
 * Props: open, onClose, title, wide, children
 */
export default function ActionModal({ open, onClose, title, children }) {
  const overlayRef = useRef(null)

  useEffect(() => {
    if (!open) return
    const handler = (e) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-[3000] flex items-center justify-center animate-[fadeScaleIn_150ms_ease-out]"
      style={{ background: 'rgba(0,0,0,0.35)' }}
      onClick={(e) => { if (e.target === overlayRef.current) onClose() }}
    >
      <div className="bg-elevated border border-border rounded-lg shadow-glass-lg w-[90%] max-w-[520px] max-h-[80vh] flex flex-col animate-[slideUp_200ms_ease-out]">
        <div className="flex items-center justify-between px-[18px] py-3.5 border-b border-border flex-shrink-0">
          <h3 className="m-0 text-base font-semibold text-foreground">{title}</h3>
          <button
            className="w-8 h-8 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded-sm transition-all duration-fast hover:bg-surface-hover"
            onClick={onClose}
            title="关闭"
          >
            <X size={18} />
          </button>
        </div>
        <div className="p-[18px] overflow-auto flex-1">
          {children}
        </div>
      </div>
    </div>
  )
}
