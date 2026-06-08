import Modal from './Modal'

const SHORTCUT_GROUPS = [
  { title: '搜索', items: [
    { keys: 'Ctrl+L', desc: '聚焦搜索框' },
    { keys: 'Esc', desc: '清空搜索（有搜索词时）' },
    { keys: '任意字母键', desc: '自动聚焦搜索框' },
  ] },
  { title: '编辑', items: [
    { keys: 'Ctrl+E', desc: '在编辑器中打开选中条目' },
    { keys: 'Ctrl+C', desc: '复制选中条目' },
    { keys: 'Delete', desc: '删除选中条目' },
    { keys: 'Space', desc: '切换收藏状态' },
  ] },
  { title: '导航', items: [
    { keys: '↑  ↓', desc: '在条目列表中上下移动焦点' },
    { keys: 'Ctrl+1~9', desc: '对第 N 条条目执行默认操作' },
    { keys: 'Enter', desc: '对焦点条目执行默认操作' },
    { keys: '→', desc: '进入动作模式（展开操作按钮）' },
    { keys: 'Home  /  End', desc: '滚动到列表顶部 / 底部' },
    { keys: 'PageUp  /  PageDown', desc: '按页滚动' },
  ] },
  { title: '动作模式', hint: '选中条目后按 → 进入，可浏览并执行快捷操作', items: [
    { keys: '←  →', desc: '在操作按钮间移动焦点' },
    { keys: 'Enter', desc: '执行当前选中的操作' },
    { keys: 'Esc', desc: '退出动作模式' },
  ] },
  { title: '标签', items: [
    { keys: 'Tab  /  Shift+Tab', desc: '在标签页间切换' },
  ] },
  { title: '窗口', items: [
    { keys: 'Esc', desc: '隐藏窗口（无搜索词时）' },
    { keys: 'Alt+V', desc: '全局快捷键，显示 / 隐藏窗口（可在设置中更改）' },
  ] },
]

export default function ShortcutHelpModal({ open, onClose }) {
  return (
    <Modal open={open} onClose={onClose} title="快捷键说明" size="sm">
      <div className="space-y-4">
        {SHORTCUT_GROUPS.map(group => (
          <div key={group.title}>
            <div className="text-sm font-semibold text-foreground mb-1.5">{group.title}</div>
            {group.hint && (
              <div className="text-[11px] text-muted mb-1.5 leading-[1.4]">{group.hint}</div>
            )}
            <div className="flex flex-col gap-1">
              {group.items.map(item => (
                <div key={item.keys} className="flex items-baseline gap-2 text-xs">
                  <kbd className="px-1.5 py-[2px] rounded font-mono font-medium flex-shrink-0"
                    style={{ color: 'var(--color-primary)', background: 'var(--color-primary-alpha-08)', border: '1px solid var(--color-primary-alpha-12)', minWidth: '56px', textAlign: 'center' }}
                  >{item.keys}</kbd>
                  <span className="text-muted leading-[1.4]">{item.desc}</span>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </Modal>
  )
}

