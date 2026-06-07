import { useState, useRef, useEffect } from 'react'
import { Search, X, Code2, ArrowUpDown } from 'lucide-react'
import { useClipboard, SORT_OPTIONS } from '../context/ClipboardContext'

export default function SearchBar({ search, onSearchChange, inputRef: externalRef, isRegex, onToggleRegex, styles }) {
  const internalRef = useRef(null)
  const inputRef = externalRef || internalRef

  const { sortField, sortOrder, setSort, sortLabel } = useClipboard()
  const [sortOpen, setSortOpen] = useState(false)
  const [hoveredKey, setHoveredKey] = useState(null)
  const sortRef = useRef(null)

  useEffect(() => {
    if (!sortOpen) return
    const handler = (e) => {
      if (sortRef.current && !sortRef.current.contains(e.target)) {
        setSortOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [sortOpen])

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  return (
    <div style={styles.searchBox}>
      <Search size={18} style={styles.searchIcon} />
      <input
        ref={inputRef}
        style={styles.searchInput}
        type="text"
        placeholder={isRegex ? '正则搜索...' : '搜索剪贴板历史...'}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        autoFocus
      />
      {search && (
        <button style={styles.clearBtn} onClick={() => { onSearchChange(''); inputRef.current?.focus() }}>
          <X size={16} />
        </button>
      )}

      {/* Sort control — integrated into search bar */}
      <div ref={sortRef} style={{ position: 'relative', display: 'inline-flex' }}>
        <button
          onClick={() => setSortOpen(!sortOpen)}
          title={`排序: ${sortLabel}`}
          style={{
            width: '36px', height: '36px', display: 'flex', alignItems: 'center',
            justifyContent: 'center', border: 'none', background: 'transparent',
            color: sortOpen ? 'var(--color-primary)' : 'var(--color-muted)',
            cursor: 'pointer', borderRadius: 'var(--radius-md)',
            transition: 'all var(--transition-fast)', flexShrink: 0,
          }}
        >
          <ArrowUpDown size={16} />
        </button>
        {sortOpen && (
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
                  onClick={() => { setSort(field, order); setSortOpen(false) }}
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

      <button
        style={isRegex ? { ...styles.regexBtn, ...styles.regexBtnActive } : styles.regexBtn}
        onClick={() => onToggleRegex(!isRegex)}
        title={isRegex ? '正则模式（点击关闭）' : '正则模式（点击开启）'}
      >
        <Code2 size={16} />
      </button>
    </div>
  )
}
