import TransformModal from '../components/TransformModal'

function decodeURL(input) {
  if (!input.trim()) return { decoded: '', error: null }
  try {
    return { decoded: decodeURIComponent(input.trim()), error: null }
  } catch (e) {
    return { decoded: '', error: '无效的 URL 编码' }
  }
}

function hasPercentEncoding(s) {
  return /%[0-9A-Fa-f]{2}/.test(s)
}

export default {
  id: 'url_decode',
  label: 'URL 解码',
  icon: 'Url',
  priority: 35,
  trigger: '包含 %XX 编码字符的文本',
  desc: '在弹窗中解码为可读字符串，支持编辑和复制结果',
  detect(content) {
    const s = content.trim()
    if (!s || s.length > 5000) return false
    return hasPercentEncoding(s)
  },
  Component: ({ content }) => (
    <TransformModal content={content} decode={decodeURL} inputLabel="URL 编码原文" outputLabel="解码结果" />
  ),
}
