import { Service as FileService } from '../../bindings/jpaste/internal/fileop'
import { formBody, label as lbl, displayValue, primaryBtn } from './actionStyles'

function isFilePath(path) {
  const cleaned = path.trim().replace(/[\\/]+$/, '')
  const last = cleaned.split(/[\\/]/).pop()
  return last.includes('.')
}

function FolderModal({ content, entryId, onClose }) {
  const path = content.trim()
  const isFile = isFilePath(path)

  const open = async () => {
    try {
      await FileService.OpenInExplorer(entryId, isFile)
      onClose()
    } catch (e) {
      console.error('Failed to open:', e)
    }
  }

  return (
    <div style={formBody}>
      <div style={lbl}>
        {isFile ? '将在资源管理器中定位此文件：' : '将使用资源管理器打开：'}
      </div>
      <div style={displayValue}>{path}</div>
      <button style={primaryBtn} onClick={open}>
        {isFile ? '打开文件所在位置' : '打开文件夹'}
      </button>
    </div>
  )
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
  Component: FolderModal,
}
