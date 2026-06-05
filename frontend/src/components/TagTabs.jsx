import { useState } from 'react'

export default function TagTabs({ tags, activeTag, onTagChange, styles }) {
  const [hoveredTag, setHoveredTag] = useState(null)

  return (
    <div style={styles.tabBar}>
      {tags.map(tag => {
        const active = activeTag === tag.id
        const hovered = hoveredTag === tag.id
        return (
          <button
            key={tag.id}
            style={{
              ...styles.tab,
              ...(active ? styles.tabActive : hovered ? {
                background: 'var(--color-primary-alpha-06)',
                color: 'var(--color-foreground)',
              } : {}),
            }}
            onFocus={(e) => e.currentTarget.blur()}
            onClick={() => onTagChange(tag.id)}
            onMouseDown={(e) => e.preventDefault()}
            onMouseEnter={() => setHoveredTag(tag.id)}
            onMouseLeave={() => setHoveredTag(null)}
          >
            {tag.label}
          </button>
        )
      })}
    </div>
  )
}
