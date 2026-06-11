import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Events } from '@wailsio/runtime'
import { ArrowLeft, ChevronUp, ChevronDown, ChevronRight, Trash2, Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen, Terminal, Radio, Link, Clock } from 'lucide-react'
import { useApp } from '../context/AppContext'
import { useClipboard } from '../context/ClipboardContext'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { log } from '../logger'
import { getAll } from '../actions'
import { formatBytes } from '../utils/format'
import ToggleSwitch from '../components/ToggleSwitch'
import Modal from '../components/Modal'

const MODS = ['Ctrl', 'Alt', 'Shift', 'Win']

const THEMES = [
  { id: 'a', label: '冷调极简', desc: '青碧主色 · 清爽高效', colors: ['#0D9488', '#F0FDFA', '#FFFFFF'] },
  { id: 'b', label: '靛蓝专注', desc: 'Indigo 经典 · 生产力优先', colors: ['#6366F1', '#F8FAFC', '#FFFFFF'] },
  { id: 'c', label: '深色沉浸', desc: '暗色氛围 · 夜间友好', colors: ['#6C78E0', '#0A0A0A', '#141414'] },
  { id: 'd', label: '梦幻浅紫', desc: '淡紫主色 · 温柔知性', colors: ['#8B5CF6', '#F8F6FF', '#FFFFFF'] },
  { id: 'e', label: '柔粉轻语', desc: '粉嫩主色 · 甜美清新', colors: ['#EC4899', '#FFF5F7', '#FFFFFF'] },
]

function parseHotkey(hotkey) {
  const parts = hotkey.split('+').map(p => p.trim())
  const mods = []
  let key = ''
  for (const p of parts) {
    const found = MODS.find(m => m.toLowerCase() === p.toLowerCase())
    if (found) mods.push(found)
    else key = p
  }
  return { mods, key }
}

