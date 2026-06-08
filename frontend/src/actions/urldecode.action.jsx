import { useState, useMemo } from 'react'
import { formBody, label as lbl, textarea, output, errorMsg, resultText, copyBtn, copyBtnSuccess } from './actionStyles'

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
  const [copied, setCopied] = useState(false)
  const { decoded, error } = useMemo(() => decodeURL(input), [input])

  const handleCopy = async () => {
    if (!decoded || error) return
    try {
      await navigator.clipboard.writeText(decoded)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('复制失败:', err)
    }
  }

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
      <div style={{ ...output, position: 'relative' }}>
        {error ? (
          <span style={errorMsg}>{error}</span>
        ) : (
          <>
            <pre style={resultText}>{decoded}</pre>
            {decoded && (
              <button
                style={copied ? copyBtnSuccess : copyBtn}
                onClick={handleCopy}
                onMouseEnter={e => { if (!copied) e.target.style.opacity = '1' }}
                onMouseLeave={e => { if (!copied) e.target.style.opacity = '0.8' }}
              >
                {copied ? '已复制' : '复制'}
              </button>
            )}
          </>
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
