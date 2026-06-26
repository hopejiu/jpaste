import { Github } from 'lucide-react';

export default function Footer() {
  return (
    <footer className="bg-white border-t border-primary-100/50 py-12 mt-16">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-muted text-sm">
            <div className="w-6 h-6 rounded-md bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center text-white font-bold text-xs">
              jP
            </div>
            <span>jPaste &mdash; Windows 剪贴板增强工具</span>
          </div>
          <div className="flex items-center gap-6">
            <a
              href="https://github.com/hopejiu/jpaste"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 text-muted hover:text-primary-600 transition-colors text-sm"
            >
              <Github size={16} />
              <span>GitHub</span>
            </a>
            <a
              href="https://github.com/hopejiu/jpaste/releases"
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted hover:text-primary-600 transition-colors text-sm"
            >
              下载
            </a>
          </div>
        </div>
        <div className="mt-6 text-center text-xs text-muted/60">
          <p>Windows 10/11 &bull; 开源免费 &bull; Built with Wails + React</p>
        </div>
      </div>
    </footer>
  );
}
