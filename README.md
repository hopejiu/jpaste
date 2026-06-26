# jPaste

Windows 剪贴板管理器，基于 Wails v3 + React。

> 事件驱动监听，支持文本/图片/文件/URL 分类，即时搜索，键盘全操作。

## 功能

- **事件驱动捕获** — `WM_CLIPBOARDUPDATE` 消息窗口，无轮询
- **多格式支持** — 文本 (`CF_UNICODETEXT`)、图片 (`CF_DIB`)、文件路径 (`CF_HDROP`)
- **来源追踪** — 记录每条剪贴板内容的来源应用和窗口标题
- **标签过滤** — 全部 / 文本 / 图片 / 网址 / 文件 / 收藏，位掩码分类
- **即时搜索** — 全文搜索 + 正则表达式搜索，支持更新时间/字符串长度排序（升降序）
- **粘贴顺序控制** — 两种模式：正常、队列（先进先出），底部面板一键切换
- **收藏保护** — 自动清理不会删除收藏条目；清空全部时可选保留收藏
- **内容识别操作** — JSON 查看器、数学计算、Base64 解编码、URL 打开、Unicode 转换
- **图片查看** — 独立窗口，自适应尺寸，缩放/拖拽，支持 ← → 切换图片
- **文件路径** — 自动识别路径文本，支持路径文本复制和文件粘贴
- **Toast 通知** — 剪贴板变化时右下角无框通知（支持重复内容通知）
- **全局热键** — 默认 `Alt+V` 切换窗口显隐
- **可配置** — 复制/粘贴默认操作、保留天数、自动启动、通知开关、排序偏好

## 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+L` | 聚焦搜索框（全选） |
| `Ctrl+E` | 在编辑器中打开（优先 VS Code） |
| `Ctrl+1~9` | 执行对应条目默认操作 |
| `Ctrl+C` | 复制选中条目 |
| `←` / `→` / `Tab` | 切换标签页 |
| `↑` / `↓` | 上下移动条目焦点 |
| `Enter` | 执行焦点条目默认操作 |
| `Delete` | 删除焦点条目 |
| `Space` | 切换收藏 |
| `Home` / `End` | 滚动到列表顶部/底部 |
| `PageUp` / `PageDown` | 翻页滚动 |
| `Esc` | 清空搜索 → 隐藏窗口 |
| `Alt+V` | 全局热键（Go 端） |
| `F12` | 打开开发者工具（DevTools） |

## 官网

[https://hopejiu.github.io/jpaste/](https://hopejiu.github.io/jpaste/) — 功能展示、使用文档、常见问题。

源码在 [`website/`](website/) 目录，基于 Vite + React + TypeScript + Tailwind。

## 下载

从 [Releases](https://github.com/hopejiu/jpaste/releases) 下载最新的 `jpaste.exe`。便携式，无需安装。

## 开发

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# 安装前端依赖
cd frontend && npm install && cd ..

# 开发模式（热重载前端 + Go）
wails3 dev

# 生产构建
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
| 后端 | Go 1.25 + Wails v3 + SQLite (`modernc.org/sqlite`) |
| 前端 | React 18 + Vite 8 + React Router + Lucide Icons |
| 剪贴板 | `lxn/win` — Win32 API，格式枚举/来源检测/粘贴模拟 |
| 热键 | `golang.design/x/hotkey` — 全局键盘钩子（Service 结构体封装） |
| 存储 | `%APPDATA%/jPaste/clipboard.db` + `%APPDATA%/jPaste/images/` |
| 共享工具 | `internal/util` — `SelfWriteTracker`（自写入跟踪）、`FormatInt`（整数转字符串，避免引入 fmt） |
| 共享类型 | `internal/viewers` — 为四个查看器服务包提供统一 `CreateWindowFunc` 类型 |
## 架构

```
┌──────────────────────────────────────────────┐
│  React 前端 (WebView2)                        │
│  MainPage · SettingsPage · JsonViewPage        │
│  ImageViewPage ─ ToastPage (独立窗口)          │
│          ↕ Wails Bindings + Events.Emit        │
├──────────────────────────────────────────────┤
│  Go 后端                                      │
│  Clipboard · History · Sync · Settings         │
│  FileOp · FiloStack · Hotkey (Service 结构体)  │
│  ImageStore · Curl/Json/Ws/ImageViewer        │
│  Toast (事件驱动通知) · Log (日志中继)          │
│  Viewers (共享 CreateWindowFunc 类型)          │
│  Util (SelfWriteTracker · FormatInt)           │
│  SQLite + 图片文件存储                          │
│  系统托盘 + 全局热键                           │
└──────────────────────────────────────────────┘
```

### 关键设计

- **Toast 通知**: 预创建的隐藏无框窗口，通过离屏定位避免闪烁。WebView2 始终保持渲染，收到事件后移入可视区域，3 秒后移回屏幕外。与主窗口路由完全隔离。
- **事件系统**: Go 端通过 `app.Event.Emit` 广播事件，前端通过 `@wailsio/runtime` 的 `Events.On` 监听。前端日志通过 `Events.Emit('frontend-log', ...)` 回传后端写入统一日志文件。
- **剪贴板监控**: 消息窗口 (`HWND_MESSAGE`) + `AddClipboardFormatListener`，无需轮询。
- **粘贴顺序控制**: 键盘钩子 (`WH_KEYBOARD_LL`) 拦截 Ctrl+V，从内部队列中弹出内容。
- **自写入跟踪**: `SelfWriteTracker`（`internal/util/tracker.go`）统一管理 clipboard 和 filostack 的自写入检测，消除代码克隆。
- **前端通用组件**:
  - `Modal.jsx` — 通用模态框（ESC/遮罩关闭、尺寸变体），替代 4 处手写的遮罩层
  - `TransformModal.jsx` — 解码类操作的通用 UI（输入→解码→输出+复制）
  - `useEscapeHide.js` — 共享的 Escape 隐藏窗口钩子，替代 3 处重复实现

## License

MIT
