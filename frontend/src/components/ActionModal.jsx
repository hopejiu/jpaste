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
      style={styles.overlay}
      onClick={(e) => { if (e.target === overlayRef.current) onClose() }}
    >
      <div style={styles.modal}>
        <div style={styles.header}>
          <h3 style={styles.title}>{title}</h3>
          <button style={styles.closeBtn} onClick={onClose} title="关闭">
            <X size={18} />
          </button>
        </div>
        <div style={styles.content}>
          {children}
        </div>
      </div>
    </div>
  )
}

const styles = {
  overlay: {
    position: 'fixed', inset: 0, zIndex: 3000,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    background: 'rgba(0,0,0,0.35)',
    animation: 'fadeIn 150ms ease-out',
  },
  modal: {
    background: 'var(--color-elevated)',
    borderRadius: 'var(--radius-lg, 12px)',
    boxShadow: '0 8px 40px rgba(0,0,0,0.18)',
    width: '90%', maxWidth: '520px', maxHeight: '80vh',
    display: 'flex', flexDirection: 'column',
    animation: 'slideUp 200ms ease-out',
  },
  header: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '14px 18px', borderBottom: '1px solid var(--color-border)',
    flexShrink: 0,
  },
  title: {
    margin: 0, fontSize: 'var(--font-size-base)', fontWeight: 600,
    color: 'var(--color-foreground)',
  },
  closeBtn: {
    width: '32px', height: '32px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', border: 'none', background: 'transparent',
    color: 'var(--color-muted)', cursor: 'pointer', borderRadius: 'var(--radius-sm)',
  },
  content: {
    padding: '18px', overflow: 'auto', flex: 1,
  },
}
