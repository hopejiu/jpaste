import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, ChevronUp, ChevronDown, Trash2 } from 'lucide-react'
import { useApp } from '../context/AppContext'
import { useClipboard } from '../context/ClipboardContext'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { log } from '../logger'
import { getAll } from '../actions'
import { formatBytes } from '../utils/format'
import ToggleSwitch from '../components/ToggleSwitch'
import { styles } from './SettingsPage.styles'

const MODS = ['Ctrl', 'Alt', 'Shift', 'Win']

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

  useEffect(() => {
    HistoryService.GetStats()
      .then(s => { if (s) setStats(s) })
      .catch(() => {})
  }, [])

  // Hotkey UI state.
  const parsed = parseHotkey(local.hotkey)
  const [mods, setMods] = useState(parsed.mods)
  const [key, setKey] = useState(parsed.key)

  useEffect(() => {
    setLocal({ ...settings })
    const p = parseHotkey(settings.hotkey)
    setMods(p.mods)
    setKey(p.key)
  }, [settings])

  const handleSave = async (updates) => {
    const updated = { ...local, ...updates }
    setLocal(updated)
    await saveSettings(updated)
    setSaved(true)
    setTimeout(() => setSaved(false), 1500)
  }

  const updateHotkey = useCallback((newMods, newKey) => {
    setMods(newMods)
    setKey(newKey)
    const sorted = [...newMods].sort((a, b) => MODS.indexOf(a) - MODS.indexOf(b))
    const hk = newKey ? [...sorted, newKey].join('+') : sorted.join('+')
    handleSave({ hotkey: hk })
  }, [])

  const toggleMod = (m) => {
    const next = mods.includes(m) ? mods.filter(x => x !== m) : [...mods, m]
    updateHotkey(next, key)
  }

  const handleKeyInput = (e) => {
    const val = e.target.value.replace(/[^a-zA-Z0-9]/g, '').slice(-1).toUpperCase()
    if (val) updateHotkey(mods, val)
  }

  // Pre-compute sorted action modules (replaces JSX IIFE).
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
      // Timeout safety: avoid UI freeze if backend hangs.
      await Promise.race([
        clearAll(keepFavorites),
        new Promise((_, reject) => setTimeout(() => reject(new Error('clearAll timeout')), 10000)),
      ])
      // Fetch real stats after deletion (some entries may remain if keepFavorites).
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

  return (
    <div style={styles.container} tabIndex={0}>
      <div style={styles.header}>
        <button style={styles.backBtn} onClick={() => navigate('/')}>
          <ArrowLeft size={20} />
        </button>
        <h2 style={styles.title}>设置</h2>
        {saved && <span style={styles.savedBadge}>已保存</span>}
      </div>

      <div style={styles.content}>
        {/* Hotkey */}
        <div style={styles.group}>
          <div style={styles.label}>全局快捷键</div>
          <div style={styles.desc}>显示/隐藏 jPaste 窗口</div>
          <div style={styles.modRow}>
            {MODS.map(m => (
              <button
                key={m}
                style={{ ...styles.modChip, ...(mods.includes(m) ? styles.modChipActive : {}) }}
                onClick={() => toggleMod(m)}
              >
                {m}
              </button>
            ))}
          </div>
          <div style={{ ...styles.label, marginTop: '12px' }}>按键</div>
          <input
            style={styles.keyInput}
            value={key}
            onChange={handleKeyInput}
            placeholder="输入字母..."
            maxLength={1}
          />
          <div style={styles.hotkeyPreview}>{displayKey || '未设置'}</div>
        </div>

        {/* Retain Days */}
        <div style={styles.group}>
          <div style={styles.row}>
            <div>
              <div style={styles.label}>保留时长</div>
              <div style={styles.desc}>超过以下天数的记录自动删除</div>
            </div>
            <div style={styles.retainControl}>
              <input
                type="range" min="1" max="90"
                value={local.retain_days}
                onChange={(e) => setLocal({ ...local, retain_days: parseInt(e.target.value) })}
                onMouseUp={() => handleSave({ retain_days: local.retain_days })}
                style={styles.slider}
              />
              <span style={styles.retainValue}>{local.retain_days} 天</span>
            </div>
          </div>
          <div style={styles.statsRow}>
            <span>{stats.count.toLocaleString()} 条记录</span>
            <span style={styles.statsDot}>·</span>
            <span>{formatBytes(stats.total_bytes)}</span>
          </div>
          <button
            style={{ ...styles.clearAllBtn, ...(clearing ? { opacity: 0.6 } : {}) }}
            onClick={() => setShowClearModal(true)}
            disabled={clearing || stats.count === 0}
          >
            <Trash2 size={14} />
            {clearing ? '清空中...' : '清空全部历史'}
          </button>
        </div>

        {/* Default Action */}
        <div style={styles.group}>
          <div style={styles.label}>默认操作</div>
          <div style={styles.desc}>点击或按 Ctrl+数字 时的行为</div>
          <div style={styles.radioGroup}>
            <label
              style={{ ...styles.radioLabel, ...(local.default_action === 'copy' ? styles.radioActive : {}) }}
              onClick={() => handleSave({ default_action: 'copy' })}
            >
              <input type="radio" name="action" value="copy" checked={local.default_action === 'copy'} onChange={() => {}} style={styles.radio} />
              复制到剪贴板
            </label>
            <label
              style={{ ...styles.radioLabel, ...(local.default_action === 'paste' ? styles.radioActive : {}) }}
              onClick={() => handleSave({ default_action: 'paste' })}
            >
              <input type="radio" name="action" value="paste" checked={local.default_action === 'paste'} onChange={() => {}} style={styles.radio} />
              自动粘贴（复制 + Ctrl+V）
            </label>
          </div>
        </div>

        {/* Toggle Settings */}
        {[
          { key: 'notify_enabled', label: '剪贴板通知', desc: '捕获到新剪贴板内容时显示通知' },
          { key: 'auto_start', label: '开机自启', desc: '登录时自动启动 jPaste' },
          { key: 'start_minimized', label: '启动时最小化', desc: '启动后最小化到系统托盘（不弹出窗口）' },
        ].map(({ key: k, label, desc }) => (
          <div style={styles.group} key={k}>
            <div style={styles.row}>
              <div>
                <div style={styles.label}>{label}</div>
                <div style={styles.desc}>{desc}</div>
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
        <div style={styles.group}>
          <div style={styles.label}>操作模块</div>
          <div style={styles.desc}>启用/禁用并调整按钮显示顺序</div>
          <div style={styles.actionList}>
            {sortedActions.map((action, idx) => (
              <div key={action.id} style={styles.actionItem}>
                <div style={styles.actionItemLeft}>
                  <ToggleSwitch
                    checked={action.config.enabled}
                    onChange={() => toggleAction(action)}
                    label={`切换${action.label}`}
                  />
                  <span style={{ ...styles.actionName, opacity: action.config.enabled ? 1 : 0.4 }}>
                    {action.label}
                  </span>
                </div>
                <div style={styles.actionItemRight}>
                  <button style={styles.priorityBtn} onClick={() => moveAction(idx, 'up')} disabled={idx === 0} title="上移">
                    <ChevronUp size={14} style={{ opacity: idx === 0 ? 0.3 : 1 }} />
                  </button>
                  <button style={styles.priorityBtn} onClick={() => moveAction(idx, 'down')} disabled={idx === sortedActions.length - 1} title="下移">
                    <ChevronDown size={14} style={{ opacity: idx === sortedActions.length - 1 ? 0.3 : 1 }} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Clear All Confirmation Modal */}
      {showClearModal && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          zIndex: 3000,
        }} onClick={() => setShowClearModal(false)}>
          <div style={{
            background: 'var(--color-surface)', borderRadius: 'var(--radius-lg)',
            padding: '24px', width: '340px', maxWidth: '90vw',
            boxShadow: '0 8px 32px rgba(0,0,0,0.2)',
          }} onClick={e => e.stopPropagation()}>
            <h3 style={{ margin: '0 0 8px', fontSize: 'var(--font-size-lg)', fontWeight: 600 }}>
              清空剪贴板历史
            </h3>
            <p style={{ margin: '0 0 20px', fontSize: 'var(--font-size-sm)', color: 'var(--color-muted)', lineHeight: 1.5 }}>
              共有 <strong>{stats.count.toLocaleString()}</strong> 条记录。选择清空方式：
            </p>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              <button
                onClick={() => doClearAll(false)}
                style={{
                  padding: '10px 16px', borderRadius: 'var(--radius-md)',
                  border: '1px solid var(--color-border)', background: 'var(--color-surface)',
                  color: 'var(--color-foreground)', cursor: 'pointer',
                  fontSize: 'var(--font-size-sm)', fontFamily: 'inherit',
                  textAlign: 'left', transition: 'background var(--transition-fast)',
                }}
                onMouseEnter={e => e.currentTarget.style.background = 'var(--color-surface-hover)'}
                onMouseLeave={e => e.currentTarget.style.background = 'var(--color-surface)'}
              >
                <div style={{ fontWeight: 600 }}>全部删除</div>
                <div style={{ fontSize: '12px', color: 'var(--color-muted)', marginTop: '2px' }}>
                  删除所有记录（包括收藏），不可撤销
                </div>
              </button>
              <button
                onClick={() => doClearAll(true)}
                style={{
                  padding: '10px 16px', borderRadius: 'var(--radius-md)',
                  border: '1px solid var(--color-border)', background: 'var(--color-surface)',
                  color: 'var(--color-foreground)', cursor: 'pointer',
                  fontSize: 'var(--font-size-sm)', fontFamily: 'inherit',
                  textAlign: 'left', transition: 'background var(--transition-fast)',
                }}
                onMouseEnter={e => e.currentTarget.style.background = 'var(--color-surface-hover)'}
                onMouseLeave={e => e.currentTarget.style.background = 'var(--color-surface)'}
              >
                <div style={{ fontWeight: 600 }}>保留收藏</div>
                <div style={{ fontSize: '12px', color: 'var(--color-muted)', marginTop: '2px' }}>
                  只删除未收藏的记录，收藏内容保留
                </div>
              </button>
              <button
                onClick={() => setShowClearModal(false)}
                style={{
                  padding: '8px', borderRadius: 'var(--radius-md)',
                  border: 'none', background: 'transparent',
                  color: 'var(--color-muted)', cursor: 'pointer',
                  fontSize: 'var(--font-size-sm)', fontFamily: 'inherit',
                  marginTop: '4px',
                }}
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
