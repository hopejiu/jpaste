# Design System Master File

> **LOGIC:** When building a specific page, first check `design-system/jpaste/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.

---

**Project:** jPaste
**Generated:** 2026-06-02
**Category:** Desktop Utility (Clipboard Manager)

---

## Global Rules

### Color Palette

| Role | Hex | CSS Variable |
|------|-----|--------------|
| Primary | `#6366F1` | `--color-primary` |
| On Primary | `#FFFFFF` | `--color-on-primary` |
| Secondary | `#4F46E5` | `--color-secondary` |
| Accent/CTA | `#06B6D4` | `--color-accent` |
| Background | `#F8FAFC` | `--color-background` |
| Surface | `#FFFFFF` | `--color-surface` |
| Surface Hover | `#F1F5F9` | `--color-surface-hover` |
| Foreground | `#0F172A` | `--color-foreground` |
| Muted | `#64748B` | `--color-muted` |
| Border | `rgba(0,0,0,0.08)` | `--color-border` |
| Destructive | `#EF4444` | `--color-destructive` |
| Ring | `rgba(99,102,241,0.24)` | `--color-ring` |

**Color Notes:** Modern indigo primary + cyan accent. Light mode. Clean, airy, professional.

### Typography

- **Font Family:** Inter (all weights)
- **Weights:** 300 Light, 400 Regular, 500 Medium, 600 Semibold, 700 Bold
- **Scale:** 12 / 13 / 14 / 16 / 18 / 20 / 24 / 32
- **Mood:** Modern, clean, precision, professional, high-end utility

```css
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
```

### Spacing Variables

| Token | Value | Usage |
|-------|-------|-------|
| `--space-1` | `4px` | Tight icon gaps |
| `--space-2` | `8px` | Inline spacing, small gaps |
| `--space-3` | `12px` | List item inner padding |
| `--space-4` | `16px` | Standard padding |
| `--space-6` | `24px` | Section padding |
| `--space-8` | `32px` | Page margins |

All spacing follows 4px baseline grid.

### Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | `6px` | Tags, badges |
| `--radius-md` | `8px` | Inputs, buttons, list items |
| `--radius-lg` | `12px` | Cards, modals |
| `--radius-full` | `9999px` | Pills, avatars |

### Animation

| Scenario | Duration | Easing |
|----------|----------|--------|
| Window show (fade+scale) | 250ms | `cubic-bezier(0.16, 1, 0.3, 1)` (spring-out) |
| Window hide (fade out) | 150ms | `ease-in` |
| List item hover | 150ms | `ease` |
| Filter results (instant) | 0ms | - |
| Toast enter | 200ms | `ease-out` |
| Toast exit | 150ms | `ease-in` |
| Setting toggle | 150ms | `ease` |
| Button press | 100ms | `ease` |

---

## Component Specs

### Search Input (Spotlight-style)

```css
.search-input {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: 12px 16px;
  font-size: 16px;
  color: var(--color-foreground);
  caret-color: var(--color-primary);
  outline: none;
  transition: border-color 200ms ease, box-shadow 200ms ease;
}
.search-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
}
.search-input::placeholder {
  color: var(--color-muted);
}
```

### Clipboard List Item

```css
.clip-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background 150ms ease;
}
.clip-item:hover {
  background: var(--color-surface-hover);
}
.clip-item:active {
  transform: scale(0.985);
}
.clip-shortcut {
  min-width: 28px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(99, 102, 241, 0.15);
  color: var(--color-primary);
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}
.clip-content {
  flex: 1;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-foreground);
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
}
.clip-time {
  font-size: 12px;
  color: var(--color-muted);
  white-space: nowrap;
}
```

### Buttons

```css
.btn-primary {
  background: var(--color-primary);
  color: white;
  padding: 8px 20px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 500;
  transition: all 150ms ease;
  cursor: pointer;
  border: none;
}
.btn-primary:hover { opacity: 0.9; }
.btn-primary:active { transform: scale(0.97); }

.btn-ghost {
  background: transparent;
  color: var(--color-foreground);
  padding: 8px 12px;
  border-radius: var(--radius-md);
  font-size: 14px;
  transition: background 150ms ease;
  cursor: pointer;
  border: none;
}
.btn-ghost:hover { background: var(--color-surface-hover); }

.btn-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--color-muted);
  cursor: pointer;
  transition: all 150ms ease;
  border: none;
  background: transparent;
}
.btn-icon:hover {
  background: var(--color-surface-hover);
  color: var(--color-foreground);
}
```

### Settings Controls

```css
.settings-group {
  border-bottom: 1px solid var(--color-border);
  padding: var(--space-4) 0;
}
.settings-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3) 0;
}
.settings-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-foreground);
}
.settings-desc {
  font-size: 12px;
  color: var(--color-muted);
  margin-top: 2px;
}

/* Toggle Switch */
.toggle {
  width: 44px;
  height: 24px;
  border-radius: 9999px;
  background: var(--color-muted);
  transition: background 150ms ease;
  cursor: pointer;
  position: relative;
}
.toggle.active {
  background: var(--color-primary);
}
.toggle::after {
  content: '';
  position: absolute;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: white;
  top: 3px;
  left: 3px;
  transition: transform 150ms ease;
}
.toggle.active::after {
  transform: translateX(20px);
}
```

---

## Anti-Patterns (Do NOT Use)

- ❌ Emojis as icons — use Lucide Icons exclusively
- ❌ Missing cursor:pointer on clickable elements
- ❌ Layout-shifting hovers that push content
- ❌ Low contrast text (< 4.5:1)
- ❌ Instant state changes without transitions (150-300ms)
- ❌ Invisible focus states — always show focus ring
- ❌ Animate width/height — use transform/opacity only

---

## Pre-Delivery Checklist

- [ ] No emojis as icons (Lucide Icons, stroke-width 1.5)
- [ ] All hex colors mapped to CSS custom properties
- [ ] Focus states visible on all interactive elements
- [ ] `prefers-reduced-motion` respected (disable animations)
- [ ] Touch targets ≥ 36px for all clickable elements
- [ ] Hover states with smooth transitions (150-300ms)
- [ ] No horizontal overflow
