import { useState, useMemo } from 'react'
import { formBody, label as lbl, textarea, output, errorMsg, resultText } from './actionStyles'

function decodeUnicode(input) {
  if (!input.trim()) return { decoded: '', error: null }
  try {
    const decoded = input.replace(/\\u([0-9a-fA-F]{4})/g, (_, hex) =>
      String.fromCharCode(parseInt(hex, 16))
    )
    return { decoded, error: null }
  } catch (e) {
    return { decoded: '', error: e.message }
  }
}

function UnicodeModal({ content, onClose }) {
  const [input, setInput] = useState(content.trim())
  const { decoded, error } = useMemo(() => decodeUnicode(input), [input])

  return (
    <div style={formBody}>
      <div style={lbl}>Unicode 转义原文</div>
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
  id: 'unicode',
  label: 'Unicode 解码',
  icon: 'Languages',
  priority: 20,
  detect(content) {
    return /\\u[0-9a-fA-F]{4}/.test(content) && content.length <= 2000
  },
  Component: UnicodeModal,
}
