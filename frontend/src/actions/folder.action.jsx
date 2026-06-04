import { Service as FileService } from '../../bindings/jpaste/internal/fileop'

function isFilePath(path) {
  const cleaned = path.trim().replace(/[\\/]+$/, '')
  const last = cleaned.split(/[\\/]/).pop()
  return last.includes('.')
}

function openFolder(content, entryId) {
  const isFile = isFilePath(content)
  FileService.OpenInExplorer(entryId, isFile).catch(e => console.error('Failed to open:', e))
}

export default {
  id: 'folder',
  label: '打开文件夹',
  icon: 'FolderOpen',
  priority: 60,
  detect(content) {
    const s = content.trim()
    return /^[A-Za-z]:[\\/]|^\\\\/.test(s) && s.length <= 260
  },
  handler: openFolder,
}
