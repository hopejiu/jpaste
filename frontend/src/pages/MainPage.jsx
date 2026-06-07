import { useState, useEffect, useRef, useCallback } from 'react'
import { useClipboard, TAGS } from '../context/ClipboardContext'
import { useApp } from '../context/AppContext'
import { Service as FileService } from '../../bindings/jpaste/internal/fileop'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { Service as ImageViewerService } from '../../bindings/jpaste/internal/imageviewer'
import { Service as FiloService } from '../../bindings/jpaste/internal/filostack'
import { getById } from '../actions'
import { useActionDetection } from '../hooks/useActionDetection'
import { useKeyboardNavigation } from '../hooks/useKeyboardNavigation'
import { useContextMenu } from '../hooks/useContextMenu'
import SearchBar from '../components/SearchBar'
import TitleBar from '../components/TitleBar'
import TagTabs from '../components/TagTabs'
import EntryList from '../components/EntryList'
import ActionModal from '../components/ActionModal'
import { styles } from './MainPage.styles'
import { log } from '../logger'

export default function MainPage() {
  const {
    entries, search, setSearch,
    activeTag, setActiveTag,
    hasMore, loading, loadMore, isRegex, toggleRegex,
    useEntry, deleteEntry, toggleFavorite,
  } = useClipboard()

  const { settings, setPasteOrder } = useApp()

  const [focusedIdx, setFocusedIdx] = useState(-1)
  const [modal, setModal] = useState(null)
  const [animatingId, setAnimatingId] = useState(null)

  const inputRef = useRef(null)
  const listRef = useRef(null)

  // Lazy-loaded image thumbnails for list items.
  const thumbnailsRef = useRef({})
  const [, setThumbTick] = useState(0)
  const thumbObserverRef = useRef(null)

  const { ctxMenu, showCtxMenu, hideCtxMenu } = useContextMenu()
  const closeModal = useCallback(() => setModal(null), [])

  const { detectedMap, observeItem } = useActionDetection(entries, settings.action_config, listRef)

  // Define handleTagChange before handleKeyDown (referenced by it).
  const handleTagChange = useCallback((tag) => {
    setActiveTag(tag)
    setFocusedIdx(-1)
  }, [setActiveTag])

  const handleOpenEditor = useCallback(async (id) => {
    try {
      await FileService.OpenInEditor(id)
    } catch (err) {
      log.error('MainPage', 'Failed to open in editor:', err)
    }
  }, [])

  const handleKeyDown = useKeyboardNavigation({
    entries, focusedIdx, settings, useEntry, setSearch, setFocusedIdx, inputRef, modal, closeModal,
    activeTag, tags: TAGS, onTagChange: handleTagChange, search, listRef,
    deleteEntry, toggleFavorite, onOpenEditor: handleOpenEditor,
  })

  // Auto-focus search + scroll to top on mount.
  useEffect(() => {
    listRef.current?.scrollTo(0, 0)
    inputRef.current?.focus()
  }, [])

  // Re-focus search + scroll to top on window shown.
  useEffect(() => {
    const handler = () => {
      listRef.current?.scrollTo(0, 0)
      inputRef.current?.focus()
    }
    window.addEventListener('focus', handler)
    return () => window.removeEventListener('focus', handler)
  }, [])

  // Lazy-load image thumbnails when they scroll into view.
  useEffect(() => {
    const loadThumb = async (entryId) => {
      const cur = thumbnailsRef.current
      if (cur[entryId]?.url || cur[entryId]?.loading) return
      cur[entryId] = { url: '', loading: true, error: false }
      setThumbTick(t => t + 1)
      try {
        const url = await HistoryService.GetImageDataURL(entryId)
        cur[entryId] = { url, loading: false, error: false }
      } catch {
        cur[entryId] = { url: '', loading: false, error: true }
      }
      setThumbTick(t => t + 1)
    }

    thumbObserverRef.current = new IntersectionObserver((observed) => {
      for (const obs of observed) {
        if (obs.isIntersecting) {
          const id = parseInt(obs.target.dataset.thumbId, 10)
          if (id) loadThumb(id)
        }
      }
    }, { root: listRef.current, rootMargin: '200px' })

    return () => thumbObserverRef.current?.disconnect()
  }, [])

  // Register image entries with thumbnail observer when entries change.
  useEffect(() => {
    const observer = thumbObserverRef.current
    if (!observer || !listRef.current) return
    const items = listRef.current.querySelectorAll('[data-thumb-id]')
    for (const item of items) observer.observe(item)
  }, [entries])

  // --- Handlers ---

  const handleSearchChange = useCallback((term) => {
    setSearch(term)
    setFocusedIdx(-1)
  }, [setSearch])

  const handleSelect = useCallback((entry) => {
    setAnimatingId(entry.id)
    setTimeout(() => setAnimatingId(null), 600)
    useEntry(entry.id, settings.default_action)
  }, [useEntry, settings.default_action])

  const handleImageClick = useCallback((entry) => {
    ImageViewerService.OpenImageViewer(entry.id, activeTag, search)
  }, [activeTag, search])

  const handleActionClick = useCallback((actionId, entry) => {
    const action = getById(actionId)
    if (action?.handler) {
      action.handler(entry.content, entry.id)
      return
    }
    setModal({ actionId, entry })
  }, [])

  const handleCopy = useCallback((id) => {
    useEntry(id, 'copy')
  }, [useEntry])

  const handlePaste = useCallback((id) => {
    useEntry(id, 'paste')
  }, [useEntry])

  // --- Stack/Queue popup ---
  const [stackItems, setStackItems] = useState([])
  const [showPopup, setShowPopup] = useState(false)
  const popupRef = useRef(null)

  const fetchItems = useCallback(async () => {
    try {
      const items = await FiloService.GetItems()
      setStackItems(items || [])
    } catch { setStackItems([]) }
  }, [])

  const handlePopupEnter = useCallback(() => {
    setShowPopup(true)
    fetchItems()
  }, [fetchItems])

  const handlePopupLeave = useCallback(() => {
    setShowPopup(false)
  }, [])

  const isNonNormal = settings.paste_order === 'stack' || settings.paste_order === 'queue'

  const modalAction = modal ? getById(modal.actionId) : null

  const popupStyles = {
    position: 'absolute',
    bottom: '100%',
    right: '0',
    marginBottom: '6px',
    minWidth: '200px',
    maxWidth: '280px',
    maxHeight: '200px',
    overflowY: 'auto',
    background: 'var(--color-elevated)',
    border: '1px solid var(--color-border)',
    borderRadius: 'var(--radius-md)',
    boxShadow: '0 4px 16px rgba(0,0,0,0.12)',
    padding: '8px 0',
    zIndex: 2000,
    animation: 'slideDown 120ms ease-out',
  }

  const modeLabels = { stack: '栈', queue: '队列' }

  return (
    <div style={styles.container} onKeyDown={handleKeyDown} tabIndex={0}>
      <TitleBar />
      {/* Header */}
      <div style={styles.header}>
        <SearchBar
          search={search}
          onSearchChange={handleSearchChange}
          inputRef={inputRef}
          isRegex={isRegex}
          onToggleRegex={toggleRegex}
          styles={{
            searchBox: styles.searchBox,
            searchIcon: styles.searchIcon,
            searchInput: styles.searchInput,
            clearBtn: styles.clearBtn,
            regexBtn: styles.regexBtn,
            regexBtnActive: styles.regexBtnActive,
          }}
        />
      </div>


      {/* Tag Tabs */}
      <TagTabs
        tags={TAGS}
        activeTag={activeTag}
        onTagChange={handleTagChange}
        styles={{
          tabBar: styles.tabBar,
          tab: styles.tab,
          tabActive: styles.tabActive,
        }}
      />

      {/* Entry List */}
      <EntryList
        entries={entries}
        focusedIdx={focusedIdx}
        hasMore={hasMore}
        loading={loading}
        detectedMap={detectedMap}
        thumbnailsRef={thumbnailsRef}
        animatingId={animatingId}
        styles={styles}
        search={search}
        listRef={listRef}
        onLoadMore={loadMore}
        onFocus={setFocusedIdx}
        onSelect={handleSelect}
        onImageClick={handleImageClick}
        onActionClick={handleActionClick}
        onCopy={handleCopy}
        onPaste={handlePaste}
        onToggleFavorite={toggleFavorite}
        onOpenEditor={handleOpenEditor}
        onDelete={deleteEntry}
        onContextMenu={showCtxMenu}
        ctxMenu={ctxMenu}
        hideCtxMenu={hideCtxMenu}
        observeItem={observeItem}
      />

      {/* Footer */}
      <div style={styles.footer}>
        <span style={styles.footerText}>Ctrl+L搜索 · Ctrl+E编辑 · Del删除 · Space收藏 · Esc隐藏</span>
        <div
          ref={popupRef}
          style={{ position: 'relative', display: 'flex', alignItems: 'center', gap: '2px', flexShrink: 0 }}
          onMouseEnter={handlePopupEnter}
          onMouseLeave={handlePopupLeave}
        >
          {['normal', 'stack', 'queue'].map(mode => {
            const active = (settings.paste_order || 'normal') === mode
            const label = mode === 'normal' ? '正常' : mode === 'stack' ? '栈' : '队列'
            const title = mode === 'normal' ? '正常粘贴'
              : mode === 'stack' ? '栈模式：Ctrl+V 倒序粘贴（后进先出）'
              : '队列模式：Ctrl+V 顺序粘贴（先进先出）'
            return (
              <button
                key={mode}
                onClick={() => setPasteOrder(mode)}
                title={title}
                style={{
                  fontSize: '11px', padding: '1px 6px', borderRadius: '4px',
                  border: '1px solid var(--color-border)',
                  background: active ? 'var(--color-primary-alpha-12)' : 'transparent',
                  color: active ? 'var(--color-primary)' : 'var(--color-muted)',
                  cursor: 'pointer', fontFamily: 'inherit',
                  transition: 'all var(--transition-fast)',
                }}
              >
                {label}
              </button>
            )
          })}
          {/* Stack/Queue hover popup */}
          {showPopup && isNonNormal && (
            <div style={popupStyles}>
              <div style={{
                padding: '4px 12px 6px', fontSize: '11px', fontWeight: 600,
                color: 'var(--color-primary)', borderBottom: '1px solid var(--color-border)',
                whiteSpace: 'nowrap',
              }}>
                {modeLabels[settings.paste_order]} · {stackItems.length} 项
              </div>
              {stackItems.length === 0 ? (
                <div style={{ padding: '12px 16px', fontSize: '12px', color: 'var(--color-muted)', textAlign: 'center' }}>
                  暂无内容
                </div>
              ) : (
                stackItems.map((item, idx) => {
                  const isNext = settings.paste_order === 'stack' ? idx === stackItems.length - 1 : idx === 0
                  return (
                    <div key={idx} style={{
                      display: 'flex', alignItems: 'center', gap: '6px',
                      padding: '5px 12px', fontSize: '12px',
                      color: isNext ? 'var(--color-foreground)' : 'var(--color-muted)',
                      fontWeight: isNext ? 500 : 400,
                      lineHeight: 1.4,
                    }}>
                      <span style={{
                        flexShrink: 0, fontSize: '10px',
                        color: isNext ? 'var(--color-primary)' : 'transparent',
                        width: '14px', textAlign: 'center',
                      }}>
                        {isNext ? '▶' : ''}
                      </span>
                      <span style={{
                        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                      }}>
                        {item}
                      </span>
                    </div>
                  )
                })
              )}
              {/* Explanation */}
              <div style={{
                padding: '6px 12px 4px', fontSize: '10px', color: 'var(--color-muted)',
                borderTop: '1px solid var(--color-border)', lineHeight: 1.4,
              }}>
                {settings.paste_order === 'stack'
                  ? '▶ 下一个将粘贴（后进先出）· 复制图片/文件将自动退出栈模式'
                  : '▶ 下一个将粘贴（先进先出）· 复制图片/文件将自动退出队列模式'}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Action Modal */}
      <ActionModal
        open={!!modal}
        onClose={closeModal}
        title={modalAction?.label || ''}
      >
        {modalAction?.Component && (
          <modalAction.Component
            content={modal.entry.content}
            entryId={modal.entry.id}
            onClose={closeModal}
          />
        )}
      </ActionModal>

    </div>
  )
}
