import { useState, useMemo } from 'react'
import { formBody, label as lbl, textarea, output, errorMsg, resultText } from './actionStyles'

function decodeB64(input) {
  if (!input.trim()) return { decoded: '', error: null }
  try {
    const decoded = atob(input.trim())
    return { decoded, error: null }
  } catch (e) {
    return { decoded: '', error: '无效的 Base64 编码' }
  }
}

function Base64Modal({ content, onClose }) {
  const [input, setInput] = useState(content.trim())
  const { decoded, error } = useMemo(() => decodeB64(input), [input])

  return (
    <div style={formBody}>
      <div style={lbl}>Base64 原文</div>
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
  id: 'base64',
  label: 'Base64 解码',
  icon: 'Binary',
  priority: 30,
  detect(content) {
    const s = content.trim()
    return /^[A-Za-z0-9+/]+=*$/.test(s) && s.length >= 4 && s.length % 4 === 0
  },
  Component: Base64Modal,
}
