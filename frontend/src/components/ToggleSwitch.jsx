export default function ToggleSwitch({ checked, onChange, label }) {
  return (
    <button
      className={`toggle-track${checked ? ' active' : ''}`}
      style={{
        background: checked ? 'var(--color-primary)' : 'var(--color-muted)',
        border: 'none', cursor: 'pointer', flexShrink: 0,
      }}
      onClick={onChange}
      aria-label={label}
      role="switch"
      aria-checked={checked}
    />
  )
}
