import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, ChevronUp, ChevronDown, Globe, CheckCircle, XCircle, Trash2 } from 'lucide-react'
import { useClipboard } from '../context/ClipboardContext'
import { Service as SyncService } from '../../bindings/jpaste/internal/sync'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
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
  const { settings, saveSettings, wdConfig, refreshWdConfig, clearAll } = useClipboard()
  const navigate = useNavigate()
  const [local, setLocal] = useState({ ...settings })
  const [saved, setSaved] = useState(false)

  // WebDAV config state.
  const [wdUrl, setWdUrl] = useState(wdConfig.url)
  const [wdUser, setWdUser] = useState(wdConfig.username)
  const [wdPass, setWdPass] = useState(wdConfig.password)
  const [wdPassDirty, setWdPassDirty] = useState(false)
  const [wdEnabled, setWdEnabled] = useState(wdConfig.enabled)
  const [wdTesting, setWdTesting] = useState(false)
  const [wdSaving, setWdSaving] = useState(false)
  const [wdTestResult, setWdTestResult] = useState(null)

  // Stats state.
  const [stats, setStats] = useState({ count: 0, total_bytes: 0 })
  const [clearing, setClearing] = useState(false)

  useEffect(() => {
    setWdUrl(wdConfig.url)
    setWdUser(wdConfig.username)
    setWdPass(wdConfig.password)
    setWdEnabled(wdConfig.enabled)
    setWdPassDirty(false)
  }, [wdConfig])

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

  const handleSaveWd = async (toggleEnabled) => {
    const enabled = toggleEnabled !== undefined ? toggleEnabled : wdEnabled
    const cfg = { url: wdUrl, username: wdUser, password: wdPassDirty ? wdPass : '••••••••', enabled }
    if (toggleEnabled !== undefined) setWdEnabled(toggleEnabled)
    try {
      await SyncService.SaveConfig(cfg)
      setWdPassDirty(false)
      setSaved(true)
      setWdSaving(true)
      await refreshWdConfig()
      setTimeout(() => { setSaved(false); setWdSaving(false) }, 1500)
    } catch (err) {
      console.error('[wd] SaveConfig error:', err)
      setWdTestResult({ ok: false, msg: '保存失败: ' + (err?.toString() || '未知错误') })
      if (toggleEnabled !== undefined) setWdEnabled(!toggleEnabled)
    }
  }

  const handleTestWd = async () => {
    setWdTesting(true)
    setWdTestResult(null)
    try {
      await SyncService.TestConnection({ url: wdUrl, username: wdUser, password: wdPass })
      setWdTestResult({ ok: true, msg: '连接成功' })
    } catch (err) {
      setWdTestResult({ ok: false, msg: err.toString() })
    } finally {
      setWdTesting(false)
    }
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

  const handleClearAll = async () => {
    if (!window.confirm('确定要清空全部剪贴板历史吗？此操作不可撤销。')) return
    setClearing(true)
    try {
      await clearAll()
      setStats({ count: 0, total_bytes: 0 })
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
            onClick={handleClearAll}
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

        {/* WebDAV Sync */}
        <div style={styles.group}>
          <div style={styles.row}>
            <div>
              <div style={styles.label}>WebDAV 同步</div>
              <div style={styles.desc}>同步剪贴板历史和设置到坚果云</div>
            </div>
            <Globe size={18} style={{ color: 'var(--color-muted)', flexShrink: 0 }} />
          </div>
          <div style={{ marginTop: '12px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <input style={styles.wdInput} type="url" placeholder="WebDAV 地址 (例: https://dav.jianguoyun.com/dav/)" value={wdUrl} onChange={e => setWdUrl(e.target.value)} />
            <input style={styles.wdInput} type="text" placeholder="账户名" value={wdUser} onChange={e => setWdUser(e.target.value)} />
            <input style={styles.wdInput} type="password" placeholder="应用密码（非登录密码）" value={wdPass} onChange={e => { setWdPass(e.target.value); setWdPassDirty(true) }} />
          </div>
          {wdTestResult && (
            <div style={{ marginTop: '8px', padding: '8px 12px', borderRadius: 'var(--radius-md)', fontSize: 'var(--font-size-xs)', display: 'flex', alignItems: 'center', gap: '6px', background: wdTestResult.ok ? 'rgba(16,185,129,0.08)' : 'rgba(239,68,68,0.08)', color: wdTestResult.ok ? '#059669' : '#DC2626' }}>
              {wdTestResult.ok ? <CheckCircle size={14} /> : <XCircle size={14} />}
              {wdTestResult.msg}
            </div>
          )}
          <div style={{ ...styles.row, paddingTop: '10px' }}>
            <div>
              <div style={styles.label}>启用同步</div>
              <div style={styles.desc}>{wdEnabled ? '自动同步剪贴板记录和配置' : '暂停同步'}</div>
            </div>
            <ToggleSwitch checked={wdEnabled} onChange={() => handleSaveWd(!wdEnabled)} label="切换同步" />
          </div>
          <div style={{ marginTop: '8px', display: 'flex', gap: '8px' }}>
            <button style={styles.wdBtn} onClick={handleTestWd} disabled={wdTesting || !wdUrl || !wdUser || !wdPass}>
              {wdTesting ? '测试中…' : '测试连接'}
            </button>
            <button style={{ ...styles.wdBtn, ...styles.wdBtnPrimary, ...(wdSaving ? { opacity: 0.7 } : {}) }} onClick={() => handleSaveWd()} disabled={!wdUrl || !wdUser}>
              {wdSaving ? '已保存 ✓' : '保存'}
            </button>
          </div>
        </div>

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
    </div>
  )
}
