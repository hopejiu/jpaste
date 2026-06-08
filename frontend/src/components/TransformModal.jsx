import { useState, useMemo } from 'react'
import { formBody, label as lbl, textarea, output, errorMsg, resultText, copyBtn, copyBtnSuccess } from '../actions/actionStyles'

/**
 * Reusable modal component for decode-type actions (base64, url-decode, unicode).
 * Props:
 *   content      - initial input text
 *   decode       - function(input: string) => { decoded: string, error: string|null }
 *   inputLabel   - label for the input textarea
 *   outputLabel  - label for the output area
 *   showCopy     - show copy button (default true)
 */
export default function TransformModal({ content, decode, inputLabel, outputLabel, showCopy = true }) {
  const [input, setInput] = useState(content.trim())
  const [copied, setCopied] = useState(false)
  const { decoded, error } = useMemo(() => decode(input), [input, decode])

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
      <div style={lbl}>{inputLabel}</div>
      <textarea
        style={textarea}
        value={input}
        onChange={e => setInput(e.target.value)}
        spellCheck={false}
        autoFocus
      />
      <div style={lbl}>{outputLabel}</div>
      <div style={{ ...output, position: 'relative' }}>
        {error ? (
          <span style={errorMsg}>{error}</span>
        ) : (
          <>
            <pre style={resultText}>{decoded}</pre>
            {showCopy && decoded && (
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
