import { useState, useCallback, useMemo } from 'react'
import { Minus, Copy, Check } from 'lucide-react'
import JsonView from '@uiw/react-json-view'

function formatJSON(text) {
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch { return null }
}

function compressJSON(text) {
  try {
    return JSON.stringify(JSON.parse(text))
  } catch { return null }
}

function JsonModal({ content, onClose }) {
  const [raw, setRaw] = useState(() => formatJSON(content) || content)
  const [copied, setCopied] = useState(false)

  // Parse for tree view.
  let parsed, parseError
  try {
    parsed = JSON.parse(raw)
  } catch (e) {
    parseError = e.message
  }

  const handleInput = useCallback((e) => {
    setRaw(e.target.value)
  }, [])

  const handleFormat = useCallback(() => {
    const fmt = formatJSON(raw)
    if (fmt) setRaw(fmt)
  }, [raw])

  const handleCompress = useCallback(() => {
    const min = compressJSON(raw)
    if (min) setRaw(min)
  }, [raw])

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(raw)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch (e) {
      // Fallback: use textarea selection
      const ta = document.getElementById('json-editor-textarea')
      if (ta) {
        ta.select()
        document.execCommand('copy')
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      }
    }
  }, [raw])

  return (
    <div style={styles.container}>
      {/* Toolbar */}
      <div style={styles.toolbar}>
        <div style={styles.toolbarLeft}>
          <button style={styles.toolBtn} onClick={handleFormat} title="格式化 (美化)">
            {'{}'} 格式化
          </button>
          <button style={styles.toolBtn} onClick={handleCompress} title="压缩为一行">
            <Minus size={14} /> 压缩
          </button>
        </div>
        <div style={styles.toolbarRight}>
          <button style={{ ...styles.toolBtn, ...(copied ? styles.toolBtnActive : {}) }} onClick={handleCopy}>
            {copied ? <Check size={14} /> : <Copy size={14} />}
            {copied ? ' 已复制' : ' 复制'}
          </button>
        </div>
      </div>

      {/* Split pane */}
      <div style={styles.split}>
        {/* Left: editable raw text */}
        <div style={styles.pane}>
          <div style={styles.paneLabel}>编辑</div>
          <textarea
            id="json-editor-textarea"
            style={styles.textarea}
            value={raw}
            onChange={handleInput}
            spellCheck={false}
            placeholder="在此编辑 JSON..."
          />
          {parseError && (
            <div style={styles.errorBadge}>JSON 错误: {parseError}</div>
          )}
        </div>

        {/* Right: tree viewer */}
        <div style={styles.pane}>
          <div style={styles.paneLabel}>预览</div>
          <div style={styles.viewer}>
            {parsed ? (
              <JsonView
                value={parsed}
                collapsed={1}
                displayDataTypes={false}
                enableClipboard={false}
                indentWidth={18}
                style={{
                  fontFamily: 'monospace',
                  fontSize: '13px',
                  lineHeight: 1.65,
                }}
              />
            ) : (
              <div style={styles.placeholder}>输入有效 JSON 后在右侧预览</div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default {
  id: 'json',
  label: '查看 JSON',
  icon: 'Braces',
  priority: 40,
  detect(content) {
    const s = content.trim()
    if (!s.startsWith('{') && !s.startsWith('[')) return false
    try {
      JSON.parse(s)
      return true
    } catch {
      return false
    }
  },
  Component: JsonModal,
}

const styles = {
  container: { display: 'flex', flexDirection: 'column', height: '100%', margin: '-8px' },
  toolbar: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '4px 0', gap: '8px', flexShrink: 0,
  },
  toolbarLeft: { display: 'flex', gap: '6px' },
  toolbarRight: { display: 'flex', gap: '6px' },
  toolBtn: {
    display: 'inline-flex', alignItems: 'center', gap: '4px',
    padding: '5px 12px', fontSize: 'var(--font-size-xs)', fontWeight: 500,
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-sm)',
    background: 'var(--color-surface)', color: 'var(--color-foreground)',
    cursor: 'pointer', fontFamily: 'inherit',
    transition: 'all var(--transition-fast)',
  },
  toolBtnActive: {
    borderColor: '#4ADE80', color: '#4ADE80', background: 'rgba(74,222,128,0.1)',
  },
  split: {
    display: 'flex', gap: '12px', flex: 1, minHeight: 0,
    marginTop: '8px',
  },
  pane: {
    flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column',
    position: 'relative',
  },
  paneLabel: {
    fontSize: 'var(--font-size-xs)', fontWeight: 500,
    color: 'var(--color-muted)', marginBottom: '6px', flexShrink: 0,
  },
  textarea: {
    flex: 1, minHeight: 0, width: '100%', padding: '12px',
    fontSize: '13px', fontFamily: 'monospace', lineHeight: 1.6,
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    background: 'var(--color-surface)', color: 'var(--color-foreground)',
    resize: 'none', outline: 'none', tabSize: 2,
  },
  viewer: {
    flex: 1, overflow: 'auto', minHeight: 0,
    border: '1px solid var(--color-border)', borderRadius: 'var(--radius-md)',
    background: 'var(--color-surface)', padding: '10px 12px',
  },
  placeholder: {
    color: 'var(--color-muted)', fontSize: 'var(--font-size-sm)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    height: '100%', minHeight: '80px',
  },
  errorBadge: {
    position: 'absolute', bottom: '4px', left: '8px', right: '8px',
    padding: '4px 10px', fontSize: '11px',
    color: 'var(--color-destructive)', background: 'rgba(239,68,68,0.08)',
    borderRadius: 'var(--radius-sm)', fontFamily: 'monospace',
  },
}
