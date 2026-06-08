import { useState, useMemo } from 'react'
import { formBody, textarea, output, errorMsg } from './actionStyles'

function MathModal({ content, onClose }) {
  const [expr, setExpr] = useState(content.trim())
  const { error, result } = useMemo(() => safeEval(expr.replace(/\s/g, '')), [expr])

  return (
    <div style={formBody}>
      <textarea
        style={{ ...textarea, fontSize: 'var(--font-size-lg)', minHeight: '80px' }}
        value={expr}
        onChange={e => setExpr(e.target.value)}
        placeholder="输入算式..."
        autoFocus
        spellCheck={false}
      />
      <div style={{ ...output, minHeight: '48px', fontSize: 'var(--font-size-base)' }}>
        {error ? (
          <span style={errorMsg}>{error}</span>
        ) : result !== null ? (
          <span style={styles.resultValue}>= {result}</span>
        ) : (
          <span style={styles.hint}>输入算式后自动计算</span>
        )}
      </div>
    </div>
  )
}

function safeEval(expr) {
  try {
    const sanitized = expr.trim()
    if (!sanitized) return { error: null, result: null }
    if (/[^0-9+\-*/%.()]/.test(sanitized)) {
      return { error: '表达式包含不支持的字符', result: null }
    }
    const fn = new Function(`return (${sanitized})`)
    const result = fn()
    if (typeof result !== 'number' || !isFinite(result)) {
      return { error: '计算结果无效', result: null }
    }
    return { error: null, result }
  } catch (e) {
    return { error: e.message, result: null }
  }
}

const styles = {
  resultValue: { fontSize: 'var(--font-size-xl)', fontWeight: 700, color: 'var(--color-primary)' },
  hint: { fontSize: 'var(--font-size-sm)', color: 'var(--color-muted)' },
}

export default {
  id: 'math',
  label: '计算',
  icon: 'Calculator',
  priority: 50,
  trigger: '纯数字和运算符的表达式',
  desc: '在弹窗中实时计算并显示结果，可编辑表达式重新计算',
  detect(content) {
    const s = content.trim()
    return /^[\d+\-*/%.()\s]+$/.test(s) && /[+\-*/%]/.test(s) && /\d/.test(s) && s.length <= 200
  },
  Component: MathModal,
}
