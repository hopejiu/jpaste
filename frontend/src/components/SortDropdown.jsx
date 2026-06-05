import { useState, useRef, useEffect } from 'react'
import { ArrowUpDown } from 'lucide-react'
import { useClipboard, SORT_OPTIONS } from '../context/ClipboardContext'

export default function SortDropdown({ style }) {
  const { sortField, sortOrder, setSort, sortLabel } = useClipboard()
  const [open, setOpen] = useState(false)
  const [hoveredKey, setHoveredKey] = useState(null)
  const ref = useRef(null)

  useEffect(() => {
    if (!open) return
    const handler = (e) => {
      if (ref.current && !ref.current.contains(e.target)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} style={{ position: 'relative', flexShrink: 0, ...style }}>
      <button
        onClick={() => setOpen(!open)}
        title="排序"
        style={{
          width: '36px', height: '36px', display: 'flex', alignItems: 'center',
          justifyContent: 'center', border: 'none', background: 'transparent',
          color: 'var(--color-muted)', cursor: 'pointer', borderRadius: 'var(--radius-md)',
          transition: 'all var(--transition-fast)',
        }}
      >
        <ArrowUpDown size={16} />
      </button>

      {open && (
        <div style={{
          position: 'absolute', right: 0, top: '100%', marginTop: '4px',
          minWidth: '120px', background: 'var(--color-elevated)',
          border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
          boxShadow: '0 4px 16px rgba(0,0,0,0.12)', padding: '4px', zIndex: 2000,
        }}>
          {SORT_OPTIONS.map(({ field, order, label }) => {
            const key = `${field}-${order}`
            const active = sortField === field && sortOrder === order
            const hovered = hoveredKey === key && !active
            return (
              <button
                key={key}
                onClick={() => { setSort(field, order); setOpen(false) }}
                style={{
                  display: 'block', width: '100%', padding: '6px 12px',
                  fontSize: 'var(--font-size-sm)', textAlign: 'left',
                  border: 'none', borderRadius: 'var(--radius-sm)',
                  background: active ? 'var(--color-primary-alpha-12)'
                    : hovered ? 'var(--color-surface-hover)' : 'transparent',
                  color: active ? 'var(--color-primary)' : 'var(--color-foreground)',
                  cursor: 'pointer', fontFamily: 'inherit',
                  transition: 'background var(--transition-fast)',
                }}
                onMouseEnter={() => setHoveredKey(key)}
                onMouseLeave={() => setHoveredKey(null)}
              >
                {label}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
