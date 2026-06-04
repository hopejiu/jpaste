# jPaste

Windows 剪贴板管理器，基于 Wails v3 + React。

> 事件驱动监听，支持文本/图片/文件/URL 分类，即时搜索，键盘全操作。

## 功能

- **事件驱动捕获** — `WM_CLIPBOARDUPDATE` 消息窗口，无轮询
- **多格式支持** — 文本 (`CF_UNICODETEXT`)、图片 (`CF_DIB`)、文件路径 (`CF_HDROP`)
- **来源追踪** — 记录每条剪贴板内容的来源应用和窗口标题
- **标签过滤** — 全部 / 文本 / 图片 / 网址 / 文件 / 收藏，位掩码分类
- **即时搜索** — 全文搜索 + 正则表达式搜索
- **内容识别操作** — JSON 查看器、数学计算、Base64 解编码、URL 打开、Unicode 转换
- **图片查看** — 独立窗口，自适应尺寸，缩放/拖拽，支持 ← → 切换图片
- **文件路径** — 自动识别路径文本，支持路径文本复制和文件粘贴
- **WebDAV 同步** — TODO，跨设备双向合并（计划支持坚果云）
- **全局热键** — 默认 `Alt+V` 切换窗口显隐
- **可配置** — 复制/粘贴默认操作、保留天数、自动启动、通知开关

## 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+L` | 聚焦搜索框（全选） |
| `Ctrl+E` | 在编辑器中打开（优先 VS Code） |
| `Ctrl+1~9` | 执行对应条目默认操作 |
| `Ctrl+Enter` | 强制粘贴（覆盖默认操作） |
| `←` / `→` / `Tab` | 切换标签页 |
| `↑` / `↓` | 上下移动条目焦点 |
| `Enter` | 执行焦点条目默认操作 |
| `Delete` | 删除焦点条目 |
| `Space` | 切换收藏 |
| `Home` / `End` | 滚动到列表顶部/底部 |
| `PageUp` / `PageDown` | 翻页滚动 |
| `Esc` | 清空搜索 → 隐藏窗口 |
| `Alt+V` | 全局热键（Go 端） |

## 下载

从 [Releases](https://github.com/wangxianyu/jPaste/releases) 下载最新的 `jpaste.exe`。便携式，无需安装。

## 开发

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# 开发模式（热重载）
wails3 dev

# 构建
wails3 build
# 输出: bin/jpaste.exe
```

### 应用图标

源文件: `jpaste-logo.svg`。一键生成所有图标资源：

```powershell
.\scripts\update-icon-svg.ps1
```

生成文件:
- `paste.png` — 托盘图标，`main.go` 通过 `//go:embed` 嵌入
- `build/windows/icon.ico` — 多分辨率（16–256px）应用图标
- `wails_windows_amd64.syso` — Windows 构建资源

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go + Wails v3 + SQLite (`modernc.org/sqlite`) |
| 前端 | React 18 + Vite + React Router + Lucide Icons |
| 剪贴板 | `lxn/win` — Win32 API，格式枚举/来源检测/粘贴模拟 |
| 存储 | `%APPDATA%/jPaste/clipboard.db` + `%APPDATA%/jPaste/images/` |
| 同步 | WebDAV（TODO，双向合并） |

## 架构

```
┌─────────────────────────────────────────┐
│  React 前端 (WebView)                    │
│  MainPage · SettingsPage  · JsonViewPage │
│  ImageViewPage                           │
│          ↕ Wails Bindings                │
├─────────────────────────────────────────┤
│  Go 后端                                 │
│  Clipboard · History  · Sync · Settings  │
│  FileOp   · ImageStore · JsonViewer      │
│  SQLite + 图片文件 · WebDAV (TODO)        │
│  系统托盘 + 全局热键                     │
└─────────────────────────────────────────┘
```

## License

MIT
