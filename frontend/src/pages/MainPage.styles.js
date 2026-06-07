export const styles = {
  container: { display: 'flex', flexDirection: 'column', height: '100vh', outline: 'none' },
  header: {
    display: 'flex', alignItems: 'center', padding: '12px 16px', gap: '8px',
    borderBottom: '1px solid var(--color-border)', flexShrink: 0,
    background: 'var(--color-surface)',
  },
  searchBox: { flex: 1, position: 'relative', display: 'flex', alignItems: 'center' },
  searchIcon: { position: 'absolute', left: '12px', color: 'var(--color-muted)', pointerEvents: 'none' },
  searchInput: {
    width: '100%', height: '40px', background: 'var(--color-surface)',
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    padding: '0 36px 0 38px', fontSize: 'var(--font-size-base)',
    color: 'var(--color-foreground)', outline: 'none',
    transition: 'border-color var(--transition-fast)', fontFamily: 'inherit',
  },
  clearBtn: {
    position: 'absolute', right: '6px', width: '28px', height: '28px',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    border: 'none', background: 'transparent', color: 'var(--color-muted)',
    cursor: 'pointer', borderRadius: 'var(--radius-sm)',
  },
  regexBtn: {
    width: '36px', height: '36px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', border: 'none', background: 'transparent',
    color: 'var(--color-muted)', cursor: 'pointer', borderRadius: 'var(--radius-md)',
    flexShrink: 0, transition: 'all var(--transition-fast)',
  },
  regexBtnActive: {
    color: 'var(--color-primary)', background: 'var(--color-primary-alpha-12)',
  },
  tabBar: {
    display: 'flex', gap: '4px', padding: '8px 16px',
    borderBottom: '1px solid var(--color-border)', flexShrink: 0, overflowX: 'auto',
    background: 'var(--color-background)',
  },
  tab: {
    padding: '5px 14px', fontSize: 'var(--font-size-xs)', fontWeight: 500,
    border: '1px solid transparent', borderRadius: 'var(--radius-full)',
    background: 'transparent', color: 'var(--color-muted)',
    cursor: 'pointer', whiteSpace: 'nowrap', fontFamily: 'inherit',
    transition: 'all var(--transition-fast)', outline: 'none',
  },
  tabActive: {
    background: 'var(--color-primary-alpha-12)', color: 'var(--color-primary)',
  },
  list: { flex: 1, overflowY: 'auto', padding: '4px 0' },
  loading: {
    textAlign: 'center', padding: '12px',
    fontSize: 'var(--font-size-xs)', color: 'var(--color-muted)',
  },
  empty: {
    display: 'flex', flexDirection: 'column', alignItems: 'center',
    justifyContent: 'center', height: '200px', padding: '24px', textAlign: 'center',
  },
  emptyTitle: {
    fontSize: 'var(--font-size-lg)', fontWeight: 500,
    color: 'var(--color-foreground)', marginBottom: '8px',
  },
  emptyDesc: {
    fontSize: 'var(--font-size-sm)', color: 'var(--color-muted)',
    lineHeight: 1.6, maxWidth: '300px',
  },
  item: {
    display: 'flex', gap: '10px', padding: '10px 16px', cursor: 'pointer',
    transition: 'background var(--transition-fast)',
    borderBottom: '1px solid var(--color-border)',
  },
  itemFocused: { background: 'var(--color-surface-hover)' },
  shortcut: {
    minWidth: '24px', height: '22px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', background: 'var(--color-primary-alpha-15)',
    color: 'var(--color-primary)', borderRadius: '4px',
    fontSize: 'var(--font-size-xs)', fontWeight: 600, marginTop: '1px', flexShrink: 0,
  },
  itemContent: { flex: 1, minWidth: 0 },
  itemText: {
    fontSize: 'var(--font-size-sm)', lineHeight: 1.55, color: 'var(--color-foreground)',
    whiteSpace: 'pre-wrap', wordBreak: 'break-word', overflow: 'hidden',
  },
  itemMeta: { display: 'flex', alignItems: 'center', gap: '4px', marginTop: '6px' },
  itemTime: { flex: 1, display: 'flex', gap: '8px', alignItems: 'baseline' },
  itemRel: { fontSize: 'var(--font-size-xs)', color: 'var(--color-muted)' },
  itemAbs: { fontSize: '11px', color: 'var(--color-muted)', opacity: 0.65 },
  actionBtn: {
    width: '28px', height: '28px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', border: 'none', background: 'transparent',
    color: 'var(--color-muted)', cursor: 'pointer', borderRadius: '4px',
    transition: 'all var(--transition-fast)',
  },
  copyTextBtn: {
    width: '28px', height: '28px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', border: 'none', background: 'transparent',
    color: 'var(--color-muted)', cursor: 'pointer', borderRadius: '4px',
    transition: 'all var(--transition-fast)',
  },
  fileBadge: {
    display: 'inline-flex', alignItems: 'center', gap: '3px',
    marginLeft: '6px', padding: '1px 6px',
    background: 'var(--color-badge-file-bg)', color: 'var(--color-badge-file)',
    borderRadius: '999px', fontSize: '10px', fontWeight: 600,
    verticalAlign: 'middle', whiteSpace: 'nowrap',
  },
  footer: {
    borderTop: '1px solid var(--color-border)', padding: '8px 16px',
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    flexShrink: 0, gap: '8px',
    background: 'var(--color-background)',
  },
  footerText: { fontSize: 'var(--font-size-xs)', color: 'var(--color-muted)' },
  ctxOverlay: {
    position: 'fixed', minWidth: '160px', background: 'var(--color-elevated)',
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    boxShadow: '0 4px 16px rgba(0,0,0,0.12)', padding: '4px', zIndex: 2000,
    animation: 'slideDown 120ms ease-out',
  },
  ctxItem: {
    display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 12px',
    fontSize: 'var(--font-size-sm)', color: 'var(--color-foreground)',
    borderRadius: 'var(--radius-sm)', cursor: 'pointer',
    transition: 'background var(--transition-fast)',
  },
  ctxItemDanger: {
    display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 12px',
    fontSize: 'var(--font-size-sm)', color: 'var(--color-destructive)',
    borderRadius: 'var(--radius-sm)', cursor: 'pointer',
    transition: 'background var(--transition-fast)',
  },
  itemImagePlaceholder: {
    display: 'flex', alignItems: 'center', gap: '8px',
    padding: '8px 12px', borderRadius: 'var(--radius-md)',
    background: 'var(--color-primary-alpha-06)', fontSize: 'var(--font-size-sm)',
    color: 'var(--color-muted)',
  },
  itemImageLabel: { fontWeight: 500 },
  // Thumbnail for image-only entries — auto-sized, constrained width.
  thumbImg: {
    maxWidth: '100%', maxHeight: '160px',
    borderRadius: 'var(--radius-sm)', objectFit: 'contain',
    display: 'block', background: 'var(--color-primary-alpha-04)',
  },
  // Inline thumbnail for mixed text+image entries.
  thumbInline: {
    width: '36px', height: '36px',
    borderRadius: '4px', objectFit: 'cover',
    flexShrink: 0, marginLeft: '8px',
  },
  // Row layout for mixed text+image content.
  itemContentRow: {
    display: 'flex', gap: '8px', alignItems: 'flex-start',
  },
  // Favorite star button.
  favBtn: {
    width: '28px', height: '28px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', border: 'none', background: 'transparent',
    color: 'var(--color-muted)', cursor: 'pointer', borderRadius: '4px',
    transition: 'all var(--transition-fast)', flexShrink: 0,
  },
  favBtnActive: { color: 'var(--color-favorite)' },
  // Source application label.
  sourceApp: {
    fontSize: '11px', color: 'var(--color-muted)', opacity: 0.8,
    marginLeft: '8px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
    maxWidth: '120px',
  },
  // Checkmark animation overlay.
  checkmark: {
    position: 'absolute', top: '50%', right: '12px',
    transform: 'translateY(-50%)',     color: 'var(--color-success)',
    animation: 'fadeScaleIn 250ms ease-out',
    pointerEvents: 'none',
  },
  // Image-only entry hover style.
  itemImage: {
    cursor: 'pointer',
    transition: 'background var(--transition-fast), box-shadow var(--transition-fast)',
  },
  // Thumbnail wrapper for hover overlay positioning.
  thumbWrapper: { position: 'relative', display: 'inline-block', maxWidth: '100%' },
  // Hover overlay badge on image entries.
  thumbOverlay: {
    position: 'absolute', inset: 0, display: 'flex',
    alignItems: 'center', justifyContent: 'center',
    background: 'rgba(0,0,0,0.35)', borderRadius: 'var(--radius-sm)',
    opacity: 0, transition: 'opacity var(--transition-fast)',
    pointerEvents: 'none',
  },
  thumbOverlayVisible: { opacity: 1 },
  thumbOverlayText: {
    display: 'flex', alignItems: 'center', gap: '4px',
    color: '#fff', fontSize: 'var(--font-size-sm)', fontWeight: 600,
  },
}
