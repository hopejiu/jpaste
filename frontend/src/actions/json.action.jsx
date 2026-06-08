import { Service as JsonViewerService } from '../../bindings/jpaste/internal/jsonviewer'

function openJsonViewer(_content, entryId) {
  JsonViewerService.OpenJsonViewer(entryId)
}

export default {
  id: 'json',
  label: '查看 JSON',
  icon: 'Braces',
  priority: 40,
  trigger: '以 { 或 [ 开头的合法 JSON',
  desc: '在独立窗口中展开浏览/编辑 JSON，支持树形和代码视图',
  detect(content) {
    const s = content.trim()
    if (!s.startsWith('{') && !s.startsWith('[')) return false
    try {
      JSON.parse(s)
      return true
    } catch {
      return false
    }
  },
  handler: openJsonViewer,
  // No modal — opens a separate window via Go backend.
  Component: null,
}
