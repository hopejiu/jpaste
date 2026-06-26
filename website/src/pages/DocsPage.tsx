import { Keyboard, Settings, Search, Image, Terminal, Globe, Code, FileText } from 'lucide-react';

const sections = [
  {
    icon: Keyboard,
    title: '快捷键',
    items: [
      ['Alt+V', '显示/隐藏 jPaste 窗口（可在设置中自定义）'],
      ['Alt+数字键 (1-9)', '快速选择第 N 个历史条目'],
      ['Ctrl+L', '聚焦搜索框'],
      ['Ctrl+C', '复制选中条目'],
      ['E', '在编辑器中打开'],
      ['Del', '删除条目'],
      ['Space', '收藏/取消收藏'],
      ['Esc', '隐藏窗口'],
    ],
  },
  {
    icon: Settings,
    title: '粘贴模式',
    items: [
      ['正常模式', '复制什么就粘贴什么，与系统剪贴板行为一致'],
      ['队列模式 (FIFO)', '先复制的内容优先粘贴，像排队一样先来先出'],
    ],
  },
  {
    icon: Search,
    title: '搜索与筛选',
    content:
      '支持全文关键词搜索，按标签快速筛选（全部 / 文本 / 图片 / 网址 / 文件）。搜索结果以光标分页方式加载，每页 20 条，滚动自动加载更多。',
  },
  {
    icon: Image,
    title: '图片支持',
    content:
      '自动捕获剪贴板中的图片并保存到本地。图片条目在主列表中以缩略图显示，支持一键复制和收藏。',
  },
  {
    icon: Terminal,
    title: 'CURL 调试器',
    content:
      '检测到剪贴板内容以 curl 开头时，自动提供 CURL 调试窗口。可编辑请求参数、发送 HTTP 请求、查看响应结果，支持 JSON 响应格式化查看。',
  },
  {
    icon: Globe,
    title: 'WebSocket 调试器',
    content:
      '检测到 ws:// 或 wss:// 链接时，自动提供 WebSocket 调试窗口。可直接连接、发送消息、查看通信记录。',
  },
  {
    icon: Code,
    title: 'JSON 查看器',
    content:
      '检测到有效 JSON 内容时，自动提供 JSON 查看窗口。支持树形和代码两种视图，可编辑、搜索、排序，支持撤销/重做。',
  },
  {
    icon: FileText,
    title: '更多功能',
    content:
      'Base64 解码、Unicode 转义解码、URL 直接打开、文件夹路径打开、数学表达式求值、时间戳转换——自动识别内容类型，提供一键操作。',
  },
];

export default function DocsPage() {
  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
      <div className="text-center mb-14">
        <h1 className="text-3xl sm:text-4xl font-bold text-foreground">使用文档</h1>
        <p className="mt-3 text-muted text-lg">了解 jPaste 的全部功能与操作方式</p>
      </div>

      <div className="space-y-14">
        {sections.map((section) => (
          <section key={section.title}>
            <div className="flex items-center gap-3 mb-6">
              <div className="w-9 h-9 rounded-xl bg-primary-50 flex items-center justify-center text-primary-600">
                <section.icon size={18} />
              </div>
              <h2 className="text-2xl font-bold text-foreground">{section.title}</h2>
            </div>

            {section.items && (
              <div className="bg-white rounded-2xl border border-primary-100/50 overflow-hidden">
                <table className="w-full text-sm">
                  <tbody>
                    {section.items.map((row, i) => (
                      <tr key={i} className={i < section.items.length - 1 ? 'border-b border-primary-50' : ''}>
                        <td className="py-3.5 px-5 font-mono text-primary-700 font-medium whitespace-nowrap w-1/3">
                          {row[0]}
                        </td>
                        <td className="py-3.5 px-5 text-foreground">{row[1]}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {section.content && (
              <p className="text-muted leading-relaxed text-base bg-white rounded-2xl border border-primary-100/50 p-6">
                {section.content}
              </p>
            )}
          </section>
        ))}
      </div>
    </div>
  );
}
