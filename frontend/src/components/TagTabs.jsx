export default function TagTabs({ tags, activeTag, onTagChange, styles }) {
  return (
    <div style={styles.tabBar}>
      {tags.map(tag => (
        <button
          key={tag.id}
          style={{
            ...styles.tab,
            ...(activeTag === tag.id ? styles.tabActive : {}),
          }}
          onFocus={(e) => e.currentTarget.blur()}
          onClick={() => onTagChange(tag.id)}
          onMouseDown={(e) => e.preventDefault()}
        >
          {tag.label}
        </button>
      ))}
    </div>
  )
}
