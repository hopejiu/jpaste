import { useState, useRef, useEffect } from 'react'
import { Search, X, Code2, ArrowUpDown } from 'lucide-react'
import { useClipboard, SORT_OPTIONS } from '../context/ClipboardContext'

export default function SearchBar({ search, onSearchChange, inputRef: externalRef, isRegex, onToggleRegex }) {
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
    <div className="flex-1 relative flex items-center">
      <Search size={18} className="absolute left-3 text-muted pointer-events-none" />
      <input
        ref={inputRef}
        type="text"
        placeholder={isRegex ? '正则搜索...' : '搜索剪贴板历史...'}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        autoFocus
        className="w-full h-10 bg-surface border border-border rounded-md pl-[38px] pr-9 text-base text-foreground outline-none font-inherit transition-[border-color] duration-fast"
        style={{ fontFamily: 'inherit' }}
      />
      {search && (
        <button
          className="absolute right-[86px] w-7 h-7 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded-sm transition-all duration-fast"
          onClick={() => { onSearchChange(''); inputRef.current?.focus() }}
        >
          <X size={16} />
        </button>
      )}

      {/* Sort control */}
      <div ref={sortRef} className="relative inline-flex">
        <button
          onClick={() => setSortOpen(!sortOpen)}
          title={`排序: ${sortLabel}`}
          className={`w-9 h-9 flex items-center justify-center border-none bg-transparent cursor-pointer rounded-md transition-all duration-fast flex-shrink-0 ${sortOpen ? 'text-primary' : 'text-muted'}`}
        >
          <ArrowUpDown size={16} />
        </button>
        {sortOpen && (
          <div
            className="absolute right-0 top-full mt-1 min-w-[120px] bg-elevated border border-border rounded-md shadow-popup p-1 z-[2000] animate-[slideDown_120ms_ease-out]"
          >
            {SORT_OPTIONS.map(({ field, order, label }) => {
              const key = `${field}-${order}`
              const active = sortField === field && sortOrder === order
              const hovered = hoveredKey === key && !active
              return (
                <button
                  key={key}
                  onClick={() => { setSort(field, order); setSortOpen(false) }}
                  className={`w-full block px-3 py-1.5 text-sm text-left border-none rounded-sm cursor-pointer font-inherit transition-[background] duration-fast ${
                    active ? 'text-primary' : hovered ? 'bg-surface-hover text-foreground' : 'text-foreground'
                  }`}
                  style={{
                    background: active ? 'var(--color-primary-alpha-12)' : hovered ? 'var(--color-surface-hover)' : 'transparent',
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
        onClick={() => onToggleRegex(!isRegex)}
        title={isRegex ? '正则模式（点击关闭）' : '正则模式（点击开启）'}
        className={`w-9 h-9 flex items-center justify-center border-none bg-transparent cursor-pointer rounded-md transition-all duration-fast flex-shrink-0 ${
          isRegex ? 'text-primary bg-primary-alpha-12' : 'text-muted'
        }`}
      >
        <Code2 size={16} />
      </button>
    </div>
  )
}
