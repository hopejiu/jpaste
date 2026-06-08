import TransformModal from '../components/TransformModal'

function decodeB64(input) {
  if (!input.trim()) return { decoded: '', error: null }
  try {
    return { decoded: atob(input.trim()), error: null }
  } catch (e) {
    return { decoded: '', error: '无效的 Base64 编码' }
  }
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
  Component: ({ content }) => (
    <TransformModal content={content} decode={decodeB64} inputLabel="Base64 原文" outputLabel="解码结果" />
  ),
}
