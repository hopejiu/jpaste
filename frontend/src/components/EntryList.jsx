import { useEffect, useRef } from 'react'
import { ExternalLink, Trash2 } from 'lucide-react'
import EntryItem, { isImageOnly } from './EntryItem'

/**
 * Scrollable entry list with infinite scroll and entry rendering.
 * Context menu state lives in the parent (MainPage).
 *
 * Interface:
 *   entries, focusedIdx, hasMore, loading, detectedMap,
 *   thumbnailsRef, animatingId, styles, search,
 *   onLoadMore, onFocus, onSelect, onImageClick, onActionClick,
 *   onCopy, onPaste, onToggleFavorite, onOpenEditor, onDelete,
 *   onContextMenu, ctxMenu, hideCtxMenu, observeItem,
 *   listRef (forwarded ref for IntersectionObserver root)
 */
export default function EntryList({
  entries, focusedIdx, hasMore, loading, detectedMap,
  thumbnailsRef, animatingId, styles, search,
  onLoadMore, onFocus, onSelect, onImageClick, onActionClick,
  onCopy, onPaste, onToggleFavorite, onOpenEditor, onDelete,
  onContextMenu, ctxMenu, hideCtxMenu, observeItem,
  listRef,
}) {
  // Infinite scroll.
  useEffect(() => {
    const list = listRef.current
    if (!list) return
    const onScroll = () => {
      if (list.scrollHeight - list.scrollTop - list.clientHeight < 120) {
        onLoadMore()
      }
    }
    list.addEventListener('scroll', onScroll, { passive: true })
    return () => list.removeEventListener('scroll', onScroll)
  }, [onLoadMore])

  // Scroll focused item into view.
  useEffect(() => {
    if (focusedIdx >= 0 && listRef.current) {
      const item = listRef.current.children[focusedIdx]
      item?.scrollIntoView({ block: 'nearest' })
    }
  }, [focusedIdx, listRef])

  const handleOpenEditor = (id) => {
    hideCtxMenu()
    onOpenEditor(id)
  }

  const handleDelete = (entryId) => {
    hideCtxMenu()
    onDelete(entryId)
  }

  return (
    <div style={styles.list} ref={listRef}>
      {entries.length === 0 && !loading ? (
        <div style={styles.empty}>
          <p style={styles.emptyTitle}>{search ? '无匹配记录' : '暂无剪贴板历史'}</p>
          <p style={styles.emptyDesc}>
            {search ? '换个关键词试试' : '复制文本即可开始。jPaste 在后台监听剪贴板。'}
          </p>
        </div>
      ) : (
        entries.map((entry, idx) => (
          <EntryItem
            key={entry.id}
            entry={entry}
            idx={idx}
            isFocused={idx === focusedIdx}
            animatingId={animatingId}
            detectedActions={detectedMap[entry.id]}
            thumb={thumbnailsRef.current?.[entry.id]}
            styles={styles}
            onFocus={onFocus}
            onSelect={onSelect}
            onImageClick={onImageClick}
            onActionClick={onActionClick}
            onCopy={onCopy}
            onPaste={onPaste}
            onToggleFavorite={onToggleFavorite}
            onContextMenu={onContextMenu}
            observeItem={observeItem}
          />
        ))
      )}
      {loading && <div style={styles.loading}>加载中...</div>}
      {hasMore && !loading && <div style={styles.loading}>向下滚动加载更多</div>}

      {/* Context Menu */}
      {ctxMenu && (
        <div style={{ ...styles.ctxOverlay, left: ctxMenu.x, top: ctxMenu.y }}>
          {!isImageOnly(ctxMenu.entry) && (
            <div style={styles.ctxItem} onClick={() => handleOpenEditor(ctxMenu.entry.id)}>
              <ExternalLink size={14} />
              <span>在编辑器中打开</span>
            </div>
          )}
          <div style={styles.ctxItemDanger} onClick={() => handleDelete(ctxMenu.entry.id)}>
            <Trash2 size={14} />
            <span>删除</span>
          </div>
        </div>
      )}
    </div>
  )
}
