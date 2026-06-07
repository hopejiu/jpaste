import { useState, useMemo } from 'react'
import { formBody, label as lbl, textarea, output, errorMsg, resultText } from './actionStyles'

function decodeURL(input) {
  if (!input.trim()) return { decoded: '', error: null }
  try {
    const decoded = decodeURIComponent(input.trim())
    return { decoded, error: null }
  } catch (e) {
    return { decoded: '', error: '无效的 URL 编码' }
  }
}

function hasPercentEncoding(s) {
  return /%[0-9A-Fa-f]{2}/.test(s)
}

function URLDecodeModal({ content, onClose }) {
  const [input, setInput] = useState(content.trim())
  const { decoded, error } = useMemo(() => decodeURL(input), [input])

  return (
    <div style={formBody}>
      <div style={lbl}>URL 编码原文</div>
      <textarea
        style={textarea}
        value={input}
        onChange={e => setInput(e.target.value)}
        spellCheck={false}
        autoFocus
      />
      <div style={lbl}>解码结果</div>
      <div style={output}>
        {error ? (
          <span style={errorMsg}>{error}</span>
        ) : (
          <pre style={resultText}>{decoded}</pre>
        )}
      </div>
    </div>
  )
}

export default {
  id: 'url_decode',
  label: 'URL 解码',
  icon: 'Url',
  priority: 35,
  detect(content) {
    const s = content.trim()
    if (!s || s.length > 5000) return false
    return hasPercentEncoding(s)
  },
  Component: URLDecodeModal,
}
