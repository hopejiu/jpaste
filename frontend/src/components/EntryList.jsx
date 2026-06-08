import { useEffect } from 'react'
import EntryItem from './EntryItem'

export default function EntryList({
  entries, focusedIdx, hasMore, loading, detectedMap,
  thumbnailsRef, animatingId, search,
  onLoadMore, onFocus, onSelect, onImageClick, onActionClick,
  onCopy, onPaste, onToggleFavorite, onOpenEditor, onDelete,
  observeItem,
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



  return (
    <div className="flex-1 overflow-y-auto py-1" ref={listRef}>
      {entries.length === 0 && !loading ? (
        <div className="flex flex-col items-center justify-center h-[200px] p-6 text-center">
          <p className="text-lg font-medium text-foreground mb-2">
            {search ? '无匹配记录' : '暂无剪贴板历史'}
          </p>
          <p className="text-sm text-muted leading-[1.6] max-w-[300px]">
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
            onFocus={onFocus}
            onSelect={onSelect}
            onImageClick={onImageClick}
            onActionClick={onActionClick}
            onCopy={onCopy}
            onPaste={onPaste}
            onToggleFavorite={onToggleFavorite}
            onOpenEditor={onOpenEditor}
            onDelete={onDelete}
            observeItem={observeItem}
          />
        ))
      )}
      {loading && (
        <div className="text-center py-3 text-xs text-muted">加载中...</div>
      )}
      {hasMore && !loading && (
        <div className="text-center py-3 text-xs text-muted">向下滚动加载更多</div>
      )}
    </div>
  )
}
