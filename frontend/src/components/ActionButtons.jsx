import { Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen, Terminal, Radio } from 'lucide-react'
import { getById } from '../actions'

const ICON_MAP = {
  Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen, Terminal, Radio,
}

/**
 * Inline action button strip that appears next to existing Copy/Paste buttons.
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
            className="w-7 h-7 flex items-center justify-center border-none bg-transparent text-primary cursor-pointer rounded transition-all duration-fast"
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
