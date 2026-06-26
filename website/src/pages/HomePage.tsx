import { ArrowRight, Sparkles, Layers, Eye, Zap } from 'lucide-react';
import screenshot from '../assets/screenshot-main.png';

const features = [
  {
    icon: Layers,
    title: '历史记录',
    desc: '自动记录所有复制内容，按时间排列，支持全文搜索和标签筛选，再也不用担心丢失复制内容。',
  },
  {
    icon: Zap,
    title: '智能粘贴模式',
    desc: '支持「队列」粘贴模式，连续复制后按 FIFO 顺序依次粘贴，效率翻倍。',
  },
  {
    icon: Eye,
    title: '内容识别工具',
    desc: '自动识别 JSON、CURL、WebSocket、Base64、时间戳，一键查看/编辑/调试，无需打开额外工具。',
  },
  {
    icon: Sparkles,
    title: '全局快捷键',
    desc: 'Alt+V 一键呼出，Alt+数字快速选择，主题自定义，通知预览，完全可配置的设置面板。',
  },
];

const DOWNLOAD_URL = 'https://github.com/hopejiu/jpaste/releases';

export default function HomePage() {
  return (
    <div>
      {/* Hero */}
      <section className="relative overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-b from-primary-50/60 to-surface pointer-events-none" />
        <div className="absolute top-0 right-0 w-[600px] h-[600px] bg-gradient-to-br from-primary-300/20 to-primary-500/10 rounded-full blur-3xl pointer-events-none" />
        <div className="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pt-20 pb-24 sm:pt-28 sm:pb-32">
          <div className="max-w-3xl">
            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold text-foreground leading-tight tracking-tight">
              你的 Windows
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-primary-500 to-primary-700"> 剪贴板 </span>
              生产力工具
            </h1>
            <p className="mt-6 text-lg sm:text-xl text-muted leading-relaxed max-w-2xl">
              jPaste 是一款轻量开源的 Windows 剪贴板管理器。自动记录历史、智能粘贴模式、内容识别工具——让复制粘贴成为真正的生产力。
            </p>
            <div className="mt-8 flex flex-col sm:flex-row gap-4">
              <a
                href={DOWNLOAD_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 px-8 py-3.5 rounded-xl bg-gradient-to-r from-primary-500 to-primary-600 text-white font-semibold text-base hover:from-primary-600 hover:to-primary-700 transition-all shadow-lg shadow-primary-500/25 hover:shadow-primary-500/40"
              >
                免费下载
                <ArrowRight size={18} />
              </a>
              <a
                href={DOWNLOAD_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 px-8 py-3.5 rounded-xl border border-primary-200 text-primary-700 font-semibold text-base hover:bg-primary-50 transition-all"
              >
                查看版本
              </a>
            </div>
            <p className="mt-4 text-sm text-muted/70">Windows 10/11 &bull; 开源免费 &bull; 单文件免安装</p>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-20">
        <div className="text-center mb-14">
          <h2 className="text-3xl sm:text-4xl font-bold text-foreground">核心功能</h2>
          <p className="mt-3 text-muted text-lg">轻量而强大，一切为了更快的复制粘贴</p>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {features.map((f) => (
            <div
              key={f.title}
              className="group bg-white rounded-2xl p-6 border border-primary-100/50 hover:border-primary-200 hover:shadow-lg hover:shadow-primary-500/5 transition-all"
            >
              <div className="w-10 h-10 rounded-xl bg-primary-50 flex items-center justify-center text-primary-600 group-hover:bg-primary-100 transition-colors mb-4">
                <f.icon size={20} />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">{f.title}</h3>
              <p className="text-sm text-muted leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Screenshot */}
      <section className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pb-20">
        <div className="bg-gradient-to-br from-primary-50 to-white rounded-3xl p-6 sm:p-10 border border-primary-100/50">
          <div className="text-center mb-8">
            <h2 className="text-2xl sm:text-3xl font-bold text-foreground">界面预览</h2>
            <p className="mt-2 text-muted">简洁清晰，专注内容</p>
          </div>
          <div className="rounded-2xl overflow-hidden shadow-xl border border-primary-100/30 bg-white">
            <img
              src={screenshot}
              alt="jPaste 应用界面截图"
              className="w-full h-auto"
              loading="lazy"
            />
          </div>
        </div>
      </section>

      {/* Download CTA */}
      <section className="bg-gradient-to-br from-primary-500 to-primary-700 py-20">
        <div className="max-w-3xl mx-auto px-4 sm:px-6 text-center">
          <h2 className="text-3xl sm:text-4xl font-bold text-white">准备好了吗？</h2>
          <p className="mt-3 text-primary-100 text-lg">立即下载，让剪贴板不再只是个临时缓冲区</p>
          <a
            href={DOWNLOAD_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-8 inline-flex items-center gap-2 px-10 py-4 rounded-xl bg-white text-primary-700 font-bold text-lg hover:bg-primary-50 transition-all shadow-xl"
          >
            下载 jPaste
            <ArrowRight size={20} />
          </a>
          <p className="mt-4 text-primary-200 text-sm">Windows 10/11 &bull; 单文件 &bull; 无需安装</p>
        </div>
      </section>
    </div>
  );
}
