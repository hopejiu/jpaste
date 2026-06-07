import { useState, useEffect, useCallback } from 'react'
import { Minus, Pin, PinOff, Settings } from 'lucide-react'
import { Window } from '@wailsio/runtime'
import { IsPinned, SetPinned } from '../../bindings/jpaste/pinner'
import { useNavigate } from 'react-router-dom'
import { log } from '../logger'

export default function TitleBar() {
  const [pinned, setPinned] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    IsPinned().then(v => setPinned(!!v)).catch(() => {})
  }, [])

  const togglePin = useCallback(() => {
    const next = !pinned
    setPinned(next)
    SetPinned(next).catch(() => setPinned(!next))
  }, [pinned])

  const handleMinimise = useCallback(() => {
    Window.Minimise().catch(e => log.error('TitleBar', 'minimise failed:', e))
  }, [])

  const handleSettings = useCallback(() => {
    navigate('/settings')
  }, [navigate])

  return (
    <div
      className="glass-titlebar flex items-center h-9 px-2 flex-shrink-0 select-none"
      style={{ '--wails-draggable': 'drag' }}
    >
      {/* Left: app actions */}
      <div className="flex items-center gap-0.5">
        <TitleBtn onClick={togglePin} title={pinned ? '取消置顶' : '置顶窗口'}>
          {pinned ? <Pin size={15} /> : <PinOff size={15} />}
        </TitleBtn>
        <TitleBtn onClick={handleSettings} title="设置">
          <Settings size={15} />
        </TitleBtn>
      </div>

      {/* Center: app name */}
      <span className="flex-1 text-[13px] font-semibold text-foreground pl-1.5 flex items-center gap-1.5">
        jPaste
      </span>

      {/* Right: window controls */}
      <div className="flex items-center gap-0.5">
        <TitleBtn onClick={handleMinimise} title="最小化" mr="-4px">
          <Minus size={16} />
        </TitleBtn>
      </div>
    </div>
  )
}

function TitleBtn({ onClick, title, children, mr }) {
  const [hovered, setHovered] = useState(false)
  return (
    <button
      onClick={onClick}
      title={title}
      className="flex items-center justify-center border-none cursor-pointer font-inherit"
      style={{
        width: '32px', height: '32px', borderRadius: '6px',
        transition: 'all 120ms ease', marginRight: mr || '0',
        background: hovered ? 'var(--color-surface-hover)' : 'transparent',
        color: 'var(--color-foreground)',
        '--wails-draggable': 'no-drag',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {children}
    </button>
  )
}