export default function SettingsPage() {
  const { settings, saveSettings } = useApp()
  const { clearAll } = useClipboard()
  const navigate = useNavigate()
  const [local, setLocal] = useState({ ...settings })
  const [saved, setSaved] = useState(false)

  // Stats state.
  const [stats, setStats] = useState({ count: 0, total_bytes: 0 })
  const [clearing, setClearing] = useState(false)
  const [showClearModal, setShowClearModal] = useState(false)
  const [hoveredClearBtn, setHoveredClearBtn] = useState(null)
  const [expandedId, setExpandedId] = useState(null)

  // Track whether the notify toggle has been explicitly changed by the user
  // to avoid showing preview on initial mount when notify is already ON.
  const notifyToggled = useRef(false)

  // Cleanup: hide preview when leaving settings page.
  useEffect(() => {
    return () => {
      Events.Emit('toast-hide-preview')
    }
  }, [])

  // Global Escape handler (more reliable than container onKeyDown).
  useEffect(() => {
    const handler = (e) => {
      if (e.key !== 'Escape') return
      log.debug('SettingsPage', 'Escape pressed, showClearModal:', showClearModal)
      e.preventDefault()
      if (showClearModal) {
        log.debug('SettingsPage', 'Escape → closing clear modal')
        setShowClearModal(false)
      } else {
        log.debug('SettingsPage', 'Escape → navigating to /')
        navigate('/')
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [showClearModal, navigate])

  const ICON_MAP = { Calculator, Braces, Binary, Languages, ExternalLink, FolderOpen, Terminal, Radio, Url: Link, Clock }

  useEffect(() => {
    HistoryService.GetStats()
      .then(s => { if (s) setStats(s) })
      .catch(() => {})
  }, [])

  // Hotkey UI state.
  const parsed = parseHotkey(local.hotkey)
  const [mods, setMods] = useState(parsed.mods)
  const [key, setKey] = useState(parsed.key)
  const [hotkeyError, setHotkeyError] = useState('')
  const [hoveredMod, setHoveredMod] = useState(null)

  // Refs to avoid stale closures in useCallback.
  const localRef = useRef(local)
  localRef.current = local
  const settingsRef = useRef(settings)
  settingsRef.current = settings

  useEffect(() => {
    setLocal({ ...settings })
    const p = parseHotkey(settings.hotkey)
    setMods(p.mods)
    setKey(p.key)
    setHotkeyError('')
  }, [settings])

  // Hover state for radio labels
  const [hoveredRadio, setHoveredRadio] = useState(null)

  const handleSave = useCallback(async (updates) => {
    const current = localRef.current
    const updated = { ...current, ...updates }
    setLocal(updated)
    try {
      await saveSettings(updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 1500)
      if (updates.hotkey !== undefined) setHotkeyError('')
      if (updates.theme && updates.theme !== settings.theme) {
        window.location.reload()
      }
    } catch (err) {
      if (updates.hotkey !== undefined) {
        const raw = err.message || ''
        let msg = raw
        try { const p = JSON.parse(raw); if (p.message) msg = p.message } catch {}
        log.warn('SettingsPage', 'hotkey save failed, reverting', { hotkey: updates.hotkey, error: msg })
        setHotkeyError(msg)
        const good = settingsRef.current
        setLocal({ ...good })
        const p = parseHotkey(good.hotkey)
        log.debug('SettingsPage', 'revert to', { hotkey: good.hotkey, mods: p.mods, key: p.key })
        setMods(p.mods)
        setKey(p.key)
      }
    }
  }, [saveSettings, settings.theme])

  const updateHotkey = useCallback((newMods, newKey) => {
    if (newMods.length === 0 && !newKey) return
    log.debug('SettingsPage', 'updateHotkey', { mods: newMods, key: newKey })
    setMods(newMods)
    setKey(newKey)
    setHotkeyError('')
    const sorted = [...newMods].sort((a, b) => MODS.indexOf(a) - MODS.indexOf(b))
    const hk = newKey ? [...sorted, newKey].join('+') : sorted.join('+')
    log.debug('SettingsPage', 'updateHotkey hk', hk)
    handleSave({ hotkey: hk })
  }, [handleSave])

  const toggleMod = (m) => {
    log.debug('SettingsPage', 'toggleMod', { mod: m, currentMods: mods, currentKey: key })
    const isSelected = mods.includes(m)
    let next
    if (isSelected) {
      if (mods.length === 1) return
      next = mods.filter(x => x !== m)
    } else {
      next = [...mods, m]
    }
    updateHotkey(next, key)
  }

  const handleKeyInput = (e) => {
    const val = e.target.value.replace(/[^a-zA-Z0-9]/g, '').slice(-1).toUpperCase()
    if (val) updateHotkey(mods, val)
  }

  // Pre-compute sorted action modules.
  const sortedActions = useMemo(() => {
    const cfg = local.action_config || {}
    return [...getAll()].sort((a, b) => {
      const pa = cfg[a.id]?.priority ?? a.priority
      const pb = cfg[b.id]?.priority ?? b.priority
      return pb - pa
    }).map(action => ({
      ...action,
      config: cfg[action.id] || { enabled: true, priority: action.priority },
    }))
  }, [local.action_config])

  const moveAction = (index, direction) => {
    if (direction === 'up' && index === 0) return
    if (direction === 'down' && index === sortedActions.length - 1) return
    const targetIdx = direction === 'up' ? index - 1 : index + 1
    const cfg = { ...local.action_config }
    const a = sortedActions[index]
    const b = sortedActions[targetIdx]
    cfg[a.id] = { ...a.config, priority: b.config.priority }
    cfg[b.id] = { ...b.config, priority: a.config.priority }
    handleSave({ action_config: cfg })
  }

  const doClearAll = async (keepFavorites) => {
    setShowClearModal(false)
    setClearing(true)
    try {
      await Promise.race([
        clearAll(keepFavorites),
        new Promise((_, reject) => setTimeout(() => reject(new Error('clearAll timeout')), 10000)),
      ])
      HistoryService.GetStats().then(s => { if (s) setStats(s) }).catch(() => {})
    } catch (err) {
      log.error('SettingsPage', 'ClearAll failed:', err)
    } finally {
      setClearing(false)
    }
  }

  const toggleAction = (action) => {
    const cfg = { ...local.action_config }
    cfg[action.id] = { ...action.config, enabled: !action.config.enabled }
    handleSave({ action_config: cfg })
  }
  
  const displayKey = [mods.join('+'), key].filter(Boolean).join(' + ')

  const settingGroups = [
    {
      items: [
        { key: 'notify_enabled', label: '剪贴板通知', desc: '捕获到新剪贴板内容时显示通知' },
        { key: 'auto_start', label: '开机自启', desc: '登录时自动启动 jPaste' },
        { key: 'start_minimized', label: '启动时最小化', desc: '启动后最小化到系统托盘（不弹出窗口）' },
      ]
    }
  ]

  return (
    <div className="flex flex-col h-screen outline-none animate-[slideDown_200ms_ease-out] bg-surface" tabIndex={0}>
      {/* Header */}
      <div className="flex items-center px-4 py-3 gap-3 border-b border-border flex-shrink-0">
        <button className="w-9 h-9 flex items-center justify-center border-none bg-transparent text-foreground cursor-pointer rounded-md transition-[background] duration-fast hover:bg-surface-hover" onClick={() => navigate('/')}>
          <ArrowLeft size={20} />
        </button>
        <h2 className="text-xl font-semibold flex-1">设置</h2>
        {saved && (
          <span className="text-xs font-medium px-2.5 py-1 rounded-full" style={{ color: '#4ADE80', background: 'rgba(74,222,128,0.12)' }}>
            已保存
          </span>
        )}
      </div>

      <div className="flex-1 overflow-y-auto px-0">
        {/* Hotkey */}
        <div className="px-4 py-5 border-b border-border">
          <div className="text-base font-medium text-foreground mb-0.5">全局快捷键</div>
          <div className="text-xs text-muted mt-0.5">显示/隐藏 jPaste 窗口</div>
          <div className="flex gap-2 mt-3">
            {MODS.map(m => {
              const active = mods.includes(m)
              const hovered = hoveredMod === m
              return (
                <button
                  key={m}
                  tabIndex={-1}
                  className="px-3.5 py-1.5 text-sm font-medium border rounded-md cursor-pointer font-inherit transition-all duration-fast outline-none"
                  style={{
                    borderColor: active ? 'var(--color-primary)' : 'var(--color-border)',
                    color: active ? 'var(--color-primary)' : 'var(--color-muted)',
                    background: active
                      ? hovered ? 'var(--color-primary-alpha-12)' : 'var(--color-primary-alpha-08)'
                      : 'var(--color-surface)',
                  }}
                  onClick={() => toggleMod(m)}
                  onMouseEnter={() => setHoveredMod(m)}
                  onMouseLeave={() => setHoveredMod(null)}
                >
                  {m}
                </button>
              )
            })}
          </div>
          <div className="text-base font-medium text-foreground mt-3 mb-1">按键</div>
          <input
            className="w-20 h-9 mt-1 text-center text-lg font-semibold border border-border rounded-md bg-surface text-foreground outline-none font-inherit"
            value={key}
            onChange={handleKeyInput}
            placeholder="输入..."
            maxLength={1}
          />
          <div className="mt-2.5 text-sm text-muted font-mono">{displayKey || '未设置'}</div>
          {hotkeyError && <div className="mt-2 text-xs text-destructive leading-[1.4]">{hotkeyError}</div>}
        </div>

        {/* Retain Days */}
        <div className="px-4 py-5 border-b border-border">
          <div className="flex justify-between items-center">
            <div>
              <div className="text-base font-medium text-foreground mb-0.5">保留时长</div>
              <div className="text-xs text-muted mt-0.5">超过以下天数的记录自动删除</div>
            </div>
            <div className="flex items-center gap-2.5">
              <input
                type="range" min="1" max="90"
                value={local.retain_days}
                onChange={(e) => setLocal({ ...local, retain_days: parseInt(e.target.value) })}
                onMouseUp={() => handleSave({ retain_days: local.retain_days })}
                className="w-[120px] cursor-pointer"
                style={{ accentColor: 'var(--color-primary)' }}
              />
              <span className="text-sm font-medium text-foreground min-w-[56px]">{local.retain_days} 天</span>
            </div>
          </div>
          <div className="mt-2.5 text-xs text-muted flex gap-1.5 items-center">
            <span>{stats.count.toLocaleString()} 条记录</span>
            <span className="opacity-40">·</span>
            <span>{formatBytes(stats.total_bytes)}</span>
          </div>
          <button
            className="mt-2.5 px-3.5 py-2 flex items-center gap-1.5 text-xs font-medium border rounded-md cursor-pointer font-inherit transition-all duration-fast disabled:opacity-60"
            style={{
              borderColor: 'rgba(239,68,68,0.3)',
              background: 'rgba(239,68,68,0.06)',
              color: '#DC2626',
              opacity: clearing ? 0.6 : 1,
            }}
            onClick={() => setShowClearModal(true)}
            disabled={clearing || stats.count === 0}
          >
            <Trash2 size={14} />
            {clearing ? '清空中...' : '清空全部历史'}
          </button>
        </div>

        {/* Auto Clear Search */}
        <div className="px-4 py-5 border-b border-border">
          <div className="flex justify-between items-center">
            <div>
              <div className="text-base font-medium text-foreground mb-0.5">自动清理搜索</div>
              <div className="text-xs text-muted mt-0.5">窗口隐藏超过设定时间后，再次显示时自动清空搜索条件</div>
            </div>
            <ToggleSwitch
              checked={local.auto_clear_search}
              onChange={() => handleSave({ auto_clear_search: !local.auto_clear_search })}
              label="自动清理搜索"
            />
          </div>
          {local.auto_clear_search && (
            <div className="mt-3 flex items-center gap-2.5">
              <input
                type="range" min="0" max="300" step="5"
                value={local.auto_clear_seconds}
                onChange={(e) => setLocal({ ...local, auto_clear_seconds: parseInt(e.target.value) })}
                onMouseUp={() => handleSave({ auto_clear_seconds: local.auto_clear_seconds })}
                className="w-[120px] cursor-pointer"
                style={{ accentColor: 'var(--color-primary)' }}
              />
              <span className="text-sm font-medium text-foreground min-w-[56px]">
                {local.auto_clear_seconds === 0 ? '每次清理' : `${local.auto_clear_seconds} 秒`}
              </span>
            </div>
          )}
        </div>

        {/* Default Action */}
        <div className="px-4 py-5 border-b border-border">
          <div className="text-base font-medium text-foreground mb-0.5">默认操作</div>
          <div className="text-xs text-muted mt-0.5">点击或按 Ctrl+数字 时的行为</div>
          <div className="flex flex-col gap-1.5 mt-3">
            {[
              { value: 'copy', label: '复制到剪贴板', desc: '将内容复制到系统剪贴板' },
              { value: 'paste', label: '自动粘贴', desc: '复制 + 模拟 Ctrl+V 粘贴' },
            ].map(({ value, label, desc }) => {
              const active = local.default_action === value
              const hovered = hoveredRadio === value
              return (
                <label
                  key={value}
                  className="flex items-center gap-2.5 px-3.5 py-2.5 rounded-md border cursor-pointer text-sm transition-all duration-fast"
                  style={{
                    borderColor: active ? 'var(--color-primary)' : 'var(--color-border)',
                    background: active ? 'var(--color-primary-alpha-08)' : hovered ? 'var(--color-surface-hover)' : 'transparent',
                  }}
                  onClick={() => handleSave({ default_action: value })}
                  onMouseEnter={() => setHoveredRadio(value)}
                  onMouseLeave={() => setHoveredRadio(null)}
                >
                  <input type="radio" name="action" value={value} checked={active} onChange={() => {}} className="accent-primary" />
                  <div>
                    <div style={{ fontWeight: active ? 600 : 500, color: 'var(--color-foreground)' }}>{label}</div>
                    <div className="text-xs text-muted">{desc}</div>
                  </div>
                </label>
              )
            })}
          </div>
        </div>

        {/* Auto-hide after copy */}
        <div className="px-4 py-5 border-b border-border">
          <div className="flex justify-between items-center">
            <div>
              <div className="text-base font-medium text-foreground mb-0.5">复制后自动隐藏</div>
              <div className="text-xs text-muted mt-0.5">复制到剪贴板后自动隐藏 jPaste 窗口</div>
            </div>
            <ToggleSwitch
              checked={local.auto_hide_after_copy}
              onChange={() => handleSave({ auto_hide_after_copy: !local.auto_hide_after_copy })}
              label="复制后自动隐藏"
            />
          </div>
        </div>

        {/* Theme Selector */}
        <div className="px-4 py-5 border-b border-border">
          <div className="text-base font-medium text-foreground mb-0.5">主题</div>
          <div className="text-xs text-muted mt-0.5">切换整体视觉风格（保存后刷新）</div>
          <div className="flex flex-col gap-2 mt-3">
            {THEMES.map(t => {
              const active = (local.theme || 'a') === t.id
              return (
                <button
                  key={t.id}
                  className="flex items-center gap-3 px-3.5 py-3 rounded-md border cursor-pointer font-inherit text-left transition-all duration-fast"
                  style={{
                    borderColor: active ? 'var(--color-primary)' : 'var(--color-border)',
                    background: active ? 'var(--color-primary-alpha-06)' : 'var(--color-surface)',
                  }}
                  onClick={() => handleSave({ theme: t.id })}
                >
                  <div className="flex gap-1 flex-shrink-0">
                    {t.colors.map((c, i) => (
                      <div
                        key={i}
                        className="w-5 h-5 rounded-full"
                        style={{
                          background: c,
                          border: i === 2 && t.id === 'c' ? '1px solid rgba(255,255,255,0.15)' : '2px solid var(--color-border)',
                        }}
                      />
                    ))}
                  </div>
                  <div className="flex-1">
                    <div className="text-sm" style={{ fontWeight: active ? 600 : 500, color: 'var(--color-foreground)', marginBottom: '2px' }}>
                      {t.label}
                    </div>
                    <div className="text-[11px] text-muted">{t.desc}</div>
                  </div>
                  <div
                    className="w-4 h-4 rounded-full border-2 border-border flex items-center justify-center flex-shrink-0"
                    style={{ borderColor: active ? 'var(--color-primary)' : undefined }}
                  >
                    {active && (
                      <div className="w-2 h-2 rounded-full" style={{ background: 'var(--color-primary)' }} />
                    )}
                  </div>
                </button>
              )
            })}
          </div>
        </div>

        {/* Notify Toggle + Opacity Slider */}
        <div className="px-4 py-5 border-b border-border">
          <div className="flex justify-between items-center">
            <div>
              <div className="text-base font-medium text-foreground mb-0.5">剪贴板通知</div>
              <div className="text-xs text-muted mt-0.5">捕获到新剪贴板内容时显示通知</div>
            </div>
            <ToggleSwitch
              checked={local.notify_enabled}
              onChange={() => {
                const newVal = !local.notify_enabled
                handleSave({ notify_enabled: newVal })
                notifyToggled.current = true
                if (newVal) {
                  Events.Emit('toast-show-preview', {
                    title: 'jPaste',
                    message: '通知示例',
                    opacity: local.notify_opacity ?? 100,
                  })
                } else {
                  Events.Emit('toast-hide-preview')
                }
              }}
              label="切换剪贴板通知"
            />
          </div>

          {local.notify_enabled && (
            <div className="mt-4">
              <div className="flex items-center gap-2.5">
                <span className="text-xs text-muted whitespace-nowrap">透明度</span>
                <input
                  type="range" min="10" max="100"
                  value={local.notify_opacity}
                  onChange={(e) => {
                    const val = parseInt(e.target.value)
                    setLocal({ ...local, notify_opacity: val })
                    Events.Emit('toast-show-preview', {
                      title: 'jPaste',
                      message: '通知示例',
                      opacity: val,
                    })
                  }}
                  onMouseUp={() => handleSave({ notify_opacity: local.notify_opacity })}
                  className="flex-1 cursor-pointer"
                  style={{ accentColor: 'var(--color-primary)' }}
                />
                <span className="text-sm font-medium text-foreground min-w-[40px] text-right">{local.notify_opacity}%</span>
              </div>
            </div>
          )}
        </div>

        {/* Other Toggle Settings */}
        {[
          { key: 'auto_start', label: '开机自启', desc: '登录时自动启动 jPaste' },
          { key: 'start_minimized', label: '启动时最小化', desc: '启动后最小化到系统托盘（不弹出窗口）' },
        ].map(({ key: k, label, desc }) => (
          <div className="px-4 py-5 border-b border-border" key={k}>
            <div className="flex justify-between items-center">
              <div>
                <div className="text-base font-medium text-foreground mb-0.5">{label}</div>
                <div className="text-xs text-muted mt-0.5">{desc}</div>
              </div>
              <ToggleSwitch
                checked={local[k]}
                onChange={() => handleSave({ [k]: !local[k] })}
                label={`切换${label}`}
              />
            </div>
          </div>
        ))}

        {/* Action Modules */}
        <div className="px-4 py-5">
          <div className="text-base font-medium text-foreground mb-0.5">操作模块</div>
          <div className="text-xs text-muted mt-0.5">启用/禁用并调整按钮显示顺序</div>
          <div className="flex flex-col gap-1 mt-3">
            {sortedActions.map((action, idx) => {
              const isExpanded = expandedId === action.id
              const IconComp = ICON_MAP[action.icon]
              const disabled = !action.config.enabled
              return (
                <div key={action.id} className="flex flex-col rounded-md border border-border">
                  {/* Header row */}
                  <div className="flex justify-between items-center px-3 min-h-[44px]">
                    <div className="flex items-center gap-1.5 min-w-0">
                      <ToggleSwitch
                        checked={action.config.enabled}
                        onChange={() => toggleAction(action)}
                        label={`切换${action.label}`}
                      />
                      <button
                        className="flex items-center gap-1 border-none bg-transparent cursor-pointer font-inherit text-left min-w-0 transition-all duration-fast hover:opacity-80"
                        onClick={() => setExpandedId(isExpanded ? null : action.id)}
                      >
                        {IconComp && (
                          <IconComp
                            size={15}
                            style={{
                              color: disabled ? 'var(--color-muted)' : 'var(--color-primary)',
                              opacity: disabled ? 0.4 : 1,
                              flexShrink: 0,
                            }}
                          />
                        )}
                        <span
                          className="text-sm font-medium truncate"
                          style={{ opacity: disabled ? 0.4 : 1, color: 'var(--color-foreground)' }}
                        >
                          {action.label}
                        </span>
                        {isExpanded ? (
                          <ChevronDown size={13} style={{ color: 'var(--color-muted)', flexShrink: 0 }} />
                        ) : (
                          <ChevronRight size={13} style={{ color: 'var(--color-muted)', flexShrink: 0 }} />
                        )}
                      </button>
                    </div>
                    <div className="flex items-center gap-0.5 flex-shrink-0">
                      <button className="w-7 h-7 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded transition-all duration-fast hover:bg-surface-hover" onClick={() => moveAction(idx, 'up')} disabled={idx === 0} title="上移">
                        <ChevronUp size={14} style={{ opacity: idx === 0 ? 0.3 : 1 }} />
                      </button>
                      <button className="w-7 h-7 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded transition-all duration-fast hover:bg-surface-hover" onClick={() => moveAction(idx, 'down')} disabled={idx === sortedActions.length - 1} title="下移">
                        <ChevronDown size={14} style={{ opacity: idx === sortedActions.length - 1 ? 0.3 : 1 }} />
                      </button>
                    </div>
                  </div>
                  {/* Expanded detail */}
                  {isExpanded && (
                    <div className="px-3 pb-3 border-t border-border">
                      <div className="text-xs text-muted mt-2 leading-[1.5]">
                        <span className="font-medium text-foreground">触发：</span>{action.trigger}
                      </div>
                      <div className="text-xs text-muted mt-1 leading-[1.5]">
                        <span className="font-medium text-foreground">功能：</span>{action.desc}
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      </div>

      {/* Clear All Confirmation Modal */}
      <Modal open={showClearModal} onClose={() => setShowClearModal(false)} title="清空剪贴板历史" size="sm">
        <p className="m-0 mb-5 text-sm text-muted leading-[1.5]">
          共有 <strong>{stats.count.toLocaleString()}</strong> 条记录。选择清空方式：
        </p>
        <div className="flex flex-col gap-2">
          <button onClick={() => doClearAll(false)}
            className="px-4 py-2.5 rounded-md border border-border cursor-pointer text-sm font-inherit text-left transition-all duration-fast"
            style={{ background: hoveredClearBtn === 'all' ? 'var(--color-surface-hover)' : 'var(--color-surface)', color: 'var(--color-foreground)' }}
            onMouseEnter={() => setHoveredClearBtn('all')}
            onMouseLeave={() => setHoveredClearBtn(null)}
          >
            <div className="font-semibold">全部删除</div>
            <div className="text-xs text-muted mt-0.5">删除所有记录（包括收藏），不可撤销</div>
          </button>
          <button onClick={() => doClearAll(true)}
            className="px-4 py-2.5 rounded-md border border-border cursor-pointer text-sm font-inherit text-left transition-all duration-fast"
            style={{ background: hoveredClearBtn === 'fav' ? 'var(--color-surface-hover)' : 'var(--color-surface)', color: 'var(--color-foreground)' }}
            onMouseEnter={() => setHoveredClearBtn('fav')}
            onMouseLeave={() => setHoveredClearBtn(null)}
          >
            <div className="font-semibold">保留收藏</div>
            <div className="text-xs text-muted mt-0.5">只删除未收藏的记录，收藏内容保留</div>
          </button>
          <button onClick={() => setShowClearModal(false)}
            className="py-2 rounded-md border-none bg-transparent text-muted cursor-pointer text-sm font-inherit mt-1">取消</button>
        </div>
      </Modal>
    </div>
  )
}
