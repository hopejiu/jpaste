import { Browser } from '@wailsio/runtime'
import { formBody, label as lbl, displayValue, primaryBtn } from './actionStyles'

function UrlModal({ content, onClose }) {
  const url = content.trim()

  const open = () => {
    try {
      Browser.OpenURL(url)
      onClose()
    } catch (e) {
      console.error('Failed to open URL:', e)
    }
  }

  return (
    <div style={formBody}>
      <div style={lbl}>将使用默认浏览器打开：</div>
      <div style={{ ...displayValue, color: 'var(--color-primary)' }}>{url}</div>
      <button style={primaryBtn} onClick={open}>
        打开链接
      </button>
    </div>
  )
}

export default {
  id: 'url',
  label: '打开网址',
  icon: 'ExternalLink',
  priority: 55,
  detect(content) {
    const s = content.trim()
    return /^https?:\/\/[^\s]{3,}/.test(s) && s.length <= 2000
  },
  Component: UrlModal,
}
