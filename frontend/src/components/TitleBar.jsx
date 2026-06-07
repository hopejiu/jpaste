import { useState, useEffect, useCallback } from 'react'
import { Minus, Pin, PinOff, Settings } from 'lucide-react'
import { Window } from '@wailsio/runtime'
import { IsPinned, SetPinned } from '../../bindings/jpaste/pinner'
import { useNavigate } from 'react-router-dom'
import { log } from '../logger'

const DRAG_STYLE = { '--wails-draggable': 'drag' }
const NO_DRAG_STYLE = { '--wails-draggable': 'no-drag' }

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
      style={{
        ...DRAG_STYLE,
        height: '36px',
        display: 'flex',
        alignItems: 'center',
        padding: '0 8px',
        background: 'var(--color-surface)',
        borderBottom: '1px solid var(--color-border)',
        flexShrink: 0,
        userSelect: 'none',
      }}
    >
      {/* Left: app actions */}
      <div style={{ display: 'flex', gap: '2px', alignItems: 'center' }}>
        <TitleBtn onClick={togglePin} title={pinned ? '取消置顶' : '置顶窗口'}>
          {pinned ? <Pin size={15} /> : <PinOff size={15} />}
        </TitleBtn>
        <TitleBtn onClick={handleSettings} title="设置">
          <Settings size={15} />
        </TitleBtn>
      </div>

      {/* Center: app name */}
      <span
        style={{
          flex: 1,
          fontSize: '13px',
          fontWeight: 600,
          color: 'var(--color-foreground)',
          paddingLeft: '6px',
          display: 'flex',
          alignItems: 'center',
          gap: '6px',
        }}
      >
        jPaste
      </span>

      {/* Right: window controls */}
      <div style={{ display: 'flex', gap: '2px', alignItems: 'center' }}>
        <TitleBtn onClick={handleMinimise} title="最小化" mr="-4px">
          <Minus size={16} />
        </TitleBtn>
      </div>
    </div>
  )
}

function TitleBtn({ onClick, title, children, mr }) {
  return (
    <button
      onClick={onClick}
      title={title}
      style={{
        ...NO_DRAG_STYLE,
        width: '32px',
        height: '32px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: 'none',
        background: 'transparent',
        color: 'var(--color-foreground)',
        cursor: 'pointer',
        borderRadius: '6px',
        transition: 'all 120ms ease',
        fontFamily: 'inherit',
        marginRight: mr || '0',
      }}
      onMouseEnter={e => e.currentTarget.style.background = 'var(--color-surface-hover)'}
      onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
    >
      {children}
    </button>
  )
}
