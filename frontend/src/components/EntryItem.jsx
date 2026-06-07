import { Copy, ClipboardPaste, Star, Image, ZoomIn, CheckCircle, File, FileText } from 'lucide-react'
import { formatTime, previewContent } from '../utils/format'
import ActionButtons from './ActionButtons'

const CF_DIB = 8
const CF_DIBV5 = 17
const CF_HDROP = 15

const isImageEntry = (entry) => entry.formats?.some(f => f.format_type === CF_DIB || f.format_type === CF_DIBV5)
const isImageOnly = (entry) => entry.formats?.length > 0 && entry.formats.every(f => f.format_type === CF_DIB || f.format_type === CF_DIBV5)
const isFileEntry = (entry) => entry.formats?.some(f => f.format_type === CF_HDROP)
const extractAppName = (exe) => {
  if (!exe) return ''
  const parts = exe.split('\\')
  return parts[parts.length - 1].replace('.exe', '')
}

export default function EntryItem({
  entry, idx, isFocused, animatingId,
  detectedActions, thumb, styles,
  onFocus, onSelect, onImageClick, onActionClick,
  onCopy, onPaste, onToggleFavorite, onContextMenu, observeItem,
}) {
  const shortcut = idx < 9 ? `Ctrl+${idx + 1}` : null
  const time = formatTime(entry.updated_at)
  const hasImg = isImageEntry(entry)
  const imgOnly = isImageOnly(entry)
  const isFile = isFileEntry(entry)

  return (
    <div
      key={entry.id}
      ref={(el) => observeItem(el, entry.id, entry.content)}
      data-thumb-id={hasImg ? entry.id : undefined}
      className={`
        flex gap-2.5 px-4 py-2.5 cursor-pointer
        transition-[background] duration-fast relative
        border-b border-border
        ${isFocused ? 'bg-surface-hover' : ''}
        ${imgOnly ? 'cursor-pointer' : ''}
      `}
      onMouseEnter={() => onFocus(idx)}
      onClick={() => {
        if (imgOnly) { onImageClick(entry); return }
        onSelect(entry)
      }}
      onContextMenu={(e) => onContextMenu(e, entry)}
    >
      {shortcut && (
        <div
          className="min-w-[24px] h-[22px] flex items-center justify-center text-xs font-semibold mt-0.5 flex-shrink-0 rounded"
          style={{
            background: 'var(--color-primary-alpha-15)',
            color: 'var(--color-primary)',
          }}
        >
          {idx + 1}
        </div>
      )}

      <div className="flex-1 min-w-0">
        {imgOnly ? (
          thumb?.url ? (
            <div
              className="relative inline-block max-w-full"
              onMouseEnter={(e) => {
                const overlay = e.currentTarget.querySelector('[data-overlay]')
                if (overlay) overlay.style.opacity = '1'
              }}
              onMouseLeave={(e) => {
                const overlay = e.currentTarget.querySelector('[data-overlay]')
                if (overlay) overlay.style.opacity = '0'
              }}
            >
              <img
                src={thumb.url}
                alt=""
                className="max-w-full max-h-[160px] rounded-sm object-contain block"
                style={{ background: 'var(--color-primary-alpha-04)' }}
              />
              <div
                data-overlay
                className="absolute inset-0 flex items-center justify-center rounded-sm pointer-events-none"
                style={{ background: 'rgba(0,0,0,0.35)', opacity: 0, transition: 'opacity var(--transition-fast)' }}
              >
                <span className="flex items-center gap-1 text-white text-sm font-semibold">
                  <ZoomIn size={16} /> 点击放大
                </span>
              </div>
            </div>
          ) : (
            <div
              className="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-muted"
              style={{ background: 'var(--color-primary-alpha-06)' }}
            >
              <Image size={20} />
              <span className="font-medium">{thumb?.loading ? '加载中...' : '图片'}</span>
            </div>
          )
        ) : (
          <div className="flex gap-2 items-start">
            <div className="text-sm leading-[1.55] text-foreground whitespace-pre-wrap break-words overflow-hidden flex-1">
              {previewContent(entry.content)}
              {isFile && (
                <span
                  className="inline-flex items-center gap-0.5 ml-1.5 px-1.5 py-px rounded-full text-[10px] font-semibold align-middle whitespace-nowrap"
                  style={{ background: 'var(--color-badge-file-bg)', color: 'var(--color-badge-file)' }}
                  title="文件"
                >
                  <File size={12} /> 文件
                </span>
              )}
              {hasImg && !isFile && (
                <Image size={14} className="ml-1.5 opacity-40 align-middle inline" />
              )}
            </div>
            {hasImg && thumb?.url && (
              <img
                src={thumb.url}
                alt=""
                className="w-9 h-9 rounded object-cover flex-shrink-0 ml-2"
              />
            )}
          </div>
        )}
        <div className="flex items-center gap-1 mt-1.5">
          <span className="flex-1 flex gap-2 items-baseline">
            <span className="text-xs text-muted">{time.rel}</span>
            <span className="text-[11px] text-muted opacity-65">{time.abs}</span>
          </span>
          {entry.source_exe && (
            <span
              className="text-[11px] text-muted opacity-80 ml-2 overflow-hidden text-ellipsis whitespace-nowrap max-w-[120px]"
              title={`${entry.source_exe} — ${entry.source_title || ''}`}
            >
              {extractAppName(entry.source_exe)}{entry.source_title ? ` · ${entry.source_title.split(' - ').pop()}` : ''}
            </span>
          )}
          {!imgOnly && (
            <ActionButtons
              actionIds={detectedActions}
              onClick={(actionId) => onActionClick(actionId, entry)}
            />
          )}
          {isFile && (
            <button
              className="w-7 h-7 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded transition-all duration-fast"
              onClick={(e) => {
                e.stopPropagation()
                navigator.clipboard.writeText(entry.content)
              }}
              title="复制路径文本"
            >
              <FileText size={14} />
            </button>
          )}
          {!imgOnly && (
            <button
              className={`w-7 h-7 flex items-center justify-center border-none bg-transparent cursor-pointer rounded transition-all duration-fast flex-shrink-0 ${
                entry.is_favorite ? 'text-favorite' : 'text-muted'
              }`}
              onClick={(e) => { e.stopPropagation(); onToggleFavorite(entry.id, !entry.is_favorite) }}
              title={entry.is_favorite ? '取消收藏' : '收藏'}
            >
              <Star size={14} fill={entry.is_favorite ? 'var(--color-favorite)' : 'none'} />
            </button>
          )}
          <button
            className="w-7 h-7 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded transition-all duration-fast"
            onClick={(e) => { e.stopPropagation(); onCopy(entry.id) }}
            title="复制"
          >
            <Copy size={14} />
          </button>
          <button
            className="w-7 h-7 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded transition-all duration-fast"
            onClick={(e) => { e.stopPropagation(); onPaste(entry.id) }}
            title="粘贴"
          >
            <ClipboardPaste size={14} />
          </button>
        </div>
      </div>

      {animatingId === entry.id && (
        <div className="absolute top-1/2 right-3 -translate-y-1/2 pointer-events-none animate-[fadeScaleIn_250ms_ease-out]">
          <CheckCircle size={20} fill="var(--color-success)" color="#fff" />
        </div>
      )}
    </div>
  )
}

export { isImageOnly, isImageEntry }
