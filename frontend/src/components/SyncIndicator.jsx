import { Cloud, CloudOff, CloudRain, AlertCircle } from 'lucide-react'

export default function SyncIndicator({ status }) {
  const icons = {
    none:    <CloudOff size={14} />,
    syncing: <CloudRain size={14} />,
    ok:      <Cloud size={14} />,
    error:   <AlertCircle size={14} />,
  }
  const colors = {
    none:    'var(--color-muted)',
    syncing: '#F59E0B',
    ok:      '#10B981',
    error:   '#EF4444',
  }
  const titles = {
    none:    '同步未配置',
    syncing: '同步中…',
    ok:      '已同步',
    error:   '同步失败',
  }
  const icon = icons[status] || icons.none
  const color = colors[status] || colors.none
  const title = titles[status] || ''
  const anim = status === 'syncing' ? { animation: 'spin 1.5s linear infinite' } : {}

  return (
    <span title={title} style={{ color, display: 'flex', alignItems: 'center', transition: 'color 0.3s', ...anim }}>
      {icon}
    </span>
  )
}
