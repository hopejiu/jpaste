import { Service as FileService } from '../../bindings/jpaste/internal/fileop'
import { log } from '../logger'

function isFilePath(path) {
  const cleaned = path.trim().replace(/[\\/]+$/, '')
  const last = cleaned.split(/[\\/]/).pop()
  return last.includes('.')
}

function openFolder(content, entryId) {
  const isFile = isFilePath(content)
  return FileService.OpenInExplorer(entryId, isFile).catch(e => {
    log.error('FolderAction', 'Failed to open:', e)
    throw new Error('文件可能已被删除或移动')
  })
}

export default {
  id: 'folder',
  label: '打开文件夹',
  icon: 'FolderOpen',
  priority: 60,
  trigger: 'Windows 本地路径（如 C:\\ 或 \\\\server）',
  desc: '在资源管理器中打开文件夹，文件路径则定位到文件位置',
  detect(content) {
    const s = content.trim()
    return /^[A-Za-z]:[\\/]|^\\\\/.test(s) && s.length <= 260
  },
  handler: openFolder,
}
