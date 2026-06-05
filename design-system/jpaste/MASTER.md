# Design System Master File

> **LOGIC:** When building a specific page, first check `design-system/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.

---

**Project:** jPaste
**Updated:** 2026-06-05
**Category:** Productivity Tool (Clipboard Manager)

---

## Architecture: Multi-Theme System

jPaste supports **3 themes** switchable from Settings → 主题. Themes are stored in Go `settings.json` as `theme` field (`"a"` / `"b"` / `"c"`), applied via CSS class `theme-*` on the root `<div>`.

| ID | Name | Mood | Primary | Background | Surface |
|----|------|------|---------|------------|---------|
| `a` | 冷调极简 | 清爽、专注 | `#0D9488` 青碧 | `#F1F3F5` | `#FFFFFF` |
| `b` | 暖调高效 | 经典、生产力 | `#6366F1` Indigo | `#F0F2F5` | `#FFFFFF` |
| `c` | 深色沉浸 | OLED 黑底白字 | `#5E6AD2` 紫蓝 | `#000000` | `#161616` |

### Switching Flow

1. User selects theme in Settings
2. Go backend saves new `theme` to `settings.json`
3. Frontend calls `window.location.reload()`
4. On reload, `AppContent` reads `settings.theme`
5. Sets `document.documentElement.className = "theme-{a|b|c}"` — 使 CSS 变量级联到 `<body>`
6. AppContent div 同时设置 `className="{animClass} theme-{a|b|c}"`
7. All components reference CSS `var(--color-*)` — themes swap variable values

### Layer Architecture (三层面系统)

```
<body>               --color-background (最深)
  └─ #root
     └─ AppContent   --color-surface (中层, 主内容区)
        ├─ Header    --color-surface (surface 层)
        ├─ TabBar    --color-background (沉回背景层, 视觉分隔)
        ├─ List      --color-surface (内容区)
        ├─ Footer    --color-background (沉回背景层)
        └─ Modal     --color-elevated (浮动最高层)
```

### CSS Variables

Each theme defines the complete set:

```
--color-primary            --color-primary-hover
--color-accent             --color-background
--color-surface            --color-surface-hover
--color-elevated           --color-foreground
--color-muted              --color-border
--color-destructive        --color-ring
--color-favorite           --color-success
--color-badge-file         --color-badge-file-bg
--color-image-bg
--color-primary-alpha-04   --color-primary-alpha-06
--color-primary-alpha-08   --color-primary-alpha-12
--color-primary-alpha-15
```

Alpha variants use `color-mix(in srgb, var(--color-primary) N%, transparent)` for theme-adaptive transparency.

### Shared Tokens (same across all themes)

```
--radius-sm: 6px     --radius-md: 8px     --radius-lg: 12px
--space-1..8         4/8/12/16/24/32px
--font-size-xs..2xl  12/13/14/16/18/24px
--transition-fast    150ms ease
--transition-normal  200ms ease
```

---

## Color Palettes

### Theme A: 冷调极简

| Role | Hex | CSS Variable |
|------|-----|--------------|
| Primary | `#0D9488` | `--color-primary` |
| Accent | `#EA580C` | `--color-accent` |
| Background | `#F1F3F5` | `--color-background` |
| Surface | `#FFFFFF` | `--color-surface` |
| Elevated | `#FFFFFF` | `--color-elevated` |
| Foreground | `#1A1D23` | `--color-foreground` |
| Muted | `#868E96` | `--color-muted` |
| Favorite | `#F59E0B` | `--color-favorite` |
| Success | `#10B981` | `--color-success` |

**设计思路：** 中性暖灰背景 + 白色卡片 + 青碧主色点缀。`--color-background` 与 `--color-surface` 有明显层级差，页面不再「飘白」。

### Theme B: 暖调高效

| Role | Hex | CSS Variable |
|------|-----|--------------|
| Primary | `#6366F1` | `--color-primary` |
| Accent | `#F59E0B` | `--color-accent` |
| Background | `#F0F2F5` | `--color-background` |
| Surface | `#FFFFFF` | `--color-surface` |
| Elevated | `#FFFFFF` | `--color-elevated` |
| Foreground | `#1E2024` | `--color-foreground` |
| Muted | `#7C828E` | `--color-muted` |
| Favorite | `#F59E0B` | `--color-favorite` |
| Success | `#10B981` | `--color-success` |

**设计思路：** 同样三层结构。Indigo 主色不变，背景改为中性灰，减少「空泛感」。

### Theme C: 深色沉浸 (OLED 黑底白字)

| Role | Hex | CSS Variable |
|------|-----|--------------|
| Primary | `#5E6AD2` | `--color-primary` |
| Accent | `#7C6FFF` | `--color-accent` |
| Background | `#000000` | `--color-background` |
| Surface | `#161616` | `--color-surface` |
| Elevated | `#1E1E1E` | `--color-elevated` |
| Foreground | `#FFFFFF` | `--color-foreground` |
| Muted | `#8C8C8C` | `--color-muted` |
| Favorite | `#F59E0B` | `--color-favorite` |
| Success | `#34D399` | `--color-success` |

**设计思路：** 纯黑背景 (`#000000`) 利用 OLED 像素关闭特性，深灰表面层叠构建层次感，纯白文字保证 13.3:1 极致可读性。`--color-surface-hover: #222222` 确保交互反馈可见。

---

## Typography

- **Font:** Inter (300–700 variable weight)
- **Body size:** 14px (--font-size-base)
- **Line-height:** 1.5
- **CSS:** `@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');`

---

## Style Guidelines

**Base Style:** Flat Design with Material Layering

- Three-tier surface hierarchy: `background → surface → elevated`
  - `--color-background`: deepest layer, page base
  - `--color-surface`: cards, list items, in-page content
  - `--color-elevated`: floating elements (modals, dropdowns, context menus)
- Light themes: button hover → `filter: brightness(0.97)`
- Dark theme C: button hover → `filter: brightness(1.2)` (lighten on dark)
- State changes via color/opacity (150ms ease)
- All interactive elements have `cursor: pointer`
- Focus states visible via `--color-ring` outline
- SVG icons only (Lucide React)

---

## Sub-window Theming

| Window | Theme Mechanism |
|--------|----------------|
| Main window | `className="theme-*"` on root `<div>` |
| Toast window | Uses CSS variables, reloads on next show |
| Image viewer | Uses `--color-image-bg` |
| JSON viewer | Uses `--color-background` |

Sub-windows are separate Wails windows. Theme is applied via CSS variables inherited from `public/style.css`. No cross-window event propagation needed — theme change triggers full `location.reload()`.

---

## Anti-Patterns (Do NOT Use)

- ❌ Hardcoded hex/rgba colors in JSX — always use `var(--color-*)`
- ❌ Emojis as icons — use SVG (Lucide React)
- ❌ Missing `cursor:pointer` on clickable elements
- ❌ Layout-shifting hovers (avoid scale transforms that move content)
- ❌ Low contrast text (maintain 4.5:1 minimum)
- ❌ Instant state changes (always use 150-300ms transitions)
- ❌ Invisible focus states
