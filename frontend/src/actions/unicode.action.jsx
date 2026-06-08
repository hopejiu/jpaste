import TransformModal from '../components/TransformModal'

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

export default {
  id: 'unicode',
  label: 'Unicode 解码',
  icon: 'Languages',
  priority: 20,
  trigger: '包含 \\uXXXX 转义序列的文本',
  desc: '在弹窗中将转义序列解码为实际字符',
  detect(content) {
    return /\\u[0-9a-fA-F]{4}/.test(content) && content.length <= 2000
  },
  Component: ({ content }) => (
    <TransformModal content={content} decode={decodeUnicode} inputLabel="Unicode 转义原文" outputLabel="解码结果" showCopy={false} />
  ),
}
