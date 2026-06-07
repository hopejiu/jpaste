import { Service as WsViewerService } from '../../bindings/jpaste/internal/wssviewer'

function openWsViewer(_content, entryId) {
  WsViewerService.OpenWsViewer(entryId)
}

export default {
  id: 'ws',
  label: 'WS 调试',
  icon: 'Radio',
  priority: 35,
  detect(content) {
    const s = content.trim()
    return s.startsWith('ws://') || s.startsWith('wss://')
  },
  handler: openWsViewer,
  Component: null,
}
