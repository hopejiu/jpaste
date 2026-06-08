import { Browser } from '@wailsio/runtime'

function openUrl(content) {
  Browser.OpenURL(content.trim())
}

export default {
  id: 'url',
  label: '打开网址',
  icon: 'ExternalLink',
  priority: 55,
  trigger: '以 http:// 或 https:// 开头的网址',
  desc: '在系统默认浏览器中打开链接',
  detect(content) {
    const s = content.trim()
    return /^https?:\/\/[^\s]{3,}/.test(s) && s.length <= 2000
  },
  handler: openUrl,
}
