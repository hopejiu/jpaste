import { useState, useEffect, useRef, useCallback } from 'react'
import { useClipboard, TAGS, TAG_FAVORITE } from '../context/ClipboardContext'
import { useApp } from '../context/AppContext'
import { Service as FileService } from '../../bindings/jpaste/internal/fileop'
import { ImageViewerService } from '../../bindings/jpaste/internal/viewers'
import { Service as FiloService } from '../../bindings/jpaste/internal/filostack'
import { getById } from '../actions'
import { useActionDetection } from '../hooks/useActionDetection'
import { useKeyboardNavigation } from '../hooks/useKeyboardNavigation'
import { useImageThumbnail } from '../hooks/useImageThumbnail'
import { Copy, CheckCircle, HelpCircle } from 'lucide-react'
import SearchBar from '../components/SearchBar'
import TitleBar from '../components/TitleBar'
import TagTabs from '../components/TagTabs'
import EntryList from '../components/EntryList'
import ActionModal from '../components/ActionModal'
import Modal from '../components/Modal'
import { log } from '../logger'
import ShortcutHelpModal from '../components/ShortcutHelpModal'

export default function MainPage() {
  const {
    entries, search, setSearch,
    activeTag, setActiveTag,
    hasMore, loading, loadMore, isRegex, toggleRegex,
    useEntry, deleteEntry, toggleFavorite,
  } = useClipboard()

  const { settings, setPasteOrder } = useApp()

  const [focusedIdx, setFocusedIdx] = useState(-1)
  const [selectedActionIdx, setSelectedActionIdx] = useState(-1)
  const [modal, setModal] = useState(null)
  const [animatingId, setAnimatingId] = useState(null)
  const [errorAlert, setErrorAlert] = useState(null) // { title, message }
  const [copyAllDone, setCopyAllDone] = useState(false)
  const [showShortcutHelp, setShowShortcutHelp] = useState(false)

  // Esc to close error alert.
  useEffect(() => {
    if (!errorAlert) return
    const handler = (e) => { if (e.key === 'Escape') setErrorAlert(null) }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [errorAlert])

  // Exit action mode when focus moves to a different entry.
  useEffect(() => { setSelectedActionIdx(-1) }, [focusedIdx])

  const inputRef = useRef(null)
  const listRef = useRef(null)
  const hiddenTimeRef = useRef(null) // 记录窗口隐藏的时间戳

  // Lazy-loaded image thumbnails for list items.
  const { thumbnailsRef } = useImageThumbnail(listRef, entries)

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
      setErrorAlert({ title: '无法打开', message: '文件可能已被删除或移动' })
    }
  }, [])

  const handleKeyDown = useKeyboardNavigation({
    entries, focusedIdx, settings, useEntry, setSearch, setFocusedIdx, inputRef, modal, closeModal,
    activeTag, tags: TAGS, onTagChange: handleTagChange, search, listRef,
    deleteEntry, toggleFavorite, onOpenEditor: handleOpenEditor,
    selectedActionIdx, setSelectedActionIdx,
  })

  // Wrap handleKeyDown to intercept Esc when a modal overlay is open,
  // preventing the main handler from hiding the window.
  const wrappedHandleKeyDown = useCallback((e) => {
    if (e.key === 'Escape' && (showShortcutHelp || errorAlert)) return
    handleKeyDown(e)
  }, [handleKeyDown, showShortcutHelp, errorAlert])

  // Auto-focus search + scroll to top on mount.
  useEffect(() => {
    listRef.current?.scrollTo(0, 0)
    inputRef.current?.focus()
  }, [])

  // Re-focus search + scroll to top on window shown.
  useEffect(() => {
    const handleFocus = () => {
      // 自动清理搜索条件
      if (settings.auto_clear_search) {
        const now = Date.now()
        const hiddenTime = hiddenTimeRef.current
        const threshold = settings.auto_clear_seconds * 1000
        
        // 如果 hiddenTime 为 null（首次显示）或者距离隐藏时间超过阈值
        if (hiddenTime === null || (now - hiddenTime) >= threshold) {
          setSearch('')
          setActiveTag(TAG_ALL)
          if (isRegex) toggleRegex(false)
        }
      }
      
      hiddenTimeRef.current = null
      listRef.current?.scrollTo(0, 0)
      inputRef.current?.focus()
    }
    
    const handleBlur = () => {
      hiddenTimeRef.current = Date.now()
    }
    
    window.addEventListener('focus', handleFocus)
    window.addEventListener('blur', handleBlur)
    return () => {
      window.removeEventListener('focus', handleFocus)
      window.removeEventListener('blur', handleBlur)
    }
  }, [settings.auto_clear_search, settings.auto_clear_seconds, setSearch, setActiveTag, isRegex, toggleRegex])

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

  const handleActionClick = useCallback(async (actionId, entry) => {
    const action = getById(actionId)
    if (action?.handler) {
      try {
        await action.handler(entry.content, entry.id)
      } catch (err) {
        log.error('MainPage', 'Action failed:', actionId, err)
        if (err?.message) {
          setErrorAlert({ title: '操作失败', message: err.message })
        }
      }
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

  const handleCopyAll = useCallback(() => {
    const textContents = entries
      .filter(e => e.content && e.content.trim())
      .map(e => e.content)
      .join('\n')
    if (!textContents) return
    navigator.clipboard.writeText(textContents)
      .then(() => {
        setCopyAllDone(true)
        setTimeout(() => setCopyAllDone(false), 1500)
      })
      .catch(err => log.error('MainPage', 'Failed to copy all:', err))
  }, [entries])

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

  const modeLabels = { stack: '栈', queue: '队列' }

  return (
    <div className="flex flex-col h-screen outline-none" onKeyDown={wrappedHandleKeyDown} tabIndex={0}>
      <TitleBar />
      {/* Header with Search */}
      <div className="flex items-center px-4 py-3 gap-2 border-b border-border flex-shrink-0 bg-surface">
        <SearchBar
          search={search}
          onSearchChange={handleSearchChange}
          inputRef={inputRef}
          isRegex={isRegex}
          onToggleRegex={toggleRegex}
        />
      </div>

      {/* Tag Tabs */}
      <TagTabs
        tags={TAGS}
        activeTag={activeTag}
        onTagChange={handleTagChange}
      />

      {/* Copy All button (only in favorites view) */}
      {activeTag === TAG_FAVORITE && entries.length > 0 && (
        <div className="flex items-center justify-center px-4 py-1.5 border-b border-border bg-background">
          <button
            onClick={handleCopyAll}
            className="flex items-center gap-1.5 px-3 py-1 text-xs font-medium text-primary rounded-md border border-border cursor-pointer font-inherit transition-all duration-fast hover:bg-primary-alpha-06"
          >
            {copyAllDone ? (
              <>
                <CheckCircle size={14} />
                已复制
              </>
            ) : (
              <>
                <Copy size={14} />
                复制所有
              </>
            )}
          </button>
        </div>
      )}

      {/* Entry List */}
      <EntryList
        entries={entries}
        focusedIdx={focusedIdx}
        hasMore={hasMore}
        loading={loading}
        detectedMap={detectedMap}
        thumbnailsRef={thumbnailsRef}
        animatingId={animatingId}
        search={search}
        listRef={listRef}
        selectedActionIdx={selectedActionIdx}
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
        observeItem={observeItem}
      />

      {/* Footer */}
      <div className="border-t border-border px-4 py-2 flex items-center justify-between flex-shrink-0 gap-2 bg-background">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setShowShortcutHelp(true)}
            className="w-5 h-5 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded-sm transition-all duration-fast hover:text-foreground hover:bg-surface-hover"
            title="快捷键说明"
          >
            <HelpCircle size={14} />
          </button>
          <span className="text-xs text-muted">Ctrl+L搜索 · E编辑 · C复制 · Del删除 · Space收藏 · Esc隐藏</span>
        </div>
        <div
          ref={popupRef}
          className="relative flex items-center gap-0.5 flex-shrink-0"
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
                className="text-[11px] px-1.5 py-px rounded border border-border cursor-pointer font-inherit transition-all duration-fast"
                style={{
                  background: active ? 'var(--color-primary-alpha-12)' : 'transparent',
                  color: active ? 'var(--color-primary)' : 'var(--color-muted)',
                }}
              >
                {label}
              </button>
            )
          })}
          {/* Stack/Queue hover popup */}
          {showPopup && isNonNormal && (
            <div
              className="absolute bottom-full right-0 mb-1.5 min-w-[200px] max-w-[280px] max-h-[200px] overflow-y-auto bg-elevated border border-border rounded-md p-2 z-[2000] animate-[slideDown_120ms_ease-out]"
              style={{ boxShadow: '0 4px 16px rgba(0,0,0,0.12)' }}
            >
              <div className="px-3 pb-1.5 text-[11px] font-semibold text-primary border-b border-border whitespace-nowrap">
                {modeLabels[settings.paste_order]} · {stackItems.length} 项
              </div>
              {stackItems.length === 0 ? (
                <div className="py-3 px-4 text-xs text-muted text-center">暂无内容</div>
              ) : (
                stackItems.map((item, idx) => {
                  const isNext = settings.paste_order === 'stack' ? idx === stackItems.length - 1 : idx === 0
                  return (
                    <div
                      key={idx}
                      className={`flex items-center gap-1.5 px-3 py-1 text-xs leading-[1.4] ${
                        isNext ? 'text-foreground font-medium' : 'text-muted font-normal'
                      }`}
                    >
                      <span className="flex-shrink-0 text-[10px] w-3.5 text-center" style={{ color: isNext ? 'var(--color-primary)' : 'transparent' }}>
                        {isNext ? '▶' : ''}
                      </span>
                      <span className="overflow-hidden text-ellipsis whitespace-nowrap">{item}</span>
                    </div>
                  )
                })
              )}
              <div className="px-3 pt-1.5 pb-1 text-[10px] text-muted border-t border-border leading-[1.4]">
                {settings.paste_order === 'stack'
                  ? '▶ 下一个将粘贴（后进先出）· 复制图片/文件将自动退出栈模式'
                  : '▶ 下一个将粘贴（先进先出）· 复制图片/文件将自动退出队列模式'}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Shortcut Help Modal */}
      <ShortcutHelpModal open={showShortcutHelp} onClose={() => setShowShortcutHelp(false)} />

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

      {/* Error Alert Modal */}
      <Modal open={!!errorAlert} onClose={() => setErrorAlert(null)} title={errorAlert?.title || ''} size="sm">
        <p className="m-0 mb-5 text-sm text-muted leading-[1.5]">{errorAlert?.message}</p>
        <button
          className="w-full py-2 rounded-md border-none bg-primary text-white text-sm font-medium cursor-pointer font-inherit transition-opacity duration-fast hover:opacity-90"
          onClick={() => setErrorAlert(null)}
        >确定</button>
      </Modal>
    </div>
  )
}
