import { Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen, Terminal, Radio, Link } from 'lucide-react'
import { getById } from '../actions'

const ICON_MAP = {
  Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen, Terminal, Radio, Url: Link,
}

/**
 * Inline action button strip that appears next to existing Copy/Paste buttons.
 */
export default function ActionButtons({ actionIds, onClick, baseIdx = 0, selectedActionIdx = -1, hoverRing }) {
  if (!actionIds || actionIds.length === 0) return null

  return (
    <>
      {actionIds.map((id, i) => {
        const action = getById(id)
        if (!action) return null
        const IconComp = ICON_MAP[action.icon]
        const btnIdx = baseIdx + i
        const isSelected = selectedActionIdx === btnIdx
        return (
          <button
            key={id}
            data-action-btn={btnIdx}
            className={`w-7 h-7 flex items-center justify-center border-none bg-transparent text-primary cursor-pointer rounded transition-all duration-fast flex-shrink-0 ${hoverRing ? 'hover:ring-2 hover:ring-primary hover:ring-inset' : ''} ${isSelected ? 'ring-2 ring-primary ring-inset' : ''}`}
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
