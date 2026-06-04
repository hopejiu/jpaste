import { useState, useEffect, useRef, useCallback } from 'react'
import { useClipboard, TAGS } from '../context/ClipboardContext'
import { useApp } from '../context/AppContext'
import { Service as FileService } from '../../bindings/jpaste/internal/fileop'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { getById } from '../actions'
import { useActionDetection } from '../hooks/useActionDetection'
import { useKeyboardNavigation } from '../hooks/useKeyboardNavigation'
import { useContextMenu } from '../hooks/useContextMenu'
import SearchBar from '../components/SearchBar'
import TagTabs from '../components/TagTabs'
import EntryList from '../components/EntryList'
import ImagePreview from '../components/ImagePreview'
import ActionModal from '../components/ActionModal'
import { styles } from './MainPage.styles'

export default function MainPage() {
  const {
    entries, search, setSearch,
    activeTag, setActiveTag,
    hasMore, loading, loadMore,
    useEntry, deleteEntry, toggleFavorite,
  } = useClipboard()

  const { settings, syncStatus } = useApp()

  const [focusedIdx, setFocusedIdx] = useState(-1)
  const [modal, setModal] = useState(null)
  const [imagePreview, setImagePreview] = useState(null)
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

  const handleKeyDown = useKeyboardNavigation({
    entries, focusedIdx, settings, useEntry, setSearch, setFocusedIdx, inputRef, modal, closeModal,
    activeTag, tags: TAGS, onTagChange: handleTagChange, search, listRef,
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

  const handleImageClick = useCallback(async (entry) => {
    setImagePreview({ url: '', loading: true })
    try {
      const url = await HistoryService.GetImageDataURL(entry.id)
      setImagePreview({ url, loading: false })
    } catch {
      setImagePreview(null)
    }
  }, [])

  const handleActionClick = useCallback((actionId, entry) => {
    const action = getById(actionId)
    if (action?.handler) {
      action.handler(entry.content)
      return
    }
    setModal({ actionId, entry })
  }, [])

  const handleOpenEditor = useCallback(async (id) => {
    try {
      await FileService.OpenInEditor(id)
    } catch (err) {
      console.error('Failed to open in editor:', err)
    }
  }, [])

  const handleCopy = useCallback((id) => {
    useEntry(id, 'copy')
  }, [useEntry])

  const handlePaste = useCallback((id) => {
    useEntry(id, 'paste')
  }, [useEntry])

  const modalAction = modal ? getById(modal.actionId) : null

  return (
    <div style={styles.container} onKeyDown={handleKeyDown} tabIndex={0}>
      {/* Header */}
      <div style={styles.header}>
        <SearchBar
          search={search}
          onSearchChange={handleSearchChange}
          syncStatus={syncStatus.status}
          styles={{
            searchBox: styles.searchBox,
            searchIcon: styles.searchIcon,
            searchInput: styles.searchInput,
            clearBtn: styles.clearBtn,
            settingsBtn: styles.settingsBtn,
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
        <span style={styles.footerText}>Alt+V 切换 · Ctrl+1-9 选择 · ← → 标签 · Page↑↓ 翻页 · Esc 隐藏 · 右键更多操作</span>
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

      {/* Image Preview */}
      <ImagePreview
        imagePreview={imagePreview}
        onClose={() => setImagePreview(null)}
      />
    </div>
  )
}
