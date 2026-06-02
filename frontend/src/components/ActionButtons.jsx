import { Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen } from 'lucide-react'
import { getById } from '../actions'

const ICON_MAP = {
  Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen,
}

/**
 * Inline action button strip that appears next to existing Copy/Paste buttons.
 * Props: actionIds (string[]), onClick (actionId => void)
 */
export default function ActionButtons({ actionIds, onClick }) {
  if (!actionIds || actionIds.length === 0) return null

  return (
    <>
      {actionIds.map(id => {
        const action = getById(id)
        if (!action) return null
        const IconComp = ICON_MAP[action.icon]
        return (
          <button
            key={id}
            style={styles.btn}
            onClick={(e) => { e.stopPropagation(); onClick(id) }}
            title={action.label}
          >
            {IconComp ? <IconComp size={14} /> : null}
          </button>
        )
      })}
    </>
  )
}

const styles = {
  btn: {
    width: '28px', height: '28px', display: 'flex', alignItems: 'center',
    justifyContent: 'center', border: 'none', background: 'transparent',
    color: 'var(--color-primary)', cursor: 'pointer', borderRadius: '4px',
    transition: 'all var(--transition-fast)',
  },
}
