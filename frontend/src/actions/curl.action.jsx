import { Service as CurlViewerService } from '../../bindings/jpaste/internal/curlviewer'

// Detection: only cheap regex. Full parsing happens in the viewer page via curlconverter.
function openCurlViewer(_content, entryId) {
  CurlViewerService.OpenCurlViewer(entryId)
}

export default {
  id: 'curl',
  label: 'HTTP 调试',
  icon: 'Terminal',
  priority: 55,
  trigger: '以 "curl " 开头的命令行',
  desc: '在独立窗口中解析/编辑 HTTP 请求并发送调试，可查看响应',
  detect(content) {
    return /^\s*curl\s/i.test(content)
  },
  handler: openCurlViewer,
  Component: null,
}
