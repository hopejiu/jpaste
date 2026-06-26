import { ChevronDown } from 'lucide-react';
import { useState } from 'react';

const faqs = [
  {
    q: 'jPaste 是否免费？',
    a: '是的，jPaste 完全免费开源，采用 MIT 许可证。你可以自由使用、修改和分发。',
  },
  {
    q: '支持哪些 Windows 版本？',
    a: 'jPaste 支持 Windows 10 和 Windows 11，64 位系统。无需额外运行时，单文件即可运行。',
  },
  {
    q: '历史记录保存在哪里？会占用多少空间？',
    a: '数据存储在 %APPDATA%/jPaste/ 目录下，包含 SQLite 数据库和图片文件。默认保留 30 天，可在设置中调整保留时长。数据库体积通常只有几 MB 到几十 MB。',
  },
  {
    q: '如何卸载 jPaste？',
    a: 'jPaste 是绿色单文件，直接删除 jpaste.exe 即可。数据目录 %APPDATA%/jPaste/ 可手动删除。',
  },
  {
    q: '全局快捷键（Alt+V）和其他软件冲突怎么办？',
    a: '可以在设置页面自定义快捷键，支持字母、数字和 F1-F12 键，避开与其他软件的冲突。',
  },
  {
    q: '什么是队列模式？',
    a: '队列模式（先进先出）：连续复制 A、B、C 后，粘贴顺序为 A、B、C，像排队一样先来先出。',
  },
  {
    q: '复制图片或文件时会怎样？',
    a: '图片会被自动捕获并保存，以缩略图形式显示在列表中。复制文件路径时，jPaste 会识别为文件条目。如果处于队列模式，复制图片或文件会自动退出该模式以保护数据完整性。',
  },
  {
    q: 'jPaste 开机自启吗？',
    a: '是的，jPaste 启动后会自动最小化到系统托盘，后台运行，不干扰正常工作。如果不需要自启，可以在设置中关闭或从系统托盘退出。',
  },
  {
    q: '剪贴板内容会被上传到云端吗？',
    a: '不会。jPaste 完全离线运行，所有数据存储在本地，不会上传任何内容到云端。你的数据只属于你。',
  },
  {
    q: '如何报告 Bug 或提建议？',
    a: '欢迎在 GitHub Issues 提交反馈：github.com/hopejiu/jpaste。我们会及时查看和处理。',
  },
];

export default function FAQPage() {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  const toggle = (i: number) => setOpenIndex(openIndex === i ? null : i);

  return (
    <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
      <div className="text-center mb-14">
        <h1 className="text-3xl sm:text-4xl font-bold text-foreground">常见问题</h1>
        <p className="mt-3 text-muted text-lg">你可能想知道的答案</p>
      </div>

      <div className="space-y-3">
        {faqs.map((faq, i) => (
          <div
            key={i}
            className="bg-white rounded-2xl border border-primary-100/50 overflow-hidden transition-shadow hover:shadow-sm"
          >
            <button
              onClick={() => toggle(i)}
              className="w-full flex items-center justify-between px-6 py-4 text-left gap-4"
              aria-expanded={openIndex === i}
            >
              <span className="font-medium text-foreground">{faq.q}</span>
              <ChevronDown
                size={18}
                className={`shrink-0 text-muted transition-transform duration-200 ${
                  openIndex === i ? 'rotate-180' : ''
                }`}
              />
            </button>
            <div
              className={`overflow-hidden transition-all duration-200 ease-in-out ${
                openIndex === i ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'
              }`}
            >
              <p className="px-6 pb-4 text-muted leading-relaxed">{faq.a}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
