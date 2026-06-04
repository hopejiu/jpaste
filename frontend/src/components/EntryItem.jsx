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

/**
 * Single clipboard entry row.
 *
 * Interface:
 *   entry: ClipboardEntry
 *   idx: number
 *   isFocused: boolean
 *   animatingId: number | null
 *   detectedActions: string[]
 *   thumb: { url?: string, loading?: boolean } | undefined
 *   styles: object
 *   onFocus: (idx: number) => void
 *   onSelect: (entry) => void
 *   onImageClick: (entry) => void
 *   onActionClick: (actionId, entry) => void
 *   onCopy: (id) => void
 *   onPaste: (id) => void
 *   onToggleFavorite: (id, value) => void
 *   onContextMenu: (e, entry) => void
 *   observeItem: (el, id, content) => void
 */
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
      style={{
        ...styles.item,
        ...(isFocused ? styles.itemFocused : {}),
        ...(imgOnly ? styles.itemImage : {}),
        position: 'relative',
      }}
      onMouseEnter={() => onFocus(idx)}
      onClick={() => {
        if (imgOnly) { onImageClick(entry); return }
        onSelect(entry)
      }}
      onContextMenu={(e) => onContextMenu(e, entry)}
    >
      {shortcut && <div style={styles.shortcut}>{idx + 1}</div>}

      <div style={styles.itemContent}>
        {imgOnly ? (
          thumb?.url ? (
            <div
              style={styles.thumbWrapper}
              onMouseEnter={(e) => {
                const overlay = e.currentTarget.querySelector('[data-overlay]')
                if (overlay) overlay.style.opacity = '1'
              }}
              onMouseLeave={(e) => {
                const overlay = e.currentTarget.querySelector('[data-overlay]')
                if (overlay) overlay.style.opacity = '0'
              }}
            >
              <img src={thumb.url} alt="" style={styles.thumbImg} />
              <div data-overlay style={styles.thumbOverlay}>
                <span style={styles.thumbOverlayText}>
                  <ZoomIn size={16} /> 点击放大
                </span>
              </div>
            </div>
          ) : (
            <div style={styles.itemImagePlaceholder}>
              <Image size={20} />
              <span style={styles.itemImageLabel}>{thumb?.loading ? '加载中...' : '图片'}</span>
            </div>
          )
        ) : (
          <div style={styles.itemContentRow}>
            <div style={styles.itemText}>
              {previewContent(entry.content)}
              {isFile && <span style={styles.fileBadge} title="文件"><File size={12} /> 文件</span>}
              {hasImg && !isFile && <Image size={14} style={{ marginLeft: 6, opacity: 0.4, verticalAlign: 'middle' }} />}
            </div>
            {hasImg && thumb?.url && (
              <img src={thumb.url} alt="" style={styles.thumbInline} />
            )}
          </div>
        )}
        <div style={styles.itemMeta}>
          <span style={styles.itemTime}>
            <span style={styles.itemRel}>{time.rel}</span>
            <span style={styles.itemAbs}>{time.abs}</span>
          </span>
          {entry.source_exe && (
            <span style={styles.sourceApp} title={`${entry.source_exe} — ${entry.source_title || ''}`}>
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
              style={styles.copyTextBtn}
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
              style={{ ...styles.favBtn, ...(entry.is_favorite ? styles.favBtnActive : {}) }}
              onClick={(e) => { e.stopPropagation(); onToggleFavorite(entry.id, !entry.is_favorite) }}
              title={entry.is_favorite ? '取消收藏' : '收藏'}
            >
              <Star size={14} fill={entry.is_favorite ? '#F59E0B' : 'none'} />
            </button>
          )}
          <button
            style={styles.actionBtn}
            onClick={(e) => { e.stopPropagation(); onCopy(entry.id) }}
            title="复制"
          >
            <Copy size={14} />
          </button>
          <button
            style={styles.actionBtn}
            onClick={(e) => { e.stopPropagation(); onPaste(entry.id) }}
            title="粘贴"
          >
            <ClipboardPaste size={14} />
          </button>
        </div>
      </div>

      {animatingId === entry.id && (
        <div style={styles.checkmark}>
          <CheckCircle size={20} fill="#10B981" color="#fff" />
        </div>
      )}
    </div>
  )
}

export { isImageOnly, isImageEntry }
