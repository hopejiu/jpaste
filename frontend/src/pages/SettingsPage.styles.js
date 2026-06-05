export const styles = {
  container: { display: 'flex', flexDirection: 'column', height: '100vh', outline: 'none', animation: 'slideDown 200ms ease-out', background: 'var(--color-surface)' },
  header: { display: 'flex', alignItems: 'center', padding: '12px 16px', gap: '12px', borderBottom: '1px solid var(--color-border)', flexShrink: 0 },
  backBtn: { width: '36px', height: '36px', display: 'flex', alignItems: 'center', justifyContent: 'center', border: 'none', background: 'transparent', color: 'var(--color-foreground)', cursor: 'pointer', borderRadius: 'var(--radius-md)', transition: 'background var(--transition-fast)' },
  title: { fontSize: 'var(--font-size-xl)', fontWeight: 600, flex: 1 },
  savedBadge: { fontSize: 'var(--font-size-xs)', fontWeight: 500, color: '#4ADE80', background: 'rgba(74,222,128,0.12)', padding: '4px 10px', borderRadius: 'var(--radius-full)' },
  content: { flex: 1, overflowY: 'auto', padding: '0 16px' },
  group: { padding: '20px 0', borderBottom: '1px solid var(--color-border)' },
  row: { display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  label: { fontSize: 'var(--font-size-base)', fontWeight: 500, color: 'var(--color-foreground)', marginBottom: '2px' },
  desc: { fontSize: 'var(--font-size-xs)', color: 'var(--color-muted)', marginTop: '2px' },

  modRow: { display: 'flex', gap: '8px', marginTop: '12px' },
  modChip: {
    padding: '6px 14px', fontSize: 'var(--font-size-sm)', fontWeight: 500,
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    background: 'var(--color-surface)', color: 'var(--color-muted)',
    cursor: 'pointer', fontFamily: 'inherit', transition: 'all var(--transition-fast)',
    outline: 'none',
  },
  modChipActive: {
    borderColor: 'var(--color-primary)', color: 'var(--color-primary)',
    background: 'var(--color-primary-alpha-08)',
  },
  keyInput: {
    width: '80px', height: '36px', marginTop: '4px', textAlign: 'center',
    fontSize: 'var(--font-size-lg)', fontWeight: 600,
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    background: 'var(--color-surface)', color: 'var(--color-foreground)',
    outline: 'none', fontFamily: 'inherit',
  },
  hotkeyPreview: {
    marginTop: '10px', fontSize: 'var(--font-size-sm)', color: 'var(--color-muted)',
    fontFamily: 'monospace',
  },
  hotkeyError: {
    marginTop: '8px', fontSize: 'var(--font-size-xs)', color: '#DC2626',
    lineHeight: 1.4,
  },

  retainControl: { display: 'flex', alignItems: 'center', gap: '10px' },
  slider: { width: '120px', accentColor: 'var(--color-primary)', cursor: 'pointer' },
  retainValue: { fontSize: 'var(--font-size-sm)', fontWeight: 500, color: 'var(--color-foreground)', minWidth: '56px' },
  statsRow: { marginTop: '10px', fontSize: 'var(--font-size-xs)', color: 'var(--color-muted)', display: 'flex', gap: '6px', alignItems: 'center' },
  statsDot: { opacity: 0.4 },
  clearAllBtn: {
    marginTop: '10px', padding: '8px 14px', display: 'flex', alignItems: 'center',
    gap: '6px', fontSize: 'var(--font-size-xs)', fontWeight: 500,
    border: '1px solid rgba(239,68,68,0.3)', borderRadius: 'var(--radius-md)',
    background: 'rgba(239,68,68,0.06)', color: '#DC2626',
    cursor: 'pointer', fontFamily: 'inherit',
  },
  radioGroup: { display: 'flex', flexDirection: 'column', gap: '6px', marginTop: '12px' },
  radioLabel: { display: 'flex', alignItems: 'center', gap: '10px', padding: '10px 14px', borderRadius: 'var(--radius-md)', border: '1px solid var(--color-border)', cursor: 'pointer', fontSize: 'var(--font-size-sm)', transition: 'all var(--transition-fast)' },
  radioActive: { borderColor: 'var(--color-primary)', background: 'var(--color-primary-alpha-08)' },
  radio: { accentColor: 'var(--color-primary)' },

  // Theme selector
  themeGrid: { display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '12px' },
  themeCard: {
    display: 'flex', alignItems: 'center', gap: '12px',
    padding: '12px 14px', borderRadius: 'var(--radius-md)',
    border: '1px solid var(--color-border)',
    background: 'var(--color-surface)',
    cursor: 'pointer', fontFamily: 'inherit', textAlign: 'left',
    transition: 'all var(--transition-fast)',
  },
  themeCardActive: {
    borderColor: 'var(--color-primary)',
    background: 'var(--color-primary-alpha-06)',
  },
  themeSwatch: { display: 'flex', gap: '4px', flexShrink: 0 },
  themeColorDot: {
    width: '20px', height: '20px', borderRadius: '50%',
    border: '2px solid var(--color-border)',
  },

  actionList: { display: 'flex', flexDirection: 'column', gap: '4px', marginTop: '12px' },
  actionItem: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '10px 12px', borderRadius: 'var(--radius-md)',
    border: '1px solid var(--color-border)',
  },
  actionItemLeft: { display: 'flex', alignItems: 'center', gap: '10px' },
  actionName: { fontSize: 'var(--font-size-sm)', fontWeight: 500, color: 'var(--color-foreground)' },
  actionItemRight: { display: 'flex', alignItems: 'center', gap: '2px' },
  priorityBtn: {
    width: '28px', height: '28px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', border: 'none', background: 'transparent',
    color: 'var(--color-muted)', cursor: 'pointer', borderRadius: '4px',
  },

  wdInput: {
    width: '100%', height: '36px', padding: '0 12px',
    fontSize: 'var(--font-size-sm)', fontFamily: 'inherit',
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    background: 'var(--color-surface)', color: 'var(--color-foreground)',
    outline: 'none', transition: 'border-color var(--transition-fast)',
  },
  wdBtn: {
    flex: 1, height: '36px',
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    background: 'var(--color-surface)', color: 'var(--color-foreground)',
    fontSize: 'var(--font-size-sm)', fontWeight: 500, cursor: 'pointer',
    fontFamily: 'inherit', transition: 'all var(--transition-fast)',
  },
  wdBtnPrimary: {
    background: 'var(--color-primary)', color: '#fff', borderColor: 'var(--color-primary)',
  },
}
