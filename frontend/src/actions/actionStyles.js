// Shared styles for action modal components.
// Each action module imports only the styles it needs.

export const formBody = { display: 'flex', flexDirection: 'column', gap: '12px' }

export const label = {
  fontSize: 'var(--font-size-xs)',
  color: 'var(--color-muted)',
}

export const textarea = {
  width: '100%', minHeight: '80px', maxHeight: '260px',
  padding: '12px', borderRadius: 'var(--radius-md)',
  background: 'var(--color-surface)',
  border: '1px solid var(--color-border)',
  fontSize: 'var(--font-size-sm)', fontFamily: 'monospace',
  color: 'var(--color-foreground)',
  outline: 'none', resize: 'vertical',
  transition: 'border-color var(--transition-fast)',
}

export const output = {
  padding: '12px', borderRadius: 'var(--radius-md)',
  background: 'var(--color-surface)', minHeight: '40px',
  fontSize: 'var(--font-size-sm)', fontFamily: 'monospace',
  whiteSpace: 'pre-wrap', wordBreak: 'break-all',
  overflow: 'auto', maxHeight: '200px',
  border: '1px solid var(--color-border)',
}

export const displayValue = {
  padding: '12px', borderRadius: 'var(--radius-md)',
  background: 'var(--color-surface)',
  fontSize: 'var(--font-size-sm)', fontFamily: 'monospace',
  wordBreak: 'break-all',
  color: 'var(--color-foreground)',
}

export const primaryBtn = {
  alignSelf: 'flex-start', padding: '8px 20px',
  background: 'var(--color-primary)', color: '#fff',
  border: 'none', borderRadius: 'var(--radius-md)',
  fontSize: 'var(--font-size-sm)', fontWeight: 500,
  cursor: 'pointer',
}

export const errorMsg = {
  fontSize: 'var(--font-size-xs)',
  color: 'var(--color-destructive)',
  padding: '8px 12px', borderRadius: 'var(--radius-md)',
  background: 'rgba(239,68,68,0.08)',
}

export const resultText = {
  margin: 0, fontSize: 'var(--font-size-sm)',
  fontFamily: 'monospace', whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
}
