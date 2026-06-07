import { useState } from 'react'

export default function TagTabs({ tags, activeTag, onTagChange }) {
  const [hoveredTag, setHoveredTag] = useState(null)

  return (
    <div className="flex gap-1 px-4 py-2 border-b border-border flex-shrink-0 overflow-x-auto bg-background">
      {tags.map(tag => {
        const active = activeTag === tag.id
        const hovered = hoveredTag === tag.id
        return (
          <button
            key={tag.id}
            onClick={() => onTagChange(tag.id)}
            onMouseDown={(e) => e.preventDefault()}
            onMouseEnter={() => setHoveredTag(tag.id)}
            onMouseLeave={() => setHoveredTag(null)}
            onFocus={(e) => e.currentTarget.blur()}
            className="px-3.5 py-1 text-xs font-medium border border-transparent rounded-full cursor-pointer whitespace-nowrap font-inherit transition-all duration-fast outline-none"
            style={{
              background: active ? 'var(--color-primary-alpha-12)' : hovered ? 'var(--color-primary-alpha-06)' : 'transparent',
              color: active ? 'var(--color-primary)' : hovered ? 'var(--color-foreground)' : 'var(--color-muted)',
            }}
          >
            {tag.label}
          </button>
        )
      })}
    </div>
  )
}
