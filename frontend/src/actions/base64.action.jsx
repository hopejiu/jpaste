import { useState, useMemo } from 'react'
import { formBody, label as lbl, textarea, output, errorMsg, resultText, copyBtn, copyBtnSuccess } from './actionStyles'

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
  const [copied, setCopied] = useState(false)
  const { decoded, error } = useMemo(() => decodeB64(input), [input])

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
      <div style={lbl}>Base64 原文</div>
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
  id: 'base64',
  label: 'Base64 解码',
  icon: 'Binary',
  priority: 30,
  trigger: '符合 Base64 编码格式的字符串',
  desc: '在弹窗中解码为可读文本，支持编辑和复制结果',
  detect(content) {
    const s = content.trim()
    return /^[A-Za-z0-9+/]+=*$/.test(s) && s.length >= 4 && s.length % 4 === 0
  },
  Component: Base64Modal,
}
