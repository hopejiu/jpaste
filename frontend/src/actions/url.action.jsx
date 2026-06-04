import { Browser } from '@wailsio/runtime'

function openUrl(content) {
  Browser.OpenURL(content.trim())
}

export default {
  id: 'url',
  label: '打开网址',
  icon: 'ExternalLink',
  priority: 55,
  detect(content) {
    const s = content.trim()
    return /^https?:\/\/[^\s]{3,}/.test(s) && s.length <= 2000
  },
  handler: openUrl,
}
